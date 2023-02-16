package table_renderer

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	healthcheck "github.com/ONSdigital/dp-api-clients-go/health"
	health "github.com/ONSdigital/dp-healthcheck/healthcheck"
	dphttp "github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/v2/log"
)

const service = "table-renderer"

// Client represents a table-renderer client
type Client struct {
	cli dphttp.Clienter
	url string
}

// ErrInvalidTableRendererResponse is returned when the table-renderer service does not respond with a status 200
type ErrInvalidTableRendererResponse struct {
	responseCode int
}

// Error should be called by the user to print out the stringified version of the error
func (e ErrInvalidTableRendererResponse) Error() string {
	return fmt.Sprintf("invalid response from table-renderer service - status %d", e.responseCode)
}

// Code returns the status code received from table-renderer if an error is returned
func (e ErrInvalidTableRendererResponse) Code() int {
	return e.responseCode
}

// New creates a new instance of Client with a given table-renderer url
func New(tableRendererURL string) *Client {
	hcClient := healthcheck.NewClient(service, tableRendererURL)

	return &Client{
		cli: hcClient.Client,
		url: tableRendererURL,
	}
}

// Checker calls table-renderer health endpoint and returns a check object to the caller.
func (c *Client) Checker(ctx context.Context, check *health.CheckState) error {
	hcClient := healthcheck.Client{
		Client: c.cli,
		URL:    c.url,
		Name:   service,
	}

	return hcClient.Checker(ctx, check)
}

func (c *Client) PostBody(ctx context.Context, format string, body []byte) (resp *http.Response, err error) {
	reqURL := fmt.Sprintf("%s/render/%s", c.url, format)
	return c.post(ctx, reqURL, body)
}

func (c *Client) post(ctx context.Context, uri string, body []byte) (*http.Response, error) {
	r := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, uri, r)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}
	return c.cli.Do(ctx, req)
}

// closeResponseBody closes the response body and logs an error containing the context if unsuccessful
func closeResponseBody(ctx context.Context, resp *http.Response) {
	if err := resp.Body.Close(); err != nil {
		log.Error(ctx, "error closing http response body", err)
	}
}
