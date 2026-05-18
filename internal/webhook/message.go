package webhook

import (
	"encoding/json"

	"github.com/alexpls/untils/internal/apimessage"
	"github.com/alexpls/untils/internal/models"
)

func NewMessageTest() apimessage.WebhookTestPayload {
	return apimessage.WebhookTestPayload{
		Type: "webhook_message",
		Message: struct {
			HelloWorld string `json:"hello_world"`
			Type       string `json:"type"`
		}{
			Type:       "test",
			HelloWorld: "Glad you're here",
		},
	}
}

func NewMessageMonitorNewResults(monitor models.Monitor, newResults []models.MonitorResult, oldResult models.MonitorResult) (apimessage.WebhookNewResultsPayload, error) {
	resultMessages := make([]apimessage.Result, len(newResults))
	for i, result := range newResults {
		msg, err := apimessage.BuildResultMessage(result)
		if err != nil {
			return apimessage.WebhookNewResultsPayload{}, err
		}
		resultMessages[i] = msg
	}

	oldResultMessage, err := apimessage.BuildResultMessage(oldResult)
	if err != nil {
		return apimessage.WebhookNewResultsPayload{}, err
	}

	return apimessage.WebhookNewResultsPayload{
		Type: "webhook_message",
		Message: struct {
			Monitor    apimessage.Monitor  `json:"monitor"`
			NewResults []apimessage.Result `json:"new_results"`
			OldResult  apimessage.Result   `json:"old_result"`
			Type       string              `json:"type"`
		}{
			Type:       "new_results",
			Monitor:    apimessage.BuildMonitorMessage(monitor),
			NewResults: resultMessages,
			OldResult:  oldResultMessage,
		},
	}, nil
}

func MarshalMessageMonitorNewResults(monitor models.Monitor, newResults []models.MonitorResult, oldResult models.MonitorResult) ([]byte, error) {
	msg, err := NewMessageMonitorNewResults(monitor, newResults, oldResult)
	if err != nil {
		return nil, err
	}
	return json.Marshal(msg)
}
