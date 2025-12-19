package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/alexpls/untils_go/internal/auth"
	"github.com/alexpls/untils_go/internal/db"
	"github.com/alexpls/untils_go/internal/db/sqlc"
	"github.com/alexpls/untils_go/internal/email"
	"github.com/alexpls/untils_go/internal/llm"
	"github.com/alexpls/untils_go/internal/monitor"
	"github.com/alexpls/untils_go/internal/must"
	"github.com/alexpls/untils_go/internal/pushover"
	"github.com/alexpls/untils_go/internal/session"
	"github.com/alexpls/untils_go/internal/usersettings"
	"github.com/alexpls/untils_go/public"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmittmann/tint"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

type app struct {
	config         *config
	logger         *slog.Logger
	db             *pgxpool.Pool
	queries        *sqlc.Queries
	auth           *auth.Auth
	sessionManager *session.Manager
	monitor        *monitor.Service
	llm            *llm.Service
	river          *river.Client[pgx.Tx]
	pushoverClient *pushover.Client
	pushoverStore  *pushover.Store
	emailService   *email.Service
	validate       *validator.Validate
	userSettings   *usersettings.Service
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

	a.queries = sqlc.New()

	workers := river.NewWorkers()
	a.river = must.NoErrVal(river.NewClient(riverpgxv5.New(a.db), &river.Config{
		Logger: slog.New(slogRiverHandler).With("source", "river"),
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 50},
		},
		Workers: workers,
	}))

	llmClient := openai.NewClient(
		option.WithBaseURL("https://api.x.ai/v1"),
		option.WithAPIKey(c.xAIKey),
	)
	a.llm = llm.NewService(&llmClient, a.logger.With("source", "llm"))

	a.auth = auth.NewAuth(a.logger.With("source", "auth"), a.db, a.queries, a.validate)

	a.sessionManager = session.NewManager(a.db, a.queries, a.logger.With("source", "session"))
	a.sessionManager.StartTrim()

	a.pushoverStore = pushover.NewStore(a.db, a.queries, a.validate)
	a.pushoverClient = pushover.NewPushoverClient(c.pushoverKey, a.logger.With("source", "pushover"), a.pushoverStore)

	a.emailService = email.NewService()

	a.monitor = monitor.NewService(a.db, a.queries, a.llm, a.river, a.logger.With("source", "monitor"), a.pushoverClient, a.validate)

	a.userSettings = usersettings.NewService(a.db, a.queries)

	river.AddWorker(workers, monitor.NewCheckWorker(a.monitor, a.logger.With("source", "monitor.check_worker")))
	river.AddWorker(workers, monitor.NewValidateMonitorWorker(a.monitor, a.logger.With("source", "monitor.validate_monitor_worker")))
	river.AddWorker(workers, monitor.NewPreviewMonitorWorker(a.monitor, a.logger.With("source", "monitor.preview_monitor_worker")))
	must.NoErr(a.river.Start(ctx))

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

		a.sessionManager.StopTrim()
		dbCloser()
	}

	return a, closer
}
