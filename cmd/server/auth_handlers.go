package main

import (
	"errors"
	"net/http"

	"github.com/alexpls/untils_go/internal/auth"
	"github.com/alexpls/untils_go/internal/components/public"
)

func (a *app) signInGet(w http.ResponseWriter, r *http.Request) {
	ret := r.URL.Query().Get("return")
	data := public.SignInData{
		Return: ret,
	}
	public.SignInPage(data).Render(r.Context(), w)
}

func (a *app) signOutGet(w http.ResponseWriter, r *http.Request) {
	err := a.sessionManager.Destroy(r, w)
	if a.internalServerError(err, w) {
		a.logger.Error("error destroying session", "error", err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *app) signInPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if a.internalServerError(err, w) {
		a.logger.Error("error parsing sign in form", "error", err)
		return
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")
	ret := r.Form.Get("return")

	a.logger.Info("signing in", "email", email)

	user, err := a.auth.GetUserByEmailPassword(r.Context(), email, password)
	if !errors.Is(err, auth.ErrNoUser) && a.internalServerError(err, w) {
		a.logger.Error("error processing sign in", "error", err)
		return
	}
	if errors.Is(err, auth.ErrNoUser) {
		data := public.SignInData{Failed: true, Email: email, Return: ret}
		public.SignInPage(data).Render(r.Context(), w)
		return
	}

	sess := a.sessionManager.New(r, w)
	sess.Data.UserID = user.ID
	// TODO: ergonomics here a bit wonky, would be nicer to call Save on the session itself
	if err = a.sessionManager.Save(r); a.internalServerError(err, w) {
		a.logger.Error("error saving session", "error", err)
		return
	}

	if ret != "" {
		http.Redirect(w, r, ret, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/app", http.StatusSeeOther)
	}
}
