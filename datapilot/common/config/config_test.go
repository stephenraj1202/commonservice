package config_test

// Feature: datapilot-platform, Property 1: Config loading round-trip
// Feature: datapilot-platform, Property 2: Missing config key produces named error

import (
	"strings"
	"testing"

	"datapilot/common/config"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// setEnv sets a map of env vars and returns a cleanup function.
func setEnv(t *testing.T, vars map[string]string) func() {
	t.Helper()
	for k, v := range vars {
		t.Setenv(k, v)
	}
	return func() {} // t.Setenv handles cleanup automatically
}

// --- Unit tests ---

func TestLoadConfig_AllRequired_Success(t *testing.T) {
	setEnv(t, map[string]string{
		"SERVICE_NAME": "test-svc",
		"HTTP_PORT":    "8080",
		"MYSQL_DSN":    "user:pass@tcp(localhost:3306)/db",
		"JWT_SECRET":   "supersecret",
	})

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ServiceName != "test-svc" {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, "test-svc")
	}
	if cfg.HTTPPort != "8080" {
		t.Errorf("HTTPPort = %q, want %q", cfg.HTTPPort, "8080")
	}
	if cfg.MySQLDSN != "user:pass@tcp(localhost:3306)/db" {
		t.Errorf("MySQLDSN mismatch")
	}
	if cfg.JWTSecret != "supersecret" {
		t.Errorf("JWTSecret mismatch")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	setEnv(t, map[string]string{
		"SERVICE_NAME": "svc",
		"HTTP_PORT":    "9090",
		"MYSQL_DSN":    "dsn",
		"JWT_SECRET":   "secret",
	})

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.FileStoragePath != "/data/files" {
		t.Errorf("FileStoragePath default = %q, want /data/files", cfg.FileStoragePath)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel default = %q, want info", cfg.LogLevel)
	}
	if cfg.AllowedOrigins != "*" {
		t.Errorf("AllowedOrigins default = %q, want *", cfg.AllowedOrigins)
	}
}

func TestLoadConfig_MissingRequired_Error(t *testing.T) {
	required := []string{"SERVICE_NAME", "HTTP_PORT", "MYSQL_DSN", "JWT_SECRET"}
	all := map[string]string{
		"SERVICE_NAME": "svc",
		"HTTP_PORT":    "8080",
		"MYSQL_DSN":    "dsn",
		"JWT_SECRET":   "secret",
	}

	for _, missing := range required {
		t.Run("missing_"+missing, func(t *testing.T) {
			for k, v := range all {
				if k != missing {
					t.Setenv(k, v)
				}
			}
			_, err := config.LoadConfig()
			if err == nil {
				t.Fatalf("expected error for missing %s, got nil", missing)
			}
			if !strings.Contains(err.Error(), missing) {
				t.Errorf("error %q does not mention missing key %q", err.Error(), missing)
			}
		})
	}
}

// --- Property tests ---

// nonEmptyString generates non-empty strings suitable for config values.
func nonEmptyString() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 })
}

// Property 1: Config loading round-trip
// Validates: Requirements 1.1
func TestProperty1_ConfigRoundTrip(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)

	properties.Property("LoadConfig returns Config matching supplied env vars", prop.ForAll(
		func(svcName, port, dsn, secret, storagePath, logLevel, origins string) bool {
			t.Setenv("SERVICE_NAME", svcName)
			t.Setenv("HTTP_PORT", port)
			t.Setenv("MYSQL_DSN", dsn)
			t.Setenv("JWT_SECRET", secret)
			t.Setenv("FILE_STORAGE_PATH", storagePath)
			t.Setenv("LOG_LEVEL", logLevel)
			t.Setenv("ALLOWED_ORIGINS", origins)

			cfg, err := config.LoadConfig()
			if err != nil {
				return false
			}
			return cfg.ServiceName == svcName &&
				cfg.HTTPPort == port &&
				cfg.MySQLDSN == dsn &&
				cfg.JWTSecret == secret &&
				cfg.FileStoragePath == storagePath &&
				cfg.LogLevel == logLevel &&
				cfg.AllowedOrigins == origins
		},
		nonEmptyString(),
		nonEmptyString(),
		nonEmptyString(),
		nonEmptyString(),
		nonEmptyString(),
		nonEmptyString(),
		nonEmptyString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Property 2: Missing config key produces named error
// Validates: Requirements 1.2
func TestProperty2_MissingKeyNamedError(t *testing.T) {
	requiredKeys := []string{"SERVICE_NAME", "HTTP_PORT", "MYSQL_DSN", "JWT_SECRET"}
	allValues := map[string]string{
		"SERVICE_NAME": "svc",
		"HTTP_PORT":    "8080",
		"MYSQL_DSN":    "dsn",
		"JWT_SECRET":   "secret",
	}

	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)

	properties.Property("error message contains the name of the missing required key", prop.ForAll(
		func(missingIdx int) bool {
			missingKey := requiredKeys[missingIdx]
			for k, v := range allValues {
				if k != missingKey {
					t.Setenv(k, v)
				} else {
					t.Setenv(k, "") // ensure it's absent/empty
				}
			}

			_, err := config.LoadConfig()
			if err == nil {
				return false
			}
			return strings.Contains(err.Error(), missingKey)
		},
		gen.IntRange(0, len(requiredKeys)-1),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
