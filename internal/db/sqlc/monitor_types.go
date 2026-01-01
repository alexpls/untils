package sqlc

import (
	"encoding/json"
	"fmt"
)

type Citations []Citation

type Citation struct {
	URL          string `json:"url"`
	WebsiteTitle string `json:"website_title"`
	PageTitle    string `json:"page_title"`
}

type Date struct {
	Date          string `json:"date"`
	PastTenseVerb string `json:"past_tense_verb"`
}

type CheckResult struct {
	Success             bool      `json:"success"`
	Reason              string    `json:"reason"`
	DifferentToPrevious bool      `json:"different_to_previous"`
	ResultPlaintext     string    `json:"result_plaintext"`
	Date                Date      `json:"date"`
	Citations           Citations `json:"citations"` // TODO: change to Sources
}

type MonitorCheckEventDetails interface {
	isMonitorCheckEventDetails()
}

type MonitorCheckEventWebSearchDetails struct {
	Query string `json:"query"`
}

func (d MonitorCheckEventWebSearchDetails) isMonitorCheckEventDetails() {}

type MonitorCheckEventBrowserClickDetails struct {
}

func (d MonitorCheckEventBrowserClickDetails) isMonitorCheckEventDetails() {}

type MonitorCheckEventBrowserNavigateDetails struct {
	URL string `json:"url"`
}

func (d MonitorCheckEventBrowserNavigateDetails) isMonitorCheckEventDetails() {}

func (e *MonitorCheckEvent) DetailsStruct() (any, error) {
	switch e.Kind {
	case MonitorCheckEventKindWebSearch:
		return unmarshalMonitorEventDetails[MonitorCheckEventWebSearchDetails](e.Details)

	case MonitorCheckEventKindBrowserClick:
		return unmarshalMonitorEventDetails[MonitorCheckEventBrowserClickDetails](e.Details)

	case MonitorCheckEventKindBrowserNavigate:
		return unmarshalMonitorEventDetails[MonitorCheckEventBrowserNavigateDetails](e.Details)

	default:
		return nil, fmt.Errorf("unknown monitor check event kind: %s", e.Kind)
	}
}

func unmarshalMonitorEventDetails[T any](data json.RawMessage) (*T, error) {
	var details T
	if err := json.Unmarshal(data, &details); err != nil {
		return nil, err
	}
	return &details, nil
}
