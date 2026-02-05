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
	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/errortypes"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/pushover"
	"github.com/alexpls/untils/internal/validation"
	"github.com/starfederation/datastar-go/datastar"
)

// AuthService provides user update methods
type AuthService interface {
	UpdateUserTimezone(ctx context.Context, userID int64, params auth.UpdateUserTimezoneParams) error
}

// Handlers contains HTTP handlers for settings routes
type Handlers struct {
	queries        *models.Queries
	db             db.DB
	pushoverStore  *pushover.Store
	pushoverClient *pushover.Client
	auth           AuthService
	logger         *slog.Logger
}

// NewHandlers creates a new Handlers instance
func NewHandlers(
	queries *models.Queries,
	db db.DB,
	pushoverStore *pushover.Store,
	pushoverClient *pushover.Client,
	auth AuthService,
	logger *slog.Logger,
) *Handlers {
	return &Handlers{
		queries:        queries,
		db:             db,
		pushoverStore:  pushoverStore,
		pushoverClient: pushoverClient,
		auth:           auth,
		logger:         logger,
	}
}

// ViewSettings handles GET /app/settings
func (h *Handlers) ViewSettings(w http.ResponseWriter, r *http.Request, user *models.User) {
	integrations, err := h.queries.UserIntegrations(r.Context(), h.db, user.ID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error listing integrations", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := Settings(&SettingsViewModel{
		ConfiguredIntegrations: integrations,
	}).Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering settings component", "error", err)
	}
}

// ViewPushoverSettings handles GET /app/settings/pushover
func (h *Handlers) ViewPushoverSettings(w http.ResponseWriter, r *http.Request, user *models.User) {
	tok, err := h.pushoverStore.GetToken(r.Context(), user.ID)
	if err != nil {
		if !errors.Is(err, &errortypes.ErrNoPushoverUserToken{}) {
			h.logger.ErrorContext(r.Context(), "error getting pushover token", "error", err)
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

	if err := PushoverSettings(&data).Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering pushover settings component", "error", err)
	}
}

// UpdatePushoverSettings handles POST /app/settings/pushover
func (h *Handlers) UpdatePushoverSettings(w http.ResponseWriter, r *http.Request, user *models.User) {
	if err := r.ParseForm(); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to parse form", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	params := pushover.CreateOrUpdateTokenParams{
		Token: r.FormValue("Token"),
	}

	err := h.pushoverClient.Validate(r.Context(), params.Token)
	if err != nil {
		var validationErr *errortypes.ErrInvalidToken
		if errors.As(err, &validationErr) {
			mapped := validation.ValidationError{
				Field:   "Token",
				Message: fmt.Sprintf("Failed to validate with Pushover: %s", strings.Join(validationErr.Reasons, ", ")),
			}
			if err := PushoverSettings(&PushoverSettingsViewModel{
				Values:           params,
				ValidationErrors: validation.ValidationErrors{mapped},
			}).Render(r.Context(), w); err != nil {
				h.logger.ErrorContext(r.Context(), "error rendering pushover settings component", "error", err)
			}
			return
		}
	}

	_, err = h.pushoverStore.CreateOrUpdateToken(r.Context(), user.ID, params)
	if err != nil {
		if validationErrs := validation.MapValidationErrors(err); validationErrs != nil {
			h.logger.WarnContext(r.Context(), "failed validation when creating pushover token", "validation_errors", validationErrs)
			if err := PushoverSettings(&PushoverSettingsViewModel{
				Values:           params,
				ValidationErrors: validationErrs,
			}).Render(r.Context(), w); err != nil {
				h.logger.ErrorContext(r.Context(), "error rendering pushover settings component", "error", err)
			}
			return
		}
		h.logger.ErrorContext(r.Context(), "error creating pushover token", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/app/settings/pushover", http.StatusSeeOther)
}

// DeletePushoverSettings handles DELETE /app/settings/pushover
func (h *Handlers) DeletePushoverSettings(w http.ResponseWriter, r *http.Request, user *models.User) {
	sse := datastar.NewSSE(w, r)

	err := h.pushoverStore.DeleteToken(r.Context(), user.ID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error deleting pushover user token", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err = sse.Redirect("/app/settings/pushover"); err != nil {
		h.logger.ErrorContext(sse.Context(), "error redirecting after deleting pushover user token", "error", err)
	}
}

// ViewEmailSettings handles GET /app/settings/email
func (h *Handlers) ViewEmailSettings(w http.ResponseWriter, r *http.Request, user *models.User) {
	if err := EmailSettings(&EmailSettingsViewModel{
		Email: user.Email,
	}).Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering email settings component", "error", err)
	}
}

type timezoneUpdate struct {
	Timezone string `json:"timezone"`
}

// UpdateTimezone handles POST /app/settings/timezone
func (h *Handlers) UpdateTimezone(w http.ResponseWriter, r *http.Request, user *models.User) {
	var params timezoneUpdate
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error decoding timezone update params", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	err = h.auth.UpdateUserTimezone(r.Context(), user.ID, auth.UpdateUserTimezoneParams{
		Timezone: params.Timezone,
	})
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error updating user timezone", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
