module github.com/XSAM/otelsql/example

go 1.15

replace github.com/XSAM/otelsql => ../

require (
	github.com/XSAM/otelsql v0.4.0
	github.com/go-sql-driver/mysql v1.5.0
	go.opentelemetry.io/otel v1.0.0-RC1
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.0.0-RC1
	go.opentelemetry.io/otel/sdk v1.0.0-RC1
)
