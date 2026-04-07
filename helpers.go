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
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
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

	// [scheme://][user[:password]@][protocol([addr])]/dbname[?param1=value1&paramN=valueN]
	// Find the schema part.
	schemaIndex := strings.Index(dsn, "://")
	if schemaIndex != -1 {
		// Remove the schema part from the DSN.
		dsn = dsn[schemaIndex+3:]
	}

	// [user[:password]@][protocol([addr])]/dbname[?param1=value1&paramN=valueN]
	// Find credentials part.
	atIndex := strings.Index(dsn, "@")
	if atIndex != -1 {
		// Remove the credential part from the DSN.
		dsn = dsn[atIndex+1:]
	}

	// [protocol([addr])]/dbname[?param1=value1&paramN=valueN]
	// Find the '/' that separates the address part from the database part.
	pathIndex := strings.Index(dsn, "/")
	if pathIndex != -1 {
		// Remove the path part from the DSN.
		path := dsn[pathIndex+1:]
		// dbname[?param1=value1&paramN=valueN]
		if questionMarkIndex := strings.Index(path, "?"); questionMarkIndex != -1 {
			path = path[:questionMarkIndex]
		}
		// Extract db name
		dbName = path
		// [protocol([addr])] or [addr]
		dsn = dsn[:pathIndex]
	}

	// [protocol([addr])] or [addr]
	// Find the '(' that starts the address part.
	openParen := strings.Index(dsn, "(")
	if openParen != -1 {
		rest := dsn[openParen+1:]
		if closeParen := strings.Index(rest, ")"); closeParen != -1 {
			rest = rest[:closeParen]
		}
		// Remove the protocol part from the DSN.
		dsn = rest
	}

	// [addr]
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
