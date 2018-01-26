package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/ONSdigital/go-ns/log"
)

// Config is the configuration for this service
type Config struct {
	BindAddr                  string        `envconfig:"BIND_ADDR"`
	CORSAllowedOrigins        string        `envconfig:"CORS_ALLOWED_ORIGINS"`
	ShutdownTimeout           time.Duration `envconfig:"SHUTDOWN_TIMEOUT"`
	HealthCheckInterval       time.Duration `envconfig:"HEALTHCHECK_INTERVAL"`
	TableRendererHost         string        `envconfig:"TABLE_RENDERER_HOST"`
	ContentServerHost         string        `envconfig:"CONTENT_SERVER_HOST"`
}

var cfg *Config

// Get configures the application and returns the configuration
func Get() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &Config{
		BindAddr:                  ":23400",
		CORSAllowedOrigins:        "*",
		ShutdownTimeout:           5 * time.Second,
		HealthCheckInterval:       30 * time.Second,
		TableRendererHost:         "http://localhost:23300",
		ContentServerHost:         "http://localhost:8082",
	}

	return cfg, envconfig.Process("", cfg)
}

// Log writes all config properties to log.Debug
func (cfg *Config) Log() {
	log.Debug("Configuration", log.Data{
		"BindAddr":                  cfg.BindAddr,
		"CORSAllowedOrigins":        cfg.CORSAllowedOrigins,
		"ShutdownTimeout":           cfg.ShutdownTimeout,
		"HealthCheckInterval":       cfg.HealthCheckInterval,
		"TableRendererHost":         cfg.TableRendererHost,
		"ContentServerHost":         cfg.ContentServerHost,
	})

}
