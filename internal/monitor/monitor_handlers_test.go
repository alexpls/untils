package monitor

import (
	"context"
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

func getHandler(handler func(http.ResponseWriter, *http.Request, *models.User), user *models.User) *http.Response {
	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, user)
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	wrappedHandler(w, req)

	res := w.Result()
	return res
}
