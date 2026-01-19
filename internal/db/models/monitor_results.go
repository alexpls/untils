package models

import "fmt"

func (mr MonitorResult) Markdown() string {
	s := fmt.Sprintf("**Result:** %s\n\n", mr.Result)

	if mr.Date != nil {
		if mr.DatePastTenseVerb.Valid && mr.DatePastTenseVerb.String != "" {
			s += fmt.Sprintf("**Date:** %s %s\n\n", mr.DatePastTenseVerb.String, mr.Date.Format("January 2, 2006"))
		} else {
			s += fmt.Sprintf("**Date:** %s\n\n", mr.Date.Format("January 2, 2006"))
		}
	}

	s += fmt.Sprintf("**Latest Check:** %s\n\n", mr.LatestConfirmationAt.Format("January 2, 2006 at 3:04 PM"))

	if mr.Feedback.Valid && mr.Feedback.String != "" {
		s += fmt.Sprintf("**User Feedback:** %s\n", mr.Feedback.String)
	}

	return s
}
