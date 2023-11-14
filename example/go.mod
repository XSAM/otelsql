module github.com/XSAM/otelsql/example

go 1.20

replace github.com/XSAM/otelsql => ../

require (
	github.com/XSAM/otelsql v0.0.0
	github.com/go-sql-driver/mysql v1.7.1
	github.com/prometheus/client_golang v1.17.0
	go.opentelemetry.io/otel v1.20.0
	go.opentelemetry.io/otel/exporters/prometheus v0.43.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.20.0
	go.opentelemetry.io/otel/sdk v1.20.0
	go.opentelemetry.io/otel/sdk/metric v1.20.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/go-logr/logr v1.3.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.44.0 // indirect
	github.com/prometheus/procfs v0.11.1 // indirect
	go.opentelemetry.io/otel/metric v1.20.0 // indirect
	go.opentelemetry.io/otel/trace v1.20.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)
