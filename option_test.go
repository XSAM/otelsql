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
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestOptions(t *testing.T) {
	tracerProvider := sdktrace.NewTracerProvider()
	meterProvider := noop.NewMeterProvider()
	textMapPropagator := propagation.NewCompositeTextMapPropagator()

	dummyAttributesGetter := func(_ context.Context, _ Method, _ string, _ []driver.NamedValue) []attribute.KeyValue {
		return []attribute.KeyValue{attribute.String("foo", "bar")}
	}

	dummyErrorAttributesGetter := func(_ error) []attribute.KeyValue {
		return []attribute.KeyValue{attribute.String("errorKey", "errorVal")}
	}

	dummyOperationNameSetter := func(_ context.Context, _ Method, _ string) string {
		return "custom_operation"
	}

	testCases := []struct {
		name           string
		options        []Option
		expectedConfig config
	}{
		{
			name:           "WithTracerProvider",
			options:        []Option{WithTracerProvider(tracerProvider)},
			expectedConfig: config{TracerProvider: tracerProvider},
		},
		{
			name: "WithAttributes",
			options: []Option{
				WithAttributes(
					attribute.String("foo", "bar"),
					attribute.String("foo2", "bar2"),
				),
			},
			expectedConfig: config{Attributes: []attribute.KeyValue{
				attribute.String("foo", "bar"),
				attribute.String("foo2", "bar2"),
			}},
		},
		{
			name:           "WithSpanNameFormatter",
			options:        []Option{WithSpanNameFormatter(nil)},
			expectedConfig: config{SpanNameFormatter: nil},
		},
		{
			name:           "WithSpanOptions",
			options:        []Option{WithSpanOptions(SpanOptions{Ping: true})},
			expectedConfig: config{SpanOptions: SpanOptions{Ping: true}},
		},
		{
			name:           "WithMeterProvider",
			options:        []Option{WithMeterProvider(meterProvider)},
			expectedConfig: config{MeterProvider: meterProvider},
		},
		{
			name:           "WithSQLCommenter",
			options:        []Option{WithSQLCommenter(true)},
			expectedConfig: config{SQLCommenterEnabled: true},
		},
		{
			name:    "WithTextMapPropagator",
			options: []Option{WithTextMapPropagator(textMapPropagator)},
			expectedConfig: config{
				TextMapPropagator: textMapPropagator,
			},
		},
		{
			name:           "WithAttributesGetter",
			options:        []Option{WithAttributesGetter(dummyAttributesGetter)},
			expectedConfig: config{AttributesGetter: dummyAttributesGetter},
		},
		{
			name:           "WithInstrumentAttributesGetter",
			options:        []Option{WithInstrumentAttributesGetter(dummyAttributesGetter)},
			expectedConfig: config{InstrumentAttributesGetter: dummyAttributesGetter},
		},
		{
			name:           "WithDisableSkipErrMeasurement",
			options:        []Option{WithDisableSkipErrMeasurement(true)},
			expectedConfig: config{DisableSkipErrMeasurement: true},
		},
		{
			name:           "WithInstrumentErrorAttributesGetter",
			options:        []Option{WithInstrumentErrorAttributesGetter(dummyErrorAttributesGetter)},
			expectedConfig: config{InstrumentErrorAttributesGetter: dummyErrorAttributesGetter},
		},
		{
			name:           "WithOperationNameSetter",
			options:        []Option{WithOperationNameSetter(dummyOperationNameSetter)},
			expectedConfig: config{OperationNameSetter: dummyOperationNameSetter},
		},
		{
			name: "WithAttributes multiple calls should accumulate",
			options: []Option{
				WithAttributes(attribute.String("key1", "value1")),
				WithAttributes(attribute.String("key2", "value2")),
			},
			expectedConfig: config{Attributes: []attribute.KeyValue{
				attribute.String("key1", "value1"),
				attribute.String("key2", "value2"),
			}},
		},
		{
			name: "WithAttributes multiple calls with multiple attributes each",
			options: []Option{
				WithAttributes(
					attribute.String("key1", "value1"),
					attribute.String("key2", "value2"),
				),
				WithAttributes(
					attribute.String("key3", "value3"),
					attribute.String("key4", "value4"),
				),
			},
			expectedConfig: config{Attributes: []attribute.KeyValue{
				attribute.String("key1", "value1"),
				attribute.String("key2", "value2"),
				attribute.String("key3", "value3"),
				attribute.String("key4", "value4"),
			}},
		},
		{
			name:           "WithAttributes with empty attributes",
			options:        []Option{WithAttributes()},
			expectedConfig: config{Attributes: nil},
		},
		{
			name: "WithAttributes with empty followed by non-empty",
			options: []Option{
				WithAttributes(),
				WithAttributes(attribute.String("key1", "value1")),
			},
			expectedConfig: config{Attributes: []attribute.KeyValue{
				attribute.String("key1", "value1"),
			}},
		},
		{
			name: "WithAttributes three calls to verify order",
			options: []Option{
				WithAttributes(attribute.String("first", "1")),
				WithAttributes(attribute.String("second", "2")),
				WithAttributes(attribute.String("third", "3")),
			},
			expectedConfig: config{Attributes: []attribute.KeyValue{
				attribute.String("first", "1"),
				attribute.String("second", "2"),
				attribute.String("third", "3"),
			}},
		},
		{
			name: "WithAttributes duplicate keys should be preserved",
			options: []Option{
				WithAttributes(attribute.String("key", "value1")),
				WithAttributes(attribute.String("key", "value2")),
			},
			expectedConfig: config{Attributes: []attribute.KeyValue{
				attribute.String("key", "value1"),
				attribute.String("key", "value2"),
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var cfg config

			for _, opt := range tc.options {
				opt.Apply(&cfg)
			}

			switch {
			case tc.expectedConfig.AttributesGetter != nil:
				assert.Equal(
					t,
					tc.expectedConfig.AttributesGetter(context.Background(), "", "", nil),
					cfg.AttributesGetter(context.Background(), "", "", nil),
				)
			case tc.expectedConfig.InstrumentAttributesGetter != nil:
				assert.Equal(
					t,
					tc.expectedConfig.InstrumentAttributesGetter(context.Background(), "", "", nil),
					cfg.InstrumentAttributesGetter(context.Background(), "", "", nil),
				)
			case tc.expectedConfig.InstrumentErrorAttributesGetter != nil:
				assert.Equal(
					t,
					tc.expectedConfig.InstrumentErrorAttributesGetter(assert.AnError),
					cfg.InstrumentErrorAttributesGetter(assert.AnError),
				)
			case tc.expectedConfig.OperationNameSetter != nil:
				assert.Equal(
					t,
					tc.expectedConfig.OperationNameSetter(context.Background(), "", ""),
					cfg.OperationNameSetter(context.Background(), "", ""),
				)
			default:
				assert.Equal(t, tc.expectedConfig, cfg)
			}
		})
	}
}
