package reqcontext

import (
	"context"

	"github.com/alexpls/untils/internal/db/models"
)

type contextKey int

const (
	_ contextKey = iota
	userKey
	tzKey
	envKey
)

func ContextWithUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

func UserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(userKey).(*models.User)
	return user, ok
}

func ContextWithTimezone(ctx context.Context, tz string) context.Context {
	return context.WithValue(ctx, tzKey, tz)
}

func TimezoneFromContext(ctx context.Context) (string, bool) {
	tz, ok := ctx.Value(tzKey).(string)
	return tz, ok
}

func ContextWithEnv(ctx context.Context, env string) context.Context {
	return context.WithValue(ctx, envKey, env)
}

func EnvFromContext(ctx context.Context) (string, bool) {
	env, ok := ctx.Value(envKey).(string)
	return env, ok
}
