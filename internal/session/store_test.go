package session

import (
	"context"
	"testing"
	"time"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/testhelper"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_Save(t *testing.T) {
	pool := testhelper.TestDB(t)
	s := &store{db: pool, queries: models.New()}
	ctx := context.Background()

	t.Run("save new session", func(t *testing.T) {
		now := time.Now()
		sess := &Session{
			ID:        "test-session-id",
			CreatedAt: now,
			ExpiresAt: now.Add(24 * time.Hour),
			Data: SessionData{
				UserID: 123,
				Flash: map[Flash]string{
					FlashTypeAlert: "Welcome!",
				},
			},
		}

		err := s.save(ctx, sess)
		require.NoError(t, err)

		assert.Equal(t, 1, countSessionsWithID(t, pool, ctx, sess.ID))
	})

	t.Run("upsert existing session", func(t *testing.T) {
		now := time.Now()
		sess := &Session{
			ID:        "test-session-id-upsert",
			CreatedAt: now,
			ExpiresAt: now.Add(24 * time.Hour),
			Data: SessionData{
				UserID: 123,
				Flash:  map[Flash]string{},
			},
		}

		err := s.save(ctx, sess)
		require.NoError(t, err)

		sess.Data.UserID = 456
		sess.Data.Flash = map[Flash]string{
			FlashTypeAlert: "Updated!",
		}
		sess.ExpiresAt = now.Add(48 * time.Hour)

		err = s.save(ctx, sess)
		require.NoError(t, err)

		assert.Equal(t, 1, countSessionsWithID(t, pool, ctx, sess.ID))

		retrieved, err := s.get(ctx, sess.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(456), retrieved.Data.UserID)
		assert.Equal(t, "Updated!", retrieved.Data.Flash[FlashTypeAlert])
	})
}

func TestStore_Get(t *testing.T) {
	pool := testhelper.TestDB(t)
	s := &store{db: pool, queries: models.New()}
	ctx := context.Background()

	t.Run("get existing session", func(t *testing.T) {
		now := time.Now()
		sess := &Session{
			ID:        "test-session-id",
			CreatedAt: now,
			ExpiresAt: now.Add(24 * time.Hour),
			Data: SessionData{
				UserID: 123,
				Flash: map[Flash]string{
					FlashTypeAlert: "Test message",
				},
			},
		}

		err := s.save(ctx, sess)
		require.NoError(t, err)

		retrieved, err := s.get(ctx, sess.ID)
		require.NoError(t, err)

		assert.Equal(t, sess.ID, retrieved.ID)
		assert.Equal(t, sess.Data.UserID, retrieved.Data.UserID)
		assert.Equal(t, sess.Data.Flash[FlashTypeAlert], retrieved.Data.Flash[FlashTypeAlert])
	})

	t.Run("get non-existent session", func(t *testing.T) {
		_, err := s.get(ctx, "non-existent-id")
		assert.Error(t, err)
	})

	t.Run("get expired session", func(t *testing.T) {
		now := time.Now()
		sess := &Session{
			ID:        "expired-session-id",
			CreatedAt: now.Add(-2 * time.Hour),
			ExpiresAt: now.Add(-1 * time.Hour), // Expired 1 hour ago
			Data: SessionData{
				UserID: 123,
				Flash:  map[Flash]string{},
			},
		}

		err := s.save(ctx, sess)
		require.NoError(t, err)

		_, err = s.get(ctx, sess.ID)
		assert.Error(t, err)
	})
}

func TestStore_Destroy(t *testing.T) {
	pool := testhelper.TestDB(t)
	s := &store{db: pool, queries: models.New()}
	ctx := context.Background()

	t.Run("destroy existing session", func(t *testing.T) {
		now := time.Now()
		sess := &Session{
			ID:        "test-session-id",
			CreatedAt: now,
			ExpiresAt: now.Add(24 * time.Hour),
			Data: SessionData{
				UserID: 123,
				Flash:  map[Flash]string{},
			},
		}

		err := s.save(ctx, sess)
		require.NoError(t, err)

		err = s.destroy(ctx, sess.ID)
		require.NoError(t, err)

		_, err = s.get(ctx, sess.ID)
		assert.Error(t, err, "should get error when getting destroyed session")
	})

	t.Run("destroy non-existent session", func(t *testing.T) {
		err := s.destroy(ctx, "non-existent-id")
		assert.NoError(t, err, "destroying non-existent session should not error")
	})
}

func TestStore_Trim(t *testing.T) {
	pool := testhelper.TestDB(t)
	s := &store{db: pool, queries: models.New()}
	ctx := context.Background()

	t.Run("trim expired sessions", func(t *testing.T) {
		now := time.Now()

		expiredSess1 := &Session{
			ID:        "expired-1",
			CreatedAt: now.Add(-2 * time.Hour),
			ExpiresAt: now.Add(-1 * time.Hour),
			Data:      SessionData{UserID: 1, Flash: map[Flash]string{}},
		}
		expiredSess2 := &Session{
			ID:        "expired-2",
			CreatedAt: now.Add(-3 * time.Hour),
			ExpiresAt: now.Add(-30 * time.Minute),
			Data:      SessionData{UserID: 2, Flash: map[Flash]string{}},
		}
		validSess := &Session{
			ID:        "valid-1",
			CreatedAt: now,
			ExpiresAt: now.Add(24 * time.Hour),
			Data:      SessionData{UserID: 3, Flash: map[Flash]string{}},
		}

		require.NoError(t, s.save(ctx, expiredSess1))
		require.NoError(t, s.save(ctx, expiredSess2))
		require.NoError(t, s.save(ctx, validSess))

		rowsAffected, err := s.trim()
		require.NoError(t, err)
		assert.Equal(t, int64(2), rowsAffected)

		_, err = s.get(ctx, validSess.ID)
		assert.NoError(t, err, "valid session should still exist")

		assert.Equal(t, 1, countSessions(t, pool, ctx))
	})

	t.Run("trim with no expired sessions", func(t *testing.T) {
		now := time.Now()

		validSess := &Session{
			ID:        "valid-2",
			CreatedAt: now,
			ExpiresAt: now.Add(24 * time.Hour),
			Data:      SessionData{UserID: 1, Flash: map[Flash]string{}},
		}

		require.NoError(t, s.save(ctx, validSess))

		rowsAffected, err := s.trim()
		require.NoError(t, err)
		assert.Equal(t, int64(0), rowsAffected)

		assert.Equal(t, 1, countSessionsWithID(t, pool, ctx, validSess.ID))
	})
}

// countSessions returns the total number of sessions in the database
func countSessions(t *testing.T, pool *pgxpool.Pool, ctx context.Context) int {
	t.Helper()
	var count int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM sessions").Scan(&count)
	require.NoError(t, err)
	return count
}

// countSessionsWithID returns the number of sessions with the specified ID
func countSessionsWithID(t *testing.T, pool *pgxpool.Pool, ctx context.Context, sessionID string) int {
	t.Helper()
	var count int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM sessions WHERE id = $1", sessionID).Scan(&count)
	require.NoError(t, err)
	return count
}
