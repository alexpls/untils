package reqcontext

import (
	"context"

	"github.com/alexpls/untils/internal/db/sqlc"
)

type contextKey int

const (
	_ contextKey = iota
	userKey
	tzKey
)

func ContextWithUser(ctx context.Context, user *sqlc.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

func UserFromContext(ctx context.Context) (*sqlc.User, bool) {
	user, ok := ctx.Value(userKey).(*sqlc.User)
	return user, ok
}

func ContextWithTimezone(ctx context.Context, tz string) context.Context {
	return context.WithValue(ctx, tzKey, tz)
}

func TimezoneFromContext(ctx context.Context) (string, bool) {
	tz, ok := ctx.Value(tzKey).(string)
	return tz, ok
}
