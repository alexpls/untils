package llm

import "strings"

type Date struct {
	Date          string `json:"date"`
	PastTenseVerb string `json:"past_tense_verb"`
}

type Citations []Citation

type Citation struct {
	URL          string `json:"url"`
	WebsiteTitle string `json:"website_title"`
	PageTitle    string `json:"page_title"`
}

type CheckParams struct {
	Subject        string
	Instructions   string
	PreviousResult string
}

type CheckResponse struct {
	Answered            bool      `json:"answered"`
	Explanation         string    `json:"explanation"`
	DetectedSPA         bool      `json:"detected_spa"`
	DifferentToPrevious bool      `json:"different_to_previous"`
	ResponsePlaintext   string    `json:"response_plaintext"`
	Date                Date      `json:"date"`
	Citations           Citations `json:"citations"`
}

func sanitizeXAIOutput(in string) string {
	return strings.ReplaceAll(in, "</xai:function_call>", "")
}
