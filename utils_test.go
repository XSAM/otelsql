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

package otelsql

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	internalsemconv "github.com/XSAM/otelsql/internal/semconv"
)

func TestRecordSpanError(t *testing.T) {
	testCases := []struct {
		name          string
		opts          SpanOptions
		err           error
		expectedError bool
		nilSpan       bool
	}{
		{
			name:          "no error",
			err:           nil,
			expectedError: false,
		},
		{
			name:          "normal error",
			err:           errors.New("error"),
			expectedError: true,
		},
		{
			name:          "normal error with DisableErrSkip",
			err:           errors.New("error"),
			opts:          SpanOptions{DisableErrSkip: true},
			expectedError: true,
		},
		{
			name:          "ErrSkip error",
			err:           driver.ErrSkip,
			expectedError: true,
		},
		{
			name:          "ErrSkip error with DisableErrSkip",
			err:           driver.ErrSkip,
			opts:          SpanOptions{DisableErrSkip: true},
			expectedError: false,
		},
		{
			name:          "avoid recording error due to RecordError option",
			err:           errors.New("error"),
			opts:          SpanOptions{RecordError: func(_ error) bool { return false }},
			expectedError: false,
		},
		{
			name:          "record error returns true",
			err:           errors.New("error"),
			opts:          SpanOptions{RecordError: func(_ error) bool { return true }},
			expectedError: true,
		},
		{
			name:          "nil span",
			err:           nil,
			nilSpan:       true,
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.nilSpan {
				// Create a span
				sr, provider := newTracerProvider()
				tracer := provider.Tracer("test")
				// Not using the spans from the SDK due to the limited data access.
				_, _ = tracer.Start(context.Background(), "test")

				// Get the span
				spanList := sr.Started()
				require.Len(t, spanList, 1)
				span := spanList[0]

				// Update the span
				recordSpanError(span, tc.opts, tc.err)

				// Check result
				if tc.expectedError {
					assert.Equal(t, codes.Error, span.Status().Code)
				} else {
					assert.Equal(t, codes.Unset, span.Status().Code)
				}
			} else {
				recordSpanError(nil, tc.opts, tc.err)
			}
		})
	}
}

func newTracerProvider() (*tracetest.SpanRecorder, trace.TracerProvider) {
	var sr tracetest.SpanRecorder
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(&sr),
	)
	return &sr, provider
}

func createDummySpan(ctx context.Context, tracer trace.Tracer) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, "dummy")
	defer span.End()
	return ctx, span
}

type spanAssertionParameter struct {
	parentSpan         trace.Span
	error              bool
	expectedAttributes []attribute.KeyValue
	method             Method
	noParentSpan       bool
	ctx                context.Context
	spanNotEnded       bool
	omitSpan           bool
	attributesGetter   AttributesGetter
	query              string
	args               []driver.NamedValue
}

func assertSpanList(
	t *testing.T, spanList []sdktrace.ReadOnlySpan, parameter spanAssertionParameter,
) {
	t.Helper()

	var span sdktrace.ReadOnlySpan
	if !parameter.omitSpan {
		if !parameter.noParentSpan {
			span = spanList[1]
		} else {
			span = spanList[0]
		}
	}

	if span == nil {
		return
	}
	if parameter.spanNotEnded {
		assert.True(t, span.EndTime().IsZero())
	} else {
		assert.False(t, span.EndTime().IsZero())
	}
	assert.Equal(t, trace.SpanKindClient, span.SpanKind())

	expectedAttributes := parameter.expectedAttributes
	if parameter.attributesGetter != nil {
		expectedAttributes = append(
			expectedAttributes,
			parameter.attributesGetter(context.Background(), parameter.method, parameter.query, parameter.args)...)
	}

	assert.Equal(t, expectedAttributes, span.Attributes())
	assert.Equal(t, string(parameter.method), span.Name())
	if parameter.parentSpan != nil {
		assert.Equal(t, parameter.parentSpan.SpanContext().TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, parameter.parentSpan.SpanContext().SpanID(), span.Parent().SpanID())
	}

	if parameter.error {
		assert.Equal(t, codes.Error, span.Status().Code)
	} else {
		assert.Equal(t, codes.Unset, span.Status().Code)
	}

	if parameter.ctx != nil {
		assert.Equal(t, span.SpanContext(), trace.SpanContextFromContext(parameter.ctx))
	}
}

func getExpectedSpanCount(noParentSpan bool, omitSpan bool) int {
	if !noParentSpan {
		if !omitSpan {
			return 2
		}
		return 1
	}
	if !omitSpan {
		return 1
	}
	return 0
}

func prepareTraces(
	noParentSpan bool,
) (context.Context, *tracetest.SpanRecorder, trace.Tracer, trace.Span) {
	sr, provider := newTracerProvider()
	tracer := provider.Tracer("test")

	var dummySpan trace.Span
	ctx := context.Background()
	if !noParentSpan {
		ctx, dummySpan = createDummySpan(context.Background(), tracer)
	}
	return ctx, sr, tracer, dummySpan
}

func prepareMetrics() (sdkmetric.Reader, *sdkmetric.MeterProvider) {
	metricReader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricReader),
	)
	return metricReader, meterProvider
}

func getDummyAttributesGetter() AttributesGetter {
	return func(_ context.Context, method Method, query string, args []driver.NamedValue) []attribute.KeyValue {
		attrs := []attribute.KeyValue{
			attribute.String("method", string(method)),
			attribute.String("query", query),
		}

		for i, a := range args {
			attrs = append(attrs, attribute.String(
				fmt.Sprintf("db.args.$%d", i+1),
				fmt.Sprintf("%v", a.Value)))
		}

		return attrs
	}
}

// omit is a dummy SpanFilter function which specifies to omit the span.
var omit SpanFilter = func(_ context.Context, _ Method, _ string, _ []driver.NamedValue) bool {
	return false
}

// keep is a dummy SpanFilter function which specifies to keep the span.
var keep SpanFilter = func(_ context.Context, _ Method, _ string, _ []driver.NamedValue) bool {
	return true
}

// TODO: apply maintidx linter
//
//nolint:maintidx
func TestRecordMetric(t *testing.T) {
	tests := []struct {
		name                  string
		cfgOptions            []Option
		semConvStabilityOptIn internalsemconv.OTelSemConvStabilityOptInType
		method                Method
		query                 string
		args                  []driver.NamedValue
		err                   error
		wantMetricData        metricdata.ResourceMetrics
	}{
		{
			name:                  "metric with no error",
			semConvStabilityOptIn: internalsemconv.OTelSemConvStabilityOptInStable,
			method:                MethodConnQuery,
			query:                 "example query",
			wantMetricData: metricdata.ResourceMetrics{
				Resource: resource.Default(),
				ScopeMetrics: []metricdata.ScopeMetrics{
					{
						Scope: instrumentation.Scope{
							Name:    "github.com/XSAM/otelsql",
							Version: Version(),
						},
						Metrics: []metricdata.Metrics{
							{
								Name:        "db.client.operation.duration",
								Description: "Duration of database client operations.",
								Unit:        "s",
								Data: metricdata.Histogram[float64]{
									Temporality: metricdata.CumulativeTemporality,
									DataPoints: []metricdata.HistogramDataPoint[float64]{
										{
											Count: 1,
											Sum:   2,
											Bounds: []float64{
												0,
												5,
												10,
												25,
												50,
												75,
												100,
												250,
												500,
												750,
												1000,
												2500,
												5000,
												7500,
												10000,
											},
											BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
											Min:          metricdata.NewExtrema[float64](2),
											Max:          metricdata.NewExtrema[float64](2),
											Attributes: attribute.NewSet(
												defaultattribute,
												attribute.String("db.operation.name", string(MethodConnQuery)),
											),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:                  "metric with an error",
			semConvStabilityOptIn: internalsemconv.OTelSemConvStabilityOptInStable,
			method:                MethodConnQuery,
			query:                 "example query",
			err:                   assert.AnError,
			wantMetricData: metricdata.ResourceMetrics{
				Resource: resource.Default(),
				ScopeMetrics: []metricdata.ScopeMetrics{
					{
						Scope: instrumentation.Scope{
							Name:    "github.com/XSAM/otelsql",
							Version: Version(),
						},
						Metrics: []metricdata.Metrics{
							{
								Name:        "db.client.operation.duration",
								Description: "Duration of database client operations.",
								Unit:        "s",
								Data: metricdata.Histogram[float64]{
									Temporality: metricdata.CumulativeTemporality,
									DataPoints: []metricdata.HistogramDataPoint[float64]{
										{
											Count: 1,
											Sum:   2,
											Bounds: []float64{
												0,
												5,
												10,
												25,
												50,
												75,
												100,
												250,
												500,
												750,
												1000,
												2500,
												5000,
												7500,
												10000,
											},
											BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
											Min:          metricdata.NewExtrema[float64](2),
											Max:          metricdata.NewExtrema[float64](2),
											Attributes: attribute.NewSet(
												defaultattribute,
												attribute.String("db.operation.name", string(MethodConnQuery)),
												attribute.String("error.type", "*errors.errorString"),
											),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:                  "metric with skip error but not disabled",
			semConvStabilityOptIn: internalsemconv.OTelSemConvStabilityOptInStable,
			method:                MethodConnQuery,
			query:                 "example query",
			err:                   driver.ErrSkip,
			wantMetricData: metricdata.ResourceMetrics{
				Resource: resource.Default(),
				ScopeMetrics: []metricdata.ScopeMetrics{
					{
						Scope: instrumentation.Scope{
							Name:    "github.com/XSAM/otelsql",
							Version: Version(),
						},
						Metrics: []metricdata.Metrics{
							{
								Name:        "db.client.operation.duration",
								Description: "Duration of database client operations.",
								Unit:        "s",
								Data: metricdata.Histogram[float64]{
									Temporality: metricdata.CumulativeTemporality,
									DataPoints: []metricdata.HistogramDataPoint[float64]{
										{
											Count: 1,
											Sum:   2,
											Bounds: []float64{
												0,
												5,
												10,
												25,
												50,
												75,
												100,
												250,
												500,
												750,
												1000,
												2500,
												5000,
												7500,
												10000,
											},
											BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
											Min:          metricdata.NewExtrema[float64](2),
											Max:          metricdata.NewExtrema[float64](2),
											Attributes: attribute.NewSet(
												defaultattribute,
												attribute.String("db.operation.name", string(MethodConnQuery)),
												attribute.String("error.type", "database/sql/driver.ErrSkip"),
											),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:                  "metric with skip error but disabled",
			semConvStabilityOptIn: internalsemconv.OTelSemConvStabilityOptInStable,
			cfgOptions:            []Option{WithDisableSkipErrMeasurement(true)},
			method:                MethodConnQuery,
			query:                 "example query",
			err:                   driver.ErrSkip,
			wantMetricData: metricdata.ResourceMetrics{
				Resource: resource.Default(),
				ScopeMetrics: []metricdata.ScopeMetrics{
					{
						Scope: instrumentation.Scope{
							Name:    "github.com/XSAM/otelsql",
							Version: Version(),
						},
						Metrics: []metricdata.Metrics{
							{
								Name:        "db.client.operation.duration",
								Description: "Duration of database client operations.",
								Unit:        "s",
								Data: metricdata.Histogram[float64]{
									Temporality: metricdata.CumulativeTemporality,
									DataPoints: []metricdata.HistogramDataPoint[float64]{
										{
											Count: 1,
											Sum:   2,
											Bounds: []float64{
												0,
												5,
												10,
												25,
												50,
												75,
												100,
												250,
												500,
												750,
												1000,
												2500,
												5000,
												7500,
												10000,
											},
											BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
											Min:          metricdata.NewExtrema[float64](2),
											Max:          metricdata.NewExtrema[float64](2),
											Attributes: attribute.NewSet(
												defaultattribute,
												attribute.String("db.operation.name", string(MethodConnQuery)),
											),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:                  "metric with instrumentAttributesGetter",
			semConvStabilityOptIn: internalsemconv.OTelSemConvStabilityOptInStable,
			cfgOptions: []Option{
				WithInstrumentAttributesGetter(
					func(_ context.Context, _ Method, _ string, _ []driver.NamedValue) []attribute.KeyValue {
						return []attribute.KeyValue{attribute.String("dummyKey", "dummyVal")}
					},
				),
			},
			method: MethodConnQuery,
			query:  "example query",
			wantMetricData: metricdata.ResourceMetrics{
				Resource: resource.Default(),
				ScopeMetrics: []metricdata.ScopeMetrics{
					{
						Scope: instrumentation.Scope{
							Name:    "github.com/XSAM/otelsql",
							Version: Version(),
						},
						Metrics: []metricdata.Metrics{
							{
								Name:        "db.client.operation.duration",
								Description: "Duration of database client operations.",
								Unit:        "s",
								Data: metricdata.Histogram[float64]{
									Temporality: metricdata.CumulativeTemporality,
									DataPoints: []metricdata.HistogramDataPoint[float64]{
										{
											Count: 1,
											Sum:   2,
											Bounds: []float64{
												0,
												5,
												10,
												25,
												50,
												75,
												100,
												250,
												500,
												750,
												1000,
												2500,
												5000,
												7500,
												10000,
											},
											BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
											Min:          metricdata.NewExtrema[float64](2),
											Max:          metricdata.NewExtrema[float64](2),
											Attributes: attribute.NewSet(
												defaultattribute,
												attribute.String("dummyKey", "dummyVal"),
												attribute.String("db.operation.name", string(MethodConnQuery)),
											),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:                  "metric with instrumentErrorAttributesGetter",
			semConvStabilityOptIn: internalsemconv.OTelSemConvStabilityOptInStable,
			cfgOptions: []Option{WithInstrumentErrorAttributesGetter(func(_ error) []attribute.KeyValue {
				return []attribute.KeyValue{attribute.String("errorKey", "errorVal")}
			})},
			method: MethodConnQuery,
			query:  "example query",
			err:    assert.AnError,
			wantMetricData: metricdata.ResourceMetrics{
				Resource: resource.Default(),
				ScopeMetrics: []metricdata.ScopeMetrics{
					{
						Scope: instrumentation.Scope{
							Name:    "github.com/XSAM/otelsql",
							Version: Version(),
						},
						Metrics: []metricdata.Metrics{
							{
								Name:        "db.client.operation.duration",
								Description: "Duration of database client operations.",
								Unit:        "s",
								Data: metricdata.Histogram[float64]{
									Temporality: metricdata.CumulativeTemporality,
									DataPoints: []metricdata.HistogramDataPoint[float64]{
										{
											Count: 1,
											Sum:   2,
											Bounds: []float64{
												0,
												5,
												10,
												25,
												50,
												75,
												100,
												250,
												500,
												750,
												1000,
												2500,
												5000,
												7500,
												10000,
											},
											BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
											Min:          metricdata.NewExtrema[float64](2),
											Max:          metricdata.NewExtrema[float64](2),
											Attributes: attribute.NewSet(
												defaultattribute,
												attribute.String("errorKey", "errorVal"),
												attribute.String("db.operation.name", string(MethodConnQuery)),
												attribute.String("error.type", "*errors.errorString"),
											),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:                  "metric with instrumentErrorAttributesGetter and no error",
			semConvStabilityOptIn: internalsemconv.OTelSemConvStabilityOptInStable,
			cfgOptions: []Option{WithInstrumentErrorAttributesGetter(func(_ error) []attribute.KeyValue {
				return []attribute.KeyValue{attribute.String("errorKey", "errorVal")}
			})},
			method: MethodConnQuery,
			query:  "example query",
			wantMetricData: metricdata.ResourceMetrics{
				Resource: resource.Default(),
				ScopeMetrics: []metricdata.ScopeMetrics{
					{
						Scope: instrumentation.Scope{
							Name:    "github.com/XSAM/otelsql",
							Version: Version(),
						},
						Metrics: []metricdata.Metrics{
							{
								Name:        "db.client.operation.duration",
								Description: "Duration of database client operations.",
								Unit:        "s",
								Data: metricdata.Histogram[float64]{
									Temporality: metricdata.CumulativeTemporality,
									DataPoints: []metricdata.HistogramDataPoint[float64]{
										{
											Count: 1,
											Sum:   2,
											Bounds: []float64{
												0,
												5,
												10,
												25,
												50,
												75,
												100,
												250,
												500,
												750,
												1000,
												2500,
												5000,
												7500,
												10000,
											},
											BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
											Min:          metricdata.NewExtrema[float64](2),
											Max:          metricdata.NewExtrema[float64](2),
											Attributes: attribute.NewSet(
												defaultattribute,
												attribute.String("db.operation.name", string(MethodConnQuery)),
											),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:                  "metric with OTelSemConvStabilityOptInDup",
			semConvStabilityOptIn: internalsemconv.OTelSemConvStabilityOptInDup,
			method:                MethodConnQuery,
			query:                 "example query",
			wantMetricData: metricdata.ResourceMetrics{
				Resource: resource.Default(),
				ScopeMetrics: []metricdata.ScopeMetrics{
					{
						Scope: instrumentation.Scope{
							Name:    "github.com/XSAM/otelsql",
							Version: Version(),
						},
						Metrics: []metricdata.Metrics{
							// Legacy format metric
							{
								Name:        "db.sql.latency",
								Description: "The latency of calls in milliseconds",
								Unit:        "ms",
								Data: metricdata.Histogram[float64]{
									Temporality: metricdata.CumulativeTemporality,
									DataPoints: []metricdata.HistogramDataPoint[float64]{
										{
											Count: 1,
											Sum:   2000, // 2s converted to ms (2 * 1000)
											Bounds: []float64{
												0,
												5,
												10,
												25,
												50,
												75,
												100,
												250,
												500,
												750,
												1000,
												2500,
												5000,
												7500,
												10000,
											},
											BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0},
											Min:          metricdata.NewExtrema[float64](2000),
											Max:          metricdata.NewExtrema[float64](2000),
											Attributes: attribute.NewSet(
												defaultattribute,
												attribute.String("method", string(MethodConnQuery)),
												attribute.String("status", "ok"),
											),
										},
									},
								},
							},
							// New format metric
							{
								Name:        "db.client.operation.duration",
								Description: "Duration of database client operations.",
								Unit:        "s",
								Data: metricdata.Histogram[float64]{
									Temporality: metricdata.CumulativeTemporality,
									DataPoints: []metricdata.HistogramDataPoint[float64]{
										{
											Count: 1,
											Sum:   2,
											Bounds: []float64{
												0,
												5,
												10,
												25,
												50,
												75,
												100,
												250,
												500,
												750,
												1000,
												2500,
												5000,
												7500,
												10000,
											},
											BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
											Min:          metricdata.NewExtrema[float64](2),
											Max:          metricdata.NewExtrema[float64](2),
											Attributes: attribute.NewSet(
												defaultattribute,
												attribute.String("db.operation.name", string(MethodConnQuery)),
											),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:                  "metric with OTelSemConvStabilityOptInNone",
			semConvStabilityOptIn: internalsemconv.OTelSemConvStabilityOptInNone,
			method:                MethodConnQuery,
			query:                 "example query",
			wantMetricData: metricdata.ResourceMetrics{
				Resource: resource.Default(),
				ScopeMetrics: []metricdata.ScopeMetrics{
					{
						Scope: instrumentation.Scope{
							Name:    "github.com/XSAM/otelsql",
							Version: Version(),
						},
						Metrics: []metricdata.Metrics{
							// Only legacy format metric
							{
								Name:        "db.sql.latency",
								Description: "The latency of calls in milliseconds",
								Unit:        "ms",
								Data: metricdata.Histogram[float64]{
									Temporality: metricdata.CumulativeTemporality,
									DataPoints: []metricdata.HistogramDataPoint[float64]{
										{
											Count: 1,
											Sum:   2000, // 2s converted to ms (2 * 1000)
											Bounds: []float64{
												0,
												5,
												10,
												25,
												50,
												75,
												100,
												250,
												500,
												750,
												1000,
												2500,
												5000,
												7500,
												10000,
											},
											BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0},
											Min:          metricdata.NewExtrema[float64](2000),
											Max:          metricdata.NewExtrema[float64](2000),
											Attributes: attribute.NewSet(
												defaultattribute,
												attribute.String("method", string(MethodConnQuery)),
												attribute.String("status", "ok"),
											),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:                  "metric with OTelSemConvStabilityOptInDup and error",
			semConvStabilityOptIn: internalsemconv.OTelSemConvStabilityOptInDup,
			method:                MethodConnQuery,
			query:                 "example query",
			err:                   assert.AnError,
			wantMetricData: metricdata.ResourceMetrics{
				Resource: resource.Default(),
				ScopeMetrics: []metricdata.ScopeMetrics{
					{
						Scope: instrumentation.Scope{
							Name:    "github.com/XSAM/otelsql",
							Version: Version(),
						},
						Metrics: []metricdata.Metrics{
							// Legacy format metric with error
							{
								Name:        "db.sql.latency",
								Description: "The latency of calls in milliseconds",
								Unit:        "ms",
								Data: metricdata.Histogram[float64]{
									Temporality: metricdata.CumulativeTemporality,
									DataPoints: []metricdata.HistogramDataPoint[float64]{
										{
											Count: 1,
											Sum:   2000, // 2s converted to ms (2 * 1000)
											Bounds: []float64{
												0,
												5,
												10,
												25,
												50,
												75,
												100,
												250,
												500,
												750,
												1000,
												2500,
												5000,
												7500,
												10000,
											},
											BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0},
											Min:          metricdata.NewExtrema[float64](2000),
											Max:          metricdata.NewExtrema[float64](2000),
											Attributes: attribute.NewSet(
												defaultattribute,
												attribute.String("method", string(MethodConnQuery)),
												attribute.String("status", "error"),
											),
										},
									},
								},
							},
							// New format metric with error
							{
								Name:        "db.client.operation.duration",
								Description: "Duration of database client operations.",
								Unit:        "s",
								Data: metricdata.Histogram[float64]{
									Temporality: metricdata.CumulativeTemporality,
									DataPoints: []metricdata.HistogramDataPoint[float64]{
										{
											Count: 1,
											Sum:   2,
											Bounds: []float64{
												0,
												5,
												10,
												25,
												50,
												75,
												100,
												250,
												500,
												750,
												1000,
												2500,
												5000,
												7500,
												10000,
											},
											BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
											Min:          metricdata.NewExtrema[float64](2),
											Max:          metricdata.NewExtrema[float64](2),
											Attributes: attribute.NewSet(
												defaultattribute,
												attribute.String("db.operation.name", string(MethodConnQuery)),
												attribute.String("error.type", "*errors.errorString"),
											),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:                  "metric with OTelSemConvStabilityOptInNone and error",
			semConvStabilityOptIn: internalsemconv.OTelSemConvStabilityOptInNone,
			method:                MethodConnQuery,
			query:                 "example query",
			err:                   assert.AnError,
			wantMetricData: metricdata.ResourceMetrics{
				Resource: resource.Default(),
				ScopeMetrics: []metricdata.ScopeMetrics{
					{
						Scope: instrumentation.Scope{
							Name:    "github.com/XSAM/otelsql",
							Version: Version(),
						},
						Metrics: []metricdata.Metrics{
							// Only legacy format metric with error
							{
								Name:        "db.sql.latency",
								Description: "The latency of calls in milliseconds",
								Unit:        "ms",
								Data: metricdata.Histogram[float64]{
									Temporality: metricdata.CumulativeTemporality,
									DataPoints: []metricdata.HistogramDataPoint[float64]{
										{
											Count: 1,
											Sum:   2000, // 2s converted to ms (2 * 1000)
											Bounds: []float64{
												0,
												5,
												10,
												25,
												50,
												75,
												100,
												250,
												500,
												750,
												1000,
												2500,
												5000,
												7500,
												10000,
											},
											BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0},
											Min:          metricdata.NewExtrema[float64](2000),
											Max:          metricdata.NewExtrema[float64](2000),
											Attributes: attribute.NewSet(
												defaultattribute,
												attribute.String("method", string(MethodConnQuery)),
												attribute.String("status", "error"),
											),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:                  "metric with OTelSemConvStabilityOptInDup with ErrSkip and DisableSkipErrMeasurement",
			semConvStabilityOptIn: internalsemconv.OTelSemConvStabilityOptInDup,
			cfgOptions:            []Option{WithDisableSkipErrMeasurement(true)},
			method:                MethodConnQuery,
			query:                 "example query",
			err:                   driver.ErrSkip,
			wantMetricData: metricdata.ResourceMetrics{
				Resource: resource.Default(),
				ScopeMetrics: []metricdata.ScopeMetrics{
					{
						Scope: instrumentation.Scope{
							Name:    "github.com/XSAM/otelsql",
							Version: Version(),
						},
						Metrics: []metricdata.Metrics{
							// Legacy format metric with ok status despite error
							{
								Name:        "db.sql.latency",
								Description: "The latency of calls in milliseconds",
								Unit:        "ms",
								Data: metricdata.Histogram[float64]{
									Temporality: metricdata.CumulativeTemporality,
									DataPoints: []metricdata.HistogramDataPoint[float64]{
										{
											Count: 1,
											Sum:   2000,
											Bounds: []float64{
												0,
												5,
												10,
												25,
												50,
												75,
												100,
												250,
												500,
												750,
												1000,
												2500,
												5000,
												7500,
												10000,
											},
											BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0},
											Min:          metricdata.NewExtrema[float64](2000),
											Max:          metricdata.NewExtrema[float64](2000),
											Attributes: attribute.NewSet(
												defaultattribute,
												attribute.String("method", string(MethodConnQuery)),
												attribute.String("status", "ok"),
											),
										},
									},
								},
							},
							// New format metric without error attributes
							{
								Name:        "db.client.operation.duration",
								Description: "Duration of database client operations.",
								Unit:        "s",
								Data: metricdata.Histogram[float64]{
									Temporality: metricdata.CumulativeTemporality,
									DataPoints: []metricdata.HistogramDataPoint[float64]{
										{
											Count: 1,
											Sum:   2,
											Bounds: []float64{
												0,
												5,
												10,
												25,
												50,
												75,
												100,
												250,
												500,
												750,
												1000,
												2500,
												5000,
												7500,
												10000,
											},
											BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
											Min:          metricdata.NewExtrema[float64](2),
											Max:          metricdata.NewExtrema[float64](2),
											Attributes: attribute.NewSet(
												defaultattribute,
												attribute.String("db.operation.name", string(MethodConnQuery)),
											),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metricReader, meterProvider := prepareMetrics()
			cfg := newConfig(append(tt.cfgOptions, WithMeterProvider(meterProvider), WithAttributes(defaultattribute))...)
			cfg.SemConvStabilityOptIn = tt.semConvStabilityOptIn

			timeNow = func() time.Time {
				return time.Unix(1, 0)
			}
			t.Cleanup(func() {
				timeNow = time.Now
			})

			recordFunc := recordMetric(context.Background(), cfg.Instruments, cfg, tt.method, tt.query, tt.args)

			timeNow = func() time.Time {
				return time.Unix(3, 0)
			}
			recordFunc(tt.err)

			var metricsData metricdata.ResourceMetrics
			err := metricReader.Collect(context.Background(), &metricsData)
			require.NoError(t, err)

			metricdatatest.AssertEqual(t, tt.wantMetricData, metricsData, metricdatatest.IgnoreTimestamp())
		})
	}
}

func TestCreateSpan(t *testing.T) {
	methodName := MethodConnQuery
	query := "SELECT * FROM users"

	tests := []struct {
		name                    string
		enableDBStatement       bool
		disableQuery            bool
		customAttributesGetter  AttributesGetter
		expectedSpanName        string
		customSpanNameFormatter SpanNameFormatter
		expectedAttrs           []attribute.KeyValue
	}{
		{
			name:              "basic span with DB statement enabled",
			enableDBStatement: true,
			expectedSpanName:  string(methodName),
			expectedAttrs: []attribute.KeyValue{
				defaultattribute,
				attribute.String("db.query.text", query),
			},
		},
		{
			name:              "span with DB statement disabled",
			enableDBStatement: false,
			expectedSpanName:  string(methodName),
			expectedAttrs: []attribute.KeyValue{
				defaultattribute,
			},
		},
		{
			name:              "span with DisableQuery option",
			enableDBStatement: true,
			disableQuery:      true,
			expectedSpanName:  string(methodName),
			expectedAttrs: []attribute.KeyValue{
				defaultattribute,
			},
		},
		{
			name:              "span with custom attributes getter",
			enableDBStatement: true,
			customAttributesGetter: func(_ context.Context, _ Method, _ string, _ []driver.NamedValue) []attribute.KeyValue {
				return []attribute.KeyValue{attribute.String("custom.attr", "custom_value")}
			},
			expectedSpanName: string(methodName),
			expectedAttrs: []attribute.KeyValue{
				defaultattribute,
				attribute.String("db.query.text", query),
				attribute.String("custom.attr", "custom_value"),
			},
		},
		{
			name:              "span with custom name formatter",
			enableDBStatement: true,
			customSpanNameFormatter: func(_ context.Context, _ Method, query string) string {
				return "Custom-" + query
			},
			expectedSpanName: "Custom-SELECT * FROM users",
			expectedAttrs: []attribute.KeyValue{
				defaultattribute,
				attribute.String("db.query.text", query),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			sr, provider := newTracerProvider()
			tracer := provider.Tracer("test")

			t.Setenv("OTEL_SEMCONV_STABILITY_OPT_IN", "database")
			cfg := newConfig(WithAttributes(defaultattribute))
			cfg.Tracer = tracer

			// Customize config for test case
			if tt.disableQuery {
				cfg.SpanOptions.DisableQuery = true
			}
			if tt.customAttributesGetter != nil {
				cfg.AttributesGetter = tt.customAttributesGetter
			}
			if tt.customSpanNameFormatter != nil {
				cfg.SpanNameFormatter = tt.customSpanNameFormatter
			}

			// Act
			ctx := context.Background()
			ctx, span := createSpan(ctx, cfg, methodName, tt.enableDBStatement, query, nil)
			span.End()

			// Get the span
			spans := sr.Ended()
			require.Len(t, spans, 1)

			spanData := spans[0]
			assert.Equal(t, tt.expectedSpanName, spanData.Name())
			assert.Equal(t, trace.SpanKindClient, spanData.SpanKind())
			assert.Equal(t, span.SpanContext(), trace.SpanContextFromContext(ctx))
			assert.ElementsMatch(t, tt.expectedAttrs, spanData.Attributes())
		})
	}
}
