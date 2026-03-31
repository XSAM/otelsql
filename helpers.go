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
// It always sets [semconv.DBSystemNameKey], falling back to [semconv.DBSystemNameOtherSQL] when
// the scheme is missing or unrecognized. It makes the best effort to retrieve values for
// [semconv.ServerAddressKey], [semconv.ServerPortKey], and [semconv.DBNamespaceKey].
func AttributesFromDSN(dsn string) []attribute.KeyValue {
	scheme, serverAddress, serverPort, dbName := parseDSN(dsn)

	var attrs []attribute.KeyValue

	attrs = append(attrs, dbSystemFromScheme(scheme))

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

// dbSystemByScheme maps lowercase DSN schemes to their [semconv.DBSystemNameKey] attribute.
var dbSystemByScheme = map[string]attribute.KeyValue{
	"mysql":            semconv.DBSystemNameMySQL,
	"postgres":         semconv.DBSystemNamePostgreSQL,
	"postgresql":       semconv.DBSystemNamePostgreSQL,
	"sqlserver":        semconv.DBSystemNameMicrosoftSQLServer,
	"mssql":            semconv.DBSystemNameMicrosoftSQLServer,
	"oracle":           semconv.DBSystemNameOracleDB,
	"oracle+cx_oracle": semconv.DBSystemNameOracleDB,
	"sqlite":           semconv.DBSystemNameSqlite,
	"sqlite3":          semconv.DBSystemNameSqlite,
	"mariadb":          semconv.DBSystemNameMariaDB,
	"cockroachdb":      semconv.DBSystemNameCockroachdb,
	"cockroach":        semconv.DBSystemNameCockroachdb,
	"cassandra":        semconv.DBSystemNameCassandra,
	"redis":            semconv.DBSystemNameRedis,
	"rediss":           semconv.DBSystemNameRedis,
	"mongodb":          semconv.DBSystemNameMongoDB,
	"mongodb+srv":      semconv.DBSystemNameMongoDB,
	"clickhouse":       semconv.DBSystemNameClickhouse,
	"trino":            semconv.DBSystemNameTrino,
	"hive":             semconv.DBSystemNameHive,
	"spanner":          semconv.DBSystemNameGCPSpanner,
	"elasticsearch":    semconv.DBSystemNameElasticsearch,
	"couchbase":        semconv.DBSystemNameCouchbase,
	"influxdb":         semconv.DBSystemNameInfluxdb,
	"dynamodb":         semconv.DBSystemNameAWSDynamoDB,
	"redshift":         semconv.DBSystemNameAWSRedshift,
	"teradata":         semconv.DBSystemNameTeradata,
	"firebird":         semconv.DBSystemNameFirebirdsql,
	"firebirdsql":      semconv.DBSystemNameFirebirdsql,
	"hbase":            semconv.DBSystemNameHBase,
}

// dbSystemFromScheme maps a DSN scheme to the corresponding [semconv.DBSystemNameKey] attribute.
// It returns [semconv.DBSystemNameOtherSQL] if the scheme is not recognized or missing.
func dbSystemFromScheme(scheme string) attribute.KeyValue {
	if v, ok := dbSystemByScheme[strings.ToLower(scheme)]; ok {
		return v
	}

	return semconv.DBSystemNameOtherSQL
}

// parseDSN parses a DSN string and returns the scheme, server address, server port, and database name.
// It handles the format: [scheme://][user[:password]@][protocol([addr])]/dbname[?params]
// scheme, serverAddress and dbName are empty strings if not found. serverPort is -1 if not found.
func parseDSN(dsn string) (scheme, serverAddress string, serverPort int64, dbName string) {
	serverPort = -1

	// Extract scheme
	if i := strings.Index(dsn, "://"); i != -1 {
		scheme = dsn[:i]
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
		dsn = dsn[i+1 : len(dsn)-1]
	}

	if len(dsn) == 0 {
		return scheme, serverAddress, serverPort, dbName
	}

	// Extract host and port
	host, portStr, err := net.SplitHostPort(dsn)
	if err != nil {
		serverAddress = dsn
		return scheme, serverAddress, serverPort, dbName
	}

	serverAddress = host

	if port, err := strconv.ParseInt(portStr, 10, 64); err == nil {
		serverPort = port
	}

	return scheme, serverAddress, serverPort, dbName
}
