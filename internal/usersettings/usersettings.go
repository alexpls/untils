package usersettings

import (
	"context"
	"fmt"

	"github.com/alexpls/untils_go/internal/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewService(db *pgxpool.Pool, queries *sqlc.Queries) *Service {
	return &Service{
		db:      db,
		queries: queries,
	}
}

func (s *Service) ActiveIntegrations(ctx context.Context, userID int64) ([]*sqlc.ActiveUserIntegrationsRow, error) {
	ai, err := s.queries.ActiveUserIntegrations(ctx, s.db, userID)
	if err != nil {
		return nil, fmt.Errorf("listing active integrations: %w", err)
	}
	return ai, nil
}
