package webhook

import (
	"encoding/json"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/monitorfieldrenderers"
)

const (
	MessageType          = "webhook_message"
	MessageTestType      = "test"
	MessageNewResultType = "new_result"
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

type MonitorNewResultMessage struct {
	Type      string         `json:"type"`
	Monitor   MonitorMessage `json:"monitor"`
	NewResult ResultMessage  `json:"new_result"`
	OldResult ResultMessage  `json:"old_result"`
}

type MessageMonitorNewResult struct {
	Type    string                  `json:"type"`
	Message MonitorNewResultMessage `json:"message"`
}

func NewMessageMonitorNewResult(monitor models.Monitor, newResult, oldResult models.MonitorResult) MessageMonitorNewResult {
	return MessageMonitorNewResult{
		Type: MessageType,
		Message: MonitorNewResultMessage{
			Type: MessageNewResultType,
			Monitor: MonitorMessage{
				Type:    "monitor",
				ID:      monitor.ID,
				Subject: monitor.Subject.String,
			},
			NewResult: buildResultMessage(newResult),
			OldResult: buildResultMessage(oldResult),
		},
	}
}

func MarshalMessageMonitorNewResult(monitor models.Monitor, newResult, oldResult models.MonitorResult) ([]byte, error) {
	return json.Marshal(NewMessageMonitorNewResult(monitor, newResult, oldResult))
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
