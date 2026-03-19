package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/alexpls/untils/internal/auth"
	"github.com/alexpls/untils/internal/browser"
	"github.com/alexpls/untils/internal/dashboard"
	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/dev"
	"github.com/alexpls/untils/internal/email"
	"github.com/alexpls/untils/internal/llm"
	"github.com/alexpls/untils/internal/logging"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/monitor"
	"github.com/alexpls/untils/internal/must"
	"github.com/alexpls/untils/internal/notifications"
	"github.com/alexpls/untils/internal/pages"
	"github.com/alexpls/untils/internal/pushover"
	"github.com/alexpls/untils/internal/reqcontext"
	"github.com/alexpls/untils/internal/search"
	"github.com/alexpls/untils/internal/session"
	"github.com/alexpls/untils/internal/settings"
	"github.com/alexpls/untils/internal/types"
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
	config              *config
	logger              *slog.Logger
	db                  *pgxpool.Pool
	dbListener          *pgxlisten.Listener
	queries             *models.Queries
	auth                *auth.Auth
	sessionManager      *session.Manager
	monitor             *monitor.Service
	monitorEvents       *monitor.DBEventHandler
	monitorHandlers     *monitor.Handlers
	dashboardHandlers   *dashboard.Handlers
	pagesHandlers       *pages.Handlers
	authHandlers        *auth.Handlers
	settingsHandlers    *settings.Handlers
	llm                 *llm.Service
	river               *river.Client[pgx.Tx]
	pushoverClient      *pushover.Client
	pushoverStore       *pushover.Store
	emailService        *email.Service
	notificationService *notifications.Service
	validate            *validator.Validate
	webSearcher         search.WebSearcher
	devHandlers         *dev.Handlers
}

func createApp(c *config) (*app, context.Context, context.CancelFunc, func()) {
	ctx, cancelFn := context.WithCancel(
		reqcontext.ContextWithBuildVersion(
			reqcontext.ContextWithEnv(context.Background(), c.env), c.buildVersion,
		),
	)
	a := &app{config: c}

	a.validate = validator.New(validator.WithRequiredStructEnabled())

	// Set dev mode for public assets
	if c.env == appEnvDev {
		public.SetDevMode()
	}

	var slogHandler slog.Handler
	var slogRiverHandler slog.Handler

	switch c.env {
	case appEnvDev:
		slogHandler = logging.ContextHandler{
			Handler: tint.NewHandler(os.Stdout, &tint.Options{
				Level:      slog.LevelDebug,
				TimeFormat: time.Kitchen,
			}),
		}
		slogRiverHandler = logging.ContextHandler{
			Handler: tint.NewHandler(os.Stdout, &tint.Options{
				Level:      slog.LevelInfo,
				TimeFormat: time.Kitchen,
			}),
		}
	default:
		slogHandler = logging.ContextHandler{
			Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}),
		}
		slogRiverHandler = logging.ContextHandler{
			Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}),
		}
	}

	a.logger = slog.New(slogHandler).With("source", "server")

	if c.migrate {
		must.NoErr(runMigrations(a.logger.With("source", "db.migrate"), c.dbUrl))
	}

	pool, dbCloser := db.Connect(c.dbUrl, a.logger.With("source", "db"))
	a.db = pool

	a.dbListener = &pgxlisten.Listener{
		Connect: func(ctx context.Context) (*pgx.Conn, error) {
			return pgx.Connect(ctx, c.dbUrl)
		},
	}

	a.queries = models.New()

	workers := river.NewWorkers()

	periodicJobs := []*river.PeriodicJob{
		river.NewPeriodicJob(
			river.PeriodicInterval(session.TrimInterval),
			func() (river.JobArgs, *river.InsertOpts) {
				return session.TrimArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		),
	}

	riverLogger := slog.New(slogRiverHandler).With("source", "river")

	a.river = must.NoErrVal(river.NewClient(riverpgxv5.New(a.db), &river.Config{
		Logger: riverLogger,
		Queues: map[string]river.QueueConfig{
			river.QueueDefault:      {MaxWorkers: 50},
			types.RiverBrowserQueue: {MaxWorkers: 5},
		},
		Workers:      workers,
		PeriodicJobs: periodicJobs,
		Middleware: []rivertype.Middleware{
			newWideEventMiddleware(riverLogger),
		},
	}))

	a.webSearcher = search.NewBraveClient(c.braveKey, a.logger.With("source", "search.brave"))

	clientOptions := []option.RequestOption{
		option.WithAPIKey(c.openAIAPIKey),
	}
	if c.usesXAI() {
		clientOptions = append(clientOptions, option.WithBaseURL("https://api.x.ai/v1"))
	}
	llmClient := openai.NewClient(clientOptions...)
	llmProvider := llm.NewOpenAIProvider(&llmClient)
	llmLogger := a.logger.With("source", "llm")
	browserManager := browser.NewManager(c.chrome.maxConcurrentSessions, browser.BrowserSessionConfig{
		ChromeDevToolsURL: a.config.chrome.devToolsURL,
	}, llmLogger.With("component", "browser"))

	a.llm = llm.NewService(
		llmProvider,
		c.openAIModel,
		a.db,
		a.queries,
		llmLogger,
		a.webSearcher,
		func(ctx context.Context) (browser.BrowserSession, context.CancelFunc, error) {
			return browserManager.NewSession(ctx)
		},
	)

	a.auth = auth.NewAuth(a.logger.With("source", "auth"), a.db, a.queries, a.validate)
	must.NoErr(a.bootstrapInitialSelfHostedAdmin(ctx, a.db))

	a.sessionManager = session.NewManager(a.db, a.queries, a.logger.With("source", "session"))

	notificationCapabilities := notifications.Capabilities{
		EmailEnabled:    c.emailSendConfigured(),
		PushoverEnabled: c.pushoverConfigured(),
	}
	a.pushoverStore = pushover.NewStore(a.db, a.queries, a.validate)
	if notificationCapabilities.PushoverEnabled {
		a.pushoverClient = pushover.NewPushoverClient(c.pushoverKey, a.logger.With("source", "pushover"), a.pushoverStore)
	}

	if notificationCapabilities.EmailEnabled {
		a.emailService = email.NewService(email.SMTPConfig{
			Username: c.smtp.username,
			Password: c.smtp.password,
			Host:     c.smtp.host,
			Port:     c.smtp.port,
			From:     c.smtp.from,
		})
	}

	notificationRenderConfig := notifications.RenderConfig{
		BaseURL: c.baseURL,
	}

	a.notificationService = notifications.NewService(a.logger.With("source", "notifications.service"), notificationRenderConfig, notificationCapabilities, a.pushoverClient, a.emailService, a.db, *a.queries)

	a.monitor = monitor.NewService(a.db, a.queries, a.llm, a.river, a.logger.With("source", "monitor"), a.validate, notificationCapabilities, a.notificationService, notificationRenderConfig)
	a.monitorEvents = monitor.NewDBEventHandler()
	a.dbListener.Handle("monitor_events", a.monitorEvents)

	a.monitorHandlers = monitor.NewHandlers(a.monitor, a.monitorEvents, a.logger.With("source", "monitor.handlers"))

	a.dashboardHandlers = dashboard.NewHandlers(a.queries, a.db, a.monitorEvents, a.logger.With("source", "dashboard.handlers"))

	a.pagesHandlers = pages.NewHandlers(a.queries, a.db, a.logger.With("source", "pages.handlers"))

	a.authHandlers = auth.NewHandlers(a.auth, a.sessionManager, a.logger.With("source", "auth.handlers"))

	a.settingsHandlers = settings.NewHandlers(a.queries, a.db, notificationCapabilities, a.pushoverStore, a.pushoverClient, a.sessionManager, a.auth, a.logger.With("source", "settings.handlers"))

	a.devHandlers = dev.NewHandlers(a.logger.With("source", "dev.handlers"), notifications.NewEmailTemplateStore(notificationRenderConfig))

	river.AddWorker(workers, monitor.NewCheckWorker(a.monitor, a.logger.With("source", "monitor.check_worker")))
	river.AddWorker(workers, monitor.NewValidateMonitorWorker(a.monitor, a.logger.With("source", "monitor.validate_monitor_worker")))
	river.AddWorker(workers, a.sessionManager.NewTrimWorker(a.logger.With("source", "session.trim_worker")))
	must.NoErr(a.river.Start(ctx))

	go func() {
		if err := a.dbListener.Listen(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				a.logger.ErrorContext(ctx, "db listener error", "error", err)
			}
		}
	}()

	closer := func() {
		ctxTimeout, ctxTimeoutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		a.logger.InfoContext(ctxTimeout, "gracefully shutting down...")
		defer ctxTimeoutCancel()

		cancelFn()

		// river is the only thing we care about waiting for at the moment, but if we need to
		// wait on others in the future we'll need to handle that in a more scalable way.
		select {
		case <-a.river.Stopped():
		case <-ctxTimeout.Done():
			a.logger.ErrorContext(ctxTimeout, "timeout out while waiting for app context cancellation")
		}

		dbCloser()
	}

	return a, ctx, cancelFn, closer
}

func (a *app) bootstrapInitialSelfHostedAdmin(ctx context.Context, db models.DBTX) error {
	if a.config.appMode != appModeSelfHosted {
		return nil
	}
	if a.config.adminEmail == "" {
		return nil
	}

	userCount, err := a.queries.CountUsers(ctx, db)
	if err != nil {
		return fmt.Errorf("counting users: %w", err)
	}
	if userCount != 0 {
		return nil
	}

	user, err := a.auth.CreateUser(ctx, a.config.adminEmail, "abc123", "UTC")
	if err != nil {
		return fmt.Errorf("creating initial selfhosted admin user: %w", err)
	}

	a.logger.InfoContext(ctx, "bootstrapped initial selfhosted admin user", "user_id", user.ID, "email", user.Email)

	return nil
}
