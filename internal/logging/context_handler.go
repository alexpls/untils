package logging

import (
	"context"
	"log/slog"

	"github.com/alexpls/untils/internal/reqcontext"
)

type ContextHandler struct {
	slog.Handler
}

func (h ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	r.AddAttrs(slog.String("build_version", reqcontext.BuildVersionFromContext(ctx)))
	r.AddAttrs(slog.String("env", reqcontext.EnvFromContext(ctx)))

	if reqID, ok := reqcontext.RequestIDFromContext(ctx); ok {
		r.AddAttrs(slog.String("request_id", reqID))
	}
	if user, ok := reqcontext.UserFromContext(ctx); ok {
		r.AddAttrs(slog.Int64("user_id", user.ID))
	}

	return h.Handler.Handle(ctx, r)
}

func (h ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return ContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h ContextHandler) WithGroup(name string) slog.Handler {
	return ContextHandler{Handler: h.Handler.WithGroup(name)}
}
