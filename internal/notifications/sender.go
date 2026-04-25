package notifications

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/email"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/pushover"
	"github.com/alexpls/untils/internal/webhook"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"golang.org/x/sync/errgroup"
)

type SendParams struct {
	UserID               int64
	NotificationChannels []models.Notifier
	Message              MonitorNewResults
}

type Sender interface {
	Send(ctx context.Context, params SendParams) error
}

type Service struct {
	logger       *slog.Logger
	renderConfig RenderConfig
	capabilities Capabilities
	pushover     *pushover.Client
	email        *email.Service
	river        *river.Client[pgx.Tx]
	db           db.DB
	queries      models.Queries
}

func NewService(logger *slog.Logger, renderConfig RenderConfig, capabilities Capabilities, pushover *pushover.Client, email *email.Service, riverClient *river.Client[pgx.Tx], db db.DB, queries models.Queries) *Service {
	return &Service{
		logger:       logger,
		renderConfig: renderConfig,
		capabilities: capabilities,
		pushover:     pushover,
		email:        email,
		river:        riverClient,
		db:           db,
		queries:      queries,
	}
}

var _ Sender = &Service{}

func (s *Service) Send(ctx context.Context, params SendParams) error {
	if len(params.Message.NewResults) == 0 {
		panic("notification message must contain at least one new result")
	}

	g, ctx := errgroup.WithContext(ctx)

	for _, channel := range params.NotificationChannels {
		switch channel {
		case models.NotifierEmail:
			g.Go(func() error {
				user, err := s.queries.GetUser(ctx, s.db, params.UserID)
				if err != nil {
					return fmt.Errorf("getting user: %w", err)
				}
				return s.sendEmail(ctx, user, params)
			})
		case models.NotifierPushover:
			g.Go(func() error {
				return s.sendPushoverNotification(ctx, params)
			})
		case models.NotifierWebhook:
			g.Go(func() error {
				return s.sendWebhookNotifications(ctx, params)
			})
		default:
			return fmt.Errorf("unsupported notification channel: %s", channel)
		}
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func (s *Service) sendEmail(ctx context.Context, user *models.User, params SendParams) error {
	if !s.capabilities.EmailEnabled || s.email == nil {
		return nil
	}

	for _, message := range params.Message.singleMessages() {
		rendered, err := RenderMonitorNewResultEmail(ctx, s.renderConfig, message)
		if err != nil {
			return fmt.Errorf("rendering email notification: %w", err)
		}

		if err := s.email.Send(ctx, &email.SendParams{
			Recipient: user.Email,
			Subject:   rendered.Subject,
			Body:      rendered.TextBody,
			HTMLBody:  rendered.HTMLBody,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) sendWebhookNotifications(ctx context.Context, params SendParams) error {
	if !s.capabilities.WebhookEnabled || s.river == nil {
		return nil
	}

	targets, err := s.queries.ListWebhookTargets(ctx, s.db, params.UserID)
	if err != nil {
		return fmt.Errorf("listing webhook targets: %w", err)
	}

	newResultIDs := make([]int64, len(params.Message.NewResults))
	for i, result := range params.Message.NewResults {
		newResultIDs[i] = result.ID
	}

	for _, target := range targets {
		_, err := s.river.Insert(ctx, webhook.SendArgs{
			UserID:          params.UserID,
			WebhookTargetID: target.ID,
			MonitorID:       params.Message.Monitor.ID,
			NewResultIDs:    newResultIDs,
			OldResultID:     params.Message.OldResult.ID,
		}, nil)
		if err != nil {
			return fmt.Errorf("inserting webhook send job: %w", err)
		}
	}

	return nil
}

func (s *Service) sendPushoverNotification(ctx context.Context, params SendParams) error {
	if !s.capabilities.PushoverEnabled || s.pushover == nil {
		return nil
	}

	for _, message := range params.Message.singleMessages() {
		rendered, err := RenderMonitorNewResultPushover(message)
		if err != nil {
			return fmt.Errorf("rendering pushover notification: %w", err)
		}

		if err := s.pushover.Send(ctx, pushover.SendParams{
			UserID:  params.UserID,
			Title:   rendered.Title,
			Message: rendered.Message,
		}); err != nil {
			return err
		}
	}

	return nil
}
