package notifications

import (
	"github.com/alexpls/untils/internal/webhook"
)

func RenderMonitorNewResultWebhook(msg MonitorNewResult) (RenderedWebhook, error) {
	jsonStr, err := webhook.MarshalMessageMonitorNewResult(msg.Monitor, msg.New, msg.Old)
	if err != nil {
		return RenderedWebhook{}, err
	}

	return RenderedWebhook{Json: jsonStr}, nil
}
