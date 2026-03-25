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
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

func TestAttributesFromDSN(t *testing.T) {
	testCases := []struct {
		dsn      string
		expected []attribute.KeyValue
	}{
		{
			dsn: "mysql://root:otel_password@example.com/db",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameMySQL,
				semconv.ServerAddress("example.com"),
				semconv.DBNamespace("db"),
			},
		},
		{
			dsn: "mysql://root:otel_password@tcp(example.com)/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameMySQL,
				semconv.ServerAddress("example.com"),
				semconv.DBNamespace("db"),
			},
		},
		{
			dsn: "mysql://root:otel_password@tcp(example.com:3307)/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameMySQL,
				semconv.ServerAddress("example.com"),
				semconv.ServerPort(3307),
				semconv.DBNamespace("db"),
			},
		},
		{
			dsn: "mysql://root:otel_password@tcp([2001:db8:1234:5678:9abc:def0:0001]:3307)/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameMySQL,
				semconv.ServerAddress("2001:db8:1234:5678:9abc:def0:0001"),
				semconv.ServerPort(3307),
				semconv.DBNamespace("db"),
			},
		},
		{
			dsn: "mysql://root:otel_password@tcp(2001:db8:1234:5678:9abc:def0:0001)/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameMySQL,
				semconv.ServerAddress("2001:db8:1234:5678:9abc:def0:0001"),
				semconv.DBNamespace("db"),
			},
		},
		{
			dsn: "root:secret@tcp(mysql)/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameOtherSQL,
				semconv.ServerAddress("mysql"),
				semconv.DBNamespace("db"),
			},
		},
		{
			dsn: "root:secret@tcp(mysql:3307)/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameOtherSQL,
				semconv.ServerAddress("mysql"),
				semconv.ServerPort(3307),
				semconv.DBNamespace("db"),
			},
		},
		{
			dsn: "root:secret@/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameOtherSQL,
				semconv.DBNamespace("db"),
			},
		},
		{
			dsn: "example.com/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameOtherSQL,
				semconv.ServerAddress("example.com"),
				semconv.DBNamespace("db"),
			},
		},
		{
			dsn: "example.com:3307/db?parseTime=true",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameOtherSQL,
				semconv.ServerAddress("example.com"),
				semconv.ServerPort(3307),
				semconv.DBNamespace("db"),
			},
		},
		{
			dsn: "example.com:3307",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameOtherSQL,
				semconv.ServerAddress("example.com"),
				semconv.ServerPort(3307),
			},
		},
		{
			dsn: "example.com:",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameOtherSQL,
				semconv.ServerAddress("example.com"),
			},
		},
		{
			dsn: "example.com",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameOtherSQL,
				semconv.ServerAddress("example.com"),
			},
		},
		{
			dsn: "example.com/db",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameOtherSQL,
				semconv.ServerAddress("example.com"),
				semconv.DBNamespace("db"),
			},
		},
		{
			dsn: "postgres://root:secret@0.0.0.0:42/db?param1=value1&paramN=valueN",
			expected: []attribute.KeyValue{
				semconv.DBSystemNamePostgreSQL,
				semconv.ServerAddress("0.0.0.0"),
				semconv.ServerPort(42),
				semconv.DBNamespace("db"),
			},
		},
		{
			dsn: "postgres://root:secret@2001:db8:1234:5678:9abc:def0:0001/db?param1=value1&paramN=valueN",
			expected: []attribute.KeyValue{
				semconv.DBSystemNamePostgreSQL,
				semconv.ServerAddress("2001:db8:1234:5678:9abc:def0:0001"),
				semconv.DBNamespace("db"),
			},
		},
		{
			dsn: "postgres://root:secret@[2001:db8:1234:5678:9abc:def0:0001]:42/db?param1=value1&paramN=valueN",
			expected: []attribute.KeyValue{
				semconv.DBSystemNamePostgreSQL,
				semconv.ServerAddress("2001:db8:1234:5678:9abc:def0:0001"),
				semconv.ServerPort(42),
				semconv.DBNamespace("db"),
			},
		},
		{
			dsn: "root:secret@0.0.0.0:42/db?param1=value1&paramN=valueN",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameOtherSQL,
				semconv.ServerAddress("0.0.0.0"),
				semconv.ServerPort(42),
				semconv.DBNamespace("db"),
			},
		},
		{
			// In this case, "tcp" will be considered as the server address.
			dsn: "root:secret@tcp/db?param1=value1&paramN=valueN",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameOtherSQL,
				semconv.ServerAddress("tcp"),
				semconv.DBNamespace("db"),
			},
		},
		{
			// DSN lacking a db-name
			dsn: "sqlserver://user:pass@dbhost:1433",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameMicrosoftSQLServer,
				semconv.ServerAddress("dbhost"),
				semconv.ServerPort(1433),
			},
		},
		{
			// DSN lacking a db-name, with trailing '/'
			dsn: "postgres://user:pass@dbhost/",
			expected: []attribute.KeyValue{
				semconv.DBSystemNamePostgreSQL,
				semconv.ServerAddress("dbhost"),
			},
		},
		{
			// Unrecognized scheme falls back to OtherSQL
			dsn: "unknown://user:pass@dbhost/db",
			expected: []attribute.KeyValue{
				semconv.DBSystemNameOtherSQL,
				semconv.ServerAddress("dbhost"),
				semconv.DBNamespace("db"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.dsn, func(t *testing.T) {
			gotAttrs := AttributesFromDSN(tc.dsn)
			assert.Equal(t, tc.expected, gotAttrs)
		})
	}
}
