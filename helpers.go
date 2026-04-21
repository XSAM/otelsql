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
// It makes the best effort to retrieve values for [semconv.ServerAddressKey] and [semconv.ServerPortKey].
func AttributesFromDSN(dsn string) []attribute.KeyValue {
	addr := addrFromDSN(dsn)
	if addr == "" {
		return nil
	}

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	attrs := make([]attribute.KeyValue, 0, 2)
	if host != "" {
		attrs = append(attrs, semconv.ServerAddress(host))
	}

	if portStr != "" {
		if port, err := strconv.ParseInt(portStr, 10, 64); err == nil {
			attrs = append(attrs, semconv.ServerPortKey.Int64(port))
		}
	}

	return attrs
}

// addrFromDSN extracts the network address (host[:port] or unix-socket path)
// from a DSN string, stripping any scheme, credentials, protocol wrapper,
// and trailing dbname/query components.
func addrFromDSN(dsn string) string {
	// [scheme://][user[:password]@][protocol([addr])]/dbname[?param1=value1&paramN=valueN]
	// Strip scheme.
	if i := strings.Index(dsn, "://"); i != -1 {
		dsn = dsn[i+3:]
	}

	// Strip credentials.
	if i := strings.Index(dsn, "@"); i != -1 {
		dsn = dsn[i+1:]
	}

	// If the DSN uses the protocol(addr) form, extract addr from between
	// the parens first. Splitting on '/' up front would break on addresses
	// like unix(/tmp/mysql.sock), which contain a '/' inside the parens
	// and used to trigger an out-of-range slice panic (#624).
	openParen := strings.Index(dsn, "(")

	closeParen := strings.Index(dsn, ")")
	if openParen != -1 && closeParen > openParen {
		return dsn[openParen+1 : closeParen]
	}

	// Bare address form: addr/db?params. Trim the path suffix.
	if i := strings.Index(dsn, "/"); i != -1 {
		return dsn[:i]
	}

	return dsn
}
