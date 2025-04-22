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
	}

	for _, tc := range testCases {
		t.Run(tc.dsn, func(t *testing.T) {
			got := AttributesFromDSN(tc.dsn)
			assert.Equal(t, tc.expected, got)
		})
	}
}
