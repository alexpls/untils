package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/alexpls/untils/internal/api"
	"github.com/alexpls/untils/internal/auth"
	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/errortypes"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/notifications"
	"github.com/alexpls/untils/internal/pushover"
	"github.com/alexpls/untils/internal/session"
	"github.com/alexpls/untils/internal/validation"
	"github.com/alexpls/untils/internal/webhook"
	"github.com/starfederation/datastar-go/datastar"
)

// AuthService provides user update methods
type AuthService interface {
	UpdateUserTimezone(ctx context.Context, userID int64, params auth.UpdateUserTimezoneParams) error
	UpdateUserPassword(ctx context.Context, userID int64, params auth.UpdateUserPasswordParams) error
}

// Handlers contains HTTP handlers for settings routes
type Handlers struct {
	capabilities   notifications.Capabilities
	queries        *models.Queries
	db             db.DB
	pushoverStore  *pushover.Store
	pushoverClient *pushover.Client
	webhook        *webhook.Service
	api            *api.Service
	sessionManager *session.Manager
	auth           AuthService
	logger         *slog.Logger
}

type SettingsIntegrationViewData struct {
	Integration *models.UserIntegrationsRow
	// Whether the integration type is available. In self hosted mode
	// this depends on the config of the application.
	Enabled bool
}

func toSettingsIntegrations(integrations []*models.UserIntegrationsRow, capabilities notifications.Capabilities) []*SettingsIntegrationViewData {
	items := make([]*SettingsIntegrationViewData, 0, len(integrations))
	for _, integration := range integrations {
		items = append(items, &SettingsIntegrationViewData{
			Integration: integration,
			Enabled:     capabilities.Enabled(integration.Name),
		})
	}
	return items
}

// NewHandlers creates a new Handlers instance
func NewHandlers(
	queries *models.Queries,
	db db.DB,
	capabilities notifications.Capabilities,
	pushoverStore *pushover.Store,
	pushoverClient *pushover.Client,
	sessionManager *session.Manager,
	auth AuthService,
	webhook *webhook.Service,
	api *api.Service,
	logger *slog.Logger,
) *Handlers {
	return &Handlers{
		capabilities:   capabilities,
		queries:        queries,
		db:             db,
		pushoverStore:  pushoverStore,
		pushoverClient: pushoverClient,
		sessionManager: sessionManager,
		auth:           auth,
		webhook:        webhook,
		api:            api,
		logger:         logger,
	}
}

func (h *Handlers) internalServerError(w http.ResponseWriter, r *http.Request, msg string, err error) {
	h.logger.ErrorContext(r.Context(), msg, "error", err)
	errortypes.InternalServerError(w)
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
		Integrations: toSettingsIntegrations(integrations, h.capabilities),
	}).Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering settings component", "error", err)
	}
}

// ViewPasswordSettings handles GET /app/settings/password
func (h *Handlers) ViewPasswordSettings(w http.ResponseWriter, r *http.Request, user *models.User) {
	if err := ChangePasswordSettings(&ChangePasswordViewModel{}).Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering change password component", "error", err)
	}
}

// UpdatePassword handles POST /app/settings/password
func (h *Handlers) UpdatePassword(w http.ResponseWriter, r *http.Request, user *models.User) {
	if err := r.ParseForm(); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to parse form", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	params := auth.UpdateUserPasswordParams{
		CurrentPassword:         r.FormValue("CurrentPassword"),
		NewPassword:             r.FormValue("NewPassword"),
		NewPasswordConfirmation: r.FormValue("NewPasswordConfirmation"),
	}

	err := h.auth.UpdateUserPassword(r.Context(), user.ID, params)
	if err != nil {
		var validationErrs validation.ValidationErrors

		switch {
		case errors.Is(err, auth.ErrCurrentPasswordIncorrect):
			validationErrs = validation.ValidationErrors{
				{
					Field:   "CurrentPassword",
					Message: "Current password is incorrect",
				},
			}
		case errors.Is(err, auth.ErrPasswordConfirmationMismatch):
			validationErrs = validation.ValidationErrors{
				{
					Field:   "NewPasswordConfirmation",
					Message: "Password confirmation does not match",
				},
			}
		default:
			validationErrs = validation.MapValidationErrors(err)
		}

		if validationErrs != nil {
			if renderErr := ChangePasswordSettings(&ChangePasswordViewModel{
				ValidationErrors: validationErrs,
			}).Render(r.Context(), w); renderErr != nil {
				h.logger.ErrorContext(r.Context(), "error rendering settings component", "error", renderErr)
			}
			return
		}

		h.logger.ErrorContext(r.Context(), "error updating user password", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.sessionManager.SetFlash(r, session.FlashTypeAlert, "Password changed."); err != nil {
		h.internalServerError(w, r, "error saving session with password update flash", err)
		return
	}

	http.Redirect(w, r, "/app/settings", http.StatusSeeOther)
}

// ViewPushoverSettings handles GET /app/settings/pushover
func (h *Handlers) ViewPushoverSettings(w http.ResponseWriter, r *http.Request, user *models.User) {
	if !h.capabilities.PushoverEnabled {
		if err := PushoverSettings(&PushoverSettingsViewModel{
			Enabled: false,
		}).Render(r.Context(), w); err != nil {
			h.logger.ErrorContext(r.Context(), "error rendering pushover settings component", "error", err)
		}
		return
	}

	tok, err := h.pushoverStore.GetToken(r.Context(), user.ID)
	if err != nil {
		if !errors.Is(err, &errortypes.ErrNoPushoverUserToken{}) {
			h.logger.ErrorContext(r.Context(), "error getting pushover token", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	data := PushoverSettingsViewModel{
		Enabled: true,
		Token:   nil,
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
	if !h.capabilities.PushoverEnabled {
		http.Error(w, "pushover is not configured", http.StatusBadRequest)
		return
	}

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
				Enabled:          true,
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
				Enabled:          true,
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
	if !h.capabilities.PushoverEnabled {
		http.Error(w, "pushover is not configured", http.StatusBadRequest)
		return
	}

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
		Enabled: h.capabilities.EmailEnabled,
		Email:   user.Email,
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
