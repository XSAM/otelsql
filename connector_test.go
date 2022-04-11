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
)

type mockConnector struct {
	driver driver.Driver

	shouldError    bool
	connectContext context.Context
	connectCount   int
}

func newMockConnector(driver driver.Driver, shouldError bool) *mockConnector {
	return &mockConnector{driver: driver, shouldError: shouldError}
}

func (m *mockConnector) Connect(ctx context.Context) (driver.Conn, error) {
	m.connectContext = ctx
	m.connectCount++
	if m.shouldError {
		return nil, errors.New("connect")
	}
	return newMockConn(false), nil
}

func (m *mockConnector) Driver() driver.Driver {
	return m.driver
}

var _ driver.Connector = (*mockConnector)(nil)

func TestNewConnector(t *testing.T) {
	mConnector := newMockConnector(nil, false)
	otelDriver := &otDriver{}

	connector := newConnector(mConnector, otelDriver)

	assert.Equal(t, mConnector, connector.Connector)
	assert.Equal(t, otelDriver, connector.otDriver)
}

func TestOtConnector_Connect(t *testing.T) {
	testCases := []struct {
		name            string
		error           bool
		allowRootOption bool
		noParentSpan    bool
	}{
		{
			name: "no error",
		},
		{
			name:  "with error",
			error: true,
		},
		{
			name:         "no parent span, disallow root span",
			noParentSpan: true,
		},
		{
			name:            "no parent span, allow root span",
			noParentSpan:    true,
			allowRootOption: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare traces
			ctx, sr, tracer, dummySpan := prepareTraces(tc.noParentSpan)

			cfg := newMockConfig(t, tracer)
			mConnector := newMockConnector(nil, tc.error)
			cfg.SpanOptions.AllowRoot = tc.allowRootOption
			connector := newConnector(mConnector, &otDriver{cfg: cfg})
			conn, err := connector.Connect(ctx)
			if tc.error {
				assert.Error(t, err)
			} else {
				otelConn, ok := conn.(*otConn)
				require.True(t, ok)
				assert.IsType(t, &mockConn{}, otelConn.Conn)
			}

			spanList := sr.Ended()
			expectedSpanCount := getExpectedSpanCount(tc.allowRootOption, tc.noParentSpan, false)
			// One dummy span and one span created in Connect
			require.Equal(t, expectedSpanCount, len(spanList))

			assertSpanList(t, spanList, spanAssertionParameter{
				parentSpan:         dummySpan,
				error:              tc.error,
				expectedAttributes: cfg.Attributes,
				expectedMethod:     MethodConnectorConnect,
				allowRootOption:    tc.allowRootOption,
				noParentSpan:       tc.noParentSpan,
				ctx:                mConnector.connectContext,
			})

			assert.Equal(t, 1, mConnector.connectCount)
		})
	}
}

func TestOtConnector_Driver(t *testing.T) {
	otelDriver := &otDriver{}
	connector := newConnector(nil, otelDriver)

	assert.Equal(t, otelDriver, connector.Driver())
}
