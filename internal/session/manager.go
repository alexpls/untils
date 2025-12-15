package session

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexpls/untils_go/internal/db/sqlc"
	"github.com/jackc/pgx/v5"
)

const cookieName = "sid"
const sessionCtxKey = "sessionCtxKey"
const trimInterval = time.Hour

type Manager struct {
	store  *store
	logger *slog.Logger

	trimStopCh chan struct{}
	trimDoneCh chan struct{}
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

func (sm *Manager) StartTrim() {
	sm.logger.Info("starting session trimmer")

	if sm.trimStopCh != nil {
		sm.logger.Warn("session trimming already running")
		return
	}

	sm.trimStopCh = make(chan struct{})
	sm.trimDoneCh = make(chan struct{})

	go func() {
		defer close(sm.trimDoneCh)
		ticker := time.NewTicker(trimInterval)
		defer ticker.Stop()

		for {
			numTrimmed, err := sm.store.trim()
			if err != nil {
				sm.logger.Error("error trimming sessions", "error", err)
			} else {
				sm.logger.Info("trimmed sessions", "num_trimmed", numTrimmed)
			}

			select {
			case <-ticker.C:
				continue
			case <-sm.trimStopCh:
				return
			}
		}
	}()
}

func (sm *Manager) StopTrim() {
	if sm.trimStopCh == nil {
		return
	}
	close(sm.trimStopCh)
	<-sm.trimDoneCh
	sm.trimStopCh = nil
	sm.trimDoneCh = nil
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
