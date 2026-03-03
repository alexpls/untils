package reqcontext

import (
	"context"

	"github.com/alexpls/untils/internal/models"
)

type contextKey int

const (
	_ contextKey = iota
	buildVersionKey
	requestIDKey
	userKey
	tzKey
	envKey
	demoKey
	flashAlertKey
)

func ContextWithBuildVersion(ctx context.Context, buildVersion string) context.Context {
	return context.WithValue(ctx, buildVersionKey, buildVersion)
}

func BuildVersionFromContext(ctx context.Context) string {
	buildVersion, _ := ctx.Value(buildVersionKey).(string)
	return buildVersion
}

func ContextWithRequestID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, requestIDKey, reqID)
}

func RequestIDFromContext(ctx context.Context) (string, bool) {
	reqID, ok := ctx.Value(requestIDKey).(string)
	return reqID, ok
}

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

func EnvFromContext(ctx context.Context) string {
	env, _ := ctx.Value(envKey).(string)
	return env
}

func ContextWithDemo(ctx context.Context) context.Context {
	return context.WithValue(ctx, demoKey, true)
}

func DemoFromContext(ctx context.Context) bool {
	demo, _ := ctx.Value(demoKey).(bool)
	return demo
}

func ContextWithFlashAlert(ctx context.Context, message string) context.Context {
	return context.WithValue(ctx, flashAlertKey, message)
}

func FlashAlertFromContext(ctx context.Context) (string, bool) {
	message, ok := ctx.Value(flashAlertKey).(string)
	return message, ok
}
