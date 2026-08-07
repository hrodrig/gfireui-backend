// Package config loads gfireui-backend YAML configuration and environment overrides.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the root configuration document (gfireui-backend.yaml).
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Auth      AuthConfig      `mapstructure:"auth"`
	GFire     GFireConfig     `mapstructure:"gfire"`
	Bootstrap BootstrapConfig `mapstructure:"bootstrap"`
}

// ServerConfig holds HTTP listener settings.
type ServerConfig struct {
	Addr string `mapstructure:"addr"`
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	DSN string `mapstructure:"dsn"`
}

// AuthConfig holds JWT signing settings for browser sessions.
type AuthConfig struct {
	JWTSecret string        `mapstructure:"jwt_secret"`
	TokenTTL  time.Duration `mapstructure:"token_ttl"`
}

// GFireConfig holds upstream GFire service connection settings.
type GFireConfig struct {
	BaseURL string `mapstructure:"base_url"`
	Token   string `mapstructure:"token"`
}

// BootstrapConfig holds first-boot Administrator creation credentials.
type BootstrapConfig struct {
	AdminEmail      string `mapstructure:"admin_email"`
	AdminPassword   string `mapstructure:"admin_password"`
	AdminFirstName  string `mapstructure:"admin_first_name"`
	AdminLastName   string `mapstructure:"admin_last_name"`
}

// Defaults returns a configuration with v0.1 defaults.
func Defaults() Config {
	return Config{
		Server: ServerConfig{
			Addr: ":8090",
		},
		Auth: AuthConfig{
			TokenTTL: 24 * time.Hour,
		},
	}
}

func bindEnv(v *viper.Viper) {
	keys := []string{
		"server.addr",
		"database.dsn",
		"auth.jwt_secret",
		"auth.token_ttl",
		"gfire.base_url",
		"gfire.token",
		"bootstrap.admin_email",
		"bootstrap.admin_password",
		"bootstrap.admin_first_name",
		"bootstrap.admin_last_name",
	}
	for _, k := range keys {
		_ = v.BindEnv(k)
	}
}

func setViperDefaults(v *viper.Viper) {
	d := Defaults()
	v.SetDefault("server.addr", d.Server.Addr)
	v.SetDefault("auth.token_ttl", d.Auth.TokenTTL)
}

// Load reads configuration from file path and GFIREUI_* environment variables.
// An empty path uses defaults plus env; GFIREUI_CONFIG selects a file when path is empty.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("GFIREUI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	setViperDefaults(v)
	bindEnv(v)

	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read config: %w", err)
		}
	} else if p := os.Getenv("GFIREUI_CONFIG"); p != "" {
		v.SetConfigFile(p)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read GFIREUI_CONFIG: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}
