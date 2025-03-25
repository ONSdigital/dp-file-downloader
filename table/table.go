package table

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ONSdigital/dp-api-clients-go/v2/zebedee"
	dphandlers "github.com/ONSdigital/dp-net/v3/handlers"
	"github.com/ONSdigital/dp-net/v3/request"
	"github.com/ONSdigital/log.go/v2/log"
)

var (
	formatParam = "format"
	uriParam    = "uri"
)

// Downloader implements api.Downloader.
type Downloader struct {
	contentClient  ZebedeeClient
	rendererClient RendererClient
}

// NewDownloader returns a new Downloader using rhttp.DefaultClient
func NewDownloader(contentClient ZebedeeClient, rendererClient RendererClient) Downloader {
	return Downloader{
		contentClient:  contentClient,
		rendererClient: rendererClient,
	}
}

// Type returns the type of file returned by this downloader, a table.
func (downloader *Downloader) Type() string {
	return "table"
}

// QueryParameters returns the format and uri query parameters we require to return a table.
// 'format' is the format of the file to return - xlsx, csv or html.
// 'uri' is the location of the file that defines the table (a path that resolves to a .json file in the content server).
func (downloader *Downloader) QueryParameters() []string {
	return []string{formatParam, uriParam}
}

// Download fulfills the Request to download a table.
// The responseBody must be closed by the caller.
func (downloader *Downloader) Download(r *http.Request) (responseBody io.ReadCloser, headers map[string]string, responseStatus int, responseErr error) {
	format := r.URL.Query().Get(formatParam)
	uri := r.URL.Query().Get(uriParam)

	ctx := r.Context()
	lang, collectionID, userAccessToken := getHeaderValues(ctx, r)

	err := validateURL(format, uri)
	if err != nil {
		return nil, nil, http.StatusBadRequest, err
	}

	// call the content server to get the json definition of the table
	contentResponseBody, err := downloader.contentClient.GetResourceBody(ctx, userAccessToken, collectionID, lang, uri)
	if err != nil {
		log.Error(ctx, "error calling content server", err)
		var e zebedee.ErrInvalidZebedeeResponse
		if errors.As(err, &e) {
			if e.ActualCode == http.StatusNotFound {
				return nil, nil, http.StatusNotFound, err
			} else if e.ActualCode == http.StatusInternalServerError {
				return nil, nil, http.StatusInternalServerError, err
			}
			return nil, nil, http.StatusBadRequest, err
		}
		return nil, nil, http.StatusInternalServerError, err
	}

	// post the json definition to the renderer
	renderResponse, err := downloader.rendererClient.PostBody(ctx, format, contentResponseBody)
	if err != nil {
		log.Error(ctx, "error calling renderer server", err)
		return nil, nil, http.StatusInternalServerError, err
	}

	return renderResponse.Body, createHeaders(renderResponse, uri, format), renderResponse.StatusCode, nil
}

// createContentRequest creates the request to send to the content server, extracting headers and cookies form the source request as appropriate
func getHeaderValues(ctx context.Context, r *http.Request) (locale, collectionID, accessToken string) {
	locale = request.GetLocaleCode(r)
	collectionID, err := request.GetCollectionID(r)
	if err != nil {
		log.Error(ctx, "unexpected error when getting collection id", err)
	}
	accessToken, err = dphandlers.GetFlorenceToken(ctx, r)
	if err != nil {
		log.Error(ctx, "unexpected error when getting access token", err)
	}
	return locale, collectionID, accessToken
}

// getContentType extracts the Content-Type from the response and puts it in a map
func getContentType(response *http.Response) map[string]string {
	return map[string]string{"Content-Type": response.Header.Get("Content-Type")}
}

// createHeaders extracts the content type form the response and constructs a filename from the last path element of the uri and the format
func createHeaders(response *http.Response, uri, format string) map[string]string {
	headers := getContentType(response)
	paths := strings.Split(uri, "/")
	filename := strings.TrimSuffix(paths[len(paths)-1], ".json") + "." + format
	headers["Content-Disposition"] = "attachment; filename=\"" + filename + "\""
	return headers
}

func validateURL(format, uri string) (err error) {
	if format == "" || uri == "" {
		return errors.New("bad request")
	}
	return nil
}
