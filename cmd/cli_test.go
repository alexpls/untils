package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func envMap(values map[string]string) envLookup {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func TestNormalizeBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "strips trailing slash",
			input: "http://untils.localhost:4200/",
			want:  "http://untils.localhost:4200",
		},
		{
			name:  "keeps path",
			input: "https://untils.example.com/base/",
			want:  "https://untils.example.com/base",
		},
		{
			name:    "rejects missing host",
			input:   "http:///app",
			wantErr: true,
		},
		{
			name:    "rejects unsupported scheme",
			input:   "mailto:test@example.com",
			wantErr: true,
		},
		{
			name:    "rejects query",
			input:   "https://untils.example.com?foo=bar",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeBaseURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseServeArgsLoadsFromEnv(t *testing.T) {
	t.Parallel()

	globalCfg, serveCfg := parseServeArgs(
		[]string{"untils", "serve"},
		envMap(map[string]string{
			"ENV":            appEnvDev.String(),
			"APP_PORT":       "3322",
			"BASE_URL":       "http://localhost:3322/",
			"PG_URL":         "postgresql://postgres:postgres@db:5432/untils",
			"ADMIN_EMAIL":    "admin@example.com",
			"SMTP_FROM":      "notifications@untils.local",
			"SMTP_HOST":      "mail.local",
			"SMTP_PORT":      "2025",
			"APP_MODE":       appModeHosted.String(),
			"MIGRATE":        "true",
			"BRAVE_KEY":      "brave",
			"OPENAI_API_KEY": "openai",
			"PUSHOVER_KEY":   "pushover",
		}),
	)

	if globalCfg.env != appEnvDev {
		t.Fatalf("got env %q, want %q", globalCfg.env, appEnvDev)
	}
	if globalCfg.baseURL != "http://localhost:3322" {
		t.Fatalf("got baseURL %q, want %q", globalCfg.baseURL, "http://localhost:3322")
	}
	if globalCfg.dbUrl != "postgresql://postgres:postgres@db:5432/untils" {
		t.Fatalf("got dbUrl %q", globalCfg.dbUrl)
	}
	if globalCfg.adminEmail != "admin@example.com" {
		t.Fatalf("got adminEmail %q, want %q", globalCfg.adminEmail, "admin@example.com")
	}
	if globalCfg.appMode != appModeHosted {
		t.Fatalf("got appMode %q, want %q", globalCfg.appMode, appModeHosted)
	}
	if !globalCfg.migrate {
		t.Fatalf("expected migrate to be true")
	}
	if globalCfg.smtp.host != "mail.local" {
		t.Fatalf("got smtp host %q", globalCfg.smtp.host)
	}
	if globalCfg.smtp.port != 2025 {
		t.Fatalf("got smtp port %d, want %d", globalCfg.smtp.port, 2025)
	}
	if globalCfg.chrome.maxConcurrentSessions != 5 {
		t.Fatalf("got max concurrent browser sessions %d, want %d", globalCfg.chrome.maxConcurrentSessions, 5)
	}
	if serveCfg.port != 3322 {
		t.Fatalf("got port %d, want %d", serveCfg.port, 3322)
	}
	assert.Equal(t, "openai", globalCfg.openAIAPIKey)
	assert.Equal(t, "gpt-5.4", globalCfg.openAIModel)
}

func TestParseServeArgsLoadsBrowserMaxConcurrentSessionsFromEnv(t *testing.T) {
	t.Parallel()

	globalCfg, _ := parseServeArgs(
		[]string{"untils", "serve"},
		envMap(map[string]string{
			"BASE_URL":                        "http://localhost:3322",
			"ADMIN_EMAIL":                     "admin@example.com",
			"BROWSER_MAX_CONCURRENT_SESSIONS": "7",
		}),
	)

	assert.Equal(t, 7, globalCfg.chrome.maxConcurrentSessions)
}

func TestParseServeArgsRejectsCLIArgs(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != "serve does not accept CLI flags or args; use environment variables instead" {
			t.Fatalf("got panic %v", r)
		}
	}()

	parseServeArgs([]string{"untils", "serve", "-port=4201"}, envMap(nil))
}

func TestParseServeArgsRequiresAdminEmailInSelfHostedMode(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != "ADMIN_EMAIL is required in selfhosted mode" {
			t.Fatalf("got panic %v", r)
		}
	}()

	parseServeArgs(
		[]string{"untils", "serve"},
		envMap(map[string]string{
			"BASE_URL": "http://localhost:3322",
		}),
	)
}

func TestParseServeArgsAllowsMissingNotifierConfigInSelfHostedMode(t *testing.T) {
	t.Parallel()

	globalCfg, _ := parseServeArgs(
		[]string{"untils", "serve"},
		envMap(map[string]string{
			"APP_MODE":    appModeSelfHosted.String(),
			"BASE_URL":    "http://localhost:3322",
			"ADMIN_EMAIL": "admin@example.com",
		}),
	)

	assert.False(t, globalCfg.emailSendConfigured())
	assert.False(t, globalCfg.pushoverConfigured())
}

func TestParseServeArgsRequiresNotifierConfigInHostedMode(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != "PUSHOVER_KEY is required in hosted mode" {
			t.Fatalf("got panic %v", r)
		}
	}()

	parseServeArgs(
		[]string{"untils", "serve"},
		envMap(map[string]string{
			"APP_MODE":  appModeHosted.String(),
			"BASE_URL":  "http://localhost:3322",
			"SMTP_HOST": "mail.local",
			"SMTP_PORT": "2025",
			"SMTP_FROM": "notifications@untils.local",
		}),
	)
}

func TestParseServeArgsAllowsMissingSMTPHostInHostedMode(t *testing.T) {
	t.Parallel()

	globalCfg, _ := parseServeArgs(
		[]string{"untils", "serve"},
		envMap(map[string]string{
			"APP_MODE":     appModeHosted.String(),
			"BASE_URL":     "http://localhost:3322",
			"PUSHOVER_KEY": "pushover",
			"SMTP_PORT":    "2025",
			"SMTP_FROM":    "notifications@untils.local",
		}),
	)

	assert.False(t, globalCfg.emailSendConfigured())
}

func TestParseServeArgsRequiresCompleteSMTPConfigWhenSMTPHostIsSetInHostedMode(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != "SMTP_HOST, SMTP_PORT, and SMTP_FROM are required in hosted mode" {
			t.Fatalf("got panic %v", r)
		}
	}()

	parseServeArgs(
		[]string{"untils", "serve"},
		envMap(map[string]string{
			"APP_MODE":     appModeHosted.String(),
			"BASE_URL":     "http://localhost:3322",
			"PUSHOVER_KEY": "pushover",
			"SMTP_HOST":    "mail.local",
		}),
	)
}

func TestParseMigrateArgsLoadsFromEnv(t *testing.T) {
	t.Parallel()

	cfg := parseMigrateArgs(
		[]string{"untils", "migrate"},
		envMap(map[string]string{
			"PG_URL": "postgresql://postgres:postgres@db:5432/untils",
		}),
	)

	if cfg.dbUrl != "postgresql://postgres:postgres@db:5432/untils" {
		t.Fatalf("got dbUrl %q", cfg.dbUrl)
	}
}

func TestMigrationDriverURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{
			input: "postgresql://postgres:postgres@db:5432/untils?sslmode=disable",
			want:  "pgx5://postgres:postgres@db:5432/untils?sslmode=disable",
		},
		{
			input: "postgres://postgres:postgres@db:5432/untils",
			want:  "pgx5://postgres:postgres@db:5432/untils",
		},
		{
			input: "pgx5://postgres:postgres@db:5432/untils",
			want:  "pgx5://postgres:postgres@db:5432/untils",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := migrationDriverURL(tt.input)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
