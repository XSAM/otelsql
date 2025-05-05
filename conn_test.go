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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	testSpanFilterOmit = "spanFilterOmit"
	testSpanFilterNil  = "spanFilterNil"
	testSpanFilterKeep = "spanFilterKeep"
	testQueryString    = "query"
	testLegacy         = "legacy"
)

type MockConn interface {
	driver.Conn
	PrepareContextCount() int
	PrepareContextCtx() context.Context
	PrepareContextQuery() string

	BeginTxCtx() context.Context
	BeginTxCount() int
}

var (
	_ MockConn    = (*mockConn)(nil)
	_ driver.Conn = (*mockConn)(nil)
)

type mockConn struct {
	shouldError bool

	resetSessionCount int
	resetSessionCtx   context.Context

	beginTxCount int
	beginTxCtx   context.Context

	prepareContextCount int
	prepareContextCtx   context.Context
	prepareContextQuery string

	queryContextCount int
	queryContextCtx   context.Context
	queryContextQuery string

	execContextCount int
	execContextCtx   context.Context
	execContextQuery string

	pingCount int
	pingCtx   context.Context
}

func (m *mockConn) BeginTxCtx() context.Context {
	return m.beginTxCtx
}

func (m *mockConn) BeginTxCount() int {
	return m.beginTxCount
}

func (m *mockConn) PrepareContextCount() int {
	return m.prepareContextCount
}

func (m *mockConn) PrepareContextCtx() context.Context {
	return m.prepareContextCtx
}

func (m *mockConn) PrepareContextQuery() string {
	return m.prepareContextQuery
}

func newMockConn(shouldError bool) *mockConn {
	return &mockConn{shouldError: shouldError}
}

func (m *mockConn) ResetSession(ctx context.Context) error {
	m.resetSessionCtx = ctx
	m.resetSessionCount++
	if m.shouldError {
		return errors.New("resetSession")
	}
	return nil
}

func (m *mockConn) BeginTx(ctx context.Context, _ driver.TxOptions) (driver.Tx, error) {
	m.beginTxCount++
	m.beginTxCtx = ctx
	if m.shouldError {
		return nil, errors.New("beginTx")
	}
	return newMockTx(false), nil
}

func (m *mockConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	m.prepareContextCount++
	m.prepareContextCtx = ctx
	m.prepareContextQuery = query
	if m.shouldError {
		return nil, errors.New("prepareContext")
	}
	return newMockStmt(false), nil
}

func (m *mockConn) QueryContext(
	ctx context.Context, query string, _ []driver.NamedValue,
) (driver.Rows, error) {
	m.queryContextCount++
	m.queryContextCtx = ctx
	m.queryContextQuery = query
	if m.shouldError {
		return nil, errors.New("queryContext")
	}
	return newMockRows(false), nil
}

func (m *mockConn) ExecContext(
	ctx context.Context, query string, _ []driver.NamedValue,
) (driver.Result, error) {
	m.execContextCount++
	m.execContextCtx = ctx
	m.execContextQuery = query
	if m.shouldError {
		return nil, errors.New("execContext")
	}
	return nil, nil
}

func (m *mockConn) Ping(ctx context.Context) error {
	m.pingCount++
	m.pingCtx = ctx
	if m.shouldError {
		return errors.New("execContext")
	}
	return nil
}

func (m *mockConn) Prepare(_ string) (driver.Stmt, error) {
	return newMockStmt(false), nil
}

func (m *mockConn) Close() error {
	return nil
}

func (m *mockConn) Begin() (driver.Tx, error) {
	return newMockTx(false), nil
}

var (
	_ driver.Pinger             = (*mockConn)(nil)
	_ driver.ExecerContext      = (*mockConn)(nil)
	_ driver.QueryerContext     = (*mockConn)(nil)
	_ driver.Conn               = (*mockConn)(nil)
	_ driver.ConnPrepareContext = (*mockConn)(nil)
	_ driver.ConnBeginTx        = (*mockConn)(nil)
	_ driver.SessionResetter    = (*mockConn)(nil)
)

func TestOtConn_Ping(t *testing.T) {
	testCases := []struct {
		name             string
		error            bool
		pingOption       bool
		noParentSpan     bool
		attributesGetter AttributesGetter
	}{
		{
			name:       "ping enabled",
			pingOption: true,
		},
		{
			name:         "ping enabled with no parent span",
			pingOption:   true,
			noParentSpan: true,
		},
		{
			name:       "ping enabled with error",
			pingOption: true,
			error:      true,
		},
		{
			name: "ping disabled",
		},
		{
			name:             "with attribute getter",
			attributesGetter: getDummyAttributesGetter(),
		},
	}

	for _, spanFilterFn := range []SpanFilter{nil, omit, keep} {
		testname := testSpanFilterOmit
		if spanFilterFn == nil {
			testname = testSpanFilterNil
		} else if spanFilterFn(nil, "", "", []driver.NamedValue{}) {
			testname = testSpanFilterKeep
		}

		t.Run(testname, func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					// Prepare traces
					ctx, sr, tracer, dummySpan := prepareTraces(tc.noParentSpan)

					// New conn
					cfg := newMockConfig(t, tracer, nil)
					cfg.SpanOptions.Ping = tc.pingOption
					cfg.AttributesGetter = tc.attributesGetter
					cfg.InstrumentAttributesGetter = InstrumentAttributesGetter(tc.attributesGetter)
					cfg.SpanOptions.SpanFilter = spanFilterFn
					mc := newMockConn(tc.error)
					otelConn := newConn(mc, cfg)

					err := otelConn.Ping(ctx)
					if tc.error {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}

					spanList := sr.Ended()
					if tc.pingOption {
						omit := !filterSpan(ctx, cfg.SpanOptions, MethodConnPing, "", []driver.NamedValue{})
						expectedSpanCount := getExpectedSpanCount(tc.noParentSpan, omit)
						// One dummy span and one span created in Ping
						require.Len(t, spanList, expectedSpanCount)

						if tc.pingOption {
							assertSpanList(t, spanList, spanAssertionParameter{
								parentSpan:         dummySpan,
								error:              tc.error,
								expectedAttributes: cfg.Attributes,
								method:             MethodConnPing,
								noParentSpan:       tc.noParentSpan,
								ctx:                mc.pingCtx,
								attributesGetter:   tc.attributesGetter,
								omitSpan:           omit,
							})

							assert.Equal(t, 1, mc.pingCount)
						}
					} else {
						if !tc.noParentSpan {
							require.Len(t, spanList, 1)
						}
					}
				})
			}
		})
	}
}

func TestOtConn_ExecContext(t *testing.T) {
	query := testQueryString
	args := []driver.NamedValue{{Value: "foo"}}
	expectedAttrs := []attribute.KeyValue{semconv.DBQueryTextKey.String(query)}

	testCases := []struct {
		name             string
		error            bool
		noParentSpan     bool
		disableQuery     bool
		attrs            []attribute.KeyValue
		attributesGetter AttributesGetter
	}{
		{
			name:  "no error",
			attrs: expectedAttrs,
		},
		{
			name:         "no query db.statement",
			disableQuery: true,
		},
		{
			name:  "with error",
			error: true,
			attrs: expectedAttrs,
		},
		{
			name:         "no parent span",
			noParentSpan: true,
			attrs:        expectedAttrs,
		},
		{
			name:             "with attribute getter",
			attributesGetter: getDummyAttributesGetter(),
			attrs:            expectedAttrs,
		},
	}
	for _, spanFilterFn := range []SpanFilter{nil, omit, keep} {
		testname := testSpanFilterOmit
		if spanFilterFn == nil {
			testname = testSpanFilterNil
		} else if spanFilterFn(nil, "", "", []driver.NamedValue{}) {
			testname = testSpanFilterKeep
		}

		t.Run(testname, func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					// Prepare traces
					ctx, sr, tracer, dummySpan := prepareTraces(tc.noParentSpan)

					// New conn
					cfg := newMockConfig(t, tracer, nil)
					cfg.SpanOptions.DisableQuery = tc.disableQuery
					cfg.SpanOptions.SpanFilter = spanFilterFn
					cfg.AttributesGetter = tc.attributesGetter
					cfg.InstrumentAttributesGetter = InstrumentAttributesGetter(tc.attributesGetter)
					mc := newMockConn(tc.error)
					otelConn := newConn(mc, cfg)

					_, err := otelConn.ExecContext(ctx, query, args)
					if tc.error {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}

					spanList := sr.Ended()
					omit := !filterSpan(ctx, cfg.SpanOptions, MethodConnExec, query, args)
					expectedSpanCount := getExpectedSpanCount(tc.noParentSpan, omit)
					// One dummy span and one span created in ExecContext
					require.Len(t, spanList, expectedSpanCount)

					assertSpanList(t, spanList, spanAssertionParameter{
						parentSpan:         dummySpan,
						error:              tc.error,
						expectedAttributes: append(cfg.Attributes, tc.attrs...),
						method:             MethodConnExec,
						omitSpan:           omit,
						noParentSpan:       tc.noParentSpan,
						ctx:                mc.execContextCtx,
						attributesGetter:   tc.attributesGetter,
						query:              query,
						args:               args,
					})

					assert.Equal(t, 1, mc.execContextCount)
					assert.Equal(t, "query", mc.execContextQuery)
				})
			}
		})
	}
}

//nolint:gocognit,cyclop
func TestOtConn_QueryContext(t *testing.T) {
	query := testQueryString
	args := []driver.NamedValue{{Value: "foo"}}
	expectedAttrs := []attribute.KeyValue{semconv.DBQueryTextKey.String(query)}

	for _, omitConnQuery := range []bool{true, false} {
		var testname string
		if omitConnQuery {
			testname = "OmitConnQuery"
		}

		t.Run(testname, func(t *testing.T) {
			for _, spanFilterFn := range []SpanFilter{nil, omit, keep} {
				testname := testSpanFilterOmit
				if spanFilterFn == nil {
					testname = testSpanFilterNil
				} else if spanFilterFn(nil, "", "", []driver.NamedValue{}) {
					testname = testSpanFilterKeep
				}

				t.Run(testname, func(t *testing.T) {
					testCases := []struct {
						name             string
						error            bool
						noParentSpan     bool
						disableQuery     bool
						attrs            []attribute.KeyValue
						attributesGetter AttributesGetter
					}{
						{
							name:  "no error",
							attrs: expectedAttrs,
						},
						{
							name:         "no query db.statement",
							disableQuery: true,
						},
						{
							name:  "with error",
							error: true,
							attrs: expectedAttrs,
						},
						{
							name:         "no parent span",
							noParentSpan: true,
							attrs:        expectedAttrs,
						},
						{
							name:             "with attribute getter",
							attributesGetter: getDummyAttributesGetter(),
							attrs:            expectedAttrs,
						},
					}

					for _, tc := range testCases {
						t.Run(tc.name, func(t *testing.T) {
							// Prepare traces
							ctx, sr, tracer, dummySpan := prepareTraces(tc.noParentSpan)

							// New conn
							cfg := newMockConfig(t, tracer, nil)
							cfg.SpanOptions.DisableQuery = tc.disableQuery
							cfg.SpanOptions.OmitConnQuery = omitConnQuery
							cfg.SpanOptions.SpanFilter = spanFilterFn
							cfg.AttributesGetter = tc.attributesGetter
							cfg.InstrumentAttributesGetter = InstrumentAttributesGetter(tc.attributesGetter)
							mc := newMockConn(tc.error)
							otelConn := newConn(mc, cfg)

							rows, err := otelConn.QueryContext(ctx, query, args)
							if tc.error {
								require.Error(t, err)
							} else {
								require.NoError(t, err)
							}

							spanList := sr.Ended()
							omit := omitConnQuery
							if !omit {
								omit = !filterSpan(ctx, cfg.SpanOptions, MethodConnQuery, query, args)
							}

							expectedSpanCount := getExpectedSpanCount(tc.noParentSpan, omit)
							// One dummy span and one span created in QueryContext
							require.Len(t, spanList, expectedSpanCount)

							assertSpanList(t, spanList, spanAssertionParameter{
								parentSpan:         dummySpan,
								error:              tc.error,
								expectedAttributes: append(cfg.Attributes, tc.attrs...),
								method:             MethodConnQuery,
								noParentSpan:       tc.noParentSpan,
								ctx:                mc.queryContextCtx,
								omitSpan:           omit,
								attributesGetter:   tc.attributesGetter,
								query:              query,
								args:               args,
							})

							assert.Equal(t, 1, mc.queryContextCount)
							assert.Equal(t, "query", mc.queryContextQuery)

							if !tc.error {
								otelRows, ok := rows.(*otRows)
								require.True(t, ok)
								if dummySpan != nil && !omit {
									assert.Equal(
										t,
										dummySpan.SpanContext().TraceID(),
										otelRows.span.SpanContext().TraceID(),
									)

									// Get a span from started span list
									startedSpanList := sr.Started()
									require.Len(t, startedSpanList, expectedSpanCount+1)
									span := startedSpanList[expectedSpanCount]
									// Make sure this span is the same as the span from otelRows
									require.Equal(t, otelRows.span.SpanContext().SpanID(), span.SpanContext().SpanID())

									// The span that creates in newRows() is the child of the dummySpan
									assert.Equal(t, dummySpan.SpanContext().SpanID(), span.Parent().SpanID())
								}
							}
						})
					}
				})
			}
		})
	}
}

//nolint:gocognit,cyclop
func TestOtConn_PrepareContext(t *testing.T) {
	query := testQueryString
	expectedAttrs := []attribute.KeyValue{semconv.DBQueryTextKey.String(query)}

	for _, legacy := range []bool{true, false} {
		var testname string
		if legacy {
			testname = testLegacy
		}

		t.Run(testname, func(t *testing.T) {
			for _, omitConnPrepare := range []bool{true, false} {
				var testname string
				if omitConnPrepare {
					testname = "OmitConnPrepare"
				}

				t.Run(testname, func(t *testing.T) {
					for _, spanFilterFn := range []SpanFilter{nil, omit, keep} {
						testname := testSpanFilterOmit
						if spanFilterFn == nil {
							testname = testSpanFilterNil
						} else if spanFilterFn(nil, "", "", []driver.NamedValue{}) {
							testname = testSpanFilterKeep
						}

						t.Run(testname, func(t *testing.T) {
							testCases := []struct {
								name             string
								error            bool
								noParentSpan     bool
								disableQuery     bool
								attrs            []attribute.KeyValue
								attributesGetter AttributesGetter
							}{
								{
									name:  "no error",
									attrs: expectedAttrs,
								},
								{
									name:         "no query db.statement",
									disableQuery: true,
								},
								{
									name:  "with error",
									error: true,
									attrs: expectedAttrs,
								},
								{
									name:         "no parent span",
									noParentSpan: true,
									attrs:        expectedAttrs,
								},
								{
									name:             "with attribute getter",
									attributesGetter: getDummyAttributesGetter(),
									attrs:            expectedAttrs,
								},
							}

							for _, tc := range testCases {
								t.Run(tc.name, func(t *testing.T) {
									// Prepare traces
									ctx, sr, tracer, dummySpan := prepareTraces(tc.noParentSpan)

									// New conn
									cfg := newMockConfig(t, tracer, nil)
									cfg.SpanOptions.DisableQuery = tc.disableQuery
									cfg.SpanOptions.OmitConnPrepare = omitConnPrepare
									cfg.SpanOptions.SpanFilter = spanFilterFn
									cfg.AttributesGetter = tc.attributesGetter
									cfg.InstrumentAttributesGetter = InstrumentAttributesGetter(tc.attributesGetter)

									var mc MockConn
									if legacy {
										mc = newMockLegacyConn(tc.error)
									} else {
										mc = newMockConn(tc.error)
									}
									otelConn := newConn(mc, cfg)

									stmt, err := otelConn.PrepareContext(ctx, query)
									if tc.error {
										require.Error(t, err)
									} else {
										require.NoError(t, err)
									}

									spanList := sr.Ended()
									omit := omitConnPrepare
									if !omit {
										omit = !filterSpan(
											ctx,
											cfg.SpanOptions,
											MethodConnPrepare,
											query,
											[]driver.NamedValue{},
										)
									}
									expectedSpanCount := getExpectedSpanCount(tc.noParentSpan, omit)
									// One dummy span and one span created in PrepareContext
									require.Len(t, spanList, expectedSpanCount)

									assertSpanList(t, spanList, spanAssertionParameter{
										parentSpan:         dummySpan,
										error:              tc.error,
										expectedAttributes: append(cfg.Attributes, tc.attrs...),
										method:             MethodConnPrepare,
										noParentSpan:       tc.noParentSpan,
										ctx:                mc.PrepareContextCtx(),
										omitSpan:           omit,
										attributesGetter:   tc.attributesGetter,
										query:              query,
									})

									assert.Equal(t, 1, mc.PrepareContextCount())
									assert.Equal(t, "query", mc.PrepareContextQuery())

									if !tc.error {
										otelStmt, ok := stmt.(*otStmt)
										require.True(t, ok)
										assert.Equal(t, "query", otelStmt.query)
									}
								})
							}
						})
					}
				})
			}
		})
	}
}

//nolint:gocognit,cyclop
func TestOtConn_BeginTx(t *testing.T) {
	testCases := []struct {
		name             string
		error            bool
		noParentSpan     bool
		attributesGetter AttributesGetter
	}{
		{
			name: "no error",
		},
		{
			name:  "with error",
			error: true,
		},
		{
			name:         "no parent span",
			noParentSpan: true,
		},
		{
			name:             "with attribute getter",
			attributesGetter: getDummyAttributesGetter(),
		},
	}

	for _, legacy := range []bool{true, false} {
		var testname string
		if legacy {
			testname = testLegacy
		}

		t.Run(testname, func(t *testing.T) {
			for _, spanFilterFn := range []SpanFilter{nil, omit, keep} {
				testname := testSpanFilterOmit
				if spanFilterFn == nil {
					testname = testSpanFilterNil
				} else if spanFilterFn(nil, "", "", []driver.NamedValue{}) {
					testname = testSpanFilterKeep
				}

				t.Run(testname, func(t *testing.T) {
					for _, tc := range testCases {
						t.Run(tc.name, func(t *testing.T) {
							// Prepare traces
							ctx, sr, tracer, dummySpan := prepareTraces(tc.noParentSpan)

							// New conn
							cfg := newMockConfig(t, tracer, nil)
							cfg.SpanOptions.SpanFilter = spanFilterFn
							cfg.AttributesGetter = tc.attributesGetter

							var mc MockConn
							if legacy {
								mc = newMockLegacyConn(tc.error)
							} else {
								mc = newMockConn(tc.error)
							}
							otelConn := newConn(mc, cfg)

							tx, err := otelConn.BeginTx(ctx, driver.TxOptions{})
							if tc.error {
								require.Error(t, err)
							} else {
								require.NoError(t, err)
							}

							spanList := sr.Ended()
							omit := !filterSpan(ctx, cfg.SpanOptions, MethodConnBeginTx, "", []driver.NamedValue{})
							expectedSpanCount := getExpectedSpanCount(tc.noParentSpan, omit)
							// One dummy span and one span created in BeginTx
							require.Len(t, spanList, expectedSpanCount)

							assertSpanList(t, spanList, spanAssertionParameter{
								parentSpan:         dummySpan,
								error:              tc.error,
								expectedAttributes: cfg.Attributes,
								method:             MethodConnBeginTx,
								noParentSpan:       tc.noParentSpan,
								ctx:                mc.BeginTxCtx(),
								attributesGetter:   tc.attributesGetter,
								omitSpan:           omit,
							})

							assert.Equal(t, 1, mc.BeginTxCount())

							if !tc.error {
								otelTx, ok := tx.(*otTx)
								require.True(t, ok)

								if dummySpan != nil {
									assert.Equal(t, dummySpan.SpanContext(), trace.SpanContextFromContext(otelTx.ctx))
								}
							}
						})
					}
				})
			}
		})
	}
}

//nolint:gocognit
func TestOtConn_ResetSession(t *testing.T) {
	testCases := []struct {
		name             string
		error            bool
		noParentSpan     bool
		attributesGetter AttributesGetter
	}{
		{
			name: "no error",
		},
		{
			name:  "with error",
			error: true,
		},
		{
			name:         "no parent span",
			noParentSpan: true,
		},
		{
			name:             "with attribute getter",
			attributesGetter: getDummyAttributesGetter(),
		},
	}

	for _, omitResetSession := range []bool{false, true} {
		var testname string
		if omitResetSession {
			testname = "OmitConnResetSession"
		}
		t.Run(testname, func(t *testing.T) {
			for _, spanFilterFn := range []SpanFilter{nil, omit, keep} {
				testname := testSpanFilterOmit
				if spanFilterFn == nil {
					testname = testSpanFilterNil
				} else if spanFilterFn(nil, "", "", []driver.NamedValue{}) {
					testname = testSpanFilterKeep
				}
				t.Run(testname, func(t *testing.T) {
					for _, tc := range testCases {
						t.Run(tc.name, func(t *testing.T) {
							// Prepare traces
							ctx, sr, tracer, dummySpan := prepareTraces(tc.noParentSpan)

							// New conn
							cfg := newMockConfig(t, tracer, nil)
							cfg.SpanOptions.OmitConnResetSession = omitResetSession
							cfg.SpanOptions.SpanFilter = spanFilterFn
							cfg.AttributesGetter = tc.attributesGetter
							cfg.InstrumentAttributesGetter = InstrumentAttributesGetter(tc.attributesGetter)
							mc := newMockConn(tc.error)
							otelConn := newConn(mc, cfg)

							err := otelConn.ResetSession(ctx)
							if tc.error {
								require.Error(t, err)
							} else {
								require.NoError(t, err)
							}

							spanList := sr.Ended()
							omit := omitResetSession
							if !omit {
								omit = !filterSpan(
									ctx,
									cfg.SpanOptions,
									MethodConnResetSession,
									"",
									[]driver.NamedValue{},
								)
							}
							expectedSpanCount := getExpectedSpanCount(tc.noParentSpan, omit)
							// One dummy span and one span created in ResetSession
							require.Len(t, spanList, expectedSpanCount)

							assertSpanList(t, spanList, spanAssertionParameter{
								parentSpan:         dummySpan,
								error:              tc.error,
								expectedAttributes: cfg.Attributes,
								method:             MethodConnResetSession,
								noParentSpan:       tc.noParentSpan,
								ctx:                mc.resetSessionCtx,
								omitSpan:           omit,
								attributesGetter:   tc.attributesGetter,
							})

							assert.Equal(t, 1, mc.resetSessionCount)
						})
					}
				})
			}
		})
	}
}

func TestOtConn_Raw(t *testing.T) {
	raw := newMockConn(false)
	conn := newConn(raw, config{})

	assert.Equal(t, raw, conn.Raw())
}
