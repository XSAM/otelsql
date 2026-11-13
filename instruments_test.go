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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestNewInstruments(t *testing.T) {
	instruments, err := newInstruments(noop.NewMeterProvider().Meter("test"))
	require.NoError(t, err)

	assert.NotNil(t, instruments)
	assert.NotNil(t, instruments.duration)
}

func TestNewInstrumentsDurationBucketBoundaries(t *testing.T) {
	r := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))

	instruments, err := newInstruments(mp.Meter("test"))
	require.NoError(t, err)

	instruments.duration.RecordSet(context.Background(), 0.001, *attribute.EmptySet())

	got := &metricdata.ResourceMetrics{}
	err = r.Collect(context.Background(), got)
	require.NoError(t, err)
	require.Len(t, got.ScopeMetrics, 1)
	require.Len(t, got.ScopeMetrics[0].Metrics, 1)

	histogram, ok := got.ScopeMetrics[0].Metrics[0].Data.(metricdata.Histogram[float64])
	require.True(t, ok)
	require.Len(t, histogram.DataPoints, 1)

	assert.Equal(t, []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10},
		histogram.DataPoints[0].Bounds)
}

func TestNewDBStatsInstruments(t *testing.T) {
	instruments, err := newDBStatsInstruments(noop.NewMeterProvider().Meter("test"))
	require.NoError(t, err)

	assert.NotNil(t, instruments)
	assert.NotNil(t, instruments.connectionMaxOpen)
	assert.NotNil(t, instruments.connectionOpen)
	assert.NotNil(t, instruments.connectionWaitTotal)
	assert.NotNil(t, instruments.connectionWaitDurationTotal)
	assert.NotNil(t, instruments.connectionClosedMaxIdleTotal)
	assert.NotNil(t, instruments.connectionClosedMaxIdleTimeTotal)
	assert.NotNil(t, instruments.connectionClosedMaxLifetimeTotal)
}
