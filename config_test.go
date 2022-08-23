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
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

func TestNewConfig(t *testing.T) {
	cfg := newConfig(
		WithSpanOptions(SpanOptions{Ping: true}),
		WithAttributes(semconv.DBSystemMySQL),
		WithArgumentsAttributes(ArgumentsOptions{EnableAttributes: true}))
	assert.Equal(t, cfg.Tracer(), otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(Version()),
	))
	assert.Equal(t, config{
		TracerProvider: otel.GetTracerProvider(),
		MeterProvider:  global.MeterProvider(),
		Meter: global.MeterProvider().Meter(
			instrumentationName,
			metric.WithInstrumentationVersion(Version()),
		),
		// No need to check values of instruments in this part.
		Instruments:      cfg.Instruments,
		SpanOptions:      SpanOptions{Ping: true},
		ArgumentsOptions: ArgumentsOptions{EnableAttributes: true},
		Attributes: []attribute.KeyValue{
			semconv.DBSystemMySQL,
		},
		SpanNameFormatter: &defaultSpanNameFormatter{},
		SQLCommenter:      newCommenter(false),
	}, cfg)

	assert.Equal(t, cfg.Tracer(), otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(Version()),
	))
	assert.NotNil(t, cfg.Instruments)
}
