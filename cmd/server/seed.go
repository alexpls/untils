package main

import (
	"context"
	"errors"

	"github.com/alexpls/untils_go/internal/auth"
	"github.com/alexpls/untils_go/internal/must"
)

func (a *app) seed() {
	must.True(a.config.env == "dev")

	ctx := context.Background()

	_, err := a.auth.CreateUser(ctx, "alexpls@fastmail.com", "abc123", "Australia/Brisbane")
	if err != nil {
		if errors.Is(err, auth.ErrUserExists) {
			return
		}
		panic(err)
	}
}
