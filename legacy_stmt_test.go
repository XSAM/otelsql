package otelsql

import (
	"database/sql/driver"
	"errors"
)

var (
	_ driver.Stmt = (*mockLegacyStmt)(nil)
	_ MockStmt    = (*mockLegacyStmt)(nil)
)

func newMockLegacyStmt(shouldError bool) *mockLegacyStmt {
	return &mockLegacyStmt{shouldError: shouldError}
}

type mockLegacyStmt struct {
	shouldError bool

	execArgs  []driver.Value
	execCount int

	queryArgs  []driver.Value
	queryCount int
}

func (m *mockLegacyStmt) QueryContextCount() int {
	return m.queryCount
}

func (m *mockLegacyStmt) QueryContextArgs() []driver.NamedValue {
	return nil
}

func (m *mockLegacyStmt) QueryArgs() []driver.Value {
	return m.queryArgs
}

func (m *mockLegacyStmt) ExecArgs() []driver.Value {
	return m.execArgs
}

func (m *mockLegacyStmt) ExecContextCount() int {
	return m.execCount
}

func (m *mockLegacyStmt) ExecContextArgs() []driver.NamedValue {
	return nil
}

func (m *mockLegacyStmt) Close() error {
	return nil
}

func (m *mockLegacyStmt) NumInput() int {
	return 0
}

func (m *mockLegacyStmt) Exec(args []driver.Value) (driver.Result, error) {
	m.execArgs = args
	m.execCount++
	if m.shouldError {
		return nil, errors.New("exec")
	}
	return nil, nil
}

func (m *mockLegacyStmt) Query(args []driver.Value) (driver.Rows, error) {
	m.queryArgs = args
	m.queryCount++
	if m.shouldError {
		return nil, errors.New("query")
	}
	return nil, nil
}
