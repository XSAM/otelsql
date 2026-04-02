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
	"net"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

// AttributesFromDSN returns attributes extracted from a DSN string.
// It makes the best effort to retrieve values for
// [semconv.ServerAddressKey], [semconv.ServerPortKey], and [semconv.DBNamespaceKey].
func AttributesFromDSN(dsn string) []attribute.KeyValue {
	serverAddress, serverPort, dbName := parseDSN(dsn)

	var attrs []attribute.KeyValue

	if serverAddress != "" {
		attrs = append(attrs, semconv.ServerAddress(serverAddress))
	}

	if serverPort != -1 {
		attrs = append(attrs, semconv.ServerPortKey.Int64(serverPort))
	}

	if dbName != "" {
		attrs = append(attrs, semconv.DBNamespace(dbName))
	}

	return attrs
}

// parseDSN parses a DSN string and returns the server address, server port, and database name.
// It handles the format: [scheme://][user[:password]@][protocol([addr])]/dbname[?params]
// serverAddress and dbName are empty strings if not found. serverPort is -1 if not found.
func parseDSN(dsn string) (serverAddress string, serverPort int64, dbName string) {
	serverPort = -1

	// Extract scheme
	if i := strings.Index(dsn, "://"); i != -1 {
		dsn = dsn[i+3:]
	}

	// Skip credentials
	if i := strings.Index(dsn, "@"); i != -1 {
		dsn = dsn[i+1:]
	}

	// Extract db name
	if i := strings.Index(dsn, "/"); i != -1 {
		path := dsn[i+1:]
		if j := strings.Index(path, "?"); j != -1 {
			path = path[:j]
		}

		dbName = path
		dsn = dsn[:i]
	}

	// Skip protocol
	if i := strings.Index(dsn, "("); i != -1 {
		rest := dsn[i+1:]
		if j := strings.Index(rest, ")"); j != -1 {
			rest = rest[:j]
		}
		dsn = rest
	}

	if len(dsn) == 0 {
		return serverAddress, serverPort, dbName
	}

	// Extract host and port
	host, portStr, err := net.SplitHostPort(dsn)
	if err != nil {
		serverAddress = dsn
		return serverAddress, serverPort, dbName
	}

	serverAddress = host

	if port, err := strconv.ParseInt(portStr, 10, 64); err == nil {
		serverPort = port
	}

	return serverAddress, serverPort, dbName
}
