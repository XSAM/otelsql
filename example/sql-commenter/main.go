// Copyright Sam Xie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main implements an example application that demonstrates how to use otelsql
// with OpenTelemetry Collector for tracing and metrics collection.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"

	_ "github.com/microsoft/go-mssqldb"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/XSAM/otelsql"
)

const instrumentationName = "github.com/XSAM/otelsql/example/sqlserver"

var serviceName = semconv.ServiceNameKey.String("otelsql-example")

var DSN = "sqlserver://sa:Passw0rd@mssql:1433/instance"

// Initialize a gRPC connection to be used by both the tracer and meter
// providers.
func initConn() (*grpc.ClientConn, error) {
	// Make a gRPC connection with otel collector.
	conn, err := grpc.NewClient("otel-collector:4317",
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	return conn, nil
}

// Initializes an OTLP exporter, and configures the corresponding trace providers.
func initTracerProvider(ctx context.Context, conn *grpc.ClientConn) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			serviceName,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Shutdown will flush any remaining spans and shut down the exporter.
	return tracerProvider.Shutdown, nil
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	conn, err := initConn()
	if err != nil {
		return err
	}

	shutdownTracerProvider, err := initTracerProvider(ctx, conn)
	if err != nil {
		return err
	}
	defer func() {
		if err := shutdownTracerProvider(ctx); err != nil {
			slog.Error("failed to shutdown TracerProvider", "error", err)
		}
	}()

	db := connectDB()
	defer func() { _ = db.Close() }()

	err = runSQLQuery(ctx, db)
	if err != nil {
		return err
	}

	slog.Info("Example finished")

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func connectDB() *sql.DB {
	// Connect to database
	db, err := otelsql.Open("sqlserver", DSN, otelsql.WithSQLCommenter(true,
		propagation.NewCompositeTextMapPropagator(
			otelsql.TextAttributesPropagator{Attributes: map[string]string{string(semconv.ServiceNameKey): "otelsql-example"}},
			propagation.TraceContext{}, // Optional, if you want to propagate trace context
		),
	))
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func runSQLQuery(ctx context.Context, db *sql.DB) error {
	// Create a parent span (Optional)
	tracer := otel.GetTracerProvider()
	ctx, span := tracer.Tracer(instrumentationName).Start(ctx, "example")
	defer span.End()

	err := query(ctx, db)
	if err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func query(ctx context.Context, db *sql.DB) error {
	// Add WAITFOR DELAY to simulate a long-running query that could take 5 seconds, so we can easily catch it by querying query samples.
	_, err := db.ExecContext(ctx, `SELECT * FROM sys.dm_exec_connections; WAITFOR DELAY '00:00:05'`)
	if err != nil {
		return err
	}

	return nil
}
