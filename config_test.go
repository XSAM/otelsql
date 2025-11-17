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
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconvlegacy "go.opentelemetry.io/otel/semconv/v1.24.0"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"

	internalsemconv "github.com/XSAM/otelsql/internal/semconv"
)

func TestNewConfig(t *testing.T) {
	// Set a clean environment for the test
	t.Setenv(internalsemconv.OTelSemConvStabilityOptIn, "")

	cfg := newConfig(WithSpanOptions(SpanOptions{Ping: true}), WithAttributes(semconv.DBSystemNameMySQL))

	// Compare function result
	assert.Equal(
		t,
		defaultSpanNameFormatter(context.Background(), "foo", "bar"),
		cfg.SpanNameFormatter(context.Background(), "foo", "bar"),
	)

	// Verify DBQueryTextAttributes exists and returns expected format
	assert.NotNil(t, cfg.DBQueryTextAttributes)
	attrs := cfg.DBQueryTextAttributes("SELECT 1")
	assert.Len(t, attrs, 1)
	assert.Contains(t, attrs[0].Key, string(semconvlegacy.DBStatementKey))

	// Ignore function compares for test equality check
	cfg.SpanNameFormatter = nil
	cfg.DBQueryTextAttributes = nil

	assert.Equal(t, config{
		TracerProvider: otel.GetTracerProvider(),
		Tracer: otel.GetTracerProvider().Tracer(
			instrumentationName,
			trace.WithInstrumentationVersion(Version()),
		),
		MeterProvider: otel.GetMeterProvider(),
		Meter: otel.GetMeterProvider().Meter(
			instrumentationName,
			metric.WithInstrumentationVersion(Version()),
		),
		// No need to check values of instruments in this part.
		Instruments: cfg.Instruments,
		SpanOptions: SpanOptions{Ping: true},
		Attributes: []attribute.KeyValue{
			semconv.DBSystemNameMySQL,
		},
		SQLCommenter:          newCommenter(false, otel.GetTextMapPropagator()),
		SemConvStabilityOptIn: internalsemconv.OTelSemConvStabilityOptInNone,
	}, cfg)
	assert.NotNil(t, cfg.Instruments)
}

func TestConfigSemConvStabilityOptIn(t *testing.T) {
	testCases := []struct {
		name          string
		envValue      string
		expectedOptIn internalsemconv.OTelSemConvStabilityOptInType
	}{
		{
			name:          "none",
			envValue:      "",
			expectedOptIn: internalsemconv.OTelSemConvStabilityOptInNone,
		},
		{
			name:          "database/dup",
			envValue:      "database/dup",
			expectedOptIn: internalsemconv.OTelSemConvStabilityOptInDup,
		},
		{
			name:          "database",
			envValue:      "database",
			expectedOptIn: internalsemconv.OTelSemConvStabilityOptInStable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use t.Setenv which automatically cleans up after the test
			t.Setenv(internalsemconv.OTelSemConvStabilityOptIn, tc.envValue)

			// Create new config
			cfg := newConfig()

			// Check that SemConvStabilityOptIn is correctly set
			assert.Equal(t, tc.expectedOptIn, cfg.SemConvStabilityOptIn)

			// Check that DBQueryTextAttributes is initialized
			assert.NotNil(t, cfg.DBQueryTextAttributes)

			// Test with a sample query to verify it returns the expected attributes format
			const query = "SELECT * FROM test"

			attrs := cfg.DBQueryTextAttributes(query)

			// Verify format of returned attributes based on opt-in type
			switch tc.expectedOptIn {
			case internalsemconv.OTelSemConvStabilityOptInNone:
				assert.Equal(t, []attribute.KeyValue{
					semconvlegacy.DBStatementKey.String(query),
				}, attrs)
			case internalsemconv.OTelSemConvStabilityOptInDup:
				assert.Equal(t, []attribute.KeyValue{
					semconvlegacy.DBStatementKey.String(query),
					semconv.DBQueryTextKey.String(query),
				}, attrs)
			case internalsemconv.OTelSemConvStabilityOptInStable:
				assert.Equal(t, []attribute.KeyValue{
					semconv.DBQueryTextKey.String(query),
				}, attrs)
			}
		})
	}
}
