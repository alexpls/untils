package main

import "github.com/alexpls/untils/internal/constants"

const (
	appEnvDev         = constants.EnvDev
	appEnvProd        = constants.EnvProd
	appModeSelfHosted = constants.ModeSelfHosted
	appModeHosted     = constants.ModeHosted
)

type config struct {
	buildVersion string
	env          constants.Env
	appMode      constants.Mode
	migrate      bool
	baseURL      string
	dbUrl        string
	demoUserID   int64
	xAIKey       string
	openAIKey    string
	braveKey     string
	pushoverKey  string
	chrome       struct {
		devToolsURL string
	}
	smtp struct {
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
