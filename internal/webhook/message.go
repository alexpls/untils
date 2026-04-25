package webhook

import (
	"encoding/json"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/monitorfieldrenderers"
)

const (
	MessageType          = "webhook_message"
	MessageTestType       = "test"
	MessageNewResultsType = "new_results"
)

type MessageTest struct {
	Type    string             `json:"type"`
	Message MessageTestMessage `json:"message"`
}

type MessageTestMessage struct {
	Type       string `json:"type"`
	HelloWorld string `json:"hello_world"`
}

func NewMessageTest() MessageTest {
	return MessageTest{
		Type: MessageType,
		Message: MessageTestMessage{
			Type:       MessageTestType,
			HelloWorld: "Glad you're here",
		},
	}
}

type ResultMessage struct {
	Type     string               `json:"type"`
	ID       int64                `json:"id"`
	Headline string               `json:"headline"`
	Subtitle string               `json:"subtitle"`
	Fields   []ResultFieldMessage `json:"fields"`
}

type ResultFieldMessage struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type MonitorMessage struct {
	Type    string `json:"type"`
	ID      int64  `json:"id"`
	Subject string `json:"subject"`
}

type MonitorNewResultsMessage struct {
	Type       string          `json:"type"`
	Monitor    MonitorMessage  `json:"monitor"`
	NewResults []ResultMessage `json:"new_results"`
	OldResult  ResultMessage   `json:"old_result"`
}

type MessageMonitorNewResults struct {
	Type    string                   `json:"type"`
	Message MonitorNewResultsMessage `json:"message"`
}

func NewMessageMonitorNewResults(monitor models.Monitor, newResults []models.MonitorResult, oldResult models.MonitorResult) MessageMonitorNewResults {
	resultMessages := make([]ResultMessage, len(newResults))
	for i, result := range newResults {
		resultMessages[i] = buildResultMessage(result)
	}

	return MessageMonitorNewResults{
		Type: MessageType,
		Message: MonitorNewResultsMessage{
			Type: MessageNewResultsType,
			Monitor: MonitorMessage{
				Type:    "monitor",
				ID:      monitor.ID,
				Subject: monitor.Subject.String,
			},
			NewResults: resultMessages,
			OldResult:  buildResultMessage(oldResult),
		},
	}
}

func MarshalMessageMonitorNewResults(monitor models.Monitor, newResults []models.MonitorResult, oldResult models.MonitorResult) ([]byte, error) {
	return json.Marshal(NewMessageMonitorNewResults(monitor, newResults, oldResult))
}

func buildResultMessage(result models.MonitorResult) ResultMessage {
	renderer := monitorfieldrenderers.TextRenderer{}
	renderCtx := models.MonitorFieldsRenderContext{}

	fields := make([]ResultFieldMessage, len(result.Data.Fields))
	for i, f := range result.Data.Fields {
		fields[i] = ResultFieldMessage{
			Type:  "result_field",
			Name:  f.Name,
			Value: f.Value,
		}
	}

	return ResultMessage{
		Type:     "result",
		ID:       result.ID,
		Headline: result.MustRenderHeadline(renderer, renderCtx),
		Subtitle: result.MustRenderSubtitle(renderer, renderCtx),
		Fields:   fields,
	}
}
