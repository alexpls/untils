package models

type Citations []Citation

type Citation struct {
	URL          string `json:"url"`
	WebsiteTitle string `json:"website_title"`
	PageTitle    string `json:"page_title"`
	// FaviconURL is the URL of the favicon for the cited website.
	//
	// Empty string if no favicon is available.
	//
	// TODO: plumbing the favicon URL through the LLM like this is
	// wasteful and error-prone.
	FaviconURL string `json:"favicon_url"`
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
