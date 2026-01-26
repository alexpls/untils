package monitor

import (
	"context"
	"errors"
	"fmt"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/notifications"
	"github.com/jackc/pgx/v5"
)

func (s *Service) ListMonitorNotifiers(ctx context.Context, mon *models.Monitor) ([]*models.MonitorNotifier, error) {
	notifiers, err := s.queries.ListMonitorNotifiers(ctx, s.db, mon.ID)
	if err != nil {
		return nil, fmt.Errorf("listing monitor notifiers: %w", err)
	}
	return notifiers, nil
}

func (s *Service) CreateMonitorNotifier(ctx context.Context, mon *models.Monitor, notifierType models.Notifier) (*models.MonitorNotifier, error) {
	return db.WithTxV(s.db, ctx, func(tx pgx.Tx) (*models.MonitorNotifier, error) {
		return s.createMonitorNotifierTx(ctx, tx, mon, notifierType)
	})
}

func (s *Service) createMonitorNotifierTx(ctx context.Context, tx pgx.Tx, mon *models.Monitor, notifierType models.Notifier) (*models.MonitorNotifier, error) {
	switch notifierType {
	case models.NotifierPushover:
		_, err := s.queries.GetPushoverUserToken(ctx, tx, mon.UserID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNotifierNotConfigured
			}
			return nil, fmt.Errorf("checking pushover token: %w", err)
		}
	}

	notifier, err := s.queries.CreateMonitorNotifier(ctx, tx, &models.CreateMonitorNotifierParams{
		MonitorID: mon.ID,
		Type:      notifierType,
	})
	if err != nil {
		return nil, fmt.Errorf("creating monitor notifier: %w", err)
	}
	return notifier, nil
}

func (s *Service) DeleteMonitorNotifier(ctx context.Context, mon *models.Monitor, notifierType models.Notifier) error {
	err := s.queries.DeleteMonitorNotifier(ctx, s.db, &models.DeleteMonitorNotifierParams{
		MonitorID: mon.ID,
		Type:      notifierType,
	})
	if err != nil {
		return fmt.Errorf("deleting monitor notifier: %w", err)
	}
	return nil
}

type SendNotificationsParams struct {
	Monitor              *models.Monitor
	NewResult, OldResult string
}

func (s *Service) SendNotifications(ctx context.Context, params SendNotificationsParams) error {
	if params.Monitor.Status != models.MonitorStatusActive {
		s.logger.WarnContext(ctx, "skipping notifications for inactive monitor", "monitor_id", params.Monitor.ID)
		return nil
	}

	notifiers, err := s.ListMonitorNotifiers(ctx, params.Monitor)
	if err != nil {
		return fmt.Errorf("listing monitor notifiers: %w", err)
	}

	var notificationChannels []models.Notifier
	for _, notifier := range notifiers {
		notificationChannels = append(notificationChannels, notifier.Type)
	}

	if err := s.notificationSender.Send(ctx, notifications.SendParams{
		UserID:               params.Monitor.UserID,
		NotificationChannels: notificationChannels,
		Message: notifications.MonitorNewResult{
			Subject: params.Monitor.Subject.String,
			New:     params.NewResult,
			Old:     params.OldResult,
		},
	}); err != nil {
		return fmt.Errorf("sending notifications: %w", err)
	}

	return nil
}

func (s *Service) enableAllNotifiers(ctx context.Context, tx pgx.Tx, mon *models.Monitor) error {
	integrations, err := s.queries.UserIntegrations(ctx, tx, mon.UserID)
	if err != nil {
		return err
	}

	for _, integration := range integrations {
		if !integration.Configured {
			continue
		}

		if _, err := s.createMonitorNotifierTx(ctx, tx, mon, integration.Name); err != nil {
			return err
		}
	}

	return nil
}
