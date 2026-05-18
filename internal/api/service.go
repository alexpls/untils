package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
)

const tokenPrefix = "untils.api."

var ErrInvalidToken = errors.New("invalid api token")
var ErrNotFound = errors.New("api resource not found")

type Service struct {
	db       db.DB
	queries  *models.Queries
	validate *validator.Validate
	logger   *slog.Logger

	// bgWrites tracks fire-and-forget background DB writes (e.g. last_used_at
	// updates) so callers (graceful shutdown, tests) can wait for them to finish.
	bgWrites sync.WaitGroup
}

func NewService(db db.DB, queries *models.Queries, validate *validator.Validate, logger *slog.Logger) *Service {
	return &Service{
		db:       db,
		queries:  queries,
		validate: validate,
		logger:   logger,
	}
}

type CreateTokenParams struct {
	UserID int64  `validate:"required"`
	Name   string `validate:"required,max=100"`
}

type CreatedToken struct {
	Token *models.ApiToken
	Key   string
}

func (s *Service) CreateToken(ctx context.Context, params CreateTokenParams) (*CreatedToken, error) {
	params.Name = strings.TrimSpace(params.Name)
	if err := s.validate.Struct(params); err != nil {
		return nil, err
	}

	for {
		key := tokenPrefix + strings.ToLower(rand.Text())
		hash := tokenHash(key)

		token, err := s.queries.CreateAPIToken(ctx, s.db, &models.CreateAPITokenParams{
			KeyHash: hash,
			UserID:  params.UserID,
			Name:    params.Name,
		})
		if err != nil {
			if db.IsUniqueViolation(err, "idx_api_tokens_key_hash") {
				continue
			}
			return nil, fmt.Errorf("creating api token: %w", err)
		}

		return &CreatedToken{Token: token, Key: key}, nil
	}
}

func (s *Service) ListTokens(ctx context.Context, userID int64) ([]*models.ApiToken, error) {
	return s.queries.ListAPITokens(ctx, s.db, userID)
}

func (s *Service) DeleteToken(ctx context.Context, userID int64, tokenID string) error {
	id, err := strconv.ParseInt(tokenID, 10, 64)
	if err != nil || id <= 0 {
		return ErrNotFound
	}

	rows, err := s.queries.DeleteAPIToken(ctx, s.db, &models.DeleteAPITokenParams{
		UserID: userID,
		ID:     id,
	})
	if err != nil {
		return fmt.Errorf("deleting api token: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) AuthenticateToken(ctx context.Context, key string) (*models.ApiToken, error) {
	secret, ok := strings.CutPrefix(key, tokenPrefix)
	if !ok || secret == "" {
		return nil, ErrInvalidToken
	}

	token, err := s.queries.GetAPITokenByKeyHash(ctx, s.db, tokenHash(key))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("getting api token by hash: %w", err)
	}

	// Update last_used_at asynchronously with a detached context so a transient
	// DB write failure (or request cancellation) doesn't break authentication.
	detachedCtx := context.WithoutCancel(ctx)
	s.bgWrites.Go(func() {
		if err := s.queries.UpdateAPITokenLastUsedAt(detachedCtx, s.db, token.ID); err != nil {
			s.logger.ErrorContext(detachedCtx, "updating api token last used time", "error", err, "token_id", token.ID)
		}
	})

	return token, nil
}

// WaitForBackgroundWrites blocks until all in-flight background DB writes
// initiated by this service have completed. Intended for graceful shutdown
// and for deterministic assertions in tests.
func (s *Service) WaitForBackgroundWrites() {
	s.bgWrites.Wait()
}

func TokenPreview() string {
	return tokenPrefix + "..."
}

func tokenHash(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
