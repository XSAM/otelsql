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
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestOptions(t *testing.T) {
	tracerProvider := sdktrace.NewTracerProvider()
	meterProvider := noop.NewMeterProvider()

	dummyAttributesGetter := func(_ context.Context, _ Method, _ string, _ []driver.NamedValue) []attribute.KeyValue {
		return []attribute.KeyValue{attribute.String("foo", "bar")}
	}

	dummyErrorAttributesGetter := func(_ error) []attribute.KeyValue {
		return []attribute.KeyValue{attribute.String("errorKey", "errorVal")}
	}

	testCases := []struct {
		name           string
		option         Option
		expectedConfig config
	}{
		{
			name:           "WithTracerProvider",
			option:         WithTracerProvider(tracerProvider),
			expectedConfig: config{TracerProvider: tracerProvider},
		},
		{
			name: "WithAttributes",
			option: WithAttributes(
				attribute.String("foo", "bar"),
				attribute.String("foo2", "bar2"),
			),
			expectedConfig: config{Attributes: []attribute.KeyValue{
				attribute.String("foo", "bar"),
				attribute.String("foo2", "bar2"),
			}},
		},
		{
			name:           "WithSpanNameFormatter",
			option:         WithSpanNameFormatter(nil),
			expectedConfig: config{SpanNameFormatter: nil},
		},
		{
			name:           "WithSpanOptions",
			option:         WithSpanOptions(SpanOptions{Ping: true}),
			expectedConfig: config{SpanOptions: SpanOptions{Ping: true}},
		},
		{
			name:           "WithMeterProvider",
			option:         WithMeterProvider(meterProvider),
			expectedConfig: config{MeterProvider: meterProvider},
		},
		{
			name:           "WithSQLCommenter",
			option:         WithSQLCommenter(true),
			expectedConfig: config{SQLCommenterEnabled: true},
		},
		{
			name:           "WithAttributesGetter",
			option:         WithAttributesGetter(dummyAttributesGetter),
			expectedConfig: config{AttributesGetter: dummyAttributesGetter},
		},
		{
			name:           "WithInstrumentAttributesGetter",
			option:         WithInstrumentAttributesGetter(dummyAttributesGetter),
			expectedConfig: config{InstrumentAttributesGetter: dummyAttributesGetter},
		},
		{
			name:           "WithDisableSkipErrMeasurement",
			option:         WithDisableSkipErrMeasurement(true),
			expectedConfig: config{DisableSkipErrMeasurement: true},
		},
		{
			name:           "WithInstrumentErrorAttributesGetter",
			option:         WithInstrumentErrorAttributesGetter(dummyErrorAttributesGetter),
			expectedConfig: config{InstrumentErrorAttributesGetter: dummyErrorAttributesGetter},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var cfg config

			tc.option.Apply(&cfg)

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
			default:
				assert.Equal(t, tc.expectedConfig, cfg)
			}
		})
	}
}
