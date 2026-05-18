package reqcontext

import (
	"context"

	"github.com/alexpls/untils/internal/constants"
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
	plausibleSnippetTagKey
	flashAlertKey
	baseURLKey
	apiTokenKey
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

func ContextWithEnv(ctx context.Context, env constants.Env) context.Context {
	return context.WithValue(ctx, envKey, env)
}

func EnvFromContext(ctx context.Context) constants.Env {
	env, _ := ctx.Value(envKey).(constants.Env)
	return env
}

func ContextWithDemo(ctx context.Context) context.Context {
	return context.WithValue(ctx, demoKey, true)
}

func DemoFromContext(ctx context.Context) bool {
	demo, _ := ctx.Value(demoKey).(bool)
	return demo
}

func ContextWithPlausibleSnippetTag(ctx context.Context, plausibleSnippetTag string) context.Context {
	return context.WithValue(ctx, plausibleSnippetTagKey, plausibleSnippetTag)
}

func PlausibleSnippetTagFromContext(ctx context.Context) string {
	v, _ := ctx.Value(plausibleSnippetTagKey).(string)
	return v
}

func ContextWithFlashAlert(ctx context.Context, message string) context.Context {
	return context.WithValue(ctx, flashAlertKey, message)
}

func FlashAlertFromContext(ctx context.Context) (string, bool) {
	message, ok := ctx.Value(flashAlertKey).(string)
	return message, ok
}

func ContextWithBaseURL(ctx context.Context, baseURL string) context.Context {
	return context.WithValue(ctx, baseURLKey, baseURL)
}

func BaseURLFromContext(ctx context.Context) string {
	baseURL, _ := ctx.Value(baseURLKey).(string)
	return baseURL
}

func ContextWithAPIToken(ctx context.Context, token *models.ApiToken) context.Context {
	return context.WithValue(ctx, apiTokenKey, token)
}

func APITokenFromContext(ctx context.Context) (*models.ApiToken, bool) {
	token, ok := ctx.Value(apiTokenKey).(*models.ApiToken)
	return token, ok && token != nil
}
