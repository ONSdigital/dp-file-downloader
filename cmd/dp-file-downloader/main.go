package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	healthcheck "github.com/ONSdigital/dp-api-clients-go/health"
	"github.com/ONSdigital/dp-api-clients-go/zebedee"
	"github.com/ONSdigital/dp-file-downloader/api"
	table_renderer "github.com/ONSdigital/dp-file-downloader/clients/table-renderer"
	"github.com/ONSdigital/dp-file-downloader/config"
	"github.com/ONSdigital/dp-file-downloader/table"
	health "github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/log.go/log"
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
		log.Event(ctx, "unable to retrieve service configuration", log.FATAL, log.Error(err))
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
		log.Event(ctx, "failed to create service version information", log.ERROR, log.Error(err))
		//return err
	}

	apiRouterCli := healthcheck.NewClient("api-router", cfg.APIRouterURL)

	zc := zebedee.NewWithHealthClient(apiRouterCli)
	tabrend := table_renderer.New(cfg.TableRendererHost)

	healthcheck := health.New(versionInfo, cfg.HealthCheckCriticalTimeout, cfg.HealthCheckInterval)

	if err = registerCheckers(ctx, &healthcheck, tabrend, apiRouterCli); err != nil {
		os.Exit(1)
	}

	healthcheck.Start(ctx)

	apiErrors := make(chan error, 1)

	tableDownloader := table.NewDownloader(zc, tabrend)

	api.StartDownloaderAPI(ctx, cfg.BindAddr, cfg.CORSAllowedOrigins, apiErrors, &healthcheck, &tableDownloader)

	// Gracefully shutdown the application closing any open resources.
	gracefulShutdown := func() {
		log.Event(ctx, fmt.Sprintf("Shutdown with timeout: %s", cfg.ShutdownTimeout), log.INFO)
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)

		var gracefulShutdown bool

		go func() {
			defer cancel()
			var hasShutdownErrs bool

			log.Event(ctx, "stop health checkers", log.INFO)
			healthcheck.Stop()

			if err = api.Close(ctx); err != nil {
				log.Event(ctx, "error with graceful shutdown", log.ERROR, log.Error(err))
			}

			if !hasShutdownErrs {
				gracefulShutdown = true
			}
		}()

		// wait for timeout or success (via cancel)
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			log.Event(ctx, "context deadline exceeded", log.WARN, log.Error(ctx.Err()))
			os.Exit(1)
		}

		if !gracefulShutdown {
			err = errors.New("failed to shutdown gracefully")
			log.Event(ctx, "failed to shutdown gracefully ", log.ERROR, log.Error(err))
			os.Exit(1)
		}

		log.Event(ctx, "graceful shutdown complete", log.INFO, log.Data{"context": ctx.Err()})
	}

	select {
	case err := <-apiErrors:
		log.Event(ctx, "api error received", log.ERROR, log.Error(err))
		gracefulShutdown()
	case signal := <-signals:
		log.Event(ctx, "os signal received", log.INFO, log.Data{"signal": signal})
		gracefulShutdown()
	}

}

func registerCheckers(ctx context.Context, h *health.HealthCheck, r *table_renderer.Client, apiRouterCli *healthcheck.Client) (err error) {

	hasErrors := false

	if err = h.AddCheck("frontend renderer", r.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "failed to add frontend renderer checker", log.ERROR, log.Error(err))
	}

	if err = h.AddCheck("API router", apiRouterCli.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "failed to add API router health checker", log.ERROR, log.Error(err))
	}

	if hasErrors {
		return errors.New("Error(s) registering checkers for healthcheck")
	}

	return nil
}
