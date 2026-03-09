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
	dbUrl        string
	demoUserID   int64
	xAIKey       string
	openAIKey    string
	braveKey     string
	pushoverKey  string
	smtp         struct {
		username string
		password string
		host     string
		port     int
	}
}

type serveConfig struct {
	port int
}

func (c *config) servesPublicPages() bool {
	return c.appMode == appModeHosted
}
