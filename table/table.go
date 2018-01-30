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
	destTokenHeader = "X-Florence-Token"
	srcTokenCookie  = "access_token"
	formatParam     = "format"
	uriParam        = "uri"
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
	contentRequest, err := http.NewRequest("GET", downloader.contentHost+"/resource?uri="+uri, nil)
	if err != nil {
		log.ErrorR(r, err, log.Data{"_message": "Unable to create HttpRequest to call content server"})
		return nil, "", http.StatusInternalServerError, err
	}
	copyHeaders(r, contentRequest)
	contentRequest.Header.Set("Accept", "application/json")
	contentResponse, err := downloader.contentClient.Do(contentRequest)
	if err != nil {
		log.ErrorR(r, err, log.Data{"_message": "Error calling content server", "uri": uri})
		return nil, "", http.StatusInternalServerError, err
	}
	if contentResponse.StatusCode != 200 {
		err = fmt.Errorf("Unexpected response from content server. Status=%d", contentResponse.StatusCode)
		log.ErrorR(r, err, log.Data{"uri": uri})
		return contentResponse.Body, contentResponse.Header.Get("Content-Type"), contentResponse.StatusCode, err
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

// copyHeaders copies headers from the source request to the destination, and sets X-Florence-Token if there's an access_token cookie in the source.
func copyHeaders(source *http.Request, dest *http.Request) {
	for name, headers := range source.Header {
		name = strings.ToLower(name)
		for _, value := range headers {
			dest.Header.Add(name, value)
		}
	}
	// if we have an access token cookie, copy it to a header for onward requests
	cookie, err := source.Cookie(srcTokenCookie)
	if err == nil { // we get an error if the cookie isn't present
		dest.Header.Add(destTokenHeader, cookie.Value)
	}
}
