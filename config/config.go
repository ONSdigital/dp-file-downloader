package config

import (
	"context"
	"time"

	"github.com/ONSdigital/log.go/v2/log"
	"github.com/kelseyhightower/envconfig"
)

// Config is the configuration for this service
type Config struct {
	BindAddr                   string        `envconfig:"BIND_ADDR"`
	CORSAllowedOrigins         string        `envconfig:"CORS_ALLOWED_ORIGINS"`
	ShutdownTimeout            time.Duration `envconfig:"SHUTDOWN_TIMEOUT"`
	HealthCheckCriticalTimeout time.Duration `envconfig:"HEALTHCHECK_CRITICAL_TIMEOUT"`
	HealthCheckInterval        time.Duration `envconfig:"HEALTHCHECK_INTERVAL"`
	OTBatchTimeout             time.Duration `encconfig:"OTEL_BATCH_TIMEOUT"`
	OTServiceName              string        `envconfig:"OTEL_SERVICE_NAME"`
	OTExporterOTLPEndpoint     string        `envconfig:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	TableRendererHost          string        `envconfig:"TABLE_RENDERER_HOST"`
	ContentServerHost          string        `envconfig:"CONTENT_SERVER_HOST"`
	APIRouterURL               string        `envconfig:"API_ROUTER_URL"`
}

var cfg *Config

// Get configures the application and returns the configuration
func Get() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &Config{
		BindAddr:                   ":23400",
		CORSAllowedOrigins:         "*",
		ShutdownTimeout:            5 * time.Second,
		HealthCheckCriticalTimeout: 90 * time.Second,
		HealthCheckInterval:        30 * time.Second,
		OTBatchTimeout:             5 * time.Second,
		OTExporterOTLPEndpoint:     "localhost:4317",
		OTServiceName:              "dp-file-downloader",
		TableRendererHost:          "http://localhost:23300",
		ContentServerHost:          "http://localhost:8082",
		APIRouterURL:               "http://localhost:23200/v1",
	}

	return cfg, envconfig.Process("", cfg)
}

// Log writes all config properties to log.Debug
func (cfg *Config) Log(ctx context.Context) {
	log.Info(ctx, "Configuration", log.Data{
		"BindAddr":                   cfg.BindAddr,
		"CORSAllowedOrigins":         cfg.CORSAllowedOrigins,
		"ShutdownTimeout":            cfg.ShutdownTimeout,
		"HealthCheckCriticalTimeout": cfg.HealthCheckCriticalTimeout,
		"HealthCheckInterval":        cfg.HealthCheckInterval,
		"TableRendererHost":          cfg.TableRendererHost,
		"ContentServerHost":          cfg.ContentServerHost,
		"APIRouterURL":               cfg.APIRouterURL,
	})
}
