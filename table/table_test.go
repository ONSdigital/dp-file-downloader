package table_test

import (
	"context"
	"strings"
	"testing"

	"net/http"

	"errors"
	"io"

	"github.com/ONSdigital/dp-api-clients-go/v2/zebedee"
	"github.com/ONSdigital/dp-file-downloader/table"
	"github.com/ONSdigital/dp-file-downloader/table/testdata"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	requestURI            = "/foo/bar.json"
	requestFormat         = "html"
	expectedDisposition   = "attachment; filename=\"bar.html\""
	expectedTitleFilename = "attachment; filename=\"my-table-title.html\""
	accessToken           = "myAccessToken"
	uriParam              = "&uri="
	expectedContentType   = "text/html"
	expectedContent       = "renderServerResponse"
	contentServerResponse = "contentServerResponse"
	baseURL               = "http://localhost/download/table?format="
)

const (
	accessTokenLabel = "access_token"
	testUrl          = "test/url"
)

func createZebedeeClientMock(body string, err error) *testdata.ZebedeeClientMock {
	return &testdata.ZebedeeClientMock{
		GetResourceBodyFunc: func(ctx context.Context, userAccessToken string, collectionID string, lang string, uri string) ([]byte, error) {
			return []byte(body), err
		},
	}
}

func createTableRenderClientMock(status int, testBody, contentType string, err error) *testdata.RendererClientMock {
	header := http.Header{}
	header.Add("Content-Type", contentType)
	return &testdata.RendererClientMock{
		PostBodyFunc: func(ctx context.Context, format string, body []byte) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(testBody)), Header: header}, err
		},
	}
}

func TestSuccessfulDownload(t *testing.T) {
	t.Parallel()
	Convey("Given a TableDownloader and a request to download a table", t, func() {
		initialRequest, err := http.NewRequest("GET", baseURL+requestFormat+uriParam+requestURI, http.NoBody)
		initialRequest.AddCookie(&http.Cookie{Name: accessTokenLabel, Value: accessToken})
		So(err, ShouldBeNil)

		contentClient := createZebedeeClientMock(contentServerResponse, nil)
		renderClient := createTableRenderClientMock(http.StatusOK, expectedContent, expectedContentType, nil)

		testObj := table.NewDownloader(contentClient, renderClient)

		Convey("When Download is invoked ", func() {
			responseBody, responseHeaders, responseStatus, responseErr := testObj.Download(initialRequest)

			Convey("contentClient should be invoked correctly", func() {
				So(len(contentClient.GetResourceBodyCalls()), ShouldEqual, 1)
			})

			Convey("renderClient should be invoked correctly", func() {
				So(len(renderClient.PostBodyCalls()), ShouldEqual, 1)
			})

			Convey("The correct response should be returned", func() {
				So(responseErr, ShouldBeNil)
				So(responseStatus, ShouldEqual, http.StatusOK)
				So(responseHeaders["Content-Type"], ShouldEqual, expectedContentType)
				So(responseHeaders["Content-Disposition"], ShouldEqual, expectedDisposition)
				So(readString(responseBody, t), ShouldEqual, expectedContent)
			})
		})
	})
}

func TestSuccessfulDownloadUsesFigureTitleAsFilename(t *testing.T) {
	t.Parallel()
	Convey("Given a TableDownloader and a request to download a table with a title in the figure JSON", t, func() {
		figureJSON := `{"title":"my-table-title","filename":"ignored","version":"1","uri":"/foo/bar.json"}`

		initialRequest, err := http.NewRequest("GET", baseURL+requestFormat+uriParam+requestURI, http.NoBody)
		initialRequest.AddCookie(&http.Cookie{Name: accessTokenLabel, Value: accessToken})
		So(err, ShouldBeNil)

		contentClient := createZebedeeClientMock(figureJSON, nil)
		renderClient := createTableRenderClientMock(http.StatusOK, expectedContent, expectedContentType, nil)

		testObj := table.NewDownloader(contentClient, renderClient)

		Convey("When Download is invoked", func() {
			responseBody, responseHeaders, responseStatus, responseErr := testObj.Download(initialRequest)

			Convey("The filename in response headers should use the figure title", func() {
				So(responseErr, ShouldBeNil)
				So(responseStatus, ShouldEqual, http.StatusOK)
				So(responseHeaders["Content-Type"], ShouldEqual, expectedContentType)
				So(responseHeaders["Content-Disposition"], ShouldEqual, expectedTitleFilename)
				So(readString(responseBody, t), ShouldEqual, expectedContent)
			})
		})
	})
}

func TestSuccessfulDownloadFallsBackToURIFilenameWhenFigureTitleEmpty(t *testing.T) {
	t.Parallel()
	Convey("Given a TableDownloader and a request to download a table with no title in the figure JSON", t, func() {
		figureJSON := `{"title":"","filename":"ignored","version":"1","uri":"/foo/bar.json"}`

		initialRequest, err := http.NewRequest("GET", baseURL+requestFormat+uriParam+requestURI, http.NoBody)
		initialRequest.AddCookie(&http.Cookie{Name: accessTokenLabel, Value: accessToken})
		So(err, ShouldBeNil)

		contentClient := createZebedeeClientMock(figureJSON, nil)
		renderClient := createTableRenderClientMock(http.StatusOK, expectedContent, expectedContentType, nil)

		testObj := table.NewDownloader(contentClient, renderClient)

		Convey("When Download is invoked", func() {
			responseBody, responseHeaders, responseStatus, responseErr := testObj.Download(initialRequest)

			Convey("The filename in response headers should use the URI path", func() {
				So(responseErr, ShouldBeNil)
				So(responseStatus, ShouldEqual, http.StatusOK)
				So(responseHeaders["Content-Type"], ShouldEqual, expectedContentType)
				So(responseHeaders["Content-Disposition"], ShouldEqual, expectedDisposition)
				So(readString(responseBody, t), ShouldEqual, expectedContent)
			})
		})
	})
}

func TestSuccessfulDownloadFallsBackToURIFilenameWhenFigureUnmarshalFails(t *testing.T) {
	t.Parallel()
	Convey("Given a TableDownloader and a request to download a table with invalid figure JSON", t, func() {
		invalidFigureJSON := "not-valid-json"

		initialRequest, err := http.NewRequest("GET", baseURL+requestFormat+uriParam+requestURI, http.NoBody)
		initialRequest.AddCookie(&http.Cookie{Name: accessTokenLabel, Value: accessToken})
		So(err, ShouldBeNil)

		contentClient := createZebedeeClientMock(invalidFigureJSON, nil)
		renderClient := createTableRenderClientMock(http.StatusOK, expectedContent, expectedContentType, nil)

		testObj := table.NewDownloader(contentClient, renderClient)

		Convey("When Download is invoked", func() {
			responseBody, responseHeaders, responseStatus, responseErr := testObj.Download(initialRequest)

			Convey("The response should still succeed and use the default URI filename", func() {
				So(responseErr, ShouldBeNil)
				So(responseStatus, ShouldEqual, http.StatusOK)
				So(responseHeaders["Content-Type"], ShouldEqual, expectedContentType)
				So(responseHeaders["Content-Disposition"], ShouldEqual, expectedDisposition)
				So(readString(responseBody, t), ShouldEqual, expectedContent)
			})
		})
	})
}

func TestSuccessfulDownloadForSpecificCollection(t *testing.T) {
	t.Parallel()
	Convey("Given a TableDownloader and a request to download a table, with a cookie identifying a collection", t, func() {
		contentCollection := "myCollection"

		initialRequest, err := http.NewRequest("GET", baseURL+requestFormat+uriParam+requestURI, http.NoBody)
		initialRequest.AddCookie(&http.Cookie{Name: accessTokenLabel, Value: accessToken})
		initialRequest.AddCookie(&http.Cookie{Name: "collection", Value: contentCollection})
		So(err, ShouldBeNil)

		contentClient := createZebedeeClientMock(contentServerResponse, nil)
		renderClient := createTableRenderClientMock(http.StatusOK, expectedContent, expectedContentType, nil)

		testObj := table.NewDownloader(contentClient, renderClient)

		Convey("When Download is invoked ", func() {
			responseBody, responseHeaders, responseStatus, responseErr := testObj.Download(initialRequest)

			Convey("contentClient should be invoked correctly", func() {
				So(len(contentClient.GetResourceBodyCalls()), ShouldEqual, 1)
			})

			Convey("renderClient should be invoked correctly", func() {
				So(len(renderClient.PostBodyCalls()), ShouldEqual, 1)
			})

			Convey("The correct response should be returned", func() {
				So(responseErr, ShouldBeNil)
				So(responseStatus, ShouldEqual, http.StatusOK)
				So(responseHeaders["Content-Type"], ShouldEqual, expectedContentType)
				So(responseHeaders["Content-Disposition"], ShouldEqual, expectedDisposition)
				So(readString(responseBody, t), ShouldEqual, expectedContent)
			})
		})
	})
}

func TestMissingContent(t *testing.T) {
	t.Parallel()
	Convey("Given a TableDownloader and a request to download content that doesn't exist", t, func() {
		requestURI := "/foo/bar"

		initialRequest, err := http.NewRequest("GET", baseURL+requestFormat+uriParam+requestURI, http.NoBody)
		initialRequest.AddCookie(&http.Cookie{Name: accessTokenLabel, Value: accessToken})
		So(err, ShouldBeNil)

		contentClient := createZebedeeClientMock("", zebedee.ErrInvalidZebedeeResponse{ActualCode: http.StatusNotFound, URI: testUrl})
		renderClient := createTableRenderClientMock(http.StatusOK, "", "", nil)

		testObj := table.NewDownloader(contentClient, renderClient)

		Convey("When Download is invoked ", func() {
			responseBody, _, responseStatus, responseErr := testObj.Download(initialRequest)

			Convey("A 404 response should be returned", func() {
				So(responseErr, ShouldNotBeNil)
				So(responseStatus, ShouldEqual, http.StatusNotFound)
				So(responseBody, ShouldBeNil)
			})
		})
	})
}

func TestContentServerError(t *testing.T) {
	t.Parallel()
	Convey("Given the content server doesn't respond", t, func() {
		initialRequest, err := http.NewRequest("GET", "http://localhost/download/table?format=html&uri=/foo/bar", http.NoBody)
		So(err, ShouldBeNil)

		expectedErr := zebedee.ErrInvalidZebedeeResponse{ActualCode: http.StatusInternalServerError, URI: testUrl}

		contentClient := createZebedeeClientMock("", zebedee.ErrInvalidZebedeeResponse{ActualCode: http.StatusInternalServerError, URI: testUrl})
		renderClient := createTableRenderClientMock(http.StatusOK, "", "", nil)

		testObj := table.NewDownloader(contentClient, renderClient)

		Convey("When Download is invoked ", func() {
			_, _, responseStatus, responseErr := testObj.Download(initialRequest)

			Convey("An error should be returned", func() {
				So(responseErr, ShouldResemble, expectedErr)
				So(responseStatus, ShouldEqual, http.StatusInternalServerError)
			})
		})
	})
}

func TestRenderServerError(t *testing.T) {
	t.Parallel()
	Convey("Given the render service is down", t, func() {
		initialRequest, err := http.NewRequest("GET", "http://localhost/download/table?format=html&uri=/foo/bar", http.NoBody)
		So(err, ShouldBeNil)

		expectedErr := errors.New("The render server is down")

		contentClient := createZebedeeClientMock("contentServerResponse", nil)
		renderClient := createTableRenderClientMock(http.StatusOK, "", "", expectedErr)

		testObj := table.NewDownloader(contentClient, renderClient)

		Convey("When Download is invoked ", func() {
			_, _, responseStatus, responseErr := testObj.Download(initialRequest)

			Convey("An error should be returned", func() {
				So(responseErr, ShouldEqual, expectedErr)
				So(responseStatus, ShouldEqual, http.StatusInternalServerError)
			})
		})
	})
}

func TestBadlyFormedRequest(t *testing.T) {
	t.Parallel()
	Convey("Given a TableDownloader and a badly formed request", t, func() {
		requestURI := "ghjghjkghj"
		requestFormat := "html"

		initialRequest, err := http.NewRequest("GET", "http://localhost/download/table?format="+requestFormat+uriParam+requestURI, http.NoBody)
		initialRequest.AddCookie(&http.Cookie{Name: accessTokenLabel, Value: accessToken})
		So(err, ShouldBeNil)

		contentClient := createZebedeeClientMock("", zebedee.ErrInvalidZebedeeResponse{ActualCode: http.StatusBadRequest, URI: testUrl})
		renderClient := createTableRenderClientMock(http.StatusOK, "", "", nil)

		testObj := table.NewDownloader(contentClient, renderClient)

		Convey("When Download is invoked ", func() {
			responseBody, _, responseStatus, responseErr := testObj.Download(initialRequest)

			Convey("A 400 response should be returned", func() {
				So(responseErr, ShouldNotBeNil)
				So(responseStatus, ShouldEqual, http.StatusBadRequest)
				So(responseBody, ShouldBeNil)
			})
		})
	})
}

func readString(reader io.Reader, _ *testing.T) string {
	So(reader, ShouldNotBeNil)
	bytes, e := io.ReadAll(reader)
	So(e, ShouldBeNil)
	return string(bytes)
}
