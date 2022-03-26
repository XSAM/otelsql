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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric/global"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/metric/export/aggregation"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"github.com/XSAM/otelsql"
)

const instrumentationName = "github.com/XSAM/otelsql/example"

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
	c := controller.New(
		processor.NewFactory(
			selector.NewWithHistogramDistribution(),
			aggregation.CumulativeTemporalitySelector(),
			processor.WithMemory(true),
		),
		controller.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			serviceName,
		)),
	)
	metricExporter, err := prometheus.New(prometheus.Config{}, c)
	if err != nil {
		log.Fatalf("failed to install metric exporter, %v", err)
	}
	global.SetMeterProvider(metricExporter.MeterProvider())

	http.HandleFunc("/", metricExporter.ServeHTTP)
	go func() {
		_ = http.ListenAndServe(":2222", nil)
	}()
	fmt.Println("Prometheus server running on :2222")
}

func main() {
	initTracer()
	initMeter()

	// Register an OTel driver
	driverName, err := otelsql.Register("mysql", semconv.DBSystemMySQL.Value.AsString())
	if err != nil {
		panic(err)
	}

	// Connect to database
	db, err := sql.Open(driverName, mysqlDSN)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = otelsql.RegisterDBStatsMetrics(db, semconv.DBSystemMySQL.Value.AsString())
	if err != nil {
		panic(err)
	}

	err = query(db)
	if err != nil {
		panic(err)
	}

	fmt.Println("Example finished updating, please visit :2222")

	select {}
}

func query(db *sql.DB) error {
	// Create a span
	tracer := otel.GetTracerProvider()
	ctx, span := tracer.Tracer(instrumentationName).Start(context.Background(), "example")
	defer span.End()

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
