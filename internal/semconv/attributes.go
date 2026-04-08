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

// Package semconv contains semantic convention definitions and utilities for database attributes
// used by the otelsql package.
package semconv

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// DBQueryTextAttributes returns the database query text attributes
// using the stable db.query.text semantic convention.
func DBQueryTextAttributes(query string) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.DBQueryTextKey.String(query),
	}
}

// ErrorTypeAttributes converts an error to a slice of attribute.KeyValue.
func ErrorTypeAttributes(err error) []attribute.KeyValue {
	if err == nil {
		return nil
	}

	// Handle common driver errors with specific error types
	switch {
	case errors.Is(err, driver.ErrBadConn):
		return []attribute.KeyValue{semconv.ErrorTypeKey.String("database/sql/driver.ErrBadConn")}
	case errors.Is(err, driver.ErrSkip):
		return []attribute.KeyValue{semconv.ErrorTypeKey.String("database/sql/driver.ErrSkip")}
	case errors.Is(err, driver.ErrRemoveArgument):
		return []attribute.KeyValue{semconv.ErrorTypeKey.String("database/sql/driver.ErrRemoveArgument")}
	}

	t := reflect.TypeOf(err)

	var value string

	if t.PkgPath() == "" && t.Name() == "" {
		// Likely a builtin type.
		value = t.String()
	} else {
		value = fmt.Sprintf("%s.%s", t.PkgPath(), t.Name())
	}

	if value == "" {
		return []attribute.KeyValue{semconv.ErrorTypeOther}
	}

	return []attribute.KeyValue{semconv.ErrorTypeKey.String(value)}
}
