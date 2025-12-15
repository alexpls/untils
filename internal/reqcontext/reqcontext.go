package reqcontext

import (
	"context"

	"github.com/alexpls/untils_go/internal/db/sqlc"
)

type contextKey int

const (
	_ contextKey = iota
	userKey
)

func ContextWithUser(ctx context.Context, user *sqlc.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

func UserFromContext(ctx context.Context) (*sqlc.User, bool) {
	user, ok := ctx.Value(userKey).(*sqlc.User)
	return user, ok
}
