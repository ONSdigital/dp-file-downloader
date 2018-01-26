package config

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSpec(t *testing.T) {
	Convey("Given an environment with no environment variables set", t, func() {
		cfg, err := Get()

		Convey("When the config values are retrieved", func() {

			Convey("There should be no error returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("The values should be set to the expected defaults", func() {
				So(cfg.BindAddr, ShouldEqual, ":23400")
				So(cfg.ShutdownTimeout, ShouldEqual, 5*time.Second)
				So(cfg.CORSAllowedOrigins, ShouldEqual, "*")
				So(cfg.HealthCheckInterval, ShouldEqual, 30*time.Second)
				So(cfg.TableRendererHost, ShouldEqual, "http://localhost:23300")
				So(cfg.ContentServerHost, ShouldEqual, "http://localhost:8082")
			})
		})
	})
}
