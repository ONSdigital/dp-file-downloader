package api

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/ONSdigital/dp-file-downloader/config"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	dphttp "github.com/ONSdigital/dp-net/http"
	dpotelgo "github.com/ONSdigital/dp-otel-go"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var httpServer *dphttp.Server

// DownloaderAPI manages requests to download files, calling the necessary backend services to fulfill the request
type DownloaderAPI struct {
	router *mux.Router
}

//cannot use "go:generate moq -out testdata/mock_downloader.go -pkg testdata . Downloader" here
// as moq can't handle build tags and incorrectly believes there are duplicate methods in gorilla/mux
// https://github.com/matryer/moq/issues/47

// Downloader defines the functions that are assigned to handle get requests
type Downloader interface {
	// Download retrieves/creates the file requested in the http.Request , returning:
	// body - a reader with the contents of the file. This must be closed by the caller.
	// headers - should include Content-Type and Content-Disposition
	// status - the http status code - should be 200 unless there was an error
	// err - any error that occurred during processing
	Download(r *http.Request) (body io.ReadCloser, headers map[string]string, status int, err error)
	// Type returns the (conceptual) type of file downloaded - forms part of the request path handled by this Downloader
	Type() string
	// QueryParameters returns the names of query parameters required by this Downloader
	QueryParameters() []string
}

// StartDownloaderAPI manages all the routes configured to the downloader
func StartDownloaderAPI(ctx context.Context, cfg *config.Config, errorChan chan error, hc *healthcheck.HealthCheck, downloaders ...Downloader) *DownloaderAPI {
	router := mux.NewRouter()
	otelHandler := otelhttp.NewHandler(router, "/")
	router.Use(otelmux.Middleware(cfg.OTServiceName))

	api := routes(ctx, router, hc, downloaders...)

	httpServer = dphttp.NewServer(cfg.BindAddr, otelHandler)
	// Disable this here to allow main to manage graceful shutdown of the entire app.
	httpServer.HandleOSSignals = false

	go func() {
		log.Info(ctx, "starting file downloader...")

		// Set up OpenTelemetry
		otelConfig := dpotelgo.Config{
			OtelBatchTimeout:         cfg.OTBatchTimeout,
			OtelExporterOtlpEndpoint: cfg.OTExporterOTLPEndpoint,
			OtelServiceName:          cfg.OTServiceName,
		}

		otelShutdown, oErr := dpotelgo.SetupOTelSDK(ctx, otelConfig)
		if oErr != nil {
			log.Fatal(ctx, "error setting up OpenTelemetry - hint: ensure OTEL_EXPORTER_OTLP_ENDPOINT is set", oErr)
		}
		// Handle shutdown properly so nothing leaks.
		defer func() {
			oErr = errors.Join(oErr, otelShutdown(context.Background()))
			errorChan <- oErr
		}()

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(ctx, "error occurred when running ListenAndServe", err)
			errorChan <- err
		}
	}()

	return api
}

// createCORSHandler wraps the router in a CORS handler that responds to OPTIONS requests and returns the headers necessary to allow CORS-enabled clients to work
func createCORSHandler(allowedOrigins string, router *mux.Router) http.Handler {
	headersOk := handlers.AllowedHeaders([]string{"Accept", "Content-Type", "Access-Control-Allow-Origin", "Access-Control-Allow-Methods", "X-Requested-With"})
	originsOk := handlers.AllowedOrigins([]string{allowedOrigins})
	methodsOk := handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"})

	return handlers.CORS(originsOk, headersOk, methodsOk)(router)
}

// routes contain all endpoints for the downloader
func routes(ctx context.Context, router *mux.Router, hc *healthcheck.HealthCheck, downloaders ...Downloader) *DownloaderAPI {
	api := DownloaderAPI{router: router}

	api.router.StrictSlash(true).Path("/health").HandlerFunc(hc.Handler)

	for _, d := range downloaders {
		path := "/download/" + d.Type()
		api.router.Path(path).Methods("GET").HandlerFunc(handleDownload(d.Download))
		log.Info(ctx, "handling GET method on path "+path, log.Data{"query_parameters": d.QueryParameters()})
	}

	return &api
}

// Close represents the graceful shutting down of the http server
func Close(ctx context.Context) error {
	if err := httpServer.Shutdown(ctx); err != nil {
		return err
	}

	log.Info(ctx, "graceful shutdown of http server complete")
	return nil
}

// handleDownload accepts a Downloader.Download function and wraps it in a handler that writes the content to an http.ResponseWriter.
func handleDownload(handler func(r *http.Request) (io.ReadCloser, map[string]string, int, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, request *http.Request) {
		reader, headers, status, err := handler(request)
		ctx := request.Context()
		defer func() {
			if reader != nil {
				err := reader.Close()
				if err != nil {
					log.Error(ctx, "unable to close reader cleanly", err)
				}
			}
		}()
		if err != nil {
			log.Error(ctx, "handleDownload: Error returned from handler", err, log.Data{"request:": request})
			if status < 400 {
				status = http.StatusInternalServerError
			}
			http.Error(w, err.Error(), status)
		} else {
			for key, value := range headers {
				w.Header().Add(key, value)
			}
			w.WriteHeader(status)
			// write body
			_, err := io.Copy(w, reader)
			if err != nil {
				log.Error(ctx, "handleDownload: Error while copying from reader", err, log.Data{"request:": request})
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}
}
