package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	healthcheckapi "github.com/ONSdigital/dp-api-clients-go/health"
	"github.com/ONSdigital/dp-api-clients-go/zebedee"
	"github.com/ONSdigital/dp-file-downloader/api"
	tableRenderer "github.com/ONSdigital/dp-file-downloader/clients/table-renderer"
	"github.com/ONSdigital/dp-file-downloader/config"
	"github.com/ONSdigital/dp-file-downloader/table"
	health "github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/log.go/v2/log"
)

var (
	// BuildTime represents the time in which the service was built
	BuildTime string
	// GitCommit represents the commit (SHA-1) hash of the service that is running
	GitCommit string
	// Version represents the version of the service that is running
	Version string
)

func main() {
	log.Namespace = "dp-file-downloader"

	ctx := context.Background()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	cfg, err := config.Get()
	if err != nil {
		log.Fatal(ctx, "unable to retrieve service configuration", err)
		os.Exit(1)
	}

	cfg.Log(ctx)

	// Healthcheck version Info
	versionInfo, err := health.NewVersionInfo(
		BuildTime,
		GitCommit,
		Version,
	)
	if err != nil {
		log.Error(ctx, "failed to create service version information", err)
	}

	apiRouterCli := healthcheckapi.NewClient("api-router", cfg.APIRouterURL)

	zc := zebedee.NewWithHealthClient(apiRouterCli)
	tabrend := tableRenderer.New(cfg.TableRendererHost)

	healthcheck := health.New(versionInfo, cfg.HealthCheckCriticalTimeout, cfg.HealthCheckInterval)

	if err = registerCheckers(ctx, &healthcheck, tabrend, apiRouterCli); err != nil {
		os.Exit(1)
	}

	healthcheck.Start(ctx)

	apiErrors := make(chan error, 1)

	tableDownloader := table.NewDownloader(zc, tabrend)

	api.StartDownloaderAPI(ctx, cfg, apiErrors, &healthcheck, &tableDownloader)

	// Gracefully shutdown the application closing any open resources.
	gracefulShutdown := func() {
		log.Info(ctx, fmt.Sprintf("Shutdown with timeout: %s", cfg.ShutdownTimeout))
		gracefulCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)

		var gracefulShutdown bool

		go func() {
			defer cancel()
			var hasShutdownErrs bool

			log.Info(gracefulCtx, "stop health checkers")
			healthcheck.Stop()

			if err = api.Close(gracefulCtx); err != nil {
				log.Error(gracefulCtx, "error closing api", err)
				hasShutdownErrs = true
			}

			if !hasShutdownErrs {
				gracefulShutdown = true
			}
		}()

		// wait for timeout or success (via cancel)
		<-gracefulCtx.Done()
		if gracefulCtx.Err() == context.DeadlineExceeded {
			log.Warn(gracefulCtx, "context deadline exceeded, enforcing shutdown", log.FormatErrors([]error{gracefulCtx.Err()}))
			os.Exit(1)
		}

		if !gracefulShutdown {
			err = errors.New("failed to close dependencies; failed to shutdown gracefully")
			log.Error(gracefulCtx, "failed to shutdown gracefully", err)
			os.Exit(1)
		}

		log.Info(gracefulCtx, "graceful shutdown complete", log.Data{"context": gracefulCtx.Err()})
	}

	select {
	case err := <-apiErrors:
		log.Error(ctx, "api error received", err)
		gracefulShutdown()
	case sig := <-signals:
		log.Info(ctx, "os signal received", log.Data{"signal": sig})
		gracefulShutdown()
	}
}

func registerCheckers(ctx context.Context, h *health.HealthCheck, r *tableRenderer.Client, apiRouterCli *healthcheckapi.Client) (err error) {
	hasErrors := false

	if err = h.AddCheck("frontend renderer", r.Checker); err != nil {
		hasErrors = true
		log.Error(ctx, "failed to add frontend renderer checker", err)
	}

	if err = h.AddCheck("API router", apiRouterCli.Checker); err != nil {
		hasErrors = true
		log.Error(ctx, "failed to add API router health checker", err)
	}

	if hasErrors {
		return errors.New("Error(s) registering checkers for healthcheck")
	}

	return nil
}
