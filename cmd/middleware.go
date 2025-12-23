package main

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/alexpls/untils_go/internal/db/sqlc"
	"github.com/alexpls/untils_go/internal/reqcontext"
	"github.com/alexpls/untils_go/internal/session"
)

type HandlerFuncWithUser func(http.ResponseWriter, *http.Request, *sqlc.User)

// TODO: dump this in favor of more conventional http.Handler args
func (a *app) requireAuth(next HandlerFuncWithUser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess := session.FromRequest(r)
		if sess.Data.IsSignedIn() {
			user, ok := reqcontext.UserFromContext(r.Context())
			if !ok {
				a.internalServerError(fmt.Errorf("can't find signed in user in context"), w)
			} else {
				next(w, r, user)
			}
		} else {
			ret := url.QueryEscape(r.URL.String())
			http.Redirect(w, r, "/sign_in?return="+ret, http.StatusSeeOther)
		}
	}
}

func (a *app) requireAuth2(next http.Handler) http.HandlerFunc {
	return a.requireAuth(func(w http.ResponseWriter, r *http.Request, _ *sqlc.User) {
		next.ServeHTTP(w, r)
	})
}

func (a *app) setUserContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := session.FromRequest(r)
		if s == nil || s.Data.UserID == 0 {
			next.ServeHTTP(w, r)
		} else {
			user, err := a.auth.GetUser(r.Context(), s.Data.UserID)
			if a.internalServerError(err, w) {
				a.logger.Error("tried to set user context on a missing user", "user_id", s.Data.UserID)
				return
			}
			next.ServeHTTP(w, r.WithContext(reqcontext.ContextWithUser(r.Context(), user)))
		}
	})
}

func (a *app) setTimezoneContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tz string

		if cookie, err := r.Cookie("tz"); err == nil && cookie.Value != "" {
			if val, err := url.QueryUnescape(cookie.Value); err == nil {
				if loc, err := time.LoadLocation(val); err == nil {
					tz = loc.String()
				} else {
					a.logger.Warn("invalid timezone in cookie", "tz", val)
				}
			}
		}

		next.ServeHTTP(w, r.WithContext(reqcontext.ContextWithTimezone(r.Context(), tz)))
	})
}

func (a *app) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		a.logger.Info("http request",
			"method", r.Method,
			"uri", r.URL.RequestURI())

		rec := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(rec, r)

		duration := time.Since(start)
		ms := float64(duration) / float64(time.Millisecond)
		a.logger.Info("http request done",
			"status", rec.status,
			"took", fmt.Sprintf("%.3fms", ms))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func applyMiddleware(h http.Handler, handlers ...func(http.Handler) http.Handler) http.Handler {
	for i := len(handlers) - 1; i >= 0; i-- {
		h = handlers[i](h)
	}
	return h
}
