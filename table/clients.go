package table

import (
	"context"
	"net/http"
)

//go:generate moq -out testdata/zebedeeclient.go -pkg testdata . ZebedeeClient
//go:generate moq -out testdata/tablerendererclient.go -pkg testdata . RendererClient

type ZebedeeClient interface {
	GetResourceBody(ctx context.Context, userAccessToken, collectionID, lang, uri string) ([]byte, error)
}

type RendererClient interface {
	PostBody(ctx context.Context, format string, body []byte) (resp *http.Response, err error)
}
