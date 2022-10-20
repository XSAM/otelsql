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
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type mockRows struct {
	shouldError bool

	closeCount, nextCount int
	nextDest              []driver.Value
}

func (m *mockRows) Columns() []string {
	return nil
}

func (m *mockRows) Close() error {
	m.closeCount++
	if m.shouldError {
		return errors.New("close")
	}
	return nil
}

func (m *mockRows) Next(dest []driver.Value) error {
	m.nextDest = dest
	m.nextCount++
	if m.shouldError {
		return errors.New("next")
	}
	return nil
}

func newMockRows(shouldError bool) *mockRows {
	return &mockRows{shouldError: shouldError}
}

var _ driver.Rows = (*mockRows)(nil)

func TestOtRows_Close(t *testing.T) {
	testCases := []struct {
		name  string
		error bool
	}{
		{
			name: "no error",
		},
		{
			name:  "with error",
			error: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare traces
			ctx, sr, tracer, _ := prepareTraces(false)

			mr := newMockRows(tc.error)
			cfg := newMockConfig(t, tracer)

			// New rows
			rows := newRows(ctx, mr, cfg)
			// Close
			err := rows.Close()

			spanList := sr.Ended()
			// A span created in newRows()
			require.Equal(t, 2, len(spanList))
			span := spanList[1]
			assert.False(t, span.EndTime().IsZero())

			assert.Equal(t, 1, mr.closeCount)
			if tc.error {
				require.Error(t, err)
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Len(t, span.Events(), 1)
			} else {
				require.NoError(t, err)
				assert.Equal(t, codes.Unset, span.Status().Code)
			}
		})
	}
}

func TestOtRows_Next(t *testing.T) {
	testCases := []struct {
		name           string
		error          bool
		rowsNextOption bool
	}{
		{
			name: "no error",
		},
		{
			name:  "with error",
			error: true,
		},
		{
			name:           "with RowsNextOption",
			rowsNextOption: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare traces
			ctx, sr, tracer, _ := prepareTraces(false)

			mr := newMockRows(tc.error)
			cfg := newMockConfig(t, tracer)
			cfg.SpanOptions.RowsNext = tc.rowsNextOption

			// New rows
			rows := newRows(ctx, mr, cfg)
			// Next
			err := rows.Next([]driver.Value{"test"})

			spanList := sr.Started()
			// A span created in newRows()
			require.Equal(t, 2, len(spanList))
			span := spanList[1]
			assert.True(t, span.EndTime().IsZero())

			assert.Equal(t, 1, mr.nextCount)
			assert.Equal(t, []driver.Value{"test"}, mr.nextDest)
			var expectedEventCount int
			if tc.error {
				require.Error(t, err)
				assert.Equal(t, codes.Error, span.Status().Code)
				expectedEventCount++
			} else {
				require.NoError(t, err)
				assert.Equal(t, codes.Unset, span.Status().Code)
			}

			if tc.rowsNextOption {
				expectedEventCount++
			}
			assert.Len(t, span.Events(), expectedEventCount)
		})
	}
}

func TestNewRows(t *testing.T) {
	for _, omitRows := range []bool{true, false} {
		var testname string
		if omitRows {
			testname = "OmitRows"
		}

		t.Run(testname, func(t *testing.T) {
			testCases := []struct {
				name             string
				noParentSpan     bool
				attributesGetter AttributesGetter
			}{
				{
					name: "default config",
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

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					// Prepare traces
					ctx, sr, tracer, dummySpan := prepareTraces(tc.noParentSpan)

					mr := newMockRows(false)
					cfg := newMockConfig(t, tracer)
					cfg.SpanOptions.OmitRows = omitRows
					cfg.AttributesGetter = tc.attributesGetter

					// New rows
					rows := newRows(ctx, mr, cfg)

					spanList := sr.Started()
					expectedSpanCount := getExpectedSpanCount(tc.noParentSpan, omitRows)
					// One dummy span and one span created in newRows()
					require.Equal(t, expectedSpanCount, len(spanList))

					// Convert []sdktrace.ReadWriteSpan to []sdktrace.ReadOnlySpan explicitly due to the limitation of Go
					var readOnlySpanList []sdktrace.ReadOnlySpan
					for _, v := range spanList {
						readOnlySpanList = append(readOnlySpanList, v)
					}

					assertSpanList(t, readOnlySpanList, spanAssertionParameter{
						parentSpan:         dummySpan,
						error:              false,
						expectedAttributes: cfg.Attributes,
						method:             MethodRows,
						noParentSpan:       tc.noParentSpan,
						spanNotEnded:       true,
						omitSpan:           omitRows,
						attributesGetter:   tc.attributesGetter,
					})

					assert.Equal(t, mr, rows.Rows)
				})
			}
		})
	}
}
