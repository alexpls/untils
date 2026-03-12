package main

import (
	"flag"
	"fmt"
	"io"
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
	name         string
	description  string
	flagName     string
	envVar       string
	defaultValue *string
	apply        func(string) error
	validate     func() error
}

func subcommand() string {
	return subcommandFromArgs(os.Args)
}

func subcommandFromArgs(args []string) string {
	if len(args) < 2 || !slices.Contains(validSubcommands, args[1]) {
		var allowedStr string
		for i, a := range validSubcommands {
			allowedStr += "'" + a + "'"
			if i < len(validSubcommands)-1 {
				allowedStr += " or "
			}
		}
		panic(fmt.Sprintf("unknown subcommand. specify %s.", allowedStr))
	}
	return args[1]
}

func parseServe() (*config, *serveConfig) {
	return parseServeArgs(os.Args, os.LookupEnv)
}

func parseServeArgs(args []string, lookupEnv envLookup) (*config, *serveConfig) {
	must.True(args[1] == "serve")

	gc := config{buildVersion: buildVersion()}
	sc := serveConfig{}

	loadConfigProperties("serve", args[2:], lookupEnv, append(globalProperties(&gc), serveProperties(&sc)...))

	if gc.baseURL == "" {
		gc.baseURL = fmt.Sprintf("http://localhost:%d", sc.port)
	}

	validateGlobalConfig(&gc)
	validateServeConfig(&sc)

	return &gc, &sc
}

func parseSeed() *config {
	return parseSeedArgs(os.Args, os.LookupEnv)
}

func parseSeedArgs(args []string, lookupEnv envLookup) *config {
	must.True(args[1] == "seed")

	gc := config{buildVersion: buildVersion()}

	loadConfigProperties("seed", args[2:], lookupEnv, globalProperties(&gc))

	if gc.baseURL == "" {
		gc.baseURL = "http://localhost:4200"
	}

	validateGlobalConfig(&gc)

	return &gc
}

func globalProperties(c *config) []configProperty {
	return []configProperty{
		enumProperty(
			"env",
			"environment",
			"env",
			"ENV",
			appEnvProd.String(),
			func(value string) error {
				c.env = constants.Env(value)
				return nil
			},
			appEnvDev.String(), appEnvProd.String(),
		),
		enumProperty(
			"app mode",
			"application mode",
			"app-mode",
			"APP_MODE",
			appModeSelfHosted.String(),
			func(value string) error {
				c.appMode = constants.Mode(value)
				return nil
			},
			appModeSelfHosted.String(), appModeHosted.String(),
		),
		boolProperty(
			"migrate",
			"run pending database migrations during startup",
			"migrate",
			"MIGRATE",
			"false",
			func(value bool) { c.migrate = value },
			nil,
		),
		stringProperty(
			"base url",
			"public application base url",
			"base-url",
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
			"database url",
			"postgresql connection url",
			"db",
			"PG_URL",
			"",
			func(value string) { c.dbUrl = value },
			nil,
		),
		int64Property(
			"demo user id",
			"user id used for demo-mode requests",
			"demo-user-id",
			"DEMO_USER_ID",
			"0",
			func(value int64) { c.demoUserID = value },
			nil,
		),
		stringProperty(
			"x.ai api key",
			"x.ai API key",
			"xai-key",
			"XAI_KEY",
			"",
			func(value string) { c.xAIKey = value },
			nil,
		),
		stringProperty(
			"openai api key",
			"OpenAI API key",
			"openai-key",
			"OPENAI_KEY",
			"",
			func(value string) { c.openAIKey = value },
			nil,
		),
		stringProperty(
			"brave api key",
			"Brave search API key",
			"brave-key",
			"BRAVE_KEY",
			"",
			func(value string) { c.braveKey = value },
			nil,
		),
		stringProperty(
			"pushover api key",
			"Pushover API key",
			"pushover-key",
			"PUSHOVER_KEY",
			"",
			func(value string) { c.pushoverKey = value },
			nil,
		),
		stringProperty(
			"smtp username",
			"smtp username",
			"smtp-username",
			"SMTP_USERNAME",
			"",
			func(value string) { c.smtp.username = value },
			nil,
		),
		stringProperty(
			"smtp password",
			"smtp password",
			"smtp-password",
			"SMTP_PASSWORD",
			"",
			func(value string) { c.smtp.password = value },
			nil,
		),
		stringProperty(
			"smtp host",
			"smtp host",
			"smtp-host",
			"SMTP_HOST",
			"127.0.0.1",
			func(value string) { c.smtp.host = value },
			nil,
		),
		intProperty(
			"smtp port",
			"smtp port",
			"smtp-port",
			"SMTP_PORT",
			"1025",
			func(value int) { c.smtp.port = value },
			nil,
		),
		stringProperty(
			"smtp from",
			"smtp from email address",
			"smtp-from",
			"SMTP_FROM",
			"notifications@untils.com",
			func(value string) { c.smtp.from = value },
			func() error {
				if c.smtp.from == "" {
					return fmt.Errorf("smtp-from is required")
				}
				return nil
			},
		),
		stringProperty(
			"chrome devtools url",
			"chrome devtools url",
			"chrome-devtools-url",
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
			"app port",
			"http server port",
			"port",
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
			"database url",
			"postgresql connection url",
			"db",
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

func stringProperty(name string, description string, flagName string, envVar string, defaultValue string, assign func(string), validate func() error) configProperty {
	return configProperty{
		name:         name,
		description:  description,
		flagName:     flagName,
		envVar:       envVar,
		defaultValue: stringPtr(defaultValue),
		apply: func(value string) error {
			assign(value)
			return nil
		},
		validate: validate,
	}
}

func intProperty(name string, description string, flagName string, envVar string, defaultValue string, assign func(int), validate func() error) configProperty {
	return configProperty{
		name:         name,
		description:  description,
		flagName:     flagName,
		envVar:       envVar,
		defaultValue: stringPtr(defaultValue),
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

func int64Property(name string, description string, flagName string, envVar string, defaultValue string, assign func(int64), validate func() error) configProperty {
	return configProperty{
		name:         name,
		description:  description,
		flagName:     flagName,
		envVar:       envVar,
		defaultValue: stringPtr(defaultValue),
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

func boolProperty(name string, description string, flagName string, envVar string, defaultValue string, assign func(bool), validate func() error) configProperty {
	return configProperty{
		name:         name,
		description:  description,
		flagName:     flagName,
		envVar:       envVar,
		defaultValue: stringPtr(defaultValue),
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

func enumProperty(name string, description string, flagName string, envVar string, defaultValue string, assign func(string) error, allowed ...string) configProperty {
	return configProperty{
		name:         name,
		description:  description,
		flagName:     flagName,
		envVar:       envVar,
		defaultValue: stringPtr(defaultValue),
		apply: func(value string) error {
			if !slices.Contains(allowed, value) {
				return fmt.Errorf("must be one of %s", strings.Join(allowed, ", "))
			}
			return assign(value)
		},
	}
}

func stringPtr(value string) *string {
	return &value
}

func loadConfigProperties(name string, args []string, lookupEnv envLookup, properties []configProperty) {
	for _, property := range properties {
		if property.defaultValue == nil {
			continue
		}
		must.NoErr(property.apply(*property.defaultValue))
	}

	flagSet := flag.NewFlagSet(name, flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)

	providedFlags := map[string]bool{}
	for _, property := range properties {
		if property.flagName == "" {
			continue
		}

		property := property
		flagSet.Func(property.flagName, propertyUsage(property), func(value string) error {
			providedFlags[property.flagName] = true
			if err := property.apply(value); err != nil {
				return fmt.Errorf("%s: %w", property.flagName, err)
			}
			return nil
		})
	}

	if err := flagSet.Parse(args); err != nil {
		panic(err.Error())
	}

	for _, property := range properties {
		if property.envVar == "" {
			continue
		}
		if property.flagName != "" && providedFlags[property.flagName] {
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

func propertyUsage(property configProperty) string {
	parts := []string{property.description}
	if property.envVar != "" {
		parts = append(parts, "env: "+property.envVar)
	}
	if property.defaultValue != nil && *property.defaultValue != "" {
		parts = append(parts, "default: "+*property.defaultValue)
	}
	return strings.Join(parts, "; ")
}

func validateGlobalConfig(c *config) {
	if c.env != appEnvProd && c.env != appEnvDev {
		panic("env must be either prod or dev")
	}
	if c.appMode != appModeSelfHosted && c.appMode != appModeHosted {
		panic("app-mode must be either selfhosted or hosted")
	}
	if c.baseURL == "" {
		panic("base-url is required")
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

	mc := migrateConfig{}
	loadConfigProperties("migrate", args[2:], lookupEnv, migrateProperties(&mc))

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
