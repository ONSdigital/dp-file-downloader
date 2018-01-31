package api

import (
	"context"

	"github.com/ONSdigital/go-ns/healthcheck"
	"github.com/ONSdigital/go-ns/log"
	"github.com/ONSdigital/go-ns/server"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"io"
	"net/http"
)

var httpServer *server.Server

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
func StartDownloaderAPI(bindAddr string, allowedOrigins string, errorChan chan error, downloaders ...Downloader) *DownloaderAPI {
	router := mux.NewRouter()
	api := routes(router, downloaders...)

	httpServer = server.New(bindAddr, createCORSHandler(allowedOrigins, router))
	// Disable this here to allow main to manage graceful shutdown of the entire app.
	httpServer.HandleOSSignals = false

	go func() {
		log.Debug("Starting file downloader...", nil)
		if err := httpServer.ListenAndServe(); err != nil {
			log.ErrorC("api", err, log.Data{"MethodInError": "httpServer.ListenAndServe()"})
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
func routes(router *mux.Router, downloaders ...Downloader) *DownloaderAPI {
	api := DownloaderAPI{router: router}

	router.Path("/healthcheck").Methods("GET").HandlerFunc(healthcheck.Do)

	for _, d := range downloaders {
		queries := []string{}
		for _, q := range d.QueryParameters() {
			queries = append(queries, q, "{"+q+"}")
		}
		path := "/download/" + d.Type()
		api.router.Path(path).Methods("GET").Queries(queries...).HandlerFunc(handleDownload(d.Download))
		log.Debug("Handling GET method on path "+path, log.Data{"query_parameters": d.QueryParameters()})
	}

	return &api
}

// Close represents the graceful shutting down of the http server
func Close(ctx context.Context) error {
	if err := httpServer.Shutdown(ctx); err != nil {
		return err
	}

	log.Info("graceful shutdown of http server complete", nil)
	return nil
}

// handleDownload accepts a Downloader.Download function and wraps it in a handler that writes the content to an http.ResponseWriter.
func handleDownload(handler func(r *http.Request) (io.ReadCloser, map[string]string, int, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, request *http.Request) {
		reader, headers, status, err := handler(request)
		defer func() {
			err := reader.Close()
			if err != nil {
				log.ErrorR(request, err, log.Data{"_message": "Unable to close reader cleanly"})
			}
		}()
		if err != nil {
			log.ErrorR(request, err, log.Data{"_message": "handleDownload: Error returned from handler", "request:": request})
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
				log.ErrorR(request, err, log.Data{"_message": "handleDownload: Error while copying from reader", "request:": request})
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}
}
