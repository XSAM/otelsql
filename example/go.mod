module github.com/XSAM/otelsql/example

go 1.15

replace github.com/XSAM/otelsql => ../

require (
	github.com/XSAM/otelsql v0.0.0
	github.com/go-sql-driver/mysql v1.6.0
	go.opentelemetry.io/otel v1.6.1
	go.opentelemetry.io/otel/exporters/prometheus v0.28.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.6.1
	go.opentelemetry.io/otel/metric v0.28.0
	go.opentelemetry.io/otel/sdk v1.6.1
	go.opentelemetry.io/otel/sdk/metric v0.28.0
)
