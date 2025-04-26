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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
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
				tracer.Start(context.Background(), "test")

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

func newMockConfig(t *testing.T, tracer trace.Tracer) config {
	// TODO: use mock meter instead of noop meter
	meter := noop.NewMeterProvider().Meter("test")

	instruments, err := newInstruments(meter)
	require.NoError(t, err)

	return config{
		Tracer:                tracer,
		Meter:                 meter,
		Instruments:           instruments,
		Attributes:            []attribute.KeyValue{defaultattribute},
		SpanNameFormatter:     defaultSpanNameFormatter,
		SQLCommenter:          newCommenter(false),
		SemConvStabilityOptIn: internalsemconv.OTelSemConvStabilityOptInStable,
		DBQueryTextAttributes: internalsemconv.NewDBQueryTextAttributes(internalsemconv.OTelSemConvStabilityOptInStable),
	}
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
	var span sdktrace.ReadOnlySpan
	if !parameter.omitSpan {
		if !parameter.noParentSpan {
			span = spanList[1]
		} else {
			span = spanList[0]
		}
	}

	if span != nil {
		if parameter.spanNotEnded {
			assert.True(t, span.EndTime().IsZero())
		} else {
			assert.False(t, span.EndTime().IsZero())
		}
		assert.Equal(t, trace.SpanKindClient, span.SpanKind())

		expectedAttributes := parameter.expectedAttributes
		if parameter.attributesGetter != nil {
			expectedAttributes = append(expectedAttributes, parameter.attributesGetter(context.Background(), parameter.method, parameter.query, parameter.args)...)
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

func TestRecordMetric(t *testing.T) {
	methodAttr := attribute.String("method", string(MethodConnQuery))
	testAttr := attribute.String("dummyKey", "dummyVal")
	testErrorAttr := attribute.String("errorKey", "errorVal")

	dummyAttributesGetter := func(_ context.Context, _ Method, _ string, _ []driver.NamedValue) []attribute.KeyValue {
		return []attribute.KeyValue{testAttr}
	}

	dummyErrorAttributesGetter := func(_ error) []attribute.KeyValue {
		return []attribute.KeyValue{testErrorAttr}
	}

	type args struct {
		ctx    context.Context
		cfg    config
		method Method
		query  string
		args   []driver.NamedValue
	}
	tests := []struct {
		name          string
		args          args
		recordErr     error
		expectedAttrs attribute.Set
	}{
		{
			name: "metric with no error",
			args: args{
				cfg:    newConfig(),
				method: MethodConnQuery,
				query:  "example query",
			},
			recordErr:     nil,
			expectedAttrs: attribute.NewSet(methodAttr, statusAttr("ok")),
		},
		{
			name: "metric with an error",
			args: args{
				cfg:    newConfig(),
				method: MethodConnQuery,
				query:  "example query",
			},
			recordErr:     assert.AnError,
			expectedAttrs: attribute.NewSet(methodAttr, statusAttr("error")),
		},
		{
			name: "metric with skip error but not disabled",
			args: args{
				cfg:    newConfig(),
				method: MethodConnQuery,
				query:  "example query",
			},
			recordErr:     driver.ErrSkip,
			expectedAttrs: attribute.NewSet(methodAttr, statusAttr("error")),
		},
		{
			name: "metric with skip error but disabled",
			args: args{
				cfg:    newConfig(WithDisableSkipErrMeasurement(true)),
				method: MethodConnQuery,
				query:  "example query",
			},
			recordErr:     driver.ErrSkip,
			expectedAttrs: attribute.NewSet(methodAttr, statusAttr("ok")),
		},
		{
			name: "metric with instrumentAttributesGetter",
			args: args{
				cfg:    newConfig(WithInstrumentAttributesGetter(dummyAttributesGetter)),
				method: MethodConnQuery,
				query:  "example query",
			},
			recordErr:     nil,
			expectedAttrs: attribute.NewSet(testAttr, methodAttr, statusAttr("ok")),
		},
		{
			name: "metric with instrumentErrorAttributesGetter",
			args: args{
				cfg:    newConfig(WithInstrumentErrorAttributesGetter(dummyErrorAttributesGetter)),
				method: MethodConnQuery,
				query:  "example query",
			},
			recordErr:     assert.AnError,
			expectedAttrs: attribute.NewSet(testErrorAttr, methodAttr, statusAttr("error")),
		},
		{
			name: "metric with instrumentErrorAttributesGetter and no error",
			args: args{
				cfg:    newConfig(WithInstrumentErrorAttributesGetter(dummyErrorAttributesGetter)),
				method: MethodConnQuery,
				query:  "example query",
			},
			recordErr:     nil,
			expectedAttrs: attribute.NewSet(methodAttr, statusAttr("ok")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLatency := &float64HistogramMock{}
			mockInstruments := &instruments{
				legacyLatency: mockLatency,
			}
			recordFunc := recordMetric(tt.args.ctx, mockInstruments, tt.args.cfg, tt.args.method, tt.args.query, tt.args.args)
			recordFunc(tt.recordErr)
			assert.Equal(t, tt.expectedAttrs, mockLatency.attrs)
		})
	}
}

type float64HistogramMock struct {
	// Add metric.Float64Histogram so we only need to implement the function we care about for the mock
	metric.Float64Histogram
	attrs attribute.Set
}

func (m *float64HistogramMock) Record(_ context.Context, _ float64, opts ...metric.RecordOption) {
	m.attrs = metric.NewRecordConfig(opts).Attributes()
}

func statusAttr(status string) attribute.KeyValue {
	return attribute.String("status", status)
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

			// Use newMockConfig instead of manual config creation
			cfg := newMockConfig(t, tracer)

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
