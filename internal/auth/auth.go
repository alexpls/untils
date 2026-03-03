package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/alexpls/untils/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Auth struct {
	logger   *slog.Logger
	db       models.DBTX
	queries  *models.Queries
	validate *validator.Validate
}

func NewAuth(logger *slog.Logger, db models.DBTX, queries *models.Queries, validate *validator.Validate) *Auth {
	return &Auth{
		logger:   logger,
		db:       db,
		queries:  queries,
		validate: validate,
	}
}

var (
	ErrNoUser                       = errors.New("no user found")
	ErrUserExists                   = errors.New("user already exists")
	ErrCurrentPasswordIncorrect     = errors.New("current password is incorrect")
	ErrPasswordConfirmationMismatch = errors.New("password confirmation does not match")
)

func (a *Auth) CreateUser(ctx context.Context, email string, password string, timezone string) (*models.User, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	now := time.Now()
	user, err := a.queries.CreateUser(ctx, a.db, &models.CreateUserParams{
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
		Timezone:     timezone,
	})
	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) && pgerr.Code == pgerrcode.UniqueViolation {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return user, nil
}

func (a *Auth) GetUserByEmailPassword(ctx context.Context, email string, password string) (*models.User, error) {
	user, err := a.queries.GetUserByEmail(ctx, a.db, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoUser
		}
		return nil, fmt.Errorf("finding user: %w", err)
	}

	match, err := argon2id.ComparePasswordAndHash(password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("checking password hash: %w", err)
	}
	if !match {
		return nil, ErrNoUser
	}

	return user, nil
}

func (a *Auth) GetUser(ctx context.Context, id int64) (*models.User, error) {
	user, err := a.queries.GetUser(ctx, a.db, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoUser
		}
		return nil, fmt.Errorf("finding user by id: %w", err)
	}
	return user, nil
}

type UpdateUserTimezoneParams struct {
	Timezone string `validate:"required,timezone"`
}

func (a *Auth) UpdateUserTimezone(ctx context.Context, id int64, params UpdateUserTimezoneParams) error {
	if err := a.validate.Struct(params); err != nil {
		return err
	}

	_, err := a.queries.UpdateUserTimezone(ctx, a.db, &models.UpdateUserTimezoneParams{
		UserID:   id,
		Timezone: params.Timezone,
	})
	if err != nil {
		return fmt.Errorf("updating user timezone: %w", err)
	}
	return nil
}

type UpdateUserPasswordParams struct {
	CurrentPassword         string `validate:"required"`
	NewPassword             string `validate:"required"`
	NewPasswordConfirmation string `validate:"required"`
}

func (a *Auth) UpdateUserPassword(ctx context.Context, id int64, params UpdateUserPasswordParams) error {
	if err := a.validate.Struct(params); err != nil {
		return err
	}

	if params.NewPassword != params.NewPasswordConfirmation {
		return ErrPasswordConfirmationMismatch
	}

	user, err := a.queries.GetUser(ctx, a.db, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNoUser
		}
		return fmt.Errorf("finding user by id: %w", err)
	}

	match, err := argon2id.ComparePasswordAndHash(params.CurrentPassword, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("checking password hash: %w", err)
	}
	if !match {
		return ErrCurrentPasswordIncorrect
	}

	newHash, err := argon2id.CreateHash(params.NewPassword, argon2id.DefaultParams)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	_, err = a.queries.UpdateUserPasswordHash(ctx, a.db, &models.UpdateUserPasswordHashParams{
		PasswordHash: newHash,
		UserID:       id,
	})
	if err != nil {
		return fmt.Errorf("updating user password hash: %w", err)
	}

	return nil
}
