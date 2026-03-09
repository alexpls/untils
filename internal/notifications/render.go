package notifications

import (
	"bytes"
	"context"
	"fmt"
)

type MonitorNewResult struct {
	Subject string
	New     string
	Old     string
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
				Subject: "Kubernetes release notes",
				New:     "Kubernetes v1.35 release notes published",
				Old:     "Kubernetes v1.34 release notes published",
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

	return RenderedNotification{
		Email:    emailRender,
		Pushover: RenderMonitorNewResultPushover(msg),
	}, nil
}

func RenderMonitorNewResultEmail(ctx context.Context, msg MonitorNewResult) (RenderedEmail, error) {
	subject := fmt.Sprintf("Monitor changed: %s", msg.Subject)
	textBody := fmt.Sprintf("New: %s\n\nOld: %s", msg.New, msg.Old)

	var htmlBody bytes.Buffer
	if err := MonitorNewResultEmail(MonitorNewResultEmailData{
		Subject: subject,
		New:     msg.New,
		Old:     msg.Old,
	}).Render(ctx, &htmlBody); err != nil {
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

func RenderMonitorNewResultPushover(msg MonitorNewResult) RenderedPushover {
	return RenderedPushover{
		Title:   fmt.Sprintf("Monitor changed: %s", msg.Subject),
		Message: fmt.Sprintf("New: %s\n\nOld: %s", msg.New, msg.Old),
	}
}
