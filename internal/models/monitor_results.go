package models

import (
	"fmt"
	"strings"
)

func (mr MonitorResultsWithLatestCheck) Markdown() string {
	var sb strings.Builder

	_, _ = fmt.Fprintf(&sb, "**Result:** %s\n\n", mr.Result)

	if mr.Date != nil && !mr.Date.IsZero() {
		if mr.DatePastTenseVerb.Valid && mr.DatePastTenseVerb.String != "" {
			_, _ = fmt.Fprintf(&sb, "**Result date:** %s %s\n", mr.DatePastTenseVerb.String, mr.Date.Format("January 2, 2006"))
		} else {
			_, _ = fmt.Fprintf(&sb, "**Result date:** %s\n", mr.Date.Format("January 2, 2006"))
		}
	}

	if mr.LatestConfirmationAt != nil {
		_, _ = fmt.Fprintf(&sb, "**Latest check ran at:** %s\n", mr.LatestConfirmationAt.Format("January 2, 2006 at 3:04 PM"))
	}

	if mr.Feedback.Valid && mr.Feedback.String != "" {
		_, _ = fmt.Fprintf(&sb, "**User feedback:** %s\n", mr.Feedback.String)
	}

	if mr.Citations != nil && len(*mr.Citations) > 0 {
		_, _ = sb.WriteString("**Sources used:**\n")
		for _, citation := range *mr.Citations {
			_, _ = fmt.Fprintf(&sb, "- %s\n", citation.URL)
		}
	}

	return sb.String()
}
