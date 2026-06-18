package config

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	AppEnv      string `envconfig:"APP_ENV" default:"development"`
	AppPort     string `envconfig:"APP_PORT" default:"8080"`
	AppLogLevel string `envconfig:"APP_LOG_LEVEL" default:"debug"`

	DatabaseURL     string `envconfig:"DATABASE_URL" required:"true"`
	TestDatabaseURL string `envconfig:"TEST_DATABASE_URL"`

	CorsAllowedOrigins []string `envconfig:"CORS_ALLOWED_ORIGINS" required:"true"`
}

func Load() (*Config, error) {
	// In dev, load .env if present. In prod (Fly), env vars come from Fly secrets.
	// Search upward from the current dir so it works regardless of where the
	// process runs (e.g. `go test` runs from each package's own directory).
	if path := findEnvFile(); path != "" {
		_ = godotenv.Load(path)
	}

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// findEnvFile walks up from the current working directory looking for a .env
// file, returning its path (or "" if none is found before the filesystem root).
// This lets tests, which run from their own package directory, still find the
// project-root .env.
func findEnvFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, ".env")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "" // reached filesystem root
		}
		dir = parent
	}
}
