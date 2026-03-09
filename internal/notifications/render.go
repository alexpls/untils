package notifications

import (
	"bytes"
	"context"
	"fmt"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/monitorfieldrenderers"
	"github.com/jackc/pgx/v5/pgtype"
)

type MonitorNewResult struct {
	Monitor models.Monitor
	New     models.MonitorResult
	Old     models.MonitorResult
}

type RenderedEmail struct {
	TemplateKey  string
	TemplateName string
	Subject      string
	TextBody     string
	HTMLBody     string
}

type RenderedPushover struct {
	Title   string
	Message string
}

type RenderedNotification struct {
	Email    RenderedEmail
	Pushover RenderedPushover
}

type EmailTemplateDefinition struct {
	Key       string
	Name      string
	DummyData MonitorNewResult
	Render    func(context.Context, MonitorNewResult) (RenderedEmail, error)
}

type EmailTemplateStore struct {
	templates []EmailTemplateDefinition
}

func NewEmailTemplateStore() *EmailTemplateStore {
	templates := []EmailTemplateDefinition{
		{
			Key:  "new_result",
			Name: "New result",
			DummyData: MonitorNewResult{
				Monitor: models.Monitor{Subject: pgtype.Text{String: "Kubernetes release notes", Valid: true}},
				New:     models.MonitorResult{Headline: "Kubernetes v1.35 release notes published"},
				Old:     models.MonitorResult{Headline: "Kubernetes v1.34 release notes published"},
			},
			Render: RenderMonitorNewResultEmail,
		},
	}

	return &EmailTemplateStore{
		templates: templates,
	}
}

func (s *EmailTemplateStore) Templates() []EmailTemplateDefinition {
	return append([]EmailTemplateDefinition(nil), s.templates...)
}

func (s *EmailTemplateStore) Template(key string) (EmailTemplateDefinition, bool) {
	for _, t := range s.templates {
		if t.Key == key {
			return t, true
		}
	}
	return EmailTemplateDefinition{}, false
}

func RenderMonitorNewResult(ctx context.Context, msg MonitorNewResult) (RenderedNotification, error) {
	emailRender, err := RenderMonitorNewResultEmail(ctx, msg)
	if err != nil {
		return RenderedNotification{}, err
	}

	pushoverRender, err := RenderMonitorNewResultPushover(msg)
	if err != nil {
		return RenderedNotification{}, err
	}

	return RenderedNotification{
		Email:    emailRender,
		Pushover: pushoverRender,
	}, nil
}

func RenderMonitorNewResultEmail(ctx context.Context, data MonitorNewResult) (RenderedEmail, error) {
	subject := fmt.Sprintf("New result: %s", data.Monitor.Subject.String)
	newHeadline, err := renderMonitorNewResultHeadline(data.New)
	if err != nil {
		return RenderedEmail{}, fmt.Errorf("rendering new result headline: %w", err)
	}
	oldHeadline, err := renderMonitorNewResultHeadline(data.Old)
	if err != nil {
		return RenderedEmail{}, fmt.Errorf("rendering old result headline: %w", err)
	}
	textBody := fmt.Sprintf("New: %s\n\nOld: %s", newHeadline, oldHeadline)

	var htmlBody bytes.Buffer
	if err := MonitorNewResultEmail(data).Render(ctx, &htmlBody); err != nil {
		return RenderedEmail{}, fmt.Errorf("rendering html email: %w", err)
	}

	return RenderedEmail{
		TemplateKey:  "new_result",
		TemplateName: "New result",
		Subject:      subject,
		TextBody:     textBody,
		HTMLBody:     htmlBody.String(),
	}, nil
}

func RenderMonitorNewResultPushover(msg MonitorNewResult) (RenderedPushover, error) {
	newHeadline, err := renderMonitorNewResultHeadline(msg.New)
	if err != nil {
		return RenderedPushover{}, fmt.Errorf("rendering new result headline: %w", err)
	}
	oldHeadline, err := renderMonitorNewResultHeadline(msg.Old)
	if err != nil {
		return RenderedPushover{}, fmt.Errorf("rendering old result headline: %w", err)
	}

	return RenderedPushover{
		Title:   fmt.Sprintf("New result: %s", msg.Monitor.Subject.String),
		Message: fmt.Sprintf("New: %s\n\nOld: %s", newHeadline, oldHeadline),
	}, nil
}

func renderMonitorNewResultHeadline(result models.MonitorResult) (string, error) {
	return result.RenderHeadline(monitorfieldrenderers.TextRenderer{}, models.MonitorFieldsRenderContext{})
}

func monitorResultURLFields(result models.MonitorResult) []models.MonitorUpdateField {
	urlFields := make([]models.MonitorUpdateField, 0, len(result.Data.Fields))
	for _, field := range result.Data.Fields {
		if field.Type == models.MonitorSchemaFieldTypeURL && field.Value != "" {
			urlFields = append(urlFields, field)
		}
	}
	return urlFields
}

func monitorResultPagePath(result models.MonitorResult) string {
	return fmt.Sprintf("/app/monitors/%d", result.MonitorID)
}
