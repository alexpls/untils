package monitor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/notifications"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListGet(t *testing.T) {
	t.Parallel()

	t.Run("with no monitors", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		deps := setupTestDeps(ctx, t)

		_, err := deps.service.db.Exec(ctx, "delete from monitors where user_id = $1", deps.fixtures.User.ID)
		require.NoError(t, err)

		res := getHandler(deps.handlers.ListMonitors, deps.fixtures.User)
		page, _ := io.ReadAll(res.Body)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, string(page), "No monitors to show")
	})

	t.Run("with monitors", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		deps := setupTestDeps(ctx, t)

		res := getHandler(deps.handlers.ListMonitors, deps.fixtures.User)
		page, _ := io.ReadAll(res.Body)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, string(page), deps.fixtures.Monitor.Subject.String)
	})
}

func TestSendDevNotification(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)
	tx := deps.service.db.(pgx.Tx) //nolint:forcetypeassert
	result := createMonitorResultFixture(t, ctx, tx, deps.fixtures.Monitor.ID, "Notification result")

	sender := &notificationSenderCapture{}
	deps.service.notificationSender = sender

	req := httptest.NewRequest("POST", fmt.Sprintf("/app/dev/monitors/%d/results/%d/send_notification", deps.fixtures.Monitor.ID, result.ID), nil)
	res := getHandlerForRequest(func(w http.ResponseWriter, r *http.Request) {
		r.SetPathValue("monitor_id", fmt.Sprint(deps.fixtures.Monitor.ID))
		r.SetPathValue("result_id", fmt.Sprint(result.ID))
		deps.handlers.SendDevNotification(w, r, deps.fixtures.User)
	}, req)

	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Len(t, sender.calls, 1)
	assert.Equal(t, deps.fixtures.User.ID, sender.calls[0].UserID)
	assert.Equal(t, []models.Notifier{models.NotifierEmail}, sender.calls[0].NotificationChannels)
	assert.Equal(t, deps.fixtures.Monitor.ID, sender.calls[0].Message.Monitor.ID)
	assert.Equal(t, result.ID, sender.calls[0].Message.NewResults[0].ID)
}

func TestViewMonitorActivityPagination(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)

	mon, err := deps.service.queries.UpdateMonitorStatus(ctx, deps.service.db, &models.UpdateMonitorStatusParams{
		Status: models.MonitorStatusActive,
		UserID: deps.fixtures.User.ID,
		ID:     deps.fixtures.Monitor.ID,
	})
	require.NoError(t, err)

	resultIDs := make([]int64, 0, monitorActivityPageSize+1)
	for i := 1; i <= monitorActivityPageSize+1; i++ {
		check, err := deps.service.queries.CreateMonitorCheck(ctx, deps.service.db, &models.CreateMonitorCheckParams{
			MonitorID:    mon.ID,
			Status:       models.MonitorCheckStatusScheduled,
			ScheduledFor: deps.fixtures.Check.ScheduledFor,
		})
		require.NoError(t, err)

		err = deps.service.queries.UpdateMonitorCheckSuccess(ctx, deps.service.db, &models.UpdateMonitorCheckSuccessParams{
			ID:     check.ID,
			Result: &models.CheckResult{},
		})
		require.NoError(t, err)

		headline := fmt.Sprintf("Result %d", i)
		result, err := deps.service.queries.CreateMonitorResult(ctx, deps.service.db, &models.CreateMonitorResultParams{
			MonitorID: mon.ID,
			Headline:  headline,
			Subtitle:  "",
			Data:      models.MonitorUpdateData{Fields: models.MonitorUpdateFields{}},
			Citations: &models.Citations{},
		})
		require.NoError(t, err)

		err = deps.service.queries.CreateMonitorResultCheck(ctx, deps.service.db, &models.CreateMonitorResultCheckParams{
			MonitorResultID: result.ID,
			MonitorCheckID:  check.ID,
		})
		require.NoError(t, err)

		resultIDs = append(resultIDs, result.ID)
	}

	t.Run("first page shows configured page size", func(t *testing.T) {
		res := getHandlerForRequest(func(w http.ResponseWriter, r *http.Request) {
			r.SetPathValue("monitor_id", fmt.Sprint(mon.ID))
			deps.handlers.ViewMonitor(w, r, deps.fixtures.User)
		}, httptest.NewRequest("GET", fmt.Sprintf("/app/monitors/%d", mon.ID), nil))
		page, _ := io.ReadAll(res.Body)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, string(page), fmt.Sprintf("/app/monitors/%d/results/%d/hide", mon.ID, resultIDs[len(resultIDs)-1]))
		assert.Contains(t, string(page), fmt.Sprintf("/app/monitors/%d/results/%d/hide", mon.ID, resultIDs[len(resultIDs)-2]))
		assert.NotContains(t, string(page), fmt.Sprintf("/app/monitors/%d/results/%d/hide", mon.ID, resultIDs[0]))
		assert.Contains(t, string(page), fmt.Sprintf("/app/monitors/%d?page=1", mon.ID))
	})

	t.Run("second page shows older results", func(t *testing.T) {
		res := getHandlerForRequest(func(w http.ResponseWriter, r *http.Request) {
			r.SetPathValue("monitor_id", fmt.Sprint(mon.ID))
			deps.handlers.ViewMonitor(w, r, deps.fixtures.User)
		}, httptest.NewRequest("GET", fmt.Sprintf("/app/monitors/%d?page=1", mon.ID), nil))
		page, _ := io.ReadAll(res.Body)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, string(page), fmt.Sprintf("/app/monitors/%d/results/%d/hide", mon.ID, resultIDs[0]))
		assert.NotContains(t, string(page), fmt.Sprintf("/app/monitors/%d/results/%d/hide", mon.ID, resultIDs[len(resultIDs)-1]))
		assert.NotContains(t, string(page), fmt.Sprintf("/app/monitors/%d/results/%d/hide", mon.ID, resultIDs[len(resultIDs)-2]))
		assert.Contains(t, string(page), fmt.Sprintf("/app/monitors/%d?page=0", mon.ID))
	})
}

func TestViewMonitorChecksPagination(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)

	mon, err := deps.service.queries.CreateMonitor(ctx, deps.service.db, &models.CreateMonitorParams{
		UserID:  deps.fixtures.User.ID,
		Subject: pgtype.Text{String: "Pagination monitor checks", Valid: true},
	})
	require.NoError(t, err)

	mon, err = deps.service.queries.UpdateMonitorStatus(ctx, deps.service.db, &models.UpdateMonitorStatusParams{
		Status: models.MonitorStatusActive,
		UserID: deps.fixtures.User.ID,
		ID:     mon.ID,
	})
	require.NoError(t, err)

	checkIDs := make([]int64, 0, monitorChecksPageSize+1)
	for i := 1; i <= monitorChecksPageSize+1; i++ {
		check, err := deps.service.queries.CreateMonitorCheck(ctx, deps.service.db, &models.CreateMonitorCheckParams{
			MonitorID:    mon.ID,
			Status:       models.MonitorCheckStatusScheduled,
			ScheduledFor: deps.fixtures.Check.ScheduledFor.Add(time.Duration(i) * time.Hour),
		})
		require.NoError(t, err)

		err = deps.service.queries.UpdateMonitorCheckSuccess(ctx, deps.service.db, &models.UpdateMonitorCheckSuccessParams{
			ID:     check.ID,
			Result: &models.CheckResult{},
		})
		require.NoError(t, err)

		checkIDs = append(checkIDs, check.ID)
	}

	t.Run("first page shows configured page size", func(t *testing.T) {
		res := getHandlerForRequest(func(w http.ResponseWriter, r *http.Request) {
			r.SetPathValue("monitor_id", fmt.Sprint(mon.ID))
			deps.handlers.ViewMonitorChecks(w, r, deps.fixtures.User)
		}, httptest.NewRequest("GET", fmt.Sprintf("/app/monitors/%d/checks", mon.ID), nil))
		page, _ := io.ReadAll(res.Body)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, string(page), fmt.Sprintf("/app/checks/%d", checkIDs[len(checkIDs)-1]))
		assert.Contains(t, string(page), fmt.Sprintf("/app/checks/%d", checkIDs[len(checkIDs)-2]))
		assert.NotContains(t, string(page), fmt.Sprintf("/app/checks/%d", checkIDs[0]))
		assert.Contains(t, string(page), fmt.Sprintf("/app/monitors/%d/checks?page=1", mon.ID))
	})

	t.Run("second page shows older checks", func(t *testing.T) {
		res := getHandlerForRequest(func(w http.ResponseWriter, r *http.Request) {
			r.SetPathValue("monitor_id", fmt.Sprint(mon.ID))
			deps.handlers.ViewMonitorChecks(w, r, deps.fixtures.User)
		}, httptest.NewRequest("GET", fmt.Sprintf("/app/monitors/%d/checks?page=1", mon.ID), nil))
		page, _ := io.ReadAll(res.Body)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, string(page), fmt.Sprintf("/app/checks/%d", checkIDs[0]))
		assert.NotContains(t, string(page), fmt.Sprintf("/app/checks/%d", checkIDs[len(checkIDs)-1]))
		assert.NotContains(t, string(page), fmt.Sprintf("/app/checks/%d", checkIDs[len(checkIDs)-2]))
		assert.Contains(t, string(page), fmt.Sprintf("/app/monitors/%d/checks?page=0", mon.ID))
	})
}

func TestViewMonitorNotificationsHidesDisabledIntegrations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)
	deps.service.capabilities.EmailEnabled = false

	res := getHandlerForRequest(func(w http.ResponseWriter, r *http.Request) {
		r.SetPathValue("monitor_id", fmt.Sprint(deps.fixtures.Monitor.ID))
		deps.handlers.ViewMonitorNotifications(w, r, deps.fixtures.User)
	}, httptest.NewRequest("GET", fmt.Sprintf("/app/monitors/%d/notifications", deps.fixtures.Monitor.ID), nil))
	page, _ := io.ReadAll(res.Body)

	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.NotContains(t, string(page), "Notify by <strong>Email</strong>")
}

func getHandler(handler func(http.ResponseWriter, *http.Request, *models.User), user *models.User) *http.Response {
	return getHandlerForRequest(func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, user)
	}, httptest.NewRequest("GET", "/", nil))
}

func getHandlerForRequest(handler func(http.ResponseWriter, *http.Request), req *http.Request) *http.Response {
	w := httptest.NewRecorder()
	handler(w, req)

	res := w.Result()
	return res
}

type notificationSenderCapture struct {
	calls []notifications.SendParams
}

func (s *notificationSenderCapture) Send(ctx context.Context, params notifications.SendParams) error {
	s.calls = append(s.calls, params)
	return nil
}
