package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/alexpls/untils/internal/must"
)

var validSubcommands = []string{"serve", "seed", "migrate"}

func subcommand() string {
	if len(os.Args) < 2 || !slices.Contains(validSubcommands, os.Args[1]) {
		var allowedStr string
		for i, a := range validSubcommands {
			allowedStr += "'" + a + "'"
			if i < len(validSubcommands)-1 {
				allowedStr += " or "
			}
		}
		panic(fmt.Sprintf("unknown subcommand. specify %s.", allowedStr))
	}
	return os.Args[1]
}

func parseServe() (*config, *serveConfig) {
	must.True(os.Args[1] == "serve")

	f := flag.NewFlagSet("serve", flag.ExitOnError)

	gc := config{}
	globalFlags(&gc, f)

	sc := serveConfig{}
	serveFlags(&sc, f)

	_ = f.Parse(os.Args[2:])

	if gc.baseURL == "" {
		gc.baseURL = fmt.Sprintf("http://localhost:%d", sc.port)
	}

	validateGlobalConfig(&gc)
	validateServeConfig(&sc)

	return &gc, &sc
}

func parseSeed() *config {
	must.True(os.Args[1] == "seed")

	f := flag.NewFlagSet("seed", flag.ExitOnError)

	gc := config{}
	globalFlags(&gc, f)

	_ = f.Parse(os.Args[2:])

	if gc.baseURL == "" {
		gc.baseURL = "http://localhost:4200"
	}

	validateGlobalConfig(&gc)

	return &gc
}

func globalFlags(c *config, f *flag.FlagSet) {
	c.buildVersion = buildVersion()

	f.StringVar(&c.env, "env", "prod", "environment (dev, prod)")
	f.Func("app-mode", "application mode (selfhosted, hosted)", func(value string) error {
		c.appMode = appMode(value)
		return nil
	})
	c.appMode = appModeSelfHosted
	f.StringVar(&c.baseURL, "base-url", "", "public application base url")
	f.StringVar(&c.dbUrl, "db", "", "postgresql connection url")
	f.Int64Var(&c.demoUserID, "demo-user-id", 0, "user id used for demo-mode requests")
	f.StringVar(&c.xAIKey, "xai-key", "", "x.ai API key")
	f.StringVar(&c.openAIKey, "openai-key", "", "OpenAI API key")
	f.StringVar(&c.braveKey, "brave-key", "", "Brave search API key")
	f.StringVar(&c.pushoverKey, "pushover-key", "", "Pushover API key")
	f.StringVar(&c.smtp.username, "smtp-username", "", "smtp username")
	f.StringVar(&c.smtp.password, "smtp-password", "", "smtp password")
	f.StringVar(&c.smtp.host, "smtp-host", "127.0.0.1", "smtp host")
	f.IntVar(&c.smtp.port, "smtp-port", 1025, "smtp port")
}

func serveFlags(c *serveConfig, f *flag.FlagSet) {
	f.IntVar(&c.port, "port", 4200, "http server port")
}

func validateGlobalConfig(c *config) {
	if c.env != "prod" && c.env != "dev" {
		panic("env must be either prod or dev")
	}
	if c.appMode != appModeSelfHosted && c.appMode != appModeHosted {
		panic("app-mode must be either selfhosted or hosted")
	}
	baseURL, err := normalizeBaseURL(c.baseURL)
	if err != nil {
		panic(err.Error())
	}
	c.baseURL = baseURL
}

func validateServeConfig(c *serveConfig) {
	// no validations yet
}

type migrateConfig struct {
	dbUrl string
}

func parseMigrate() *migrateConfig {
	must.True(os.Args[1] == "migrate")

	f := flag.NewFlagSet("migrate", flag.ExitOnError)

	mc := migrateConfig{}
	f.StringVar(&mc.dbUrl, "db", "", "postgresql connection url")

	_ = f.Parse(os.Args[2:])

	if mc.dbUrl == "" {
		panic("db url is required")
	}

	return &mc
}

func buildVersion() string {
	var revision string
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	for _, setting := range buildInfo.Settings {
		if setting.Key == "vcs.revision" {
			revision = setting.Value[0:7]
			break
		}
	}

	if revision == "" {
		return "unknown"
	}

	return revision
}

func normalizeBaseURL(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("base-url is required")
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("invalid base-url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("base-url must use http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("base-url must include a host")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("base-url must not include query params or fragments")
	}

	parsed.Path = strings.TrimSuffix(parsed.Path, "/")

	return parsed.String(), nil
}
