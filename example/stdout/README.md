# database/sql instrumentation stdout example

A MySQL client using database/sql with instrumentation. Then, the client prints the trace data to stdout and serves metrics data via prometheus client.

These instructions expect you have
[Docker Compose V2](https://docs.docker.com/compose/) installed.

Bring up all services to run the
example:

```sh
docker compose up -d
```

Then check the logs of `client` service to see the results:

```sh
docker compose logs client
```

Shut down the services when you are finished with the example:

```sh
docker compose down
```
