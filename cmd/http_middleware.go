package main

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/alexpls/untils/internal/logging"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/reqcontext"
	"github.com/alexpls/untils/internal/session"
)

type HandlerFuncWithUser func(http.ResponseWriter, *http.Request, *models.User)

// TODO: dump this in favor of more conventional http.Handler args
func (a *app) requireAuth(next HandlerFuncWithUser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess := session.FromRequest(r)

		// If demo mode is set, get user ID 1 and allow the request
		if reqcontext.DemoFromContext(r.Context()) {
			user, err := a.auth.GetUser(r.Context(), 1)
			if a.internalServerError(err, w) {
				a.logger.ErrorContext(r.Context(), "failed to get demo user", "user_id", 1)
				return
			}
			next(w, r, user)
			return
		}

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

func (a *app) allowDemo(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("demo") == "true" {
			ctx := reqcontext.ContextWithDemo(r.Context())
			next(w, r.WithContext(ctx))
			return
		}
		next(w, r)
	}
}

func (a *app) requireAuth2(next http.Handler) http.HandlerFunc {
	return a.requireAuth(func(w http.ResponseWriter, r *http.Request, _ *models.User) {
		next.ServeHTTP(w, r)
	})
}

func (a *app) setRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := reqcontext.RequestIDFromContext(r.Context()); ok {
			next.ServeHTTP(w, r)
			return
		}

		reqID := rand.Text()
		next.ServeHTTP(w, r.WithContext(reqcontext.ContextWithRequestID(r.Context(), reqID)))
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
				a.logger.ErrorContext(r.Context(), "tried to set user context on a missing user", "user_id", s.Data.UserID)
				return
			}
			next.ServeHTTP(w, r.WithContext(reqcontext.ContextWithUser(r.Context(), user)))
		}
	})
}

func (a *app) setFlashContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := session.FromRequest(r)
		alert := sess.Data.PopFlash(session.FlashTypeAlert)
		if alert != "" {
			if err := a.sessionManager.Save(r); err != nil {
				a.logger.ErrorContext(r.Context(), "error saving session after consuming flash", "error", err)
			}
			r = r.WithContext(reqcontext.ContextWithFlashAlert(r.Context(), alert))
		}

		next.ServeHTTP(w, r)
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
					a.logger.WarnContext(r.Context(), "invalid timezone in cookie", "tz", val)
				}
			}
		}

		next.ServeHTTP(w, r.WithContext(reqcontext.ContextWithTimezone(r.Context(), tz)))
	})
}

type HTTPLogEvent struct {
	Method     string
	URI        string
	Duration   time.Duration
	StatusCode int
	Error      string
}

func (h *HTTPLogEvent) Key() string {
	return "http"
}

func (h *HTTPLogEvent) SlogAttr() slog.Attr {
	return slog.Group(h.Key(),
		slog.String("method", h.Method),
		slog.String("uri", h.URI),
		slog.Int("status_code", h.StatusCode),
		slog.Duration("duration", h.Duration),
	)
}

func (a *app) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		events := logging.Events{}
		ctx2 := logging.ContextWithEvents(r.Context(), events)

		httpEvent := logging.GetOrCreate(events, func() *HTTPLogEvent {
			return &HTTPLogEvent{
				Method: r.Method,
				URI:    r.URL.RequestURI(),
			}
		})

		rec := &statusRecorder{
			ResponseWriter: w,
			Flusher:        w.(http.Flusher),
			status:         http.StatusOK,
		}

		next.ServeHTTP(rec, r.WithContext(ctx2))

		httpEvent.Duration = time.Since(start)
		httpEvent.StatusCode = rec.status

		if httpEvent.Error != "" {
			a.logger.LogAttrs(r.Context(), slog.LevelError, "http request", events.SlogAttrs()...)
		} else {
			a.logger.LogAttrs(r.Context(), slog.LevelInfo, "http request", events.SlogAttrs()...)
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	http.Flusher
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
