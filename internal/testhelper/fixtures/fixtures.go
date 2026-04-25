package testfixtures

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/models"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

type Fixtures struct {
	User    *models.User
	Monitor *models.Monitor
	Check   *models.MonitorCheck
}

var userEmailSeq atomic.Int64

func New(ctx context.Context, t *testing.T, db db.DB, queries *models.Queries) Fixtures {
	u := user(ctx, t, db, queries)
	m := monitor(ctx, t, db, queries, u.ID)
	c := check(ctx, t, db, queries, m.ID)

	return Fixtures{
		User:    u,
		Monitor: m,
		Check:   c,
	}
}

func user(ctx context.Context, t *testing.T, db db.DB, queries *models.Queries) *models.User {
	t.Helper()

	user, err := queries.CreateUser(ctx, db, &models.CreateUserParams{
		Email:        testEmail(t),
		PasswordHash: "supersecret",
		Timezone:     "UTC",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	require.NoError(t, err)

	return user
}

func testEmail(t *testing.T) string {
	t.Helper()

	name := strings.NewReplacer("/", "-", " ", "-", "_", "-").Replace(strings.ToLower(t.Name()))
	return fmt.Sprintf("tester+%s-%d@example.com", name, userEmailSeq.Add(1))
}

func monitor(ctx context.Context, t *testing.T, db db.DB, queries *models.Queries, userID int64) *models.Monitor {
	t.Helper()

	monitor, err := queries.CreateMonitor(ctx, db, &models.CreateMonitorParams{
		UserID:  userID,
		Subject: pgtype.Text{String: "Latest album by Taylor Swift", Valid: true},
	})
	require.NoError(t, err)

	return monitor
}

func check(ctx context.Context, t *testing.T, db db.DB, queries *models.Queries, monitorID int64) *models.MonitorCheck {
	t.Helper()

	check, err := queries.CreateMonitorCheck(ctx, db, &models.CreateMonitorCheckParams{
		MonitorID:    monitorID,
		Status:       models.MonitorCheckStatusScheduled,
		ScheduledFor: time.Now(),
	})
	require.NoError(t, err)

	return check
}
