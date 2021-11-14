module github.com/XSAM/otelsql/example

go 1.15

replace github.com/XSAM/otelsql => ../

require (
	github.com/XSAM/otelsql v0.0.0
	github.com/go-sql-driver/mysql v1.6.0
	go.opentelemetry.io/otel v1.2.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.1.0
	go.opentelemetry.io/otel/sdk v1.2.0
)
