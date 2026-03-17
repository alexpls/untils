package main

import (
	"fmt"
	"net/url"
	"os"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"

	"github.com/alexpls/untils/internal/constants"
	"github.com/alexpls/untils/internal/must"
)

var validSubcommands = []string{"serve", "seed", "migrate"}

type envLookup func(string) (string, bool)

type configProperty struct {
	envVar       string
	defaultValue string
	apply        func(string) error
	validate     func() error
}

func subcommand() string {
	return subcommandFromArgs(os.Args)
}

func subcommandFromArgs(args []string) string {
	if len(args) < 2 || !slices.Contains(validSubcommands, args[1]) {
		panic("unknown subcommand. specify serve, command, or migrate.")
	}
	return args[1]
}

func parseServe() (*config, *serveConfig) {
	return parseServeArgs(os.Args, os.LookupEnv)
}

func parseServeArgs(args []string, lookupEnv envLookup) (*config, *serveConfig) {
	must.True(args[1] == "serve")
	requireNoArgs("serve", args[2:])

	gc := config{buildVersion: buildVersion()}
	sc := serveConfig{}

	loadConfigProperties(lookupEnv, append(globalProperties(&gc), serveProperties(&sc)...))

	if gc.baseURL == "" {
		gc.baseURL = fmt.Sprintf("http://localhost:%d", sc.port)
	}

	validateGlobalConfig(&gc)
	validateServeGlobalConfig(&gc)
	validateServeConfig(&sc)

	return &gc, &sc
}

func parseSeed() *config {
	return parseSeedArgs(os.Args, os.LookupEnv)
}

func parseSeedArgs(args []string, lookupEnv envLookup) *config {
	must.True(args[1] == "seed")
	requireNoArgs("seed", args[2:])

	gc := config{buildVersion: buildVersion()}

	loadConfigProperties(lookupEnv, globalProperties(&gc))

	if gc.baseURL == "" {
		gc.baseURL = "http://localhost:4200"
	}

	validateGlobalConfig(&gc)

	return &gc
}

func globalProperties(c *config) []configProperty {
	return []configProperty{
		enumProperty(
			"ENV",
			appEnvProd.String(),
			func(value string) error {
				c.env = constants.Env(value)
				return nil
			},
			appEnvDev.String(), appEnvProd.String(),
		),
		enumProperty(
			"APP_MODE",
			appModeSelfHosted.String(),
			func(value string) error {
				c.appMode = constants.Mode(value)
				return nil
			},
			appModeSelfHosted.String(), appModeHosted.String(),
		),
		boolProperty(
			"MIGRATE",
			"true",
			func(value bool) { c.migrate = value },
			nil,
		),
		stringProperty(
			"BASE_URL",
			"",
			func(value string) { c.baseURL = value },
			func() error {
				if c.baseURL == "" {
					return nil
				}
				normalized, err := normalizeBaseURL(c.baseURL)
				if err != nil {
					return err
				}
				c.baseURL = normalized
				return nil
			},
		),
		stringProperty(
			"PG_URL",
			"",
			func(value string) { c.dbUrl = value },
			nil,
		),
		stringProperty(
			"ADMIN_EMAIL",
			"",
			func(value string) { c.adminEmail = value },
			nil,
		),
		int64Property(
			"DEMO_USER_ID",
			"0",
			func(value int64) { c.demoUserID = value },
			nil,
		),
		stringProperty(
			"OPENAI_API_KEY",
			"",
			func(value string) { c.openAIAPIKey = value },
			nil,
		),
		stringProperty(
			"OPENAI_MODEL",
			"gpt-5.4",
			func(value string) { c.openAIModel = value },
			nil,
		),
		stringProperty(
			"BRAVE_KEY",
			"",
			func(value string) { c.braveKey = value },
			nil,
		),
		stringProperty(
			"PUSHOVER_KEY",
			"",
			func(value string) { c.pushoverKey = value },
			nil,
		),
		stringProperty(
			"SMTP_USERNAME",
			"",
			func(value string) { c.smtp.username = value },
			nil,
		),
		stringProperty(
			"SMTP_PASSWORD",
			"",
			func(value string) { c.smtp.password = value },
			nil,
		),
		stringProperty(
			"SMTP_HOST",
			"",
			func(value string) { c.smtp.host = value },
			nil,
		),
		intProperty(
			"SMTP_PORT",
			"0",
			func(value int) { c.smtp.port = value },
			nil,
		),
		stringProperty(
			"SMTP_FROM",
			"",
			func(value string) { c.smtp.from = value },
			nil,
		),
		stringProperty(
			"CHROME_DEVTOOLS_URL",
			"",
			func(value string) { c.chrome.devToolsURL = value },
			func() error {
				if c.chrome.devToolsURL == "" {
					return nil
				}
				_, err := url.Parse(c.chrome.devToolsURL)
				if err != nil {
					return fmt.Errorf("chrome-devtools-url invalid: %w", err)
				}
				return nil
			},
		),
	}
}

func serveProperties(c *serveConfig) []configProperty {
	return []configProperty{
		intProperty(
			"APP_PORT",
			"4200",
			func(value int) { c.port = value },
			nil,
		),
	}
}

func migrateProperties(c *migrateConfig) []configProperty {
	return []configProperty{
		stringProperty(
			"PG_URL",
			"",
			func(value string) { c.dbUrl = value },
			func() error {
				if c.dbUrl == "" {
					return fmt.Errorf("db url is required")
				}
				return nil
			},
		),
	}
}

func stringProperty(envVar string, defaultValue string, assign func(string), validate func() error) configProperty {
	return configProperty{
		envVar:       envVar,
		defaultValue: defaultValue,
		apply: func(value string) error {
			assign(value)
			return nil
		},
		validate: validate,
	}
}

func intProperty(envVar string, defaultValue string, assign func(int), validate func() error) configProperty {
	return configProperty{
		envVar:       envVar,
		defaultValue: defaultValue,
		apply: func(value string) error {
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("must be an integer: %w", err)
			}
			assign(parsed)
			return nil
		},
		validate: validate,
	}
}

func int64Property(envVar string, defaultValue string, assign func(int64), validate func() error) configProperty {
	return configProperty{
		envVar:       envVar,
		defaultValue: defaultValue,
		apply: func(value string) error {
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("must be a 64-bit integer: %w", err)
			}
			assign(parsed)
			return nil
		},
		validate: validate,
	}
}

func boolProperty(envVar string, defaultValue string, assign func(bool), validate func() error) configProperty {
	return configProperty{
		envVar:       envVar,
		defaultValue: defaultValue,
		apply: func(value string) error {
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("must be a boolean: %w", err)
			}
			assign(parsed)
			return nil
		},
		validate: validate,
	}
}

func enumProperty(envVar string, defaultValue string, assign func(string) error, allowed ...string) configProperty {
	return configProperty{
		envVar:       envVar,
		defaultValue: defaultValue,
		apply: func(value string) error {
			if !slices.Contains(allowed, value) {
				return fmt.Errorf("must be one of %s", strings.Join(allowed, ", "))
			}
			return assign(value)
		},
	}
}

func loadConfigProperties(lookupEnv envLookup, properties []configProperty) {
	for _, property := range properties {
		must.NoErr(property.apply(property.defaultValue))
	}

	for _, property := range properties {
		if property.envVar == "" {
			continue
		}
		value, ok := lookupEnv(property.envVar)
		if !ok {
			continue
		}

		if err := property.apply(value); err != nil {
			panic(fmt.Sprintf("%s: %v", property.envVar, err))
		}
	}

	for _, property := range properties {
		if property.validate == nil {
			continue
		}
		if err := property.validate(); err != nil {
			panic(err.Error())
		}
	}
}

func validateGlobalConfig(c *config) {
	if c.env != appEnvProd && c.env != appEnvDev {
		panic("ENV must be either prod or dev")
	}
	if c.appMode != appModeSelfHosted && c.appMode != appModeHosted {
		panic("APP_MODE must be either selfhosted or hosted")
	}
	if c.baseURL == "" {
		panic("BASE_URL is required")
	}
}

func validateServeGlobalConfig(c *config) {
	if c.appMode == appModeSelfHosted && c.adminEmail == "" {
		panic("ADMIN_EMAIL is required in selfhosted mode")
	}

	if c.appMode == appModeHosted && !c.pushoverConfigured() {
		panic("PUSHOVER_KEY is required in hosted mode")
	}
	if c.appMode == appModeHosted && !c.emailSendConfigured() {
		panic("SMTP_HOST, SMTP_PORT, and SMTP_FROM are required in hosted mode")
	}
}

func validateServeConfig(c *serveConfig) {
	// no validations yet
}

type migrateConfig struct {
	dbUrl string
}

func parseMigrate() *migrateConfig {
	return parseMigrateArgs(os.Args, os.LookupEnv)
}

func parseMigrateArgs(args []string, lookupEnv envLookup) *migrateConfig {
	must.True(args[1] == "migrate")
	requireNoArgs("migrate", args[2:])

	mc := migrateConfig{}
	loadConfigProperties(lookupEnv, migrateProperties(&mc))

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

func requireNoArgs(command string, args []string) {
	if len(args) == 0 {
		return
	}

	panic(fmt.Sprintf("%s does not accept CLI flags or args; use environment variables instead", command))
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
