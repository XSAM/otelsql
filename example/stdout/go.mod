module github.com/XSAM/otelsql/example/stdout

go 1.21

replace github.com/XSAM/otelsql => ../../

require (
	github.com/XSAM/otelsql v0.0.0
	github.com/go-sql-driver/mysql v1.8.1
	github.com/prometheus/client_golang v1.19.0
	go.opentelemetry.io/otel v1.26.0
	go.opentelemetry.io/otel/exporters/prometheus v0.48.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.26.0
	go.opentelemetry.io/otel/sdk v1.26.0
	go.opentelemetry.io/otel/sdk/metric v1.26.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	go.opentelemetry.io/otel/metric v1.26.0 // indirect
	go.opentelemetry.io/otel/trace v1.26.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)
