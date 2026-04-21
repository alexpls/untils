package notifications

import (
	"encoding/json"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/monitorfieldrenderers"
)

type resultWebhookPayload struct {
	Type     string                      `json:"type"`
	ID       int64                       `json:"id"`
	Headline string                      `json:"headline"`
	Subtitle string                      `json:"subtitle"`
	Fields   []resultFieldWebhookPayload `json:"fields"`
}

type resultFieldWebhookPayload struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type monitorWebhookPayload struct {
	Type    string `json:"type"`
	ID      int64  `json:"id"`
	Subject string `json:"subject"`
}

type messageWebhookPayload struct {
	Type      string                `json:"type"`
	Monitor   monitorWebhookPayload `json:"monitor"`
	NewResult resultWebhookPayload  `json:"new_result"`
	OldResult resultWebhookPayload  `json:"old_result"`
}

type newResultWebhookPayload struct {
	Type    string                `json:"type"`
	Message messageWebhookPayload `json:"message"`
}

func RenderMonitorNewResultWebhook(msg MonitorNewResult) (RenderedWebhook, error) {
	payload := newResultWebhookPayload{
		Type: "webhook_message",
		Message: messageWebhookPayload{
			Type: "new_result",
			Monitor: monitorWebhookPayload{
				Type:    "monitor",
				ID:      msg.Monitor.ID,
				Subject: msg.Monitor.Subject.String,
			},
			NewResult: buildResultWebhookPayload(msg.New),
			OldResult: buildResultWebhookPayload(msg.Old),
		},
	}

	jsonStr, err := json.Marshal(payload)
	if err != nil {
		return RenderedWebhook{}, nil
	}

	return RenderedWebhook{Json: jsonStr}, nil
}

func buildResultWebhookPayload(result models.MonitorResult) resultWebhookPayload {
	renderer := monitorfieldrenderers.TextRenderer{}
	renderCtx := models.MonitorFieldsRenderContext{}

	headline := result.MustRenderHeadline(renderer, renderCtx)
	subtitle := result.MustRenderSubtitle(renderer, renderCtx)

	fields := make([]resultFieldWebhookPayload, len(result.Data.Fields))

	for i, f := range result.Data.Fields {
		fields[i] = resultFieldWebhookPayload{
			Type:  "result_field",
			Name:  f.Name,
			Value: f.Value,
		}
	}

	return resultWebhookPayload{
		Type:     "result",
		ID:       result.ID,
		Headline: headline,
		Subtitle: subtitle,
		Fields:   fields,
	}
}
