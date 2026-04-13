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
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// AttributesFromDSN returns attributes extracted from a DSN string.
// It makes the best effort to retrieve values for [semconv.ServerAddressKey] and [semconv.ServerPortKey].
func AttributesFromDSN(dsn string) []attribute.KeyValue {
	// [scheme://][user[:password]@][protocol([addr])][/path][?param1=value1&paramN=valueN]
	// Find the schema part.
	schemaIndex := strings.Index(dsn, "://")
	if schemaIndex != -1 {
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
	if questionMarkIndex := strings.Index(dsn, "?"); questionMarkIndex != -1 {
		// Remove queryString part from the DSN
		dsn = dsn[:questionMarkIndex]
	}

	// [protocol([addr])]/path
	// Find the '/' that separates the address part from the database part.
	pathIndex := strings.Index(dsn, "/")
	if pathIndex != -1 {
		// Remove the path part from the DSN.
		dsn = dsn[:pathIndex]
	}

	// [protocol([addr])] or [addr]
	host, port := parseHostPort(dsn)

	var attrs []attribute.KeyValue
	if host != "" {
		attrs = append(attrs, semconv.ServerAddress(host))
	}
	if port != -1 {
		attrs = append(attrs, semconv.ServerPortKey.Int64(port))
	}
	return attrs
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

// errDBNamespaceNotFound is returned by [DBNamespaceFromDSN] when no database name can be extracted from the DSN.
var errDBNamespaceNotFound = errors.New("db namespace not found in DSN")

// DBNamespaceFromDSN extracts the OpenTelemetry db.namespace resource attribute from a DSN string and returns it as
// an [semconv.DBNamespaceKey] attribute.
// It handles the format: [scheme://][user[:password]@][protocol([addr])][/path][?param1=value1&paramN=valueN]
// Returns [errDBNamespaceNotFound] if the database name is not found.
func DBNamespaceFromDSN(dsn string) (attribute.KeyValue, error) {
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
	if atIndex := strings.Index(dsn, "@"); atIndex != -1 {
		// Remove the credential part from the DSN.
		dsn = dsn[atIndex+1:]
	}

	// [protocol([addr])][/path][?param1=value1&paramN=valueN]
	// Find the '?' that separates the query string.
	var queryString string
	if questionMarkIndex := strings.Index(dsn, "?"); questionMarkIndex != -1 {
		queryString = dsn[questionMarkIndex+1:]
		// Remove queryString part from the DSN
		dsn = dsn[:questionMarkIndex]
	}

	// [protocol([addr])][/path]
	// Find the '/' that separates the address part from the path (database or instance name).
	pathIndex := strings.Index(dsn, "/")

	var path string
	if pathIndex != -1 {
		path = dsn[pathIndex+1:]
	}

	switch scheme {
	case "sqlserver", "mssql":
		if params, err := url.ParseQuery(queryString); err == nil {
			if db := params.Get("database"); db != "" {
				return semconv.DBNamespace(db), nil
			}
		}
	case "postgresql", "postgres", "mysql", "clickhouse":
		if path != "" {
			return semconv.DBNamespace(path), nil
		}
	}

	return attribute.KeyValue{}, errDBNamespaceNotFound
}
