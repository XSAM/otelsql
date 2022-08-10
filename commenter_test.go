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
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestCommenter_WithComment(t *testing.T) {
	query := "foo"

	traceID, err := trace.TraceIDFromHex("a3d3b88cf7994e554c1afbdceec1620b")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("683ec6a9a3a265fb")
	require.NoError(t, err)
	traceState, err := trace.ParseTraceState("rojo=00f067aa0ba902b7,congo=t61rcWkgMzE")
	require.NoError(t, err)
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: 0x1,
		TraceState: traceState,
	}))

	m1, err := baggage.NewMember("foo", "bar")
	require.NoError(t, err)
	b, err := baggage.New(m1)
	require.NoError(t, err)
	ctx = baggage.ContextWithBaggage(ctx, b)

	testCases := []struct {
		name     string
		enabled  bool
		ctx      context.Context
		expected string
	}{
		{
			name:     "empty context",
			enabled:  true,
			ctx:      context.Background(),
			expected: query,
		},
		{
			name:     "context with disable",
			enabled:  false,
			ctx:      ctx,
			expected: query,
		},
		{
			name:     "context",
			enabled:  true,
			ctx:      ctx,
			expected: query + " /*tracestate='rojo%3D00f067aa0ba902b7%2Ccongo%3Dt61rcWkgMzE',traceparent='00-a3d3b88cf7994e554c1afbdceec1620b-683ec6a9a3a265fb-01',baggage='foo%3Dbar'*/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := newCommenter(tc.enabled)
			c.propagator = propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})

			result := c.withComment(tc.ctx, query)
			assert.Equal(t, tc.expected, result)
		})
	}
}
