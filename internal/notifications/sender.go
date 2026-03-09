package notifications

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/email"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/pushover"
	"golang.org/x/sync/errgroup"
)

type SendParams struct {
	UserID               int64
	NotificationChannels []models.Notifier
	Message              MonitorNewResult
}

type Sender interface {
	Send(ctx context.Context, params SendParams) error
}

type Service struct {
	logger   *slog.Logger
	pushover *pushover.Client
	email    *email.Service
	db       db.DB
	queries  models.Queries
}

func NewService(logger *slog.Logger, pushover *pushover.Client, email *email.Service, db db.DB, queries models.Queries) *Service {
	return &Service{
		logger:   logger,
		pushover: pushover,
		email:    email,
		db:       db,
		queries:  queries,
	}
}

var _ Sender = &Service{}

func (s *Service) Send(ctx context.Context, params SendParams) error {
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
	rendered, err := RenderMonitorNewResultEmail(ctx, params.Message)
	if err != nil {
		return fmt.Errorf("rendering email notification: %w", err)
	}

	return s.email.Send(ctx, &email.SendParams{
		Recipient: user.Email,
		Subject:   rendered.Subject,
		Body:      rendered.TextBody,
	})
}

func (s *Service) sendPushoverNotification(ctx context.Context, params SendParams) error {
	rendered := RenderMonitorNewResultPushover(params.Message)

	return s.pushover.Send(ctx, pushover.SendParams{
		UserID:  params.UserID,
		Title:   rendered.Title,
		Message: rendered.Message,
	})
}
