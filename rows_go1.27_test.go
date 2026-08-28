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

//go:build go1.27

package otelsql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
)

type mockRowsColumnScanner struct {
	*mockRows

	nextRowErr, scanColumnErr     error
	nextRowCount, scanColumnCount int
	scanColumnIndex               int
	scanColumnDest                any
	scanColumnValue               driver.Value
}

var _ driver.RowsColumnScanner = (*mockRowsColumnScanner)(nil)

func (m *mockRowsColumnScanner) NextRow() error {
	m.nextRowCount++

	return m.nextRowErr
}

func (m *mockRowsColumnScanner) Columns() []string {
	return []string{"value"}
}

func (m *mockRowsColumnScanner) ScanColumn(scanCtx driver.ScanContext, index int, dest any) error {
	m.scanColumnCount++
	m.scanColumnIndex = index
	m.scanColumnDest = dest

	if m.scanColumnErr != nil {
		return m.scanColumnErr
	}

	if m.scanColumnValue != nil {
		return sql.ConvertAssign(scanCtx, dest, m.scanColumnValue)
	}

	return nil
}

type mockRowsColumnScannerConn struct {
	*mockConn

	rows driver.Rows
}

var _ driver.QueryerContext = (*mockRowsColumnScannerConn)(nil)

func (m *mockRowsColumnScannerConn) QueryContext(
	context.Context, string, []driver.NamedValue,
) (driver.Rows, error) {
	return m.rows, nil
}

type singleConnConnector struct {
	conn driver.Conn
}

var _ driver.Connector = (*singleConnConnector)(nil)

func (c *singleConnConnector) Connect(context.Context) (driver.Conn, error) {
	return c.conn, nil
}

func (c *singleConnConnector) Driver() driver.Driver {
	return newMockDriver(false)
}

func TestNewRows_RowsColumnScanner(t *testing.T) {
	t.Run("does not expose unsupported interface", func(t *testing.T) {
		rows := newRows(t.Context(), newMockRows(false), newConfig())

		_, ok := rows.(driver.RowsColumnScanner)
		assert.False(t, ok)
	})

	t.Run("forwards row scanning", func(t *testing.T) {
		ctx, sr, tracer, _ := prepareTraces(false)
		dest := new(string)
		mr := &mockRowsColumnScanner{mockRows: newMockRows(false)}

		cfg := newConfig()
		cfg.Tracer = tracer
		cfg.SpanOptions.RowsNext = true

		rows := newRows(ctx, mr, cfg)
		scanner, ok := rows.(driver.RowsColumnScanner)
		require.True(t, ok)

		require.NoError(t, scanner.NextRow())
		require.NoError(t, scanner.ScanColumn(driver.ScanContext{}, 2, dest))

		assert.Equal(t, 1, mr.nextRowCount)
		assert.Equal(t, 0, mr.nextCount)
		assert.Equal(t, 1, mr.scanColumnCount)
		assert.Equal(t, 2, mr.scanColumnIndex)
		assert.Same(t, dest, mr.scanColumnDest)

		spans := sr.Started()
		require.Len(t, spans, 2)
		assert.Len(t, spans[1].Events(), 1)
		assert.Equal(t, codes.Unset, spans[1].Status().Code)
	})

	t.Run("records NextRow errors", func(t *testing.T) {
		ctx, sr, tracer, _ := prepareTraces(false)
		nextRowErr := errors.New("next row")
		mr := &mockRowsColumnScanner{
			mockRows:   newMockRows(false),
			nextRowErr: nextRowErr,
		}

		cfg := newConfig()
		cfg.Tracer = tracer

		rows := newRows(ctx, mr, cfg)
		scanner, ok := rows.(driver.RowsColumnScanner)
		require.True(t, ok)
		require.ErrorIs(t, scanner.NextRow(), nextRowErr)

		spans := sr.Started()
		require.Len(t, spans, 2)
		assert.Equal(t, codes.Error, spans[1].Status().Code)
		assert.Len(t, spans[1].Events(), 1)
	})

	t.Run("does not record NextRow EOF as an error", func(t *testing.T) {
		ctx, sr, tracer, _ := prepareTraces(false)
		mr := &mockRowsColumnScanner{
			mockRows:   newMockRows(false),
			nextRowErr: io.EOF,
		}

		cfg := newConfig()
		cfg.Tracer = tracer
		cfg.SpanOptions.RowsNext = true

		rows := newRows(ctx, mr, cfg)
		scanner, ok := rows.(driver.RowsColumnScanner)
		require.True(t, ok)
		require.ErrorIs(t, scanner.NextRow(), io.EOF)

		spans := sr.Started()
		require.Len(t, spans, 2)
		assert.Equal(t, codes.Unset, spans[1].Status().Code)
		assert.Len(t, spans[1].Events(), 1)
	})
}

func TestRowsColumnScanner_DatabaseSQL(t *testing.T) {
	ctx, sr, tracer, _ := prepareTraces(false)
	mr := &mockRowsColumnScanner{
		mockRows:        newMockRows(false),
		scanColumnValue: "scanned",
	}

	cfg := newConfig()
	cfg.Tracer = tracer
	cfg.SpanOptions.RowsNext = true

	conn := &mockRowsColumnScannerConn{
		mockConn: newMockConn(false),
		rows:     mr,
	}
	db := sql.OpenDB(&singleConnConnector{conn: newConn(conn, cfg)})

	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	rows, err := db.QueryContext(ctx, testQueryString)

	require.NoError(t, err)
	defer func() {
		require.NoError(t, rows.Close())
	}()

	require.True(t, rows.Next())

	var value string
	require.NoError(t, rows.Scan(&value))
	require.NoError(t, rows.Err())

	assert.Equal(t, "scanned", value)
	assert.Equal(t, 1, mr.nextRowCount)
	assert.Equal(t, 0, mr.nextCount)
	assert.Equal(t, 1, mr.scanColumnCount)
	assert.Equal(t, 0, mr.scanColumnIndex)
	assert.Same(t, &value, mr.scanColumnDest)

	spans := sr.Started()
	require.Len(t, spans, 3)
	assert.Len(t, spans[2].Events(), 1)
	assert.Equal(t, codes.Unset, spans[2].Status().Code)
}
