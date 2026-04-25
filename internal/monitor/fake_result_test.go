package monitor

import (
	"context"
	"testing"

	"github.com/alexpls/untils/internal/models"
	"github.com/stretchr/testify/require"
)

func TestCreateFakeMonitorResultAndNotify(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)

	mon, err := deps.service.queries.UpdateMonitorStatus(ctx, deps.service.db, &models.UpdateMonitorStatusParams{
		ID:     deps.fixtures.Monitor.ID,
		UserID: deps.fixtures.User.ID,
		Status: models.MonitorStatusActive,
	})
	require.NoError(t, err)

	_, err = deps.service.queries.CreateMonitorNotifier(ctx, deps.service.db, &models.CreateMonitorNotifierParams{
		MonitorID: mon.ID,
		Type:      models.NotifierWebhook,
	})
	require.NoError(t, err)

	sender := &notificationSenderCapture{}
	deps.service.notificationSender = sender

	result, err := deps.service.CreateFakeMonitorResultAndNotify(ctx, mon)
	require.NoError(t, err)
	require.NotZero(t, result.ID)
	require.Contains(t, result.Headline, "Fake result generated at")
	require.Equal(t, "Generated in dev mode for notification testing.", result.Subtitle)

	require.Len(t, sender.calls, 1)
	require.Equal(t, []models.Notifier{models.NotifierWebhook}, sender.calls[0].NotificationChannels)
	require.Equal(t, result.ID, sender.calls[0].Message.New.ID)
}

func TestCreateFakeMonitorResultAndNotifyRequiresActiveMonitor(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)

	_, err := deps.service.CreateFakeMonitorResultAndNotify(ctx, deps.fixtures.Monitor)
	require.ErrorIs(t, err, ErrFakeMonitorResultRequiresActiveMonitor)
}
