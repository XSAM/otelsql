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
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

// exemplarTestDriverName is a mock driver of its own, so these tests do not
// change the state of the shared test driver.
const exemplarTestDriverName = "test-driver-exemplar"

func init() {
	sql.Register(exemplarTestDriverName, newMockDriver(false))
}

// The duration of a call is recorded in the context of the span created for
// that call, so the exemplar of db.client.operation.duration links to the
// database client span, not to its parent.
func TestDurationExemplarLinksToMethodSpan(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithExemplarFilter(exemplar.AlwaysOnFilter),
	)

	db, err := Open(exemplarTestDriverName, "", WithTracerProvider(tp), WithMeterProvider(mp),
		WithSpanOptions(SpanOptions{Ping: true, RowsChildOfQuery: true}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	ctx, parent := tp.Tracer("test").Start(context.Background(), "parent")

	require.NoError(t, db.PingContext(ctx))

	_, err = db.ExecContext(ctx, "exec")
	require.NoError(t, err)

	queryRows(ctx, t, db)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	tx, err = db.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	parent.End()

	checked := assertExemplarsLinkToMethodSpans(t, sr, reader, parent)

	for _, method := range []Method{
		MethodConnectorConnect, MethodConnPing, MethodConnExec, MethodConnQuery, MethodRows,
		MethodConnBeginTx, MethodTxCommit, MethodTxRollback,
	} {
		assert.True(t, checked[string(method)], "no duration recorded for %s", method)
	}
}

func TestDurationExemplarLinksToMethodSpan_Conn(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithExemplarFilter(exemplar.AlwaysOnFilter),
	)

	conn := newConn(newMockConn(false), newConfig(WithTracerProvider(tp), WithMeterProvider(mp)))

	ctx, parent := tp.Tracer("test").Start(context.Background(), "parent")

	_, err := conn.PrepareContext(ctx, "prepared")
	require.NoError(t, err)

	require.NoError(t, conn.ResetSession(ctx))

	parent.End()

	checked := assertExemplarsLinkToMethodSpans(t, sr, reader, parent)

	for _, method := range []Method{MethodConnPrepare, MethodConnResetSession} {
		assert.True(t, checked[string(method)], "no duration recorded for %s", method)
	}
}

func TestDurationExemplarLinksToMethodSpan_Stmt(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithExemplarFilter(exemplar.AlwaysOnFilter),
	)

	cfg := newConfig(WithTracerProvider(tp), WithMeterProvider(mp))
	stmt := newStmt(newMockStmt(false), cfg, "prepared", nil)

	ctx, parent := tp.Tracer("test").Start(context.Background(), "parent")

	_, err := stmt.ExecContext(ctx, nil)
	require.NoError(t, err)

	_, err = stmt.QueryContext(ctx, nil)
	require.NoError(t, err)

	parent.End()

	checked := assertExemplarsLinkToMethodSpans(t, sr, reader, parent)

	for _, method := range []Method{MethodStmtExec, MethodStmtQuery} {
		assert.True(t, checked[string(method)], "no duration recorded for %s", method)
	}
}

// queryRows runs a query and closes its rows.
func queryRows(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	rows, err := db.QueryContext(ctx, "query")
	require.NoError(t, err)

	defer func() { require.NoError(t, rows.Close()) }()

	require.NoError(t, rows.Err())
}

// assertExemplarsLinkToMethodSpans checks that every exemplar of
// db.client.operation.duration links to the span of its own method, and
// returns the methods it saw.
func assertExemplarsLinkToMethodSpans(
	t *testing.T, sr *tracetest.SpanRecorder, reader sdkmetric.Reader, parent trace.Span,
) map[string]bool {
	t.Helper()

	// Every span of a method, by span ID.
	methodOf := map[trace.SpanID]string{}
	for _, s := range sr.Ended() {
		methodOf[s.SpanContext().SpanID()] = s.Name()
	}

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	checked := map[string]bool{}

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "db.client.operation.duration" {
				continue
			}

			hist, ok := m.Data.(metricdata.Histogram[float64])
			require.True(t, ok)

			for _, dp := range hist.DataPoints {
				op, ok := dp.Attributes.Value(semconv.DBOperationNameKey)
				require.True(t, ok)
				require.NotEmpty(t, dp.Exemplars, "no exemplar for %s", op.AsString())

				for _, ex := range dp.Exemplars {
					var id trace.SpanID
					copy(id[:], ex.SpanID)
					assert.NotEqual(t, parent.SpanContext().SpanID(), id,
						"%s exemplar links to the parent span", op.AsString())
					assert.Equal(t, op.AsString(), methodOf[id],
						"%s exemplar does not link to its own span", op.AsString())
				}

				checked[op.AsString()] = true
			}
		}
	}

	return checked
}

// When no span is created for a call, the duration is still recorded, in the
// caller's context.
func TestDurationExemplarWithoutMethodSpan(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithExemplarFilter(exemplar.AlwaysOnFilter),
	)

	db, err := Open(exemplarTestDriverName, "", WithTracerProvider(tp), WithMeterProvider(mp),
		WithSpanOptions(SpanOptions{SpanFilter: func(context.Context, Method, string, []driver.NamedValue) bool {
			return false
		}}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	ctx, parent := tp.Tracer("test").Start(context.Background(), "parent")
	_, err = db.ExecContext(ctx, "exec")
	require.NoError(t, err)
	parent.End()

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	found := false

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "db.client.operation.duration" {
				continue
			}

			hist, ok := m.Data.(metricdata.Histogram[float64])
			require.True(t, ok)

			for _, dp := range hist.DataPoints {
				op, _ := dp.Attributes.Value(semconv.DBOperationNameKey)
				if op.AsString() != string(MethodConnExec) {
					continue
				}

				for _, ex := range dp.Exemplars {
					var id trace.SpanID
					copy(id[:], ex.SpanID)
					assert.Equal(t, parent.SpanContext().SpanID(), id)

					found = true
				}
			}
		}
	}

	assert.True(t, found, "no exemplar for the exec duration")
}

// dropSQLSpans samples the parent span and drops every database span.
type dropSQLSpans struct{}

func (dropSQLSpans) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	if p.Name == "parent" {
		return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample}
	}

	return sdktrace.SamplingResult{Decision: sdktrace.Drop}
}

func (dropSQLSpans) Description() string { return "dropSQLSpans" }

// When the span of a call is not sampled, the duration keeps the caller's
// context, so the default trace-based filter still records an exemplar,
// linked to the sampled parent.
func TestDurationExemplarUnsampledMethodSpan(t *testing.T) {
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(dropSQLSpans{}))
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)) // default TraceBased filter

	conn := newConn(newMockConn(false), newConfig(WithTracerProvider(tp), WithMeterProvider(mp)))

	ctx, parent := tp.Tracer("test").Start(context.Background(), "parent")
	_, err := conn.ExecContext(ctx, "exec", nil)
	require.NoError(t, err)
	parent.End()

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	found := false

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "db.client.operation.duration" {
				continue
			}

			hist, ok := m.Data.(metricdata.Histogram[float64])
			require.True(t, ok)

			for _, dp := range hist.DataPoints {
				for _, ex := range dp.Exemplars {
					var id trace.SpanID
					copy(id[:], ex.SpanID)
					assert.Equal(t, parent.SpanContext().SpanID(), id)

					found = true
				}
			}
		}
	}

	assert.True(t, found, "no exemplar for the exec duration")
}

// InstrumentAttributesGetter receives the caller's context, as
// AttributesGetter does, also when a span is created for the call.
func TestInstrumentAttributesGetterReceivesCallerContext(t *testing.T) {
	tp := sdktrace.NewTracerProvider()

	var instrumentGetterSpan, spanGetterSpan trace.SpanID

	cfg := newConfig(
		WithTracerProvider(tp),
		WithInstrumentAttributesGetter(
			func(ctx context.Context, _ Method, _ string, _ []driver.NamedValue) []attribute.KeyValue {
				instrumentGetterSpan = trace.SpanContextFromContext(ctx).SpanID()

				return nil
			}),
		WithAttributesGetter(
			func(ctx context.Context, _ Method, _ string, _ []driver.NamedValue) []attribute.KeyValue {
				spanGetterSpan = trace.SpanContextFromContext(ctx).SpanID()

				return nil
			}),
	)
	conn := newConn(newMockConn(false), cfg)

	ctx, parent := tp.Tracer("test").Start(context.Background(), "parent")
	_, err := conn.ExecContext(ctx, "exec", nil)
	require.NoError(t, err)
	parent.End()

	assert.Equal(t, parent.SpanContext().SpanID(), instrumentGetterSpan)
	assert.Equal(t, parent.SpanContext().SpanID(), spanGetterSpan)
}
