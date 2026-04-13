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
	"net/url"
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
// It handles the format: [scheme://][user[:password]@][protocol([addr])][/path][?param1=value1&paramN=valueN]
// serverAddress and dbName are empty strings if not found. serverPort is -1 if not found.
func parseDSN(dsn string) (serverAddress string, serverPort int64, dbName string) {
	// [scheme://][user[:password]@][protocol([addr])][/path][?param1=value1&paramN=valueN]
	// Find the schema part.
	var scheme string

	schemaIndex := strings.Index(dsn, "://")
	if schemaIndex != -1 {
		scheme = dsn[:schemaIndex]
		// Remove the schema part from the DSN.
		dsn = dsn[schemaIndex+3:]
	}

	// [user[:password]@][protocol([addr])][/path][?param1=value1&paramN=valueN]
	// Find credentials part.
	atIndex := strings.Index(dsn, "@")
	if atIndex != -1 {
		// Remove the credential part from the DSN.
		dsn = dsn[atIndex+1:]
	}

	// [protocol([addr])][/path][?param1=value1&paramN=valueN]
	// Find the '?' that separates the query string.
	var queryString string
	if questionMarkIndex := strings.Index(dsn, "?"); questionMarkIndex != -1 {
		queryString = dsn[questionMarkIndex+1:]
		dsn = dsn[:questionMarkIndex]
	}

	// [protocol([addr])][/path]
	// Find the '/' that separates the address part from the path (database or instance name).
	pathIndex := strings.Index(dsn, "/")

	var path string
	if pathIndex != -1 {
		path = dsn[pathIndex+1:]

		// [protocol([addr])] or [addr]
		dsn = dsn[:pathIndex]
	}

	switch scheme {
	case "sqlserver", "mssql":
		// sqlserver uses the "database" query param; the path is the instance name, not the database.
		if params, err := url.ParseQuery(queryString); err == nil {
			dbName = params.Get("database")
		}
	case "postgresql", "postgres", "mysql", "clickhouse":
		dbName = path
	}

	// [protocol([addr])]
	serverAddress, serverPort = parseHostPort(dsn)

	return serverAddress, serverPort, dbName
}

// parseHostPort extracts the server address and port from a DSN fragment.
// It handles MySQL's protocol(addr) syntax where the address is wrapped in parentheses.
// serverAddress is an empty string if not found; serverPort is -1 if not found.
func parseHostPort(dsn string) (serverAddress string, serverPort int64) {
	serverPort = -1

	// Strip MySQL's protocol(addr) wrapper, e.g. "tcp(host:3306)" → "host:3306".
	if openParen := strings.Index(dsn, "("); openParen != -1 {
		rest := dsn[openParen+1:]
		if closeParen := strings.Index(rest, ")"); closeParen != -1 {
			rest = rest[:closeParen]
		}

		dsn = rest
	}

	if len(dsn) == 0 {
		return
	}

	host, portStr, err := net.SplitHostPort(dsn)
	if err != nil {
		return dsn, serverPort
	}

	serverAddress = host

	if port, err := strconv.ParseInt(portStr, 10, 64); err == nil {
		serverPort = port
	}

	return
}
