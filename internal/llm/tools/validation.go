package tools

import "github.com/alexpls/untils/internal/models"

// noDuplicateCallsValidator returns a validation error if params matches any prior call.
func noDuplicateCallsValidator[P models.ToolParams](tc *Context, params P, errMsg string) string {
	for _, prior := range *tc.PriorCalls {
		if params.Equal(prior.Params()) {
			return errMsg
		}
	}
	return ""
}
