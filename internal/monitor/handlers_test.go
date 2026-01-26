package monitor

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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

		handler := func(w http.ResponseWriter, r *http.Request) {
			deps.handlers.ListGet(w, r, deps.fixtures.User)
		}

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		res := w.Result()
		page, _ := io.ReadAll(res.Body)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, string(page), "No monitors to show")
	})

	t.Run("with monitors", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		deps := setupTestDeps(ctx, t)

		handler := func(w http.ResponseWriter, r *http.Request) {
			deps.handlers.ListGet(w, r, deps.fixtures.User)
		}

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		res := w.Result()
		page, _ := io.ReadAll(res.Body)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, string(page), deps.fixtures.Monitor.Subject.String)
	})
}
