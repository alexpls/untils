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

	a.registerPublicRoutes(mux)

	// auth pages
	mux.HandleFunc("GET /sign_in", a.authHandlers.ViewSignIn)
	mux.HandleFunc("POST /sign_in", a.authHandlers.SignIn)
	mux.HandleFunc("GET /sign_out", a.authHandlers.SignOut)

	// dashboard
	mux.HandleFunc("GET /app", a.requireAuth(a.dashboardHandlers.ViewDashboard))

	// monitors
	mux.HandleFunc("GET /app/monitors", a.allowDemo(a.requireAuth(a.monitorHandlers.ListMonitors)))
	mux.HandleFunc("GET /app/monitors/new", a.requireAuth(a.monitorHandlers.NewMonitor))
	mux.HandleFunc("POST /app/monitors/new", a.requireAuth(a.monitorHandlers.CreateMonitor))
	mux.HandleFunc("GET /app/monitors/{monitor_id}", a.requireAuth(a.monitorHandlers.ViewMonitor))
	mux.HandleFunc("GET /app/monitors/{monitor_id}/checks", a.requireAuth(a.monitorHandlers.ViewMonitorChecks))
	mux.HandleFunc("GET /app/monitors/{monitor_id}/notifications", a.requireAuth(a.monitorHandlers.ViewMonitorNotifications))
	mux.HandleFunc("GET /app/monitors/{monitor_id}/settings", a.requireAuth(a.monitorHandlers.ViewMonitorSettings))
	mux.HandleFunc("POST /app/monitors/{monitor_id}", a.requireAuth(a.monitorHandlers.UpdateMonitor))
	mux.HandleFunc("DELETE /app/monitors/{monitor_id}", a.requireAuth(a.monitorHandlers.DeleteMonitor))
	mux.HandleFunc("POST /app/monitors/{monitor_id}/pause", a.requireAuth(a.monitorHandlers.PauseMonitor))
	mux.HandleFunc("POST /app/monitors/{monitor_id}/unpause", a.requireAuth(a.monitorHandlers.UnpauseMonitor))
	mux.HandleFunc("POST /app/monitors/{monitor_id}/activate", a.requireAuth(a.monitorHandlers.ActivateMonitor))
	mux.HandleFunc("POST /app/monitors/{monitor_id}/frequency", a.requireAuth(a.monitorHandlers.UpdateMonitorCheckFrequency))
	mux.HandleFunc("POST /app/monitors/{monitor_id}/toggle_auto_activate", a.requireAuth(a.monitorHandlers.UpdateMonitorToggleAutoActivate))
	mux.HandleFunc("POST /app/monitors/{monitor_id}/notifiers/{type}", a.requireAuth(a.monitorHandlers.UpdateMonitorNotifier))
	mux.HandleFunc("DELETE /app/monitors/{monitor_id}/notifiers/{type}", a.requireAuth(a.monitorHandlers.DeleteMonitorNotifier))
	mux.HandleFunc("GET /app/monitors/{monitor_id}/results/{result_id}/correction", a.requireAuth(a.monitorHandlers.ViewResultCorrectionModal))
	mux.HandleFunc("POST /app/monitors/{monitor_id}/results/{result_id}/correction", a.requireAuth(a.monitorHandlers.UpdateResultCorrection))
	mux.HandleFunc("POST /app/monitors/{monitor_id}/results/{result_id}/hide", a.requireAuth(a.monitorHandlers.HideResult))
	// checks
	mux.HandleFunc("GET /app/checks", a.requireAuth(a.monitorHandlers.ListChecks))
	mux.HandleFunc("GET /app/checks/{check_id}", a.requireAuth(a.monitorHandlers.ViewCheck))
	mux.HandleFunc("POST /app/checks/{check_id}/run", a.requireAuth(a.monitorHandlers.RunCheckNow))

	// settings
	mux.HandleFunc("GET /app/settings", a.requireAuth(a.settingsHandlers.ViewSettings))
	mux.HandleFunc("POST /app/settings/timezone", a.requireAuth(a.settingsHandlers.UpdateTimezone))
	mux.HandleFunc("GET /app/settings/password", a.requireAuth(a.settingsHandlers.ViewPasswordSettings))
	mux.HandleFunc("POST /app/settings/password", a.requireAuth(a.settingsHandlers.UpdatePassword))
	mux.HandleFunc("GET /app/settings/pushover", a.requireAuth(a.settingsHandlers.ViewPushoverSettings))
	mux.HandleFunc("POST /app/settings/pushover", a.requireAuth(a.settingsHandlers.UpdatePushoverSettings))
	mux.HandleFunc("DELETE /app/settings/pushover", a.requireAuth(a.settingsHandlers.DeletePushoverSettings))
	mux.HandleFunc("GET /app/settings/email", a.requireAuth(a.settingsHandlers.ViewEmailSettings))

	// favicon
	mux.Handle("GET /app/favicon", a.requireAuth2(faviconproxy.Handler(a.logger.With("source", "faviconproxy"))))

	// dev
	mux.Handle("GET /app/dev/palette", a.requireDev(a.requireAuth(a.devHandlers.ViewPalette)))
	mux.Handle("GET /app/dev/palette/monitor_draft", a.requireDev(a.requireAuth(a.devHandlers.ViewMonitorDraftPalette)))
	mux.Handle("GET /app/dev/palette/flash", a.requireDev(a.requireAuth(a.devHandlers.ViewFlashPalette)))
	mux.Handle("GET /app/dev/emails", a.requireDev(a.requireAuth(a.devHandlers.ListEmailPreviews)))
	mux.Handle("GET /app/dev/emails/{template_key}", a.requireDev(a.requireAuth(a.devHandlers.ViewEmailPreview)))
	mux.Handle("GET /app/dev/emails/{template_key}/html", a.requireDev(a.requireAuth(a.devHandlers.ViewEmailPreviewHTML)))
	mux.HandleFunc("GET /app/dev/preview_notification", a.requireDev(a.requireAuth(a.monitorHandlers.ViewNotificationPreview)))
	mux.HandleFunc("GET /app/dev/preview_notification/email", a.requireDev(a.requireAuth(a.monitorHandlers.ViewNotificationPreviewEmailHTML)))
	mux.HandleFunc("POST /app/dev/monitors/{monitor_id}/results/{result_id}/send_notification", a.requireDev(a.requireAuth(a.monitorHandlers.SendDevNotification)))

	// middleware
	csrf := http.NewCrossOriginProtection()
	sess := a.sessionManager

	return applyMiddleware(mux,
		a.setRequestID, csrf.Handler,
		a.setTimezoneContext, sess.Handler,
		a.setFlashContext, a.setContextFromAppConfig,
		a.setUserContext, a.logRequests,
	)
}

func (a *app) registerPublicRoutes(mux *http.ServeMux) {
	if a.config.servesPublicPages() {
		mux.HandleFunc("GET /docs", a.pagesHandlers.DocsHome)
		mux.HandleFunc("GET /docs/{$}", a.pagesHandlers.DocsHome)
		mux.HandleFunc("GET /docs/{doc_path...}", a.pagesHandlers.DocsPage)
		mux.HandleFunc("GET /{$}", a.pagesHandlers.Home)
		mux.HandleFunc("POST /subscribe", a.pagesHandlers.SubscribeEmail)
		return
	}

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/app", http.StatusSeeOther)
	})
}
