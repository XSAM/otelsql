# database/sql instrumentation OpenTelemetry Collector example

> This is an adapted example of https://github.com/open-telemetry/opentelemetry-go/tree/main/example/otel-collector to provide a one-stop place to play with this instrumentation and see the results visually.

A MySQL client using database/sql with instrumentation. This example shows the trace data on Jaeger and the metrics data on Prometheus server.

The complete data flow is:

```
                                             -----> Jaeger (trace)
MySQL client ---> OpenTelemetry Collector ---|
                                             -----> Prometheus (metrics)
```

These instructions expect you have
[Docker Compose V2](https://docs.docker.com/compose/) installed.

Bring up all services to run the
example:

```sh
docker compose up -d
```

Then check the logs of `client` service to make ensure it is finished:

```sh
docker compose logs client
```

Access the Jaeger UI at http://localhost:16686 and the Prometheus UI at http://localhost:9090 to see the results.

Shut down the services when you are finished with the example:

```sh
docker compose down
```
