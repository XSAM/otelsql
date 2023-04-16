package otelsql

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockLegacyConn struct {
	shouldError bool

	prepareCount int
	prepareCtx   context.Context
	prepareQuery string

	beginCount int
	beginCtx   context.Context
}

func (m *mockLegacyConn) BeginTxCtx() context.Context {
	return m.beginCtx
}

func (m *mockLegacyConn) BeginTxCount() int {
	return m.beginCount
}

func (m *mockLegacyConn) PrepareContextCount() int {
	return m.prepareCount
}

func (m *mockLegacyConn) PrepareContextCtx() context.Context {
	return m.prepareCtx
}

func (m *mockLegacyConn) PrepareContextQuery() string {
	return m.prepareQuery
}

var _ MockConn = (*mockLegacyConn)(nil)
var _ driver.Conn = (*mockLegacyConn)(nil)

func (m *mockLegacyConn) Prepare(query string) (driver.Stmt, error) {
	m.prepareCount++
	m.prepareQuery = query
	if m.shouldError {
		return nil, errors.New("prepare")
	}
	return newMockStmt(false), nil
}

func (m *mockLegacyConn) Close() error {
	return nil
}

func (m *mockLegacyConn) Begin() (driver.Tx, error) {
	m.beginCount++
	if m.shouldError {
		return nil, errors.New("begin")
	}
	return newMockTx(false), nil
}

var _ driver.Conn = (*mockLegacyConn)(nil)

func newMockLegacyConn(shouldError bool) *mockLegacyConn {
	return &mockLegacyConn{shouldError: shouldError}
}

func TestOtConn_PingWithLegacyConn(t *testing.T) {
	otelConn := newConn(newMockLegacyConn(false), config{})
	err := otelConn.Ping(context.Background())
	assert.Nil(t, err)
}

func TestOtConn_ResetSessionWithLegacyConn(t *testing.T) {
	otelConn := newConn(newMockLegacyConn(false), config{})
	err := otelConn.ResetSession(context.Background())
	assert.Nil(t, err)
}
