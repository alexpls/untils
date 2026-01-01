package pushover

import (
	"context"
	"errors"
	"fmt"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/db/sqlc"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db       *pgxpool.Pool
	queries  *sqlc.Queries
	validate *validator.Validate
}

func NewStore(db *pgxpool.Pool, queries *sqlc.Queries, validate *validator.Validate) *Store {
	return &Store{db: db, queries: queries, validate: validate}
}

var ErrNoPushoverUserToken = errors.New("no pushover user token found")

func (s *Store) GetToken(ctx context.Context, userID int64) (*sqlc.PushoverUserToken, error) {
	tok, err := s.queries.GetPushoverUserToken(ctx, s.db, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoPushoverUserToken
		}
		return nil, fmt.Errorf("getting pushover user token: %w", err)
	}
	return tok, nil
}

type CreateOrUpdateTokenParams struct {
	Token string `validate:"required"`
}

func (s *Store) CreateOrUpdateToken(ctx context.Context, userID int64, params CreateOrUpdateTokenParams) (*sqlc.PushoverUserToken, error) {
	if err := s.validate.Struct(params); err != nil {
		return nil, err
	}

	tok, err := s.queries.CreateOrUpdatePushoverUserToken(ctx, s.db, &sqlc.CreateOrUpdatePushoverUserTokenParams{
		Token:  params.Token,
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pushover token: %w", err)
	}

	return tok, nil
}

func (s *Store) DeleteToken(ctx context.Context, userID int64) error {
	err := db.WithTx(s.db, ctx, func(tx pgx.Tx) error {
		if err := s.queries.DeleteMonitorNotifiersByUserAndType(ctx, tx, &sqlc.DeleteMonitorNotifiersByUserAndTypeParams{
			UserID: userID,
			Type:   sqlc.NotifierPushover,
		}); err != nil {
			return fmt.Errorf("deleting pushover monitor notifiers: %w", err)
		}

		if err := s.queries.DeletePushoverUserToken(ctx, tx, userID); err != nil {
			return fmt.Errorf("deleting pushover user token: %w", err)
		}

		return nil
	})
	return err
}
