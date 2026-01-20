package session

import (
	"context"
	"encoding/json"

	"github.com/alexpls/untils/internal/models"
)

type store struct {
	db      models.DBTX
	queries *models.Queries
}

func (s *store) get(ctx context.Context, id string) (*Session, error) {
	sqlcSess, err := s.queries.GetSession(ctx, s.db, id)
	if err != nil {
		return &Session{}, err
	}

	var data SessionData
	if err := json.Unmarshal(sqlcSess.Data, &data); err != nil {
		return &Session{}, err
	}

	return &Session{
		ID:        sqlcSess.ID,
		CreatedAt: sqlcSess.CreatedAt,
		ExpiresAt: sqlcSess.ExpiresAt,
		Data:      data,
	}, nil
}

func (s *store) save(ctx context.Context, sess *Session) error {
	data, err := json.Marshal(sess.Data)
	if err != nil {
		return err
	}

	return s.queries.SaveSession(ctx, s.db, &models.SaveSessionParams{
		ID:        sess.ID,
		CreatedAt: sess.CreatedAt,
		ExpiresAt: sess.ExpiresAt,
		Data:      data,
	})
}

func (s *store) destroy(ctx context.Context, sessionID string) error {
	return s.queries.DestroySession(ctx, s.db, sessionID)
}

func (s *store) trim() (int64, error) {
	return s.queries.TrimExpiredSessions(context.Background(), s.db)
}
