package table

import (
	"io"
	"net/http"
	"strings"

	"fmt"

	"github.com/ONSdigital/go-ns/log"
	"github.com/ONSdigital/go-ns/rhttp"
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
	contentClient  HTTPClient
	contentHost    string
	rendererClient HTTPClient
	rendererHost   string
}

//go:generate moq -out testdata/mock_httpclient.go -pkg testdata . HTTPClient

// HTTPClient is implemented by http.Client etc.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewDownloader returns a new Downloader using rhttp.DefaultClient
func NewDownloader(contentHost string, rendererHost string) Downloader {
	return NewDownloaderWithClients(rhttp.DefaultClient, contentHost, rhttp.DefaultClient, rendererHost)
}

// NewDownloaderWithClients returns a new Downloader using the given clients
func NewDownloaderWithClients(contentClient HTTPClient, contentHost string, rendererClient HTTPClient, rendererHost string) Downloader {
	return Downloader{
		contentClient:  contentClient,
		contentHost:    contentHost,
		rendererClient: rendererClient,
		rendererHost:   rendererHost,
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
func (downloader *Downloader) Download(r *http.Request) (responseBody io.Reader, contentType string, responseStatus int, responseErr error) {

	format := r.URL.Query().Get(formatParam)
	uri := r.URL.Query().Get(uriParam)

	// call the content server to get the json definition of the table
	contentRequest, err := createContentRequest(downloader, uri, r)
	if err != nil {
		log.ErrorR(r, err, log.Data{"_message": "Unable to create HttpRequest to call content server"})
		return nil, "", http.StatusInternalServerError, err
	}
	contentResponse, err := downloader.contentClient.Do(contentRequest)
	if err != nil {
		log.ErrorR(r, err, log.Data{"_message": "Error calling content server", "uri": uri})
		return nil, "", http.StatusInternalServerError, err
	}
	if contentResponse.StatusCode != 200 {
		err = fmt.Errorf("Unexpected response from content server. Status=%d", contentResponse.StatusCode)
		log.ErrorR(r, err, log.Data{"uri": uri})
		return contentResponse.Body, contentResponse.Header.Get("Content-Type"), contentResponse.StatusCode, nil
	}

	// post the json definition to the renderer
	renderRequest, err := http.NewRequest("POST", downloader.rendererHost+"/render/"+format, contentResponse.Body)
	if err != nil {
		log.ErrorR(r, err, log.Data{"_message": "Unable to create HttpRequest to call render server"})
		return nil, "", http.StatusInternalServerError, err
	}
	copyHeaders(r, renderRequest)
	renderRequest.Header.Set("Content-Type", "application/json")
	renderResponse, err := downloader.rendererClient.Do(renderRequest)
	if err != nil {
		log.ErrorR(r, err, log.Data{"_message": "Error calling render server", "format": format})
		return nil, "", http.StatusInternalServerError, err
	}

	// return content from the renderResponse
	return renderResponse.Body, renderResponse.Header.Get("Content-Type"), renderResponse.StatusCode, nil
}

// createContentRequest creates the request to send to the content server, extracting headers and cookies form the source request as appropriate
func createContentRequest(downloader *Downloader, uri string, r *http.Request) (*http.Request, error) {
	path := "/resource"
	// append the requested collection, if one is present as a cookie
	cookie, _ := r.Cookie(collectionCookie)
	if cookie != nil {
		path += "/" + cookie.Value
	}
	contentRequest, err := http.NewRequest("GET", downloader.contentHost+ path + "?uri="+uri, nil)
	if err != nil {
		return nil, err
	}
	copyHeaders(r, contentRequest)
	contentRequest.Header.Set("Accept", "application/json")
	return contentRequest, err
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
