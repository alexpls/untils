package notifications

import (
	"github.com/alexpls/untils/internal/webhook"
)

func RenderMonitorNewResultsWebhook(msg MonitorNewResults) (RenderedWebhook, error) {
	jsonStr, err := webhook.MarshalMessageMonitorNewResults(msg.Monitor, msg.NewResults, msg.OldResult)
	if err != nil {
		return RenderedWebhook{}, err
	}

	return RenderedWebhook{Json: jsonStr}, nil
}
