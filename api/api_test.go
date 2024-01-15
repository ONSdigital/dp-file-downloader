package api

import (
	"context"
	"testing"

	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/ONSdigital/dp-file-downloader/api/testdata"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

var responseHeaders = map[string]string{
	"Content-Type":        "text/plain",
	"Content-Disposition": "attachment; filename=\"fname.ext\"",
}

var ctx = context.Background()
var hcMock = healthcheck.HealthCheck{}
var responseBody = "Mock invocation"
var queryParam = "my-query-param"
var queryValue = "foo"
var baseURL = "http://localhost:80/download/"

func TestSuccessfulDownload(t *testing.T) {
	t.Parallel()
	Convey("Given an api with a mock implementation of Downloader", t, func() {
		mockDownloader := createMockDownloader("mock", []string{queryParam}, responseBody, http.StatusOK, nil)

		api := routes(ctx, mux.NewRouter(), &hcMock, mockDownloader)

		Convey("When a route is invoked ", func() {
			url := baseURL + mockDownloader.Type() + "?" + queryParam + "=" + queryValue
			r, err := http.NewRequest("GET", url, http.NoBody)
			So(err, ShouldBeNil)

			w := httptest.NewRecorder()
			api.router.ServeHTTP(w, r)

			Convey("Download should be invoked with the request", func() {
				So(len(mockDownloader.DownloadCalls()), ShouldEqual, 1)
				So(mockDownloader.DownloadCalls()[0].R.URL, ShouldResemble, r.URL)
			})

			Convey("The correct response should be returned", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
				for key, expected := range responseHeaders {
					So(w.Header().Get(key), ShouldEqual, expected)
				}
				So(w.Body.String(), ShouldEqual, responseBody)
			})
		})
	})
}

func TestReturnsCorrectStatus(t *testing.T) {
	t.Parallel()
	Convey("Given an api with a mock implementation of Downloader", t, func() {
		mockDownloader := createMockDownloader("mock", []string{queryParam}, responseBody, http.StatusBadRequest, nil)

		api := routes(ctx, mux.NewRouter(), &hcMock, mockDownloader)

		Convey("When a route is invoked ", func() {
			url := baseURL + mockDownloader.Type() + "?" + queryParam + "=" + queryValue
			r, err := http.NewRequest("GET", url, http.NoBody)
			So(err, ShouldBeNil)

			w := httptest.NewRecorder()
			api.router.ServeHTTP(w, r)

			Convey("The correct response should be returned", func() {
				So(w.Result().StatusCode, ShouldEqual, http.StatusBadRequest)
				for key, expected := range responseHeaders {
					So(w.Header().Get(key), ShouldEqual, expected)
				}
				So(w.Body.String(), ShouldEqual, responseBody)
			})
		})
	})
}

func TestNotFound(t *testing.T) {
	t.Parallel()
	Convey("Given an api with a mock implementation of Downloader", t, func() {
		mockDownloader := createMockDownloader("mock", []string{queryParam}, responseBody, http.StatusOK, nil)

		api := routes(ctx, mux.NewRouter(), &hcMock, mockDownloader)

		Convey("When a route is invoked with the wrong type", func() {
			r, err := http.NewRequest("GET", "http://localhost/download/foo"+"?"+queryParam+"="+queryValue, http.NoBody)
			So(err, ShouldBeNil)
			r.Header.Add(queryParam, "foo")

			w := httptest.NewRecorder()
			api.router.ServeHTTP(w, r)

			Convey("Then a 404 response should be returned", func() {
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})
	})
}

func TestReturnsError(t *testing.T) {
	t.Parallel()
	Convey("Given an api with a mock implementation of Downloader that returns an error", t, func() {
		downloadError := errors.New("This is an error")
		mockDownloader := createMockDownloader("mock", []string{queryParam}, responseBody, http.StatusOK, downloadError)

		api := routes(ctx, mux.NewRouter(), &hcMock, mockDownloader)

		Convey("When a route is invoked ", func() {
			url := baseURL + mockDownloader.Type() + "?" + queryParam + "=" + queryValue
			r, err := http.NewRequest("GET", url, http.NoBody)
			So(err, ShouldBeNil)

			w := httptest.NewRecorder()
			api.router.ServeHTTP(w, r)

			Convey("The correct error response should be returned", func() {
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
				So(strings.Trim(w.Body.String(), "\n"), ShouldEqual, downloadError.Error())
			})
		})
	})
}

func TestReturnsBadRequest(t *testing.T) {
	t.Parallel()
	Convey("Given an api with a mock implementation of Downloader that returns an error with bad request", t, func() {
		downloadError := errors.New("That was a bad request")
		mockDownloader := createMockDownloader("mock", []string{queryParam}, responseBody, http.StatusBadRequest, downloadError)

		api := routes(ctx, mux.NewRouter(), &hcMock, mockDownloader)

		Convey("When a route is invoked ", func() {
			url := baseURL + mockDownloader.Type() + "?" + queryParam + "=" + queryValue
			r, err := http.NewRequest("GET", url, http.NoBody)
			So(err, ShouldBeNil)

			w := httptest.NewRecorder()
			api.router.ServeHTTP(w, r)

			Convey("The correct error response should be returned", func() {
				So(w.Code, ShouldEqual, http.StatusBadRequest)
				So(strings.Trim(w.Body.String(), "\n"), ShouldEqual, downloadError.Error())
			})
		})
	})
}

func createMockDownloader(path string, query []string, responseBody string, code int, err error) *testdata.DownloaderMock {
	return &testdata.DownloaderMock{
		QueryParametersFunc: func() []string {
			return query
		},
		TypeFunc: func() string {
			return path
		},
		DownloadFunc: func(r *http.Request) (io.ReadCloser, map[string]string, int, error) {
			return io.NopCloser(strings.NewReader(responseBody)), responseHeaders, code, err
		},
	}
}
