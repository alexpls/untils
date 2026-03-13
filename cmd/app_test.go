package main

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/alexpls/untils/internal/auth"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/testhelper"
	"github.com/go-playground/validator/v10"
)

func TestBootstrapInitialSelfHostedAdminCreatesFirstUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := testhelper.TestTx(ctx, t)
	queries := models.New()
	a := &app{
		config: &config{
			appMode:    appModeSelfHosted,
			adminEmail: "admin@example.com",
		},
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		queries:  queries,
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
	a.auth = auth.NewAuth(a.logger, db, a.queries, a.validate)

	err := a.bootstrapInitialSelfHostedAdmin(ctx, db)
	if err != nil {
		t.Fatalf("bootstrapInitialSelfHostedAdmin returned error: %v", err)
	}

	userCount, err := queries.CountUsers(ctx, db)
	if err != nil {
		t.Fatalf("CountUsers returned error: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("got userCount %d, want 1", userCount)
	}

	user, err := queries.GetUserByEmail(ctx, db, "admin@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail returned error: %v", err)
	}
	if user.Timezone != "UTC" {
		t.Fatalf("got timezone %q, want %q", user.Timezone, "UTC")
	}

	authedUser, err := a.auth.GetUserByEmailPassword(ctx, "admin@example.com", "abc123")
	if err != nil {
		t.Fatalf("GetUserByEmailPassword returned error: %v", err)
	}
	if authedUser.ID != user.ID {
		t.Fatalf("got authenticated user id %d, want %d", authedUser.ID, user.ID)
	}
}

func TestBootstrapInitialSelfHostedAdminSkipsWhenUsersExist(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := testhelper.TestTx(ctx, t)
	queries := models.New()
	a := &app{
		config: &config{
			appMode:    appModeSelfHosted,
			adminEmail: "admin@example.com",
		},
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		queries:  queries,
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
	a.auth = auth.NewAuth(a.logger, db, a.queries, a.validate)

	existingUser, err := a.auth.CreateUser(ctx, "existing@example.com", "existing-password", "UTC")
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	err = a.bootstrapInitialSelfHostedAdmin(ctx, db)
	if err != nil {
		t.Fatalf("bootstrapInitialSelfHostedAdmin returned error: %v", err)
	}

	userCount, err := queries.CountUsers(ctx, db)
	if err != nil {
		t.Fatalf("CountUsers returned error: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("got userCount %d, want 1", userCount)
	}

	user, err := queries.GetUserByEmail(ctx, db, "existing@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail returned error: %v", err)
	}
	if user.ID != existingUser.ID {
		t.Fatalf("got user id %d, want %d", user.ID, existingUser.ID)
	}

	_, err = queries.GetUserByEmail(ctx, db, "admin@example.com")
	if err == nil {
		t.Fatal("expected admin bootstrap user to be absent")
	}
}
