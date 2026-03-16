package monitor

import (
	"log/slog"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/llm"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/notifications"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type llmWorkflowBuilder interface {
	NewTriageWorkflow() llm.TriageWorkflowRunner
	NewCheckWorkflow() llm.CheckWorkflowRunner
}

type Service struct {
	db                 db.DB
	queries            *models.Queries
	llm                llmWorkflowBuilder
	river              *river.Client[pgx.Tx]
	logger             *slog.Logger
	notificationSender notifications.Sender
	notificationRender notifications.RenderConfig
	validate           *validator.Validate
}

func NewService(db db.DB,
	queries *models.Queries,
	llm llmWorkflowBuilder,
	river *river.Client[pgx.Tx],
	logger *slog.Logger,
	validate *validator.Validate,
	notificationSender notifications.Sender,
	notificationRender notifications.RenderConfig,
) *Service {
	return &Service{
		db:                 db,
		queries:            queries,
		llm:                llm,
		river:              river,
		logger:             logger,
		validate:           validate,
		notificationSender: notificationSender,
		notificationRender: notificationRender,
	}
}
