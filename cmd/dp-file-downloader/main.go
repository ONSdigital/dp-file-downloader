package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ONSdigital/dp-file-downloader/api"
	"github.com/ONSdigital/dp-file-downloader/config"
	"github.com/ONSdigital/go-ns/healthcheck"
	"github.com/ONSdigital/go-ns/log"
)

func main() {
	log.Namespace = "dp-file-downloader"

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	cfg, err := config.Get()
	if err != nil {
		log.Error(err, nil)
		os.Exit(1)
	}

	cfg.Log()

	healthTicker := healthcheck.NewTicker(
		cfg.HealthCheckInterval,
		healthcheck.NewClient("dp-table-renderer", cfg.TableRendererHost+"/healthcheck"),
	)

	apiErrors := make(chan error, 1)

	api.CreateDownloaderAPI(cfg.BindAddr, cfg.CORSAllowedOrigins, apiErrors)

	// Gracefully shutdown the application closing any open resources.
	gracefulShutdown := func() {
		log.Info(fmt.Sprintf("Shutdown with timeout: %s", cfg.ShutdownTimeout), nil)
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)

		if err = api.Close(ctx); err != nil {
			log.Info("api error", nil)
			log.Error(err, nil)
		}

		healthTicker.Close()

		cancel()

		log.Info("Shutdown complete", nil)
		os.Exit(1)
	}

	for {
		select {
		case err := <-apiErrors:
			log.ErrorC("api error received", err, nil)
			gracefulShutdown()
		case <-signals:
			log.Debug("os signal received", nil)
			gracefulShutdown()
		}
	}
}
