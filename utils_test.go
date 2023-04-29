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
	"go.opentelemetry.io/otel/metric/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
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
			opts:          SpanOptions{RecordError: func(err error) bool { return false }},
			expectedError: false,
		},
		{
			name:          "record error returns true",
			err:           errors.New("error"),
			opts:          SpanOptions{RecordError: func(err error) bool { return true }},
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
		Tracer:            tracer,
		Meter:             meter,
		Instruments:       instruments,
		Attributes:        []attribute.KeyValue{defaultattribute},
		SpanNameFormatter: &defaultSpanNameFormatter{},
		SQLCommenter:      newCommenter(false),
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

func assertSpanList(t *testing.T, spanList []sdktrace.ReadOnlySpan, parameter spanAssertionParameter) {
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

func prepareTraces(noParentSpan bool) (context.Context, *tracetest.SpanRecorder, trace.Tracer, trace.Span) {
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
	return func(ctx context.Context, method Method, query string, args []driver.NamedValue) []attribute.KeyValue {
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
