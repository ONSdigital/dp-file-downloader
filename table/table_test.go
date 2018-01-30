package table_test

import (
	"testing"

	"net/http"
	"strings"

	"errors"
	"io"
	"io/ioutil"

	"github.com/ONSdigital/dp-file-downloader/table"
	"github.com/ONSdigital/dp-file-downloader/table/testdata"
	. "github.com/smartystreets/goconvey/convey"
)

const contentHost = "content"
const renderHost = "render"

func TestSuccessfulDownload(t *testing.T) {
	t.Parallel()
	Convey("Given a TableDownloader and a request to download a table", t, func() {

		requestUri := "/foo/bar"
		requestFormat := "html"
		accessToken := "myAccessToken"
		expectedContentType := "text/html"
		expectedContent := "renderServerResponse"
		contentServerResponse := "contentServerResponse"

		initialRequest, err := http.NewRequest("GET", "http://localhost/download/table?format="+requestFormat+"&uri="+requestUri, nil)
		initialRequest.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
		So(err, ShouldBeNil)

		contentClient := createMockClient(http.StatusOK, contentServerResponse, "application/json")
		renderClient := createMockClient(http.StatusOK, expectedContent, expectedContentType)

		testObj := table.NewDownloaderWithClients(contentClient, "http://"+contentHost, renderClient, "http://"+renderHost)

		Convey("When Download is invoked ", func() {

			responseBody, contentType, responseStatus, responseErr := testObj.Download(initialRequest)

			Convey("contentClient should be invoked correctly", func() {
				So(len(contentClient.DoCalls()), ShouldEqual, 1)
				request := contentClient.DoCalls()[0]
				So(request.Req.URL.Host, ShouldEqual, contentHost)
				So(request.Req.URL.Path, ShouldEqual, "/resource")
				So(request.Req.URL.Query().Get("uri"), ShouldEqual, requestUri)
				So(request.Req.Header.Get("X-Florence-Token"), ShouldEqual, accessToken)
				So(request.Req.Header.Get("Accept-Encoding"), ShouldEqual, "application/json")
				So(request.Req.Method, ShouldEqual, "GET")
			})

			Convey("renderClient should be invoked correctly", func() {
				So(len(renderClient.DoCalls()), ShouldEqual, 1)
				request := renderClient.DoCalls()[0]
				So(request.Req.URL.Host, ShouldEqual, renderHost)
				So(request.Req.URL.Path, ShouldEqual, "/render/"+requestFormat)
				So(request.Req.Method, ShouldEqual, "POST")
				So(readString(request.Req.Body, t), ShouldEqual, contentServerResponse)
			})

			Convey("The correct response should be returned", func() {
				So(responseErr, ShouldBeNil)
				So(responseStatus, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, expectedContentType)
				So(readString(responseBody, t), ShouldEqual, expectedContent)
			})
		})
	})
}

func TestSuccessfulDownloadForSpecificCollection(t *testing.T) {
	t.Parallel()
	Convey("Given a TableDownloader and a request to download a table, with a cookie identifying a collection", t, func() {

		requestUri := "/foo/bar"
		requestFormat := "html"
		accessToken := "myAccessToken"
		contentCollection := "myCollection"
		expectedContentType := "text/html"
		expectedContent := "renderServerResponse"
		contentServerResponse := "contentServerResponse"

		initialRequest, err := http.NewRequest("GET", "http://localhost/download/table?format="+requestFormat+"&uri="+requestUri, nil)
		initialRequest.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
		initialRequest.AddCookie(&http.Cookie{Name: "collection", Value: contentCollection})
		So(err, ShouldBeNil)

		contentClient := createMockClient(http.StatusOK, contentServerResponse, "application/json")
		renderClient := createMockClient(http.StatusOK, expectedContent, expectedContentType)

		testObj := table.NewDownloaderWithClients(contentClient, "http://"+contentHost, renderClient, "http://"+renderHost)

		Convey("When Download is invoked ", func() {

			responseBody, contentType, responseStatus, responseErr := testObj.Download(initialRequest)

			Convey("contentClient should be invoked correctly", func() {
				So(len(contentClient.DoCalls()), ShouldEqual, 1)
				request := contentClient.DoCalls()[0]
				So(request.Req.URL.Host, ShouldEqual, contentHost)
				So(request.Req.URL.Path, ShouldEqual, "/resource/" + contentCollection)
				So(request.Req.URL.Query().Get("uri"), ShouldEqual, requestUri)
				So(request.Req.Header.Get("X-Florence-Token"), ShouldEqual, accessToken)
				So(request.Req.Method, ShouldEqual, "GET")
			})

			Convey("renderClient should be invoked correctly", func() {
				So(len(renderClient.DoCalls()), ShouldEqual, 1)
				request := renderClient.DoCalls()[0]
				So(request.Req.URL.Host, ShouldEqual, renderHost)
				So(request.Req.URL.Path, ShouldEqual, "/render/"+requestFormat)
				So(request.Req.Method, ShouldEqual, "POST")
				So(readString(request.Req.Body, t), ShouldEqual, contentServerResponse)
			})

			Convey("The correct response should be returned", func() {
				So(responseErr, ShouldBeNil)
				So(responseStatus, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, expectedContentType)
				So(readString(responseBody, t), ShouldEqual, expectedContent)
			})
		})
	})
}

func TestMissingContent(t *testing.T) {
	t.Parallel()
	Convey("Given a TableDownloader and a request to download content that doesn't exist", t, func() {

		requestUri := "/foo/bar"
		requestFormat := "html"
		accessToken := "myAccessToken"

		initialRequest, err := http.NewRequest("GET", "http://localhost/download/table?format="+requestFormat+"&uri="+requestUri, nil)
		initialRequest.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
		So(err, ShouldBeNil)

		expectedResponse := "Content could not be found"
		contentClient := createMockClient(http.StatusNotFound, expectedResponse, "")
		renderClient := createMockClient(http.StatusOK, "", "")

		testObj := table.NewDownloaderWithClients(contentClient, "http://"+contentHost, renderClient, "http://"+renderHost)

		Convey("When Download is invoked ", func() {

			responseBody, _, responseStatus, responseErr := testObj.Download(initialRequest)

			Convey("A 404 response should be returned", func() {
				So(responseErr, ShouldBeNil)
				So(responseStatus, ShouldEqual, http.StatusNotFound)
				So(readString(responseBody, t), ShouldEqual, expectedResponse)
			})
		})
	})
}

func TestContentServerError(t *testing.T) {
	t.Parallel()
	Convey("Given the content server doesn't respond", t, func() {

		initialRequest, err := http.NewRequest("GET", "http://localhost/download/table?format=html&uri=/foo/bar", nil)
		So(err, ShouldBeNil)

		expectedErr := errors.New("The content server is down")

		contentClient := &testdata.HttpClientMock{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return nil, expectedErr
			},
		}
		renderClient := createMockClient(http.StatusOK, "", "")

		testObj := table.NewDownloaderWithClients(contentClient, "http://"+contentHost, renderClient, "http://"+renderHost)

		Convey("When Download is invoked ", func() {

			_, _, responseStatus, responseErr := testObj.Download(initialRequest)

			Convey("An error should be returned", func() {
				So(responseErr, ShouldEqual, expectedErr)
				So(responseStatus, ShouldEqual, http.StatusInternalServerError)
			})
		})
	})
}

func TestRenderServerError(t *testing.T) {
	t.Parallel()
	Convey("Given the render service is down", t, func() {

		initialRequest, err := http.NewRequest("GET", "http://localhost/download/table?format=&uri=", nil)
		So(err, ShouldBeNil)

		expectedErr := errors.New("The render server is down")

		contentClient := createMockClient(http.StatusOK, "contentServerResponse", "application/json")
		renderClient := &testdata.HttpClientMock{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return nil, expectedErr
			},
		}

		testObj := table.NewDownloaderWithClients(contentClient, "http://"+contentHost, renderClient, "http://"+renderHost)

		Convey("When Download is invoked ", func() {

			_, _, responseStatus, responseErr := testObj.Download(initialRequest)

			Convey("An error should be returned", func() {
				So(responseErr, ShouldEqual, expectedErr)
				So(responseStatus, ShouldEqual, http.StatusInternalServerError)
			})
		})
	})
}

func createMockClient(status int, response string, contentType string) *testdata.HttpClientMock {
	header := http.Header{}
	header.Add("Content-Type", contentType)
	return &testdata.HttpClientMock{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: ioutil.NopCloser(strings.NewReader(response)), Header: header}, nil
		},
	}
}

func readString(reader io.Reader, t *testing.T) string {
	So(reader, ShouldNotBeNil)
	bytes, e := ioutil.ReadAll(reader)
	So(e, ShouldBeNil)
	return string(bytes)
}
