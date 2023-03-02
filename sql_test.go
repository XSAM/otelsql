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
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

var driverName string

const (
	testDriverName               = "test-driver"
	testDriverWithoutContextName = "test-driver-without-context"
)

func init() {
	sql.Register(testDriverName, newMockDriver(false))
	sql.Register(testDriverWithoutContextName, struct{ driver.Driver }{newMockDriver(false)})
	maxDriverSlot = 1

	var err error
	driverName, err = Register(testDriverName,
		WithAttributes(attribute.String("foo", "bar")),
	)
	if err != nil {
		panic(err)
	}
	if driverName != "test-driver-otelsql-0" {
		panic(fmt.Sprintf("expect driver name: test-driver-otelsql-0, got %s", driverName))
	}
}

func TestRegister(t *testing.T) {
	// Expected driver
	db, err := sql.Open(driverName, "")
	require.NoError(t, err)
	otelDriver, ok := db.Driver().(*otDriver)
	require.True(t, ok)
	assert.Equal(t, &mockDriver{openConnectorCount: 2}, otelDriver.driver)
	assert.ElementsMatch(t, []attribute.KeyValue{
		attribute.String("foo", "bar"),
	}, otelDriver.cfg.Attributes)

	// Exceed max slot count
	_, err = Register(testDriverName)
	assert.Error(t, err)
}

func TestWrapDriver(t *testing.T) {
	driver := WrapDriver(newMockDriver(false),
		WithAttributes(attribute.String("foo", "bar")),
	)

	// Expected driver
	otelDriver, ok := driver.(*otDriver)
	require.True(t, ok)
	assert.IsType(t, &mockDriver{}, otelDriver.driver)
	assert.ElementsMatch(t, []attribute.KeyValue{
		attribute.String("foo", "bar"),
	}, otelDriver.cfg.Attributes)
}

func TestOpen(t *testing.T) {
	testCases := []struct {
		driverName         string
		expectedDriverType interface{}
	}{
		{
			driverName:         testDriverName,
			expectedDriverType: &mockDriver{},
		},
		{
			driverName:         testDriverWithoutContextName,
			expectedDriverType: struct{ driver.Driver }{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.driverName, func(t *testing.T) {
			db, err := Open(tc.driverName, "",
				WithAttributes(attribute.String("foo", "bar")),
			)
			t.Cleanup(func() {
				assert.NoError(t, db.Close())
			})
			require.NoError(t, err)
			require.NotNil(t, db)

			_, err = db.Conn(context.Background())
			require.NoError(t, err)

			// Expected driver
			otelDriver, ok := db.Driver().(*otDriver)
			require.True(t, ok)
			assert.IsType(t, tc.expectedDriverType, otelDriver.driver)
			assert.ElementsMatch(t, []attribute.KeyValue{
				attribute.String("foo", "bar"),
			}, otelDriver.cfg.Attributes)
		})
	}
}

func TestOpenDB(t *testing.T) {
	connector, err := newMockDriver(false).OpenConnector("")
	require.NoError(t, err)

	db := OpenDB(connector, WithAttributes(attribute.String("foo", "bar")))
	require.NotNil(t, db)

	_, err = db.Conn(context.Background())
	require.NoError(t, err)

	otelDriver, ok := db.Driver().(*otDriver)
	require.True(t, ok)
	assert.IsType(t, &mockDriver{}, otelDriver.driver)
	assert.ElementsMatch(t, []attribute.KeyValue{
		attribute.String("foo", "bar"),
	}, otelDriver.cfg.Attributes)
}

func TestRegisterDBStatsMetrics(t *testing.T) {
	db, err := sql.Open(driverName, "")
	require.NoError(t, err)

	r := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))

	err = RegisterDBStatsMetrics(db, WithMeterProvider(mp))
	assert.NoError(t, err)

	// Should collect 7 metrics
	got := &metricdata.ResourceMetrics{}
	err = r.Collect(context.Background(), got)
	require.NoError(t, err)
	assert.Len(t, got.ScopeMetrics, 1)
	assert.Len(t, got.ScopeMetrics[0].Metrics, 7)
}
