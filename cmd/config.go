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
	// plausible analytics tag for page view tracking.
	// set to zero value to disable tracking.
	// e.g. pa-vckXXX
	plausibleSnippetTag string
	demoUserID          int64
	openAIAPIKey        string
	openAIModel         string
	braveKey            string
	pushoverKey         string
	chrome              struct {
		devToolsURL           string
		maxConcurrentSessions int
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

func (c *config) pushoverConfigured() bool {
	return c.pushoverKey != ""
}

func (c *config) emailSendConfigured() bool {
	return c.smtp.host != "" && c.smtp.port != 0 && c.smtp.from != ""
}
