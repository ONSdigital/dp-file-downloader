dp-file-downloader
================

Accepts GET requests to download a file (currently limited to tables in html, xlsx or csv format),
retrieves the definition of the file from the content server (Zebedee),
makes a POST request to the renderer service and returns the response to the user.

### Getting started

```
make debug
```


| Environment variable          |   Default              | Description
| ----------------------------- | -----------------------| -----------
| BIND_ADDR                     | :23400                 | The host and port to bind to                                                                    |
| CORS_ALLOWED_ORIGINS          | *                      | The allowed origins for CORS requests                                                           |
| SHUTDOWN_TIMEOUT              | 5s                     | The graceful shutdown timeout ([`time.Duration`](https://golang.org/pkg/time/#Duration) format) |
| HEALTHCHECK_INTERVAL          | 30 seconds             | Interval between health checks                                                                  |
| HEALTHCHECK_CRITICAL_TIMEOUT  | 90 seconds             | Amount of time to pass since last healthy health check to be deemed a critical failure          |
| OTEL_BATCH_TIMEOUT            | 5s                     | Interval between pushes to OT Collector                                                         |
| OTEL_EXPORTER_OTLP_ENDPOINT   | http://localhost:4317  | URL for OpenTelemetry endpoint                                                                  |
| OTEL_SERVICE_NAME             | "dp-file-downloader"   | Service name to report to telemetry tools                                                       |
| TABLE_RENDERER_HOST           | http://localhost:23300 | The hostname and port of the table renderer                                                     |
| CONTENT_SERVER_HOST           | http://localhost:8082  | The hostname and port of the content service                                                    |

### Endpoints

| url                                       | Method | Description                                          |
| ---                                       | ------ | -----------                                          |
| /download/table?format={format}&uri={uri} | GET    | Retrieves (generates) and returns the requested file |


### Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

### License

Copyright © 2016-2020, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details.
