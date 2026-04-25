package monitor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/notifications"
	"github.com/jackc/pgx/v5"
)

var ErrFakeMonitorResultRequiresActiveMonitor = errors.New("fake monitor result requires an active monitor")

func (s *Service) CreateFakeMonitorResultAndNotify(ctx context.Context, mon *models.Monitor) (*models.MonitorResult, error) {
	if mon.Status != models.MonitorStatusActive {
		return nil, ErrFakeMonitorResultRequiresActiveMonitor
	}

	type fakeResultData struct {
		result  *models.MonitorResult
		message notifications.MonitorNewResults
	}

	created, err := db.WithTxV(s.db, ctx, func(tx pgx.Tx) (*fakeResultData, error) {
		oldResult := emptyNotificationResult()
		latestResult, err := s.queries.GetLatestMonitorResult(ctx, tx, mon.ID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("getting latest monitor result: %w", err)
		}
		if latestResult != nil {
			oldResult = *latestResult
		}

		now := time.Now()
		update, err := s.fakeMonitorUpdate(ctx, tx, mon.ID, now)
		if err != nil {
			return nil, err
		}
		checkResult := &models.CheckResult{
			CheckResultBase: models.CheckResultBase{
				Success:             true,
				Reason:              "Fake result generated in dev mode.",
				DifferentToPrevious: true,
				Updates:             models.MonitorUpdateDataList{update},
			},
		}

		check, err := s.queries.CreateMonitorCheck(ctx, tx, &models.CreateMonitorCheckParams{
			MonitorID:    mon.ID,
			Status:       models.MonitorCheckStatusSuccess,
			ScheduledFor: now,
			DoneAt:       &now,
			Result:       checkResult,
		})
		if err != nil {
			return nil, fmt.Errorf("creating fake monitor check: %w", err)
		}

		citations := models.Citations{}
		result, err := s.queries.CreateMonitorResult(ctx, tx, MonitorUpdateToCreateMonitorResultParams(mon.ID, update, &citations))
		if err != nil {
			return nil, fmt.Errorf("creating fake monitor result: %w", err)
		}

		if err := s.queries.CreateMonitorResultCheck(ctx, tx, &models.CreateMonitorResultCheckParams{
			MonitorResultID: result.ID,
			MonitorCheckID:  check.ID,
		}); err != nil {
			return nil, fmt.Errorf("creating fake monitor result check link: %w", err)
		}

		return &fakeResultData{
			result:  result,
			message: newResultsNotificationMessage(*mon, []models.MonitorResult{*result}, oldResult),
		}, nil
	})
	if err != nil {
		return nil, err
	}

	if err := s.SendNotifications(ctx, SendNotificationsParams{
		Monitor: mon,
		Message: created.message,
	}); err != nil {
		return nil, err
	}

	return created.result, nil
}

func (s *Service) fakeMonitorUpdate(ctx context.Context, tx pgx.Tx, monitorID int64, now time.Time) (models.MonitorUpdateData, error) {
	fields := models.MonitorUpdateFields{}

	schema, err := s.queries.GetMonitorSchema(ctx, tx, monitorID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return models.MonitorUpdateData{}, fmt.Errorf("getting monitor schema: %w", err)
	}
	if schema != nil {
		for _, field := range schema.Data.Fields {
			fields = append(fields, models.MonitorUpdateField{
				MonitorSchemaField: field,
				Value:              fakeMonitorFieldValue(field, now),
			})
		}
	}

	return models.MonitorUpdateData{
		Headline: fmt.Sprintf("Fake result generated at %s", now.Format(time.RFC3339)),
		Subtitle: "Generated in dev mode for notification testing.",
		Fields:   fields,
	}, nil
}

func fakeMonitorFieldValue(field models.MonitorSchemaField, now time.Time) string {
	switch field.Type {
	case models.MonitorSchemaFieldTypeDate:
		return now.Format("2006-01-02")
	case models.MonitorSchemaFieldTypeURL:
		return "https://example.com/fake-result"
	default:
		return "Fake " + field.Name
	}
}
