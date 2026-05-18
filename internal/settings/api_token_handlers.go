package settings

import (
	"context"
	"errors"
	"net/http"

	"github.com/a-h/templ"
	"github.com/alexpls/untils/internal/api"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/session"
	"github.com/alexpls/untils/internal/validation"
	"github.com/starfederation/datastar-go/datastar"
)

// ViewAPITokenSettings handles GET /app/settings/api_tokens
func (h *Handlers) ViewAPITokenSettings(w http.ResponseWriter, r *http.Request, user *models.User) {
	comp, err := h.apiTokenSettingsComponent(r.Context(), user, nil, nil)
	if err != nil {
		h.internalServerError(w, r, "building api token settings component", err)
		return
	}
	if err := comp.Render(r.Context(), w); err != nil {
		h.internalServerError(w, r, "rendering api token settings component", err)
	}
}

// CreateAPIToken handles POST /app/settings/api_tokens
func (h *Handlers) CreateAPIToken(w http.ResponseWriter, r *http.Request, user *models.User) {
	if err := r.ParseForm(); err != nil {
		h.internalServerError(w, r, "error parsing form", err)
		return
	}

	created, err := h.api.CreateToken(r.Context(), api.CreateTokenParams{
		UserID: user.ID,
		Name:   r.FormValue("name"),
	})
	if err != nil {
		validationErrs := validation.MapValidationErrors(err)
		if validationErrs != nil {
			comp, compErr := h.apiTokenSettingsComponent(r.Context(), user, validationErrs, nil)
			if compErr != nil {
				h.internalServerError(w, r, "building api token settings component", compErr)
				return
			}
			if renderErr := comp.Render(r.Context(), w); renderErr != nil {
				h.internalServerError(w, r, "rendering api token settings component", renderErr)
			}
			return
		}

		h.internalServerError(w, r, "creating api token", err)
		return
	}

	comp, err := h.apiTokenSettingsComponent(r.Context(), user, nil, created)
	if err != nil {
		h.internalServerError(w, r, "building api token settings component", err)
		return
	}
	if err := comp.Render(r.Context(), w); err != nil {
		h.internalServerError(w, r, "rendering api token settings component", err)
	}
}

// DeleteAPIToken handles DELETE /app/settings/api_tokens/:token_id
func (h *Handlers) DeleteAPIToken(w http.ResponseWriter, r *http.Request, user *models.User) {
	tokenID := r.PathValue("token_id")
	if tokenID == "" {
		http.NotFound(w, r)
		return
	}

	if err := h.api.DeleteToken(r.Context(), user.ID, tokenID); err != nil {
		if errors.Is(err, api.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.internalServerError(w, r, "deleting api token", err)
		return
	}

	if err := h.sessionManager.SetFlash(r, session.FlashTypeAlert, "API token deleted"); err != nil {
		h.internalServerError(w, r, "saving session", err)
		return
	}

	sse := datastar.NewSSE(w, r)
	if err := sse.Redirect("/app/settings/api_tokens"); err != nil {
		h.logger.ErrorContext(sse.Context(), "redirecting after deleting api token", "error", err)
	}
}

func (h *Handlers) apiTokenSettingsComponent(ctx context.Context, user *models.User, validationErrs validation.ValidationErrors, created *api.CreatedToken) (templ.Component, error) {
	tokens, err := h.api.ListTokens(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return APITokenSettings(&APITokenSettingsViewModel{
		Tokens:         tokens,
		Created:        created,
		ValidationErrs: validationErrs,
	}), nil
}
