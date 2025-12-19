package components

import (
	"context"
	"strings"
	"time"

	"github.com/alexpls/untils_go/internal/db/sqlc"
	"github.com/alexpls/untils_go/internal/reqcontext"
	"github.com/alexpls/untils_go/internal/validation"
	"github.com/alexpls/untils_go/public"
)

func IsSignedIn(ctx context.Context) bool {
	_, ok := reqcontext.UserFromContext(ctx)
	return ok
}

func CurrentUser(ctx context.Context) *sqlc.User {
	u, _ := reqcontext.UserFromContext(ctx)
	return u
}

func TimezoneFromCookie(ctx context.Context) string {
	tz, _ := reqcontext.TimezoneFromContext(ctx)
	return tz
}

func AssetURL(path string) string {
	return public.AssetURL(path)
}

func FormatDateTime(ctx context.Context, t time.Time) string {
	user, ok := reqcontext.UserFromContext(ctx)
	timezone := time.UTC
	if ok {
		if loc, err := time.LoadLocation(user.Timezone); err == nil {
			timezone = loc
		}
	}
	localTime := t.In(timezone)
	return localTime.Format("Jan 2, 2006 at 3:04 PM")
}

func FormatDate(ctx context.Context, t time.Time) string {
	user, ok := reqcontext.UserFromContext(ctx)
	timezone := time.UTC
	if ok {
		if loc, err := time.LoadLocation(user.Timezone); err == nil {
			timezone = loc
		}
	}
	localTime := t.In(timezone)
	return localTime.Format("Jan 2, 2006")
}

func ValidationError(data validation.HasValidationErrors, field string) string {
	for _, val := range data.GetValidationErrors() {
		if val.Field == field {
			return val.Message
		}
	}
	return ""
}

func MaskSecret(str string) string {
	visibleChars := 3
	maskChar := "•"

	if len(str) <= visibleChars {
		return strings.Repeat(maskChar, len(str))
	}

	visible := str[0:visibleChars]
	masked := strings.Repeat(maskChar, len(str)-visibleChars)

	return visible + masked
}
