package main

import (
	"net/http"

	"github.com/alexpls/untils/internal/faviconproxy"
	"github.com/alexpls/untils/public"
)

func (a *app) routes() http.Handler {
	mux := http.NewServeMux()

	// assets
	mux.Handle("/assets/", public.Handler())

	// public pages
	mux.HandleFunc("GET /{$}", a.pagesHandlers.Home)

	// auth pages
	mux.HandleFunc("GET /sign_in", a.authHandlers.SignInGet)
	mux.HandleFunc("POST /sign_in", a.authHandlers.SignInPost)
	mux.HandleFunc("GET /sign_out", a.authHandlers.SignOutGet)

	// dashboard
	mux.HandleFunc("GET /app", a.requireAuth(a.dashboardHandlers.Get))
	mux.HandleFunc("GET /app/dashboard/events", a.requireAuth(a.dashboardHandlers.Events))

	// monitors
	mux.HandleFunc("GET /app/monitors", a.requireAuth(a.monitorHandlers.ListGet))
	mux.HandleFunc("GET /app/monitors/events", a.requireAuth(a.monitorHandlers.ListEventsGet))
	mux.HandleFunc("GET /app/monitors/new", a.requireAuth(a.monitorHandlers.NewGet))
	mux.HandleFunc("POST /app/monitors/new", a.requireAuth(a.monitorHandlers.CreatePost))
	mux.HandleFunc("GET /app/monitors/{monitor_id}", a.requireAuth(a.monitorHandlers.ViewGet))
	mux.HandleFunc("GET /app/monitors/{monitor_id}/events", a.requireAuth(a.monitorHandlers.ViewEventsGet))
	mux.HandleFunc("POST /app/monitors/{monitor_id}", a.requireAuth(a.monitorHandlers.UpdatePost))
	mux.HandleFunc("DELETE /app/monitors/{monitor_id}", a.requireAuth(a.monitorHandlers.Delete))
	mux.HandleFunc("POST /app/monitors/{monitor_id}/check", a.requireAuth(a.monitorHandlers.CheckPost))
	mux.HandleFunc("POST /app/monitors/{monitor_id}/pause", a.requireAuth(a.monitorHandlers.PausePost))
	mux.HandleFunc("POST /app/monitors/{monitor_id}/unpause", a.requireAuth(a.monitorHandlers.UnpausePost))
	mux.HandleFunc("POST /app/monitors/{monitor_id}/activate", a.requireAuth(a.monitorHandlers.ActivatePost))
	mux.HandleFunc("POST /app/monitors/{monitor_id}/schedule", a.requireAuth(a.monitorHandlers.UpdateCheckSchedule))
	mux.HandleFunc("POST /app/monitors/{monitor_id}/notifiers/{type}", a.requireAuth(a.monitorHandlers.NotifierPost))
	mux.HandleFunc("DELETE /app/monitors/{monitor_id}/notifiers/{type}", a.requireAuth(a.monitorHandlers.NotifierDelete))
	mux.HandleFunc("GET /app/monitors/{monitor_id}/results/{result_id}/feedback", a.requireAuth(a.monitorHandlers.ResultFeedbackGet))
	mux.HandleFunc("POST /app/monitors/{monitor_id}/results/{result_id}/feedback", a.requireAuth(a.monitorHandlers.ResultFeedbackPost))

	// checks
	mux.HandleFunc("GET /app/checks", a.requireAuth(a.monitorHandlers.ChecksListGet))
	mux.HandleFunc("GET /app/checks/events", a.requireAuth(a.monitorHandlers.ChecksListEventsGet))
	mux.HandleFunc("GET /app/checks/{check_id}", a.requireAuth(a.monitorHandlers.CheckViewGet))
	mux.HandleFunc("GET /app/checks/{check_id}/events", a.requireAuth(a.monitorHandlers.CheckViewEventsGet))

	// settings
	mux.HandleFunc("GET /app/settings", a.requireAuth(a.settingsHandlers.SettingsGet))
	mux.HandleFunc("POST /app/settings/timezone", a.requireAuth(a.settingsHandlers.UpdateTimezonePost))
	mux.HandleFunc("GET /app/settings/pushover", a.requireAuth(a.settingsHandlers.PushoverSettingsGet))
	mux.HandleFunc("POST /app/settings/pushover", a.requireAuth(a.settingsHandlers.PushoverSettingsPost))
	mux.HandleFunc("DELETE /app/settings/pushover", a.requireAuth(a.settingsHandlers.PushoverSettingsDelete))
	mux.HandleFunc("GET /app/settings/email", a.requireAuth(a.settingsHandlers.EmailSettingsGet))

	// favicon
	mux.Handle("GET /app/favicon", a.requireAuth2(faviconproxy.Handler(a.logger.With("source", "faviconproxy"))))

	// dev
	mux.Handle("GET /app/dev/palette", a.requireAuth(a.devHandlers.PaletteGet))

	// middleware
	csrf := http.NewCrossOriginProtection()
	sess := a.sessionManager

	return applyMiddleware(mux,
		a.setRequestID, csrf.Handler,
		a.setTimezoneContext, sess.Handler,
		a.setUserContext, a.logRequests, a.setEnvContext,
	)
}
