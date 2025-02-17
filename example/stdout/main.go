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

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	"github.com/XSAM/otelsql"
)

const instrumentationName = "github.com/XSAM/otelsql/example/stdout"

var serviceName = semconv.ServiceNameKey.String("otesql-example")

var mysqlDSN = "root:otel_password@tcp(mysql)/db?parseTime=true"

func initTracer() {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSyncer(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			serviceName,
		)),
	)

	otel.SetTracerProvider(tp)
}

func initMeter() {
	// The exporter embeds a default OpenTelemetry Reader and
	// implements prometheus.Collector, allowing it to be used as
	// both a Reader and Collector.
	metricExporter, err := prometheus.New()
	if err != nil {
		log.Fatal(err)
	}
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metricExporter),
		metric.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			serviceName,
		)),
	)
	otel.SetMeterProvider(meterProvider)

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		_ = http.ListenAndServe(":2222", nil)
	}()
	fmt.Println("Prometheus server running on :2222")
}

func main() {
	initTracer()
	initMeter()

	attrs := append(otelsql.AttributesFromDSN(mysqlDSN), semconv.DBSystemMySQL)

	// Connect to database
	db, err := otelsql.Open("mysql", mysqlDSN, otelsql.WithAttributes(
		attrs...,
	))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(
		attrs...,
	))
	if err != nil {
		panic(err)
	}

	err = run(db)
	if err != nil {
		panic(err)
	}

	fmt.Println("Example finished updating, please visit localhost:2222/metrics")

	select {}
}

func run(db *sql.DB) error {
	// Create a parent span (Optional)
	tracer := otel.GetTracerProvider()
	ctx, span := tracer.Tracer(instrumentationName).Start(context.Background(), "example")
	defer span.End()

	err := query(ctx, db)
	if err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func query(ctx context.Context, db *sql.DB) error {
	// Make a query
	rows, err := db.QueryContext(ctx, `SELECT CURRENT_TIMESTAMP`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var currentTime time.Time
	for rows.Next() {
		err = rows.Scan(&currentTime)
		if err != nil {
			return err
		}
	}
	fmt.Println(currentTime)
	return nil
}
