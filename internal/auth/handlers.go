package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/alexpls/untils/internal/reqcontext"
	"github.com/alexpls/untils/internal/session"
)

// Handlers contains HTTP handlers for auth routes
type Handlers struct {
	auth           *Auth
	sessionManager *session.Manager
	logger         *slog.Logger
}

// NewHandlers creates a new Handlers instance
func NewHandlers(auth *Auth, sessionManager *session.Manager, logger *slog.Logger) *Handlers {
	return &Handlers{
		auth:           auth,
		sessionManager: sessionManager,
		logger:         logger,
	}
}

// SignInGet handles GET /sign_in
func (h *Handlers) SignInGet(w http.ResponseWriter, r *http.Request) {
	if _, ok := reqcontext.UserFromContext(r.Context()); ok {
		http.Redirect(w, r, "/app", http.StatusSeeOther)
		return
	}

	ret := r.URL.Query().Get("return")
	data := SignInData{
		Return: ret,
	}
	if err := SignInPage(data).Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering sign in page", "error", err)
	}
}

// SignOutGet handles GET /sign_out
func (h *Handlers) SignOutGet(w http.ResponseWriter, r *http.Request) {
	err := h.sessionManager.Destroy(r, w)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error destroying session", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// SignInPost handles POST /sign_in
func (h *Handlers) SignInPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error parsing sign in form", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")
	ret := r.Form.Get("return")

	h.logger.InfoContext(r.Context(), "signing in", "email", email)

	user, err := h.auth.GetUserByEmailPassword(r.Context(), email, password)
	if errors.Is(err, ErrNoUser) {
		if err := SignInPage(SignInData{Failed: true, Email: email, Return: ret}).Render(r.Context(), w); err != nil {
			h.logger.ErrorContext(r.Context(), "error rendering sign in page", "error", err)
		}
		return
	}
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error processing sign in", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sess := h.sessionManager.New(r, w)
	sess.Data.UserID = user.ID
	if err = h.sessionManager.Save(r); err != nil {
		h.logger.ErrorContext(r.Context(), "error saving session", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if ret != "" {
		http.Redirect(w, r, ret, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/app", http.StatusSeeOther)
	}
}
