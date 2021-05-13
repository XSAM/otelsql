module github.com/XSAM/otelsql/example

go 1.15

replace github.com/XSAM/otelsql => ../

require (
	github.com/XSAM/otelsql v0.3.0
	github.com/go-sql-driver/mysql v1.5.0
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/stdout v0.20.0
	go.opentelemetry.io/otel/sdk v0.20.0
)
