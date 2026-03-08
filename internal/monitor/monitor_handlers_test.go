package monitor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexpls/untils/internal/models"
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

	for i := 1; i <= 3; i++ {
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

		result, err := deps.service.queries.CreateMonitorResult(ctx, deps.service.db, &models.CreateMonitorResultParams{
			MonitorID: mon.ID,
			Headline:  fmt.Sprintf("Result %d", i),
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
	}

	t.Run("first page shows configured page size", func(t *testing.T) {
		res := getHandlerForRequest(func(w http.ResponseWriter, r *http.Request) {
			r.SetPathValue("monitor_id", fmt.Sprint(mon.ID))
			deps.handlers.ViewMonitor(w, r, deps.fixtures.User)
		}, httptest.NewRequest("GET", fmt.Sprintf("/app/monitors/%d", mon.ID), nil))
		page, _ := io.ReadAll(res.Body)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, string(page), "Result 3")
		assert.Contains(t, string(page), "Result 2")
		assert.NotContains(t, string(page), "Result 1")
		assert.Contains(t, string(page), fmt.Sprintf("/app/monitors/%d?page=1", mon.ID))
	})

	t.Run("second page shows older results", func(t *testing.T) {
		res := getHandlerForRequest(func(w http.ResponseWriter, r *http.Request) {
			r.SetPathValue("monitor_id", fmt.Sprint(mon.ID))
			deps.handlers.ViewMonitor(w, r, deps.fixtures.User)
		}, httptest.NewRequest("GET", fmt.Sprintf("/app/monitors/%d?page=1", mon.ID), nil))
		page, _ := io.ReadAll(res.Body)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, string(page), "Result 1")
		assert.NotContains(t, string(page), "Result 2")
		assert.NotContains(t, string(page), "Result 3")
		assert.Contains(t, string(page), fmt.Sprintf("/app/monitors/%d?page=0", mon.ID))
	})
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
