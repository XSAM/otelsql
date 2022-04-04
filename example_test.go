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

package otelsql_test

import (
	"database/sql"
	"database/sql/driver"

	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"

	"github.com/XSAM/otelsql"
)

func init() {
	sql.Register("mysql", otelsql.NewMockDriver())
}

var (
	connector = driver.Connector(nil)
	dri       = otelsql.NewMockDriver()
	mysqlDSN  = "root:otel_password@db"
)

func ExampleOpen() {
	// Connect to database
	db, err := otelsql.Open("mysql", mysqlDSN, semconv.DBSystemMySQL.Value.AsString())
	if err != nil {
		panic(err)
	}
	defer db.Close()
	// Output:
}

func ExampleOpenDB() {
	// Connect to database
	db := otelsql.OpenDB(connector, semconv.DBSystemMySQL.Value.AsString())
	defer db.Close()
}

func ExampleWrapDriver() {
	otDriver := otelsql.WrapDriver(dri, semconv.DBSystemMySQL.Value.AsString())

	connector, err := otDriver.(driver.DriverContext).OpenConnector(mysqlDSN)
	if err != nil {
		panic(err)
	}

	// Connect to database
	db := sql.OpenDB(connector)
	defer db.Close()
	// Output:
}

func ExampleRegister() {
	// Register an OTel driver
	driverName, err := otelsql.Register("mysql", semconv.DBSystemMySQL.Value.AsString())
	if err != nil {
		panic(err)
	}

	// Connect to database
	db, err := otelsql.Open(driverName, mysqlDSN, semconv.DBSystemMySQL.Value.AsString())
	if err != nil {
		panic(err)
	}
	defer db.Close()
	// Output:
}
