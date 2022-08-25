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
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

type mockTx struct {
	shouldError bool

	commitCount   int
	rollbackCount int
}

func newMockTx(shouldError bool) *mockTx {
	return &mockTx{shouldError: shouldError}
}

func (m *mockTx) Commit() error {
	m.commitCount++
	if m.shouldError {
		return errors.New("commit")
	}
	return nil
}

func (m *mockTx) Rollback() error {
	m.rollbackCount++
	if m.shouldError {
		return errors.New("rollback")
	}
	return nil
}

var _ driver.Tx = (*mockTx)(nil)

var defaultattribute = attribute.Key("test").String("foo")

func TestOtTx_Commit(t *testing.T) {
	testCases := []struct {
		name         string
		error        bool
		noParentSpan bool
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare traces
			ctx, cfg, sr, dummySpan := prepareTraces(t, tc.noParentSpan)
			mt := newMockTx(tc.error)

			// New tx
			tx := newTx(ctx, mt, cfg)
			// Commit
			err := tx.Commit()
			if tc.error {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			spanList := sr.Ended()
			expectedSpanCount := getExpectedSpanCount(tc.noParentSpan, false)
			// One dummy span and one span created in tx
			require.Equal(t, expectedSpanCount, len(spanList))

			assertSpanList(t, spanList, spanAssertionParameter{
				parentSpan:         dummySpan,
				error:              tc.error,
				expectedAttributes: cfg.Attributes,
				expectedMethod:     MethodTxCommit,
				noParentSpan:       tc.noParentSpan,
			})

			assert.Equal(t, 1, mt.commitCount)
		})
	}
}

func TestOtTx_Rollback(t *testing.T) {
	testCases := []struct {
		name         string
		error        bool
		noParentSpan bool
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare traces
			ctx, cfg, sr, dummySpan := prepareTraces(t, tc.noParentSpan)
			mt := newMockTx(tc.error)

			// New tx
			tx := newTx(ctx, mt, cfg)

			// Rollback
			err := tx.Rollback()
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
				expectedAttributes: cfg.Attributes,
				expectedMethod:     MethodTxRollback,
				noParentSpan:       tc.noParentSpan,
			})

			assert.Equal(t, 1, mt.rollbackCount)
		})
	}
}
