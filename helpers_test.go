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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

//nolint:gosec
func TestAttributesFromDSN(t *testing.T) {
	testCases := []struct {
		dsn      string
		expected []attribute.KeyValue
	}{
		{
			dsn: "mysql://root:otel_password@tcp(example.com)/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("example.com"),
			},
		},
		{
			dsn: "mysql://root:otel_password@tcp(example.com:3307)/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("example.com"),
				semconv.ServerPort(3307),
			},
		},
		{
			dsn: "mysql://root:otel_password@tcp([2001:db8:1234:5678:9abc:def0:0001]:3307)/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("2001:db8:1234:5678:9abc:def0:0001"),
				semconv.ServerPort(3307),
			},
		},
		{
			dsn: "mysql://root:otel_password@tcp(2001:db8:1234:5678:9abc:def0:0001)/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("2001:db8:1234:5678:9abc:def0:0001"),
			},
		},
		{
			dsn: "sqlserver://user:pass@dbhost:1433?database=db",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("dbhost"),
				semconv.ServerPort(1433),
			},
		},
		{
			dsn: "root:secret@tcp(mysql)/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("mysql"),
			},
		},
		{
			dsn: "root:secret@tcp(mysql:3307)/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("mysql"),
				semconv.ServerPort(3307),
			},
		},
		{
			dsn:      "root:secret@/db?parseTime=true",
			expected: nil,
		},
		{
			dsn: "example.com/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("example.com"),
			},
		},
		{
			dsn: "example.com:3307/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("example.com"),
				semconv.ServerPort(3307),
			},
		},
		{
			dsn: "example.com:3307",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("example.com"),
				semconv.ServerPort(3307),
			},
		},
		{
			dsn: "example.com:",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("example.com"),
			},
		},
		{
			dsn: "example.com",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("example.com"),
			},
		},
		{
			dsn: "postgres://root:secret@0.0.0.0:42/db?param1=value1&paramN=valueN",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("0.0.0.0"),
				semconv.ServerPort(42),
			},
		},
		{
			dsn: "postgres://root:secret@2001:db8:1234:5678:9abc:def0:0001/db?param1=value1&paramN=valueN",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("2001:db8:1234:5678:9abc:def0:0001"),
			},
		},
		{
			dsn: "postgres://root:secret@[2001:db8:1234:5678:9abc:def0:0001]:42/db?param1=value1&paramN=valueN",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("2001:db8:1234:5678:9abc:def0:0001"),
				semconv.ServerPort(42),
			},
		},
		{
			dsn: "root:secret@0.0.0.0:42/db?param1=value1&paramN=valueN",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("0.0.0.0"),
				semconv.ServerPort(42),
			},
		},
		{
			// In this case, "tcp" will be considered as the server address.
			dsn: "root:secret@tcp/db?param1=value1&paramN=valueN",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("tcp"),
			},
		},
		{
			// Missing closing protocol parenthesis and no path/queryString shouldn't cause a panic
			dsn: "mysql://root:otel_password@tcp(example.com",
			expected: []attribute.KeyValue{
				semconv.ServerAddress("example.com"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.dsn, func(t *testing.T) {
			got := AttributesFromDSN(tc.dsn)
			assert.Equal(t, tc.expected, got)
		})
	}
}

//nolint:gosec
func TestDBNamespaceFromDSN(t *testing.T) {
	testCases := []struct {
		dsn      string
		expected attribute.KeyValue
		wantErr  bool
	}{
		// Standard URL-style DSNs
		{dsn: "mysql://root:pass@example.com/db", expected: semconv.DBNamespace("db")},
		{dsn: "mysql://root:pass@tcp(example.com)/db?parseTime=true", expected: semconv.DBNamespace("db")},
		{dsn: "postgres://root:secret@0.0.0.0:42/db?param1=value1", expected: semconv.DBNamespace("db")},
		{dsn: "unknown://user:pass@dbhost/db", wantErr: true}, // unknown scheme: db.namespace not extracted

		// No scheme: db.namespace not extracted
		{dsn: "root:secret@/db?parseTime=true", wantErr: true},
		{dsn: "example.com/db", wantErr: true},
		{dsn: "root:secret@tcp(mysql)/db?parseTime=true", wantErr: true},

		// Empty or missing db name
		{dsn: "example.com:3307", wantErr: true},
		{dsn: "postgres://user:pass@dbhost/", wantErr: true},
		{dsn: "sqlserver://user:pass@dbhost", wantErr: true},

		// sqlserver: database from query param
		{dsn: "sqlserver://user:pass@dbhost:1433?database=db", expected: semconv.DBNamespace("db")},
		{dsn: "sqlserver://user:pass@dbhost/SQLEXPRESS?database=db", expected: semconv.DBNamespace("db")},
		{dsn: "sqlserver://dbhost:1433?database=db", expected: semconv.DBNamespace("db")},

		// sqlserver: no database query param
		{dsn: "sqlserver://user:pass@dbhost/SQLEXPRESS", wantErr: true},
		{dsn: "sqlserver://user:pass@dbhost:1433", wantErr: true},

		// oracle: db namespace not supported
		{dsn: "oracle://user:pass@dbhost:1521/service", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.dsn, func(t *testing.T) {
			got, err := DBNamespaceFromDSN(tc.dsn)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, got)
			}
		})
	}
}

//nolint:gosec
func TestAttributesAndDBNamespaceFromDSN(t *testing.T) {
	testCases := []struct {
		dsn          string
		dbSystemName attribute.KeyValue
		expected     []attribute.KeyValue
	}{
		// Standard URL-style DSNs
		{
			dsn:          "mysql://root:pass@example.com/db",
			dbSystemName: semconv.DBSystemNameMySQL,
			expected: []attribute.KeyValue{
				semconv.ServerAddress("example.com"),
				semconv.DBNamespace("db"),
				semconv.DBSystemNameMySQL,
			}},
		// Empty or missing db name
		{
			dsn:          "mysql://root:pass@example.com",
			dbSystemName: semconv.DBSystemNameMySQL,
			expected: []attribute.KeyValue{
				semconv.ServerAddress("example.com"),
				semconv.DBSystemNameMySQL,
			}},
		//
		// sqlserver: database from query param
		{
			dsn:          "sqlserver://user:pass@dbhost:1433?database=db",
			dbSystemName: semconv.DBSystemNameMicrosoftSQLServer,

			expected: []attribute.KeyValue{
				semconv.ServerAddress("dbhost"),
				semconv.ServerPort(1433),
				semconv.DBNamespace("db"),
				semconv.DBSystemNameMicrosoftSQLServer,
			},
		},
		// sqlserver: missing db name
		{
			dsn:          "sqlserver://user:pass@dbhost/SQLEXPRESS",
			dbSystemName: semconv.DBSystemNameMicrosoftSQLServer,

			expected: []attribute.KeyValue{
				semconv.ServerAddress("dbhost"),
				semconv.DBSystemNameMicrosoftSQLServer,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.dsn, func(t *testing.T) {
			got := AttributesFromDSN(tc.dsn)
			if dbNamespace, err := DBNamespaceFromDSN(tc.dsn); err == nil {
				got = append(got, dbNamespace)
			}

			got = append(got, tc.dbSystemName)
			assert.Equal(t, tc.expected, got)
		})
	}
}
