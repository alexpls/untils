package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/alexpls/untils/internal/auth"
	"github.com/alexpls/untils/internal/db/models"
	"github.com/alexpls/untils/internal/pushover"
	"github.com/alexpls/untils/internal/validation"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/starfederation/datastar-go/datastar"
)

// AuthService provides user update methods
type AuthService interface {
	UpdateUserTimezone(ctx context.Context, userID int64, params auth.UpdateUserTimezoneParams) error
}

// Handlers contains HTTP handlers for settings routes
type Handlers struct {
	queries        *models.Queries
	pool           *pgxpool.Pool
	pushoverStore  *pushover.Store
	pushoverClient *pushover.Client
	auth           AuthService
	logger         *slog.Logger
}

// NewHandlers creates a new Handlers instance
func NewHandlers(
	queries *models.Queries,
	pool *pgxpool.Pool,
	pushoverStore *pushover.Store,
	pushoverClient *pushover.Client,
	auth AuthService,
	logger *slog.Logger,
) *Handlers {
	return &Handlers{
		queries:        queries,
		pool:           pool,
		pushoverStore:  pushoverStore,
		pushoverClient: pushoverClient,
		auth:           auth,
		logger:         logger,
	}
}

// SettingsGet handles GET /app/settings
func (h *Handlers) SettingsGet(w http.ResponseWriter, r *http.Request, user *models.User) {
	integrations, err := h.queries.UserIntegrations(r.Context(), h.pool, user.ID)
	if err != nil {
		h.logger.Error("error listing integrations", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	Settings(&SettingsViewModel{
		ConfiguredIntegrations: integrations,
	}).Render(r.Context(), w)
}

// PushoverSettingsGet handles GET /app/settings/pushover
func (h *Handlers) PushoverSettingsGet(w http.ResponseWriter, r *http.Request, user *models.User) {
	tok, err := h.pushoverStore.GetToken(r.Context(), user.ID)
	if err != nil {
		if !errors.Is(err, pushover.ErrNoPushoverUserToken) {
			h.logger.Error("error getting pushover token", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	data := PushoverSettingsViewModel{
		Token: nil,
		Values: pushover.CreateOrUpdateTokenParams{
			Token: "",
		},
	}

	if tok != nil {
		data.Token = tok
		data.Values.Token = tok.Token
	}

	PushoverSettings(&data).Render(r.Context(), w)
}

// PushoverSettingsPost handles POST /app/settings/pushover
func (h *Handlers) PushoverSettingsPost(w http.ResponseWriter, r *http.Request, user *models.User) {
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	params := pushover.CreateOrUpdateTokenParams{
		Token: r.FormValue("Token"),
	}

	err := h.pushoverClient.Validate(r.Context(), params.Token)
	if err != nil {
		var validationErr *pushover.ErrInvalidToken
		if errors.As(err, &validationErr) {
			mapped := validation.ValidationError{
				Field:   "Token",
				Message: fmt.Sprintf("Failed to validate with Pushover: %s", strings.Join(validationErr.Reasons, ", ")),
			}
			PushoverSettings(&PushoverSettingsViewModel{
				Values:           params,
				ValidationErrors: validation.ValidationErrors{mapped},
			}).Render(r.Context(), w)
			return
		}
	}

	_, err = h.pushoverStore.CreateOrUpdateToken(r.Context(), user.ID, params)
	if err != nil {
		if validationErrs := validation.MapValidationErrors(err); validationErrs != nil {
			h.logger.Warn("failed validation when creating pushover token", "validation_errors", validationErrs)
			PushoverSettings(&PushoverSettingsViewModel{
				Values:           params,
				ValidationErrors: validationErrs,
			}).Render(r.Context(), w)
			return
		}
		h.logger.Error("error creating pushover token", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/app/settings/pushover", http.StatusSeeOther)
}

// PushoverSettingsDelete handles DELETE /app/settings/pushover
func (h *Handlers) PushoverSettingsDelete(w http.ResponseWriter, r *http.Request, user *models.User) {
	sse := datastar.NewSSE(w, r)

	err := h.pushoverStore.DeleteToken(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("error deleting pushover user token", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse.Redirect("/app/settings/pushover")
}

// EmailSettingsGet handles GET /app/settings/email
func (h *Handlers) EmailSettingsGet(w http.ResponseWriter, r *http.Request, user *models.User) {
	EmailSettings(&EmailSettingsViewModel{
		Email: user.Email,
	}).Render(r.Context(), w)
}

type timezoneUpdate struct {
	Timezone string `json:"timezone"`
}

// UpdateTimezonePost handles POST /app/settings/timezone
func (h *Handlers) UpdateTimezonePost(w http.ResponseWriter, r *http.Request, user *models.User) {
	var params timezoneUpdate
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		h.logger.Error("error decoding timezone update params", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	err = h.auth.UpdateUserTimezone(r.Context(), user.ID, auth.UpdateUserTimezoneParams{
		Timezone: params.Timezone,
	})
	if err != nil {
		h.logger.Error("error updating user timezone", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
