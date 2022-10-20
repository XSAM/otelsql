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
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

type mockStmt struct {
	driver.Stmt

	shouldError bool
	queryCount  int
	execCount   int

	queryContextArgs []driver.NamedValue
	ExecContextArgs  []driver.NamedValue
}

func newMockStmt(shouldError bool) *mockStmt {
	return &mockStmt{shouldError: shouldError}
}

func (m *mockStmt) CheckNamedValue(_ *driver.NamedValue) error {
	if m.shouldError {
		return errors.New("checkNamedValue")
	}
	return nil
}

func (m *mockStmt) QueryContext(_ context.Context, args []driver.NamedValue) (driver.Rows, error) {
	m.queryContextArgs = args
	m.queryCount++
	if m.shouldError {
		return nil, errors.New("queryContext")
	}
	return nil, nil
}

func (m *mockStmt) ExecContext(_ context.Context, args []driver.NamedValue) (driver.Result, error) {
	m.ExecContextArgs = args
	m.execCount++
	if m.shouldError {
		return nil, errors.New("execContext")
	}
	return nil, nil
}

var (
	_ driver.Stmt              = (*mockStmt)(nil)
	_ driver.StmtExecContext   = (*mockStmt)(nil)
	_ driver.StmtQueryContext  = (*mockStmt)(nil)
	_ driver.NamedValueChecker = (*mockStmt)(nil)
)

func TestOtStmt_ExecContext(t *testing.T) {
	query := "query"
	args := []driver.NamedValue{{Value: "foo"}}
	expectedAttrs := []attribute.KeyValue{semconv.DBStatementKey.String(query)}
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
			ms := newMockStmt(tc.error)

			// New stmt
			cfg := newMockConfig(t, tracer)
			cfg.SpanOptions.DisableQuery = tc.disableQuery
			cfg.AttributesGetter = tc.attributesGetter
			stmt := newStmt(ms, cfg, query)
			// Exec
			_, err := stmt.ExecContext(ctx, args)
			if tc.error {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			spanList := sr.Ended()
			expectedSpanCount := getExpectedSpanCount(tc.noParentSpan, false)
			// One dummy span and a span created in tx
			require.Equal(t, expectedSpanCount, len(spanList))

			assertSpanList(t, spanList, spanAssertionParameter{
				parentSpan:         dummySpan,
				error:              tc.error,
				expectedAttributes: append(cfg.Attributes, tc.attrs...),
				method:             MethodStmtExec,
				noParentSpan:       tc.noParentSpan,
				attributesGetter:   tc.attributesGetter,
				query:              query,
				args:               args,
			})

			assert.Equal(t, 1, ms.execCount)
			assert.Equal(t, []driver.NamedValue{{Value: "foo"}}, ms.ExecContextArgs)
		})
	}
}

func TestOtStmt_QueryContext(t *testing.T) {
	query := "query"
	args := []driver.NamedValue{{Value: "foo"}}
	expectedAttrs := []attribute.KeyValue{semconv.DBStatementKey.String(query)}
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
			ms := newMockStmt(tc.error)

			// New stmt
			cfg := newMockConfig(t, tracer)
			cfg.SpanOptions.DisableQuery = tc.disableQuery
			cfg.AttributesGetter = tc.attributesGetter
			stmt := newStmt(ms, cfg, query)
			// Query
			rows, err := stmt.QueryContext(ctx, args)
			if tc.error {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			spanList := sr.Ended()
			expectedSpanCount := getExpectedSpanCount(tc.noParentSpan, false)
			// One dummy span and a span created in tx
			require.Equal(t, expectedSpanCount, len(spanList))

			assertSpanList(t, spanList, spanAssertionParameter{
				parentSpan:         dummySpan,
				error:              tc.error,
				expectedAttributes: append(cfg.Attributes, tc.attrs...),
				method:             MethodStmtQuery,
				noParentSpan:       tc.noParentSpan,
				attributesGetter:   tc.attributesGetter,
				query:              query,
				args:               args,
			})

			assert.Equal(t, 1, ms.queryCount)
			assert.Equal(t, []driver.NamedValue{{Value: "foo"}}, ms.queryContextArgs)
			if !tc.error {
				assert.IsType(t, &otRows{}, rows)
			}
		})
	}
}
