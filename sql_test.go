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
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func init() {
	sql.Register("test-driver", newMockDriver(false))
	maxDriverSlot = 1
}

func TestRegister(t *testing.T) {
	driverName, err := Register("test-driver", "test-db",
		WithAttributes(attribute.String("foo", "bar")),
	)
	require.NoError(t, err)
	assert.Equal(t, "test-driver-otelsql-0", driverName)

	// Expected driver
	db, err := sql.Open(driverName, "")
	require.NoError(t, err)
	otelDriver, ok := db.Driver().(*otDriver)
	require.True(t, ok)
	assert.Equal(t, &mockDriver{openConnectorCount: 2}, otelDriver.driver)
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.DBSystemKey.String("test-db"),
		attribute.String("foo", "bar"),
	}, otelDriver.cfg.Attributes)

	// Exceed max slot count
	_, err = Register("test-driver", "test-db")
	assert.Error(t, err)
}

func TestWrapDriver(t *testing.T) {
	driver := WrapDriver(newMockDriver(false), "test-db",
		WithAttributes(attribute.String("foo", "bar")),
	)

	// Expected driver
	otelDriver, ok := driver.(*otDriver)
	require.True(t, ok)
	assert.Equal(t, &mockDriver{}, otelDriver.driver)
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.DBSystemKey.String("test-db"),
		attribute.String("foo", "bar"),
	}, otelDriver.cfg.Attributes)
}
