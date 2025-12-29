package session

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/alexpls/untils_go/internal/db/sqlc"
	"github.com/jackc/pgx/v5"
)

const cookieName = "sid"
const sessionCtxKey = "sessionCtxKey"

type Manager struct {
	store  *store
	logger *slog.Logger
}

func NewManager(db sqlc.DBTX, queries *sqlc.Queries, logger *slog.Logger) *Manager {
	return &Manager{
		store: &store{
			db:      db,
			queries: queries,
		},
		logger: logger,
	}
}

func (sm *Manager) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, _ := r.Cookie(cookieName)
		var sessID string
		if cookie != nil {
			sessID = cookie.Value
		}

		ctx := r.Context()
		sess, err := sm.store.get(ctx, sessID)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				sm.logger.Error("error getting session", "error", err)
			}
		}

		newCtx := context.WithValue(ctx, sessionCtxKey, sess)
		rc := r.WithContext(newCtx)

		w.Header().Add("Vary", "Cookie")
		next.ServeHTTP(w, rc)
	})
}

func (sm *Manager) New(r *http.Request, w http.ResponseWriter) *Session {
	sess := FromRequest(r)
	sess.Reset()

	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    sess.ID,
		MaxAge:   sessionExpirySecs,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	w.Header().Add("Cache-Control", "no-store") // never cache cookie values
	http.SetCookie(w, cookie)

	return sess
}

func (sm *Manager) Save(r *http.Request) error {
	sess := FromRequest(r)
	if sess.ID == "" {
		panic("session must have ID")
	}
	err := sm.store.save(r.Context(), sess)
	if err != nil {
		return err
	}
	return nil
}

func (sm *Manager) Destroy(r *http.Request, w http.ResponseWriter) error {
	sess := FromRequest(r)
	err := sm.store.destroy(r.Context(), sess.ID)
	if err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name:   cookieName,
		MaxAge: -1,
	}

	w.Header().Add("Cache-Control", "no-store") // never cache cookie values
	http.SetCookie(w, cookie)
	return nil
}

func FromRequest(r *http.Request) *Session {
	sess, ok := r.Context().Value(sessionCtxKey).(*Session)
	if !ok {
		panic("no session in context")
	}
	return sess
}

func (sm *Manager) NewTrimWorker(logger *slog.Logger) *TrimWorker {
	return NewTrimWorker(sm.store, logger)
}
