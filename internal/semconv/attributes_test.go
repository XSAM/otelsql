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

package semconv

import (
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	semconvlegacy "go.opentelemetry.io/otel/semconv/v1.24.0"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

func TestNewDBQueryTextAttributes(t *testing.T) {
	const query = "SELECT * FROM users"

	tests := []struct {
		name      string
		optInType OTelSemConvStabilityOptInType
		expected  []attribute.KeyValue
	}{
		{
			name:      "none",
			optInType: OTelSemConvStabilityOptInNone,
			expected: []attribute.KeyValue{
				semconvlegacy.DBStatementKey.String(query),
			},
		},
		{
			name:      "dup",
			optInType: OTelSemConvStabilityOptInDup,
			expected: []attribute.KeyValue{
				semconvlegacy.DBStatementKey.String(query),
				semconv.DBQueryTextKey.String(query),
			},
		},
		{
			name:      "stable",
			optInType: OTelSemConvStabilityOptInStable,
			expected: []attribute.KeyValue{
				semconv.DBQueryTextKey.String(query),
			},
		},
		{
			name:      "unknown",
			optInType: OTelSemConvStabilityOptInType(999), // An undefined type
			expected: []attribute.KeyValue{
				semconvlegacy.DBStatementKey.String(query),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the function for the specified opt-in type
			fn := NewDBQueryTextAttributes(tt.optInType)

			// Call the function with the test query
			result := fn(query)

			// Verify the result matches what we expect
			assert.Equal(t, tt.expected, result)
		})
	}
}

// customError is a test error type.
type customError struct {
	msg string
}

func (e customError) Error() string {
	return e.msg
}

func TestErrorTypeAttributes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected []attribute.KeyValue
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: nil,
		},
		{
			name:     "driver.ErrBadConn",
			err:      driver.ErrBadConn,
			expected: []attribute.KeyValue{semconv.ErrorTypeKey.String("database/sql/driver.ErrBadConn")},
		},
		{
			name:     "driver.ErrSkip",
			err:      driver.ErrSkip,
			expected: []attribute.KeyValue{semconv.ErrorTypeKey.String("database/sql/driver.ErrSkip")},
		},
		{
			name:     "driver.ErrRemoveArgument",
			err:      driver.ErrRemoveArgument,
			expected: []attribute.KeyValue{semconv.ErrorTypeKey.String("database/sql/driver.ErrRemoveArgument")},
		},
		{
			name: "custom error type",
			err:  customError{msg: "test error"},
			expected: []attribute.KeyValue{
				semconv.ErrorTypeKey.String("github.com/XSAM/otelsql/internal/semconv.customError"),
			},
		},
		{
			name:     "built-in error",
			err:      errors.New("some error"),
			expected: []attribute.KeyValue{semconv.ErrorTypeKey.String("*errors.errorString")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ErrorTypeAttributes(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
