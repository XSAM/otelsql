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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestNewInstruments(t *testing.T) {
	instruments, err := newInstruments(noop.NewMeterProvider().Meter("test"))
	require.NoError(t, err)

	assert.NotNil(t, instruments)
	assert.NotNil(t, instruments.latency)
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
