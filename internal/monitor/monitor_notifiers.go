package monitor

import (
	"context"
	"errors"
	"fmt"

	"github.com/alexpls/untils_go/internal/db/sqlc"
	"github.com/alexpls/untils_go/internal/pushover"
	"github.com/jackc/pgx/v5"
)

func (s *Service) ListMonitorNotifiers(ctx context.Context, mon *sqlc.Monitor) ([]*sqlc.MonitorNotifier, error) {
	notifiers, err := s.queries.ListMonitorNotifiers(ctx, s.pool, mon.ID)
	if err != nil {
		return nil, fmt.Errorf("listing monitor notifiers: %w", err)
	}
	return notifiers, nil
}

func (s *Service) CreateMonitorNotifier(ctx context.Context, mon *sqlc.Monitor, notifierType sqlc.Notifier) (*sqlc.MonitorNotifier, error) {
	switch notifierType {
	case sqlc.NotifierPushover:
		_, err := s.queries.GetPushoverUserToken(ctx, s.pool, mon.UserID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNotifierNotConfigured
			}
			return nil, fmt.Errorf("checking pushover token: %w", err)
		}
	}

	notifier, err := s.queries.CreateMonitorNotifier(ctx, s.pool, &sqlc.CreateMonitorNotifierParams{
		MonitorID: mon.ID,
		Type:      notifierType,
	})
	if err != nil {
		return nil, fmt.Errorf("creating monitor notifier: %w", err)
	}
	return notifier, nil
}

func (s *Service) DeleteMonitorNotifier(ctx context.Context, mon *sqlc.Monitor, notifierType sqlc.Notifier) error {
	err := s.queries.DeleteMonitorNotifier(ctx, s.pool, &sqlc.DeleteMonitorNotifierParams{
		MonitorID: mon.ID,
		Type:      notifierType,
	})
	if err != nil {
		return fmt.Errorf("deleting monitor notifier: %w", err)
	}
	return nil
}

type SendNotificationsParams struct {
	Monitor *sqlc.Monitor
	Message string
}

func (s *Service) SendNotifications(ctx context.Context, params SendNotificationsParams) error {
	notifiers, err := s.ListMonitorNotifiers(ctx, params.Monitor)
	if err != nil {
		return fmt.Errorf("listing monitor notifiers: %w", err)
	}

	for _, notifier := range notifiers {
		switch notifier.Type {
		case sqlc.NotifierPushover:
			if err := s.sendPushoverNotification(ctx, params); err != nil {
				return fmt.Errorf("sending pushover notification: %w", err)
			}
		}
	}

	return nil
}

func (s *Service) sendPushoverNotification(ctx context.Context, params SendNotificationsParams) error {
	return s.pushoverClient.Send(ctx, pushover.SendParams{
		Title:   fmt.Sprintf("Monitor: %s", params.Monitor.Subject.String),
		Message: params.Message,
		UserID:  params.Monitor.UserID,
	})
}
