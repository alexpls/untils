package main

type appMode string

const (
	appModeSelfHosted appMode = "selfhosted"
	appModeHosted     appMode = "hosted"
)

type config struct {
	buildVersion string
	env          string
	appMode      appMode
	baseURL      string
	dbUrl        string
	demoUserID   int64
	xAIKey       string
	openAIKey    string
	braveKey     string
	pushoverKey  string
	chrome struct {
		devToolsURL string
	}
	smtp         struct {
		username string
		password string
		host     string
		port     int
		from     string
	}
}

type serveConfig struct {
	port int
}

func (c *config) servesPublicPages() bool {
	return c.appMode == appModeHosted
}
