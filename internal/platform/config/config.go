package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aead/chacha20poly1305"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	AppEnv      string `envconfig:"APP_ENV" default:"development"`
	AppPort     string `envconfig:"APP_PORT" default:"8080"`
	AppLogLevel string `envconfig:"APP_LOG_LEVEL" default:"debug"`

	DatabaseURL       string        `envconfig:"DATABASE_URL" required:"true"`
	TestDatabaseURL   string        `envconfig:"TEST_DATABASE_URL"`
	TokenSymmetricKey string        `envconfig:"TOKEN_SYMMETRIC_KEY" required:"true"`
	AccessTokenTTL    time.Duration `envconfig:"ACCESS_TOKEN_TTL" default:"15m"`
	RefreshTokenTTL   time.Duration `envconfig:"REFRESH_TOKEN_TTL" default:"168h"` // 7 days

	CorsAllowedOrigins []string `envconfig:"CORS_ALLOWED_ORIGINS" required:"true"`
}

func Load() (*Config, error) {
	if path := findEnvFile(); path != "" {
		_ = godotenv.Load(path)
	}

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}
func (c *Config) validate() error {
	if len(c.TokenSymmetricKey) != chacha20poly1305.KeySize {
		return fmt.Errorf(
			"TOKEN_SYMMETRIC_KEY must be exactly %d bytes, got %d",
			chacha20poly1305.KeySize, len(c.TokenSymmetricKey),
		)
	}
	if c.AccessTokenTTL >= c.RefreshTokenTTL {
		return fmt.Errorf(
			"ACCESS_TOKEN_TTL (%s) must be shorter than REFRESH_TOKEN_TTL (%s)",
			c.AccessTokenTTL, c.RefreshTokenTTL,
		)
	}
	return nil
}

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
