package settings

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/alexpls/untils/internal/errortypes"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/session"
	"github.com/alexpls/untils/internal/validation"
	"github.com/alexpls/untils/internal/webhook"
	"github.com/starfederation/datastar-go/datastar"
)

// ViewWebhookSettings handles GET /app/settings/webhook
func (h *Handlers) ViewWebhookSettings(w http.ResponseWriter, r *http.Request, user *models.User) {
	comp, err := h.webhookSettingsComponent(r.Context(), user, nil, webhookTestData{})
	if err != nil {
		h.internalServerError(w, r, "building webhook settings component", err)
		return
	}
	if err := comp.Render(r.Context(), w); err != nil {
		h.internalServerError(w, r, "rendering webhook settings component", err)
		return
	}
}

// CreateWebhook handles POST /app/settings/webhook
func (h *Handlers) CreateWebhook(w http.ResponseWriter, r *http.Request, user *models.User) {
	if err := r.ParseForm(); err != nil {
		h.internalServerError(w, r, "error parsing form", err)
		return
	}

	if err := h.webhook.CreateWebhookTarget(r.Context(), webhook.CreateWebhookTargetParams{
		UserID: user.ID,
		URL:    r.FormValue("url"),
	}); err != nil {
		if validationErrs := validation.MapValidationErrors(err); validationErrs != nil {
			comp, err := h.webhookSettingsComponent(r.Context(), user, validationErrs, webhookTestData{})
			if err != nil {
				h.internalServerError(w, r, "building webhook settings component", err)
				return
			}
			if err := comp.Render(r.Context(), w); err != nil {
				h.internalServerError(w, r, "rendering webhook settings component", err)
				return
			}
			return
		}

		if !errors.Is(err, webhook.ErrWebhookTargetAlreadyExists) {
			h.internalServerError(w, r, "creating webhook target", err)
			return
		}
	}

	if err := h.sessionManager.SetFlash(r, session.FlashTypeAlert, "Webhook added"); err != nil {
		h.internalServerError(w, r, "saving session", err)
		return
	}

	http.Redirect(w, r, "/app/settings/webhook", http.StatusSeeOther)
}

// DeleteWebhook handles DELETE /app/settings/webhook/:webhook_id
func (h *Handlers) DeleteWebhook(w http.ResponseWriter, r *http.Request, user *models.User) {
	whID := h.webhookIDPathValue(w, r)
	if whID == 0 {
		return
	}

	if err := h.webhook.DeleteWebhookTarget(r.Context(), user.ID, whID); err != nil {
		h.internalServerError(w, r, "deleting webhook target", err)
		return
	}

	if err := h.sessionManager.SetFlash(r, session.FlashTypeAlert, "Webhook deleted"); err != nil {
		h.internalServerError(w, r, "saving session", err)
		return
	}

	sse := datastar.NewSSE(w, r)
	if err := sse.Redirect("/app/settings/webhook"); err != nil {
		h.logger.ErrorContext(sse.Context(), "redirecting after deleting webhook target", "error", err)
	}
}

// TestWebhook handles POST /app/settings/webhook/:webhook_id/test
func (h *Handlers) TestWebhook(w http.ResponseWriter, r *http.Request, user *models.User) {
	whID := h.webhookIDPathValue(w, r)
	if whID == 0 {
		return
	}

	testData := webhookTestData{Loading: true, TargetID: whID}
	sse := datastar.NewSSE(w, r)

	render := func() bool {
		comp, err := h.webhookSettingsComponent(r.Context(), user, nil, testData)
		if err != nil {
			h.internalServerError(w, r, "rendering webhook settings component", err)
			return false
		}

		if err = sse.PatchElementTempl(comp); err != nil {
			h.internalServerError(w, r, "patching webhook settings component", err)
			return false
		}
		return true
	}

	if !render() {
		return
	}

	resp, testErr := h.webhook.TestWebhookTarget(r.Context(), user.ID, whID)
	testData.Loading = false
	testData.Response = resp

	if testErr != nil {
		webhookError, ok := errors.AsType[*errortypes.ErrWebhookRequest](testErr)
		if ok {
			testData.Error = webhookError.Reason
		} else {
			testData.Error = "webhook request failed"
		}
	}

	if !render() {
		return
	}
}

func (h *Handlers) webhookSettingsComponent(ctx context.Context, user *models.User, validationErrs validation.ValidationErrors, testData webhookTestData) (templ.Component, error) {
	targets, err := h.webhook.ListWebhookTargets(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return WebhookSettings(&WebhookSettingsViewModel{
		Enabled:        h.capabilities.WebhookEnabled,
		Targets:        targets,
		ValidationErrs: validationErrs,
		TestData:       testData,
	}), nil
}

func (h *Handlers) webhookIDPathValue(w http.ResponseWriter, r *http.Request) int64 {
	whID, err := strconv.ParseInt(r.PathValue("webhook_id"), 10, 64)
	if err != nil {
		h.internalServerError(w, r, "converting id path param to int", err)
		return 0
	}
	return whID
}
