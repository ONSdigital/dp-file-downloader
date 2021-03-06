package table

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ONSdigital/dp-api-clients-go/zebedee"
	dphandlers "github.com/ONSdigital/dp-net/handlers"
	"github.com/ONSdigital/dp-net/request"
	"github.com/ONSdigital/log.go/log"
)

var (
	tokenHeader      = "X-Florence-Token"
	tokenCookie      = "access_token"
	collectionCookie = "collection"
	formatParam      = "format"
	uriParam         = "uri"
)

// Downloader implements api.Downloader.
type Downloader struct {
	contentClient  ZebedeeClient
	rendererClient TableRendererClient
}

// NewDownloader returns a new Downloader using rhttp.DefaultClient
func NewDownloader(contentClient ZebedeeClient, rendererClient TableRendererClient) Downloader {
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

	// call the content server to get the json definition of the table
	contentResponseBody, err := downloader.contentClient.GetResourceBody(ctx, userAccessToken, collectionID, lang, uri)
	if err != nil {
		log.Event(ctx, "error calling content server", log.ERROR, log.Error(err))
		var e zebedee.ErrInvalidZebedeeResponse
		if errors.As(err, &e) {
			if e.ActualCode == http.StatusNotFound {
				return nil, nil, e.ActualCode, nil
			}
			return nil, nil, e.ActualCode, err
		}
		return nil, nil, http.StatusInternalServerError, err
	}

	// post the json definition to the renderer
	renderResponse, err := downloader.rendererClient.PostBody(ctx, format, contentResponseBody)
	if err != nil {
		log.Event(ctx, "error calling renderer server", log.ERROR, log.Error(err))
		return nil, nil, http.StatusInternalServerError, err
	}

	return renderResponse.Body, createHeaders(renderResponse, uri, format), renderResponse.StatusCode, nil
}

// createContentRequest creates the request to send to the content server, extracting headers and cookies form the source request as appropriate
func getHeaderValues(ctx context.Context, r *http.Request) (string, string, string) {
	locale := request.GetLocaleCode(r)
	collectionID, err := request.GetCollectionID(r)
	if err != nil {
		log.Event(ctx, "unexpected error when getting collection id", log.ERROR, log.Error(err))
	}
	accessToken, err := dphandlers.GetFlorenceToken(ctx, r)
	if err != nil {
		log.Event(ctx, "unexpected error when getting access token", log.ERROR, log.Error(err))
	}
	return locale, collectionID, accessToken
}

// copyHeaders copies headers from the source request to the destination, and sets X-Florence-Token if there's an access_token cookie in the source.
func copyHeaders(source *http.Request, dest *http.Request) {
	for name, headers := range source.Header {
		name = strings.ToLower(name)
		for _, value := range headers {
			dest.Header.Add(name, value)
		}
	}
	// if we have an access token cookie, copy it to a header for onward requests
	cookie, _ := source.Cookie(tokenCookie)
	if cookie != nil {
		dest.Header.Add(tokenHeader, cookie.Value)
	}
}

// getContentType extracts the Content-Type from the response and puts it in a map
func getContentType(response *http.Response) map[string]string {
	return map[string]string{"Content-Type": response.Header.Get("Content-Type")}
}

// createHeaders extracts the content type form the response and constructs a filename from the last path element of the uri and the format
func createHeaders(response *http.Response, uri string, format string) map[string]string {
	headers := getContentType(response)
	paths := strings.Split(uri, "/")
	filename := strings.TrimSuffix(paths[len(paths)-1], ".json") + "." + format
	headers["Content-Disposition"] = "attachment; filename=\"" + filename + "\""
	return headers
}
