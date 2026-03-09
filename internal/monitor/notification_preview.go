package monitor

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/notifications"
	"github.com/starfederation/datastar-go/datastar"
)

type NotificationPreviewTab string

const (
	NotificationPreviewTabEmail    NotificationPreviewTab = "email"
	NotificationPreviewTabPushover NotificationPreviewTab = "pushover"
)

type NotificationPreviewPageData struct {
	Monitor         *models.Monitor
	Result          *models.MonitorResult
	ActiveTab       NotificationPreviewTab
	Rendered        notifications.RenderedNotification
	EmailPreviewURL string
}

func (h *Handlers) ViewNotificationPreview(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPreviewQuery(w, r, user)
	if mon == nil {
		return
	}

	result := h.monitorResultFromPreviewQuery(w, r, mon)
	if result == nil {
		return
	}

	rendered, err := h.renderNotificationPreview(r, user, *mon, *result)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering notification preview", "monitor_id", mon.ID, "result_id", result.ID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := NotificationPreviewPageData{
		Monitor:         mon,
		Result:          result,
		ActiveTab:       notificationPreviewTabFromRequest(r),
		Rendered:        rendered,
		EmailPreviewURL: fmt.Sprintf("/app/dev/preview_notification/email?monitor_id=%d&result_id=%d", mon.ID, result.ID),
	}
	if err := NotificationPreviewPage(data).Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering notification preview page", "monitor_id", mon.ID, "result_id", result.ID, "error", err)
	}
}

func (h *Handlers) ViewNotificationPreviewEmailHTML(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPreviewQuery(w, r, user)
	if mon == nil {
		return
	}

	result := h.monitorResultFromPreviewQuery(w, r, mon)
	if result == nil {
		return
	}

	rendered, err := h.renderNotificationPreview(r, user, *mon, *result)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering notification preview html", "monitor_id", mon.ID, "result_id", result.ID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(rendered.Email.HTMLBody))
}

func (h *Handlers) SendDevNotification(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	result := h.monitorResultFromPath(w, r, mon)
	if result == nil {
		return
	}

	message, err := h.monitorNewResultNotification(r, user, *mon, *result)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error building dev notification", "monitor_id", mon.ID, "result_id", result.ID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.service.notificationSender.Send(r.Context(), notifications.SendParams{
		UserID:               user.ID,
		NotificationChannels: []models.Notifier{models.NotifierEmail},
		Message:              message,
	}); err != nil {
		h.logger.ErrorContext(r.Context(), "error sending dev notification", "monitor_id", mon.ID, "result_id", result.ID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)
	if err := sse.ExecuteScript("window.alert('Notification email sent.')"); err != nil {
		h.logger.ErrorContext(sse.Context(), "error sending dev notification success feedback", "monitor_id", mon.ID, "result_id", result.ID, "error", err)
	}
}

func (h *Handlers) renderNotificationPreview(r *http.Request, user *models.User, mon models.Monitor, result models.MonitorResult) (notifications.RenderedNotification, error) {
	message, err := h.monitorNewResultNotification(r, user, mon, result)
	if err != nil {
		return notifications.RenderedNotification{}, err
	}
	return notifications.RenderMonitorNewResult(r.Context(), message)
}

func notificationPreviewTabFromRequest(r *http.Request) NotificationPreviewTab {
	switch NotificationPreviewTab(r.URL.Query().Get("tab")) {
	case NotificationPreviewTabPushover:
		return NotificationPreviewTabPushover
	default:
		return NotificationPreviewTabEmail
	}
}

func (h *Handlers) monitorNewResultNotification(r *http.Request, user *models.User, mon models.Monitor, result models.MonitorResult) (notifications.MonitorNewResult, error) {
	oldValue, err := h.service.previousVisibleNotificationResult(r.Context(), mon.ID, result.ID)
	if err != nil {
		return notifications.MonitorNewResult{}, err
	}
	return newResultNotificationMessage(mon, result, oldValue), nil
}

func (h *Handlers) monitorFromPreviewQuery(w http.ResponseWriter, r *http.Request, user *models.User) *models.Monitor {
	monitorID := previewIDQuery(r, "monitor_id")
	if monitorID == 0 {
		http.NotFound(w, r)
		return nil
	}

	mon, err := h.service.GetMonitor(r.Context(), user.ID, monitorID)
	if err != nil {
		http.NotFound(w, r)
		return nil
	}

	return mon
}

func (h *Handlers) monitorResultFromPreviewQuery(w http.ResponseWriter, r *http.Request, mon *models.Monitor) *models.MonitorResult {
	resultID := previewIDQuery(r, "result_id")
	if resultID == 0 {
		http.NotFound(w, r)
		return nil
	}

	params := &models.GetMonitorResultParams{
		MonitorID: mon.ID,
		ResultID:  resultID,
	}
	result, err := h.service.queries.GetMonitorResult(r.Context(), h.service.db, params)
	if err != nil {
		http.NotFound(w, r)
		return nil
	}

	return result
}

func previewIDQuery(r *http.Request, name string) int64 {
	id, err := strconv.ParseInt(r.URL.Query().Get(name), 10, 64)
	if err != nil {
		return 0
	}
	return id
}
