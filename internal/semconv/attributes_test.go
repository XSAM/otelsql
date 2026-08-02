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

package semconv_test

import (
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"github.com/XSAM/otelsql/internal/semconv"
)

func TestDBQueryTextAttributes(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected []attribute.KeyValue
	}{
		{
			name:  "normal query",
			query: "SELECT * FROM users",
			expected: []attribute.KeyValue{
				otelsemconv.DBQueryTextKey.String("SELECT * FROM users"),
			},
		},
		{
			name:  "empty query",
			query: "",
			expected: []attribute.KeyValue{
				otelsemconv.DBQueryTextKey.String(""),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := semconv.DBQueryTextAttributes(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

var _ error = (*customError)(nil)

// customError is a test error type.
type customError struct {
	msg string
}

func (e customError) Error() string {
	return e.msg
}

func TestErrorTypeAttribute(t *testing.T) {
	tests := [...]struct {
		name     string
		err      error
		expected attribute.KeyValue
	}{
		{
			name:     "driver.ErrBadConn",
			err:      driver.ErrBadConn,
			expected: otelsemconv.ErrorTypeKey.String("database/sql/driver.ErrBadConn"),
		},
		{
			name:     "driver.ErrSkip",
			err:      driver.ErrSkip,
			expected: otelsemconv.ErrorTypeKey.String("database/sql/driver.ErrSkip"),
		},
		{
			name:     "driver.ErrRemoveArgument",
			err:      driver.ErrRemoveArgument,
			expected: otelsemconv.ErrorTypeKey.String("database/sql/driver.ErrRemoveArgument"),
		},
		{
			name:     "custom error type",
			err:      customError{msg: "test error"},
			expected: otelsemconv.ErrorTypeKey.String("github.com/XSAM/otelsql/internal/semconv_test.customError"),
		},
		{
			name:     "built-in error",
			err:      errors.New("some error"),
			expected: otelsemconv.ErrorTypeKey.String("*errors.errorString"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := semconv.ErrorTypeAttribute(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
