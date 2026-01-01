package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/alexpls/untils/internal/auth"
	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/db/sqlc"
	"github.com/alexpls/untils/internal/email"
	"github.com/alexpls/untils/internal/llm"
	"github.com/alexpls/untils/internal/monitor"
	"github.com/alexpls/untils/internal/must"
	"github.com/alexpls/untils/internal/pushover"
	"github.com/alexpls/untils/internal/search"
	"github.com/alexpls/untils/internal/session"
	"github.com/alexpls/untils/internal/usersettings"
	"github.com/alexpls/untils/public"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgxlisten"
	"github.com/lmittmann/tint"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
)

type app struct {
	config         *config
	logger         *slog.Logger
	db             *pgxpool.Pool
	dbListener     *pgxlisten.Listener
	queries        *sqlc.Queries
	auth           *auth.Auth
	sessionManager *session.Manager
	monitor        *monitor.Service
	monitorEvents  *monitor.DBEventHandler
	llm            *llm.Service
	river          *river.Client[pgx.Tx]
	pushoverClient *pushover.Client
	pushoverStore  *pushover.Store
	emailService   *email.Service
	validate       *validator.Validate
	userSettings   *usersettings.Service
	webSearcher    search.WebSearcher
}

func createApp(c *config) (*app, func()) {
	ctx, cancelFn := context.WithCancel(context.Background())
	a := &app{config: c}

	a.validate = validator.New(validator.WithRequiredStructEnabled())

	// Set dev mode for public assets
	if c.env == "dev" {
		public.SetDevMode()
	}

	var slogHandler slog.Handler
	var slogRiverHandler slog.Handler

	switch c.env {
	case "dev":
		slogHandler = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		})
		slogRiverHandler = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      slog.LevelInfo,
			TimeFormat: time.Kitchen,
		})
	default:
		slogHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		slogRiverHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	a.logger = slog.New(slogHandler).With("source", "server")

	pool, dbCloser := db.Connect(c.dbUrl, a.logger.With("source", "db"))
	a.db = pool

	a.dbListener = &pgxlisten.Listener{
		Connect: func(ctx context.Context) (*pgx.Conn, error) {
			return pgx.Connect(ctx, c.dbUrl)
		},
	}

	a.queries = sqlc.New()

	workers := river.NewWorkers()

	periodicJobs := []*river.PeriodicJob{
		river.NewPeriodicJob(
			river.PeriodicInterval(session.TrimInterval),
			func() (river.JobArgs, *river.InsertOpts) {
				return session.TrimArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		),
	}

	riverLogger := slog.New(slogRiverHandler).With("source", "river")

	a.river = must.NoErrVal(river.NewClient(riverpgxv5.New(a.db), &river.Config{
		Logger: riverLogger,
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 50},
		},
		Workers:      workers,
		PeriodicJobs: periodicJobs,
		Middleware: []rivertype.Middleware{
			newWideEventMiddleware(riverLogger),
		},
	}))

	a.webSearcher = search.NewBraveClient(c.braveKey, a.logger.With("source", "search.brave"))

	llmClient := openai.NewClient(
		option.WithBaseURL("https://api.x.ai/v1"),
		option.WithAPIKey(c.xAIKey),
	)
	a.llm = llm.NewService(&llmClient, a.logger.With("source", "llm"), a.webSearcher)

	a.auth = auth.NewAuth(a.logger.With("source", "auth"), a.db, a.queries, a.validate)

	a.sessionManager = session.NewManager(a.db, a.queries, a.logger.With("source", "session"))

	a.pushoverStore = pushover.NewStore(a.db, a.queries, a.validate)
	a.pushoverClient = pushover.NewPushoverClient(c.pushoverKey, a.logger.With("source", "pushover"), a.pushoverStore)

	a.emailService = email.NewService(email.SMTPConfig{
		Username: c.smtp.username,
		Password: c.smtp.password,
		Host:     c.smtp.host,
		Port:     c.smtp.port,
	})

	a.monitor = monitor.NewService(a.db, a.queries, a.llm, a.river, a.logger.With("source", "monitor"), a.pushoverClient, a.emailService, a.validate)
	a.monitorEvents = monitor.NewDBEventHandler(a.monitor)
	a.dbListener.Handle("monitor_events", a.monitorEvents)

	a.userSettings = usersettings.NewService(a.db, a.queries)

	river.AddWorker(workers, monitor.NewCheckWorker(a.monitor, a.logger.With("source", "monitor.check_worker")))
	river.AddWorker(workers, monitor.NewValidateMonitorWorker(a.monitor, a.logger.With("source", "monitor.validate_monitor_worker")))
	river.AddWorker(workers, a.sessionManager.NewTrimWorker(a.logger.With("source", "session.trim_worker")))
	must.NoErr(a.river.Start(ctx))

	go func() {
		if err := a.dbListener.Listen(ctx); err != nil {
			a.logger.Error("db listener error", "error", err)
		}
	}()

	closer := func() {
		a.logger.Info("gracefully shutting down...")

		ctxTimeout, ctxTimeoutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer ctxTimeoutCancel()

		cancelFn()

		// river is the only thing we care about waiting for at the moment, but if we need to
		// wait on others in the future we'll need to handle that in a more scalable way.
		select {
		case <-a.river.Stopped():
			a.logger.Info("river stopped cleanly")
		case <-ctxTimeout.Done():
			a.logger.Error("timeout out while waiting for app context cancellation")
		}

		dbCloser()
	}

	return a, closer
}
