package main

import (
	"strings"

	"github.com/alexpls/untils/internal/constants"
)

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
	adminEmail   string
	demoUserID   int64
	openAIAPIKey string
	openAIModel  string
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

func (c *config) usesXAI() bool {
	return strings.HasPrefix(c.openAIModel, "grok")
}
