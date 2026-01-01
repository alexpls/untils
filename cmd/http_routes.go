package main

import (
	"net/http"

	"github.com/alexpls/untils/internal/db/sqlc"
	"github.com/alexpls/untils/public"
)

func (a *app) routes() http.Handler {
	mux := http.NewServeMux()

	// assets
	mux.Handle("/assets/", public.Handler())

	// public pages
	mux.HandleFunc("GET /{$}", a.home)

	// auth pages
	mux.HandleFunc("GET /sign_in", a.signInGet)
	mux.HandleFunc("POST /sign_in", a.signInPost)
	mux.HandleFunc("GET /sign_out", a.signOutGet)

	// app
	mux.HandleFunc("GET /app", a.requireAuth(a.appHandler))

	// monitors
	mux.HandleFunc("GET /app/monitors", a.requireAuth(a.monitorListGet))
	mux.HandleFunc("GET /app/monitors/new", a.requireAuth(a.monitorNewGet))
	mux.HandleFunc("POST /app/monitors/new", a.requireAuth(a.monitorCreatePost))
	mux.HandleFunc("GET /app/monitors/{id}", a.requireAuth(a.monitorViewGet))
	mux.HandleFunc("GET /app/monitors/{id}/events", a.requireAuth(a.monitorViewEventsGet))
	mux.HandleFunc("POST /app/monitors/{id}", a.requireAuth(a.monitorUpdatePost))
	mux.HandleFunc("DELETE /app/monitors/{id}", a.requireAuth(a.monitorDelete))
	mux.HandleFunc("POST /app/monitors/{id}/check", a.requireAuth(a.monitorCheckPost))
	mux.HandleFunc("POST /app/monitors/{id}/activate", a.requireAuth(a.monitorActivatePost))
	mux.HandleFunc("POST /app/monitors/{id}/notifiers/{type}", a.requireAuth(a.monitorNotifierPost))
	mux.HandleFunc("DELETE /app/monitors/{id}/notifiers/{type}", a.requireAuth(a.monitorNotifierDelete))

	// settings
	mux.HandleFunc("GET /app/settings", a.requireAuth(a.settingsGet))
	mux.HandleFunc("POST /app/settings/timezone", a.requireAuth(a.updateTimezonePost))
	mux.HandleFunc("GET /app/settings/pushover", a.requireAuth(a.pushoverSettingsGet))
	mux.HandleFunc("POST /app/settings/pushover", a.requireAuth(a.pushoverSettingsPost))
	mux.HandleFunc("DELETE /app/settings/pushover", a.requireAuth(a.pushoverSettingsDelete))
	mux.HandleFunc("GET /app/settings/email", a.requireAuth(a.emailSettingsGet))

	// middleware
	csrf := http.NewCrossOriginProtection()
	sess := a.sessionManager

	return applyMiddleware(mux,
		csrf.Handler, a.logRequests, a.setTimezoneContext,
		sess.Handler, a.setUserContext,
	)
}

func (a *app) appHandler(w http.ResponseWriter, r *http.Request, _ *sqlc.User) {
	http.Redirect(w, r, "/app/monitors", http.StatusSeeOther)
}
