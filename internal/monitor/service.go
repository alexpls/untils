package monitor

import (
	"log/slog"

	"github.com/alexpls/untils/internal/db/sqlc"
	"github.com/alexpls/untils/internal/email"
	"github.com/alexpls/untils/internal/llm"
	"github.com/alexpls/untils/internal/pushover"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type Service struct {
	pool           *pgxpool.Pool
	queries        *sqlc.Queries
	llm            *llm.Service
	river          *river.Client[pgx.Tx]
	logger         *slog.Logger
	pushoverClient *pushover.Client // TODO: temporary, this should not be talking directly to pushover but rather a more generic notification service
	emailService   *email.Service   // TODO: ditto above TODO, but this time for email
	validate       *validator.Validate
}

func NewService(pool *pgxpool.Pool, queries *sqlc.Queries, llm *llm.Service, river *river.Client[pgx.Tx], logger *slog.Logger, pushoverClient *pushover.Client, emailService *email.Service, validate *validator.Validate) *Service {
	return &Service{
		pool:           pool,
		queries:        queries,
		llm:            llm,
		river:          river,
		logger:         logger,
		pushoverClient: pushoverClient,
		validate:       validate,
		emailService:   emailService,
	}
}
