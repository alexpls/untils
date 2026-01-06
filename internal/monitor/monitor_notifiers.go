package monitor

import (
	"context"
	"errors"
	"fmt"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/db/sqlc"
	"github.com/alexpls/untils/internal/email"
	"github.com/alexpls/untils/internal/pushover"
	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"
)

func (s *Service) ListMonitorNotifiers(ctx context.Context, mon *sqlc.Monitor) ([]*sqlc.MonitorNotifier, error) {
	notifiers, err := s.queries.ListMonitorNotifiers(ctx, s.pool, mon.ID)
	if err != nil {
		return nil, fmt.Errorf("listing monitor notifiers: %w", err)
	}
	return notifiers, nil
}

func (s *Service) CreateMonitorNotifier(ctx context.Context, mon *sqlc.Monitor, notifierType sqlc.Notifier) (*sqlc.MonitorNotifier, error) {
	return db.WithTxV(s.pool, ctx, func(tx pgx.Tx) (*sqlc.MonitorNotifier, error) {
		return s.createMonitorNotifierTx(ctx, tx, mon, notifierType)
	})
}

func (s *Service) createMonitorNotifierTx(ctx context.Context, tx pgx.Tx, mon *sqlc.Monitor, notifierType sqlc.Notifier) (*sqlc.MonitorNotifier, error) {
	switch notifierType {
	case sqlc.NotifierPushover:
		_, err := s.queries.GetPushoverUserToken(ctx, tx, mon.UserID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNotifierNotConfigured
			}
			return nil, fmt.Errorf("checking pushover token: %w", err)
		}
	}

	notifier, err := s.queries.CreateMonitorNotifier(ctx, tx, &sqlc.CreateMonitorNotifierParams{
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
	if params.Monitor.Status != sqlc.MonitorStatusActive {
		s.logger.Warn("skipping notifications for inactive monitor", "monitor_id", params.Monitor.ID)
		return nil
	}

	notifiers, err := s.ListMonitorNotifiers(ctx, params.Monitor)
	if err != nil {
		return fmt.Errorf("listing monitor notifiers: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)

	for _, notifier := range notifiers {
		// TODO: remove repetitive code in favor of interface abstraction
		switch notifier.Type {
		case sqlc.NotifierEmail:
			g.Go(func() error {
				if err := s.sendEmailNotification(ctx, params); err != nil {
					return fmt.Errorf("sending email notification: %w", err)
				}
				return nil
			})
		case sqlc.NotifierPushover:
			g.Go(func() error {
				if err := s.sendPushoverNotification(ctx, params); err != nil {
					return fmt.Errorf("sending pushover notification: %w", err)
				}
				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func (s *Service) enableAllNotifiers(ctx context.Context, tx pgx.Tx, mon *sqlc.Monitor) error {
	integrations, err := s.queries.UserIntegrations(ctx, tx, mon.UserID)
	if err != nil {
		return err
	}

	for _, integration := range integrations {
		if !integration.Active {
			continue
		}

		if _, err := s.createMonitorNotifierTx(ctx, tx, mon, integration.Name); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) sendEmailNotification(ctx context.Context, params SendNotificationsParams) error {
	u, err := s.queries.GetUser(ctx, s.pool, params.Monitor.UserID)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}

	return s.emailService.Send(ctx, &email.SendParams{
		Recipient: u.Email,
		Subject:   fmt.Sprintf("Changed monitor: %s", params.Monitor.Subject.String),
		Body:      params.Message,
	})
}

func (s *Service) sendPushoverNotification(ctx context.Context, params SendNotificationsParams) error {
	return s.pushoverClient.Send(ctx, pushover.SendParams{
		Title:   fmt.Sprintf("Monitor: %s", params.Monitor.Subject.String),
		Message: params.Message,
		UserID:  params.Monitor.UserID,
	})
}
