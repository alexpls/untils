package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	appcomponents "github.com/alexpls/untils_go/internal/components/app"
	"github.com/alexpls/untils_go/internal/db/sqlc"
	"github.com/alexpls/untils_go/internal/pushover"
	"github.com/alexpls/untils_go/internal/validation"
)

func (a *app) settingsGet(w http.ResponseWriter, r *http.Request, user *sqlc.User) {
	activeIntegrations, err := a.userSettings.ActiveIntegrations(r.Context(), user.ID)
	if a.internalServerError(err, w) {
		a.logger.Error("error listing active integrations", "error", err)
		return
	}
	appcomponents.Settings(&appcomponents.SettingsViewModel{
		ActiveIntegrations: activeIntegrations,
	}).Render(r.Context(), w)
}

func renderPushoverSettings(data appcomponents.PushoverSettingsViewModel, r *http.Request, w http.ResponseWriter) {
	appcomponents.PushoverSettings(&data).Render(r.Context(), w)
}

func (a *app) pushoverSettingsGet(w http.ResponseWriter, r *http.Request, user *sqlc.User) {
	data := appcomponents.PushoverSettingsViewModel{
		Token: nil,
		Values: pushover.CreateOrUpdateTokenParams{
			Token: "",
		},
	}

	tok, err := a.pushoverStore.GetToken(r.Context(), user.ID)
	if err != nil {
		if !errors.Is(err, pushover.ErrNoPushoverUserToken) {
			a.internalServerError(err, w)
			return
		}
	}

	if tok != nil {
		data.Token = tok
		data.Values.Token = tok.Token
	}

	renderPushoverSettings(data, r, w)
}

func (a *app) pushoverSettingsPost(w http.ResponseWriter, r *http.Request, user *sqlc.User) {
	if a.internalServerError(r.ParseForm(), w) {
		a.logger.Error("failed to parse form")
		return
	}

	params := pushover.CreateOrUpdateTokenParams{
		Token: r.FormValue("Token"),
	}

	err := a.pushoverClient.Validate(r.Context(), params.Token)
	if err != nil {
		var validationErr *pushover.ErrInvalidToken
		if errors.As(err, &validationErr) {
			mapped := validation.ValidationError{
				Field:   "Token",
				Message: fmt.Sprintf("Failed to validate with Pushover: %s", strings.Join(validationErr.Reasons, ", ")),
			}
			renderPushoverSettings(appcomponents.PushoverSettingsViewModel{
				Values:           params,
				ValidationErrors: validation.ValidationErrors{mapped},
			}, r, w)
			return
		}
	}

	_, err = a.pushoverStore.CreateOrUpdateToken(r.Context(), user.ID, params)
	if err != nil {
		if validationErrs := validation.MapValidationErrors(err); validationErrs != nil {
			a.logger.Warn("failed validation when creating pushover token", "validation_errors", validationErrs)
			renderPushoverSettings(appcomponents.PushoverSettingsViewModel{
				Values:           params,
				ValidationErrors: validationErrs,
			}, r, w)
			return
		}
		a.logger.Error("error creating pushover token", "error", err)
		a.internalServerError(err, w)
		return
	}

	http.Redirect(w, r, "/app/settings/pushover", http.StatusSeeOther)
}

func (a *app) pushoverSettingsDelete(w http.ResponseWriter, r *http.Request, user *sqlc.User) {
	err := a.pushoverStore.DeleteToken(r.Context(), user.ID)
	if a.internalServerError(err, w) {
		a.logger.Error("error deleting pushover user token", "error", err)
		return
	}

	http.Redirect(w, r, "/app/settings/pushover", http.StatusSeeOther)
}

func (a *app) emailSettingsGet(w http.ResponseWriter, r *http.Request, user *sqlc.User) {
	data := appcomponents.EmailSettingsViewModel{
		Email: user.Email,
	}

	appcomponents.EmailSettings(&data).Render(r.Context(), w)
}

