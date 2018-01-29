package table

import (
	"io"
	"net/http"
	"strings"

	"github.com/ONSdigital/go-ns/rhttp"
	"github.com/gorilla/mux"
)

// Downloader implements api.Downloader.
type Downloader struct {
	contentClient  *rhttp.Client
	contentHost    string
	rendererClient *rhttp.Client
	rendererHost   string
}

// NewDownloader returns a new Downloader using rhttp.DefaultClient
func NewDownloader(contentHost string, rendererHost string) Downloader {
	return NewDownloaderWithClients(rhttp.DefaultClient, contentHost, rhttp.DefaultClient, rendererHost)
}

// NewDownloaderWithClients returns a new Downloader using the given clients
func NewDownloaderWithClients(contentClient *rhttp.Client, contentHost string, rendererClient *rhttp.Client, rendererHost string) Downloader {
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
	return []string{"format", "uri"}
}

// Download fulfills the Request to download a table, sending the file in the ResponseWriter.
func (downloader *Downloader) Download(r *http.Request) (io.Reader, string, int, error) {

	vars := mux.Vars(r)

	format := vars["format"]
	uri := vars["uri"]
	println(format, uri)

	//renderRequest, err := http.NewRequest("GET", dl.rendererHost, nil)
	//if (err)

	return strings.NewReader(""), "foo", http.StatusOK, nil
}
