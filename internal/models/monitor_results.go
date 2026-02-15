package models

import (
	"fmt"
	"strings"
)

func (mr MonitorResult) Markdown() string {
	var sb strings.Builder

	_, _ = fmt.Fprintf(&sb, "**Result:** %s\n\n", mr.Headline)

	if mr.Subtitle != "" {
		_, _ = fmt.Fprintf(&sb, "**Result subtitle:** %s\n", mr.Subtitle)
	}

	_, _ = sb.WriteString("**Result fields:**\n")
	for _, field := range mr.Data.Fields {
		_, _ = fmt.Fprintf(&sb, "- %s: %s\n", field.Name, field.Value)
	}

	_, _ = fmt.Fprintf(&sb, "**Latest check ran at:** %s\n", mr.LastConfirmedAt.Format("January 2, 2006 at 3:04 PM"))

	if mr.Feedback.Valid {
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
