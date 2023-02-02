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
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/unit"
)

const (
	namespace = "db.sql"
)

type dbStatsInstruments struct {
	connectionMaxOpen                instrument.Int64ObservableGauge
	connectionOpen                   instrument.Int64ObservableGauge
	connectionWaitTotal              instrument.Int64ObservableCounter
	connectionWaitDurationTotal      instrument.Float64ObservableCounter
	connectionClosedMaxIdleTotal     instrument.Int64ObservableCounter
	connectionClosedMaxIdleTimeTotal instrument.Int64ObservableCounter
	connectionClosedMaxLifetimeTotal instrument.Int64ObservableCounter
}

type instruments struct {
	// The latency of calls in milliseconds
	latency instrument.Float64Histogram
}

func newInstruments(meter metric.Meter) (*instruments, error) {
	var instruments instruments
	var err error

	if instruments.latency, err = meter.Float64Histogram(
		strings.Join([]string{namespace, "latency"}, "."),
		instrument.WithDescription("The latency of calls in milliseconds"),
		instrument.WithUnit(unit.Milliseconds),
	); err != nil {
		return nil, fmt.Errorf("failed to create latency instrument, %v", err)
	}
	return &instruments, nil
}

func newDBStatsInstruments(meter metric.Meter) (*dbStatsInstruments, error) {
	var instruments dbStatsInstruments
	var err error
	subsystem := "connection"

	if instruments.connectionMaxOpen, err = meter.Int64ObservableGauge(
		strings.Join([]string{namespace, subsystem, "max_open"}, "."),
		instrument.WithDescription("Maximum number of open connections to the database"),
	); err != nil {
		return nil, fmt.Errorf("failed to create connectionMaxOpen instrument, %v", err)
	}

	if instruments.connectionOpen, err = meter.Int64ObservableGauge(
		strings.Join([]string{namespace, subsystem, "open"}, "."),
		instrument.WithDescription("The number of established connections both in use and idle"),
	); err != nil {
		return nil, fmt.Errorf("failed to create connectionOpen instrument, %v", err)
	}

	if instruments.connectionWaitTotal, err = meter.Int64ObservableCounter(
		strings.Join([]string{namespace, subsystem, "wait"}, "."),
		instrument.WithDescription("The total number of connections waited for"),
	); err != nil {
		return nil, fmt.Errorf("failed to create connectionWaitTotal instrument, %v", err)
	}

	if instruments.connectionWaitDurationTotal, err = meter.Float64ObservableCounter(
		strings.Join([]string{namespace, subsystem, "wait_duration"}, "."),
		instrument.WithDescription("The total time blocked waiting for a new connection"),
		instrument.WithUnit(unit.Milliseconds),
	); err != nil {
		return nil, fmt.Errorf("failed to create connectionWaitDurationTotal instrument, %v", err)
	}

	if instruments.connectionClosedMaxIdleTotal, err = meter.Int64ObservableCounter(
		strings.Join([]string{namespace, subsystem, "closed_max_idle"}, "."),
		instrument.WithDescription("The total number of connections closed due to SetMaxIdleConns"),
	); err != nil {
		return nil, fmt.Errorf("failed to create connectionClosedMaxIdleTotal instrument, %v", err)
	}

	if instruments.connectionClosedMaxIdleTimeTotal, err = meter.Int64ObservableCounter(
		strings.Join([]string{namespace, subsystem, "closed_max_idle_time"}, "."),
		instrument.WithDescription("The total number of connections closed due to SetConnMaxIdleTime"),
	); err != nil {
		return nil, fmt.Errorf("failed to create connectionClosedMaxIdleTimeTotal instrument, %v", err)
	}

	if instruments.connectionClosedMaxLifetimeTotal, err = meter.Int64ObservableCounter(
		strings.Join([]string{namespace, subsystem, "closed_max_lifetime"}, "."),
		instrument.WithDescription("The total number of connections closed due to SetConnMaxLifetime"),
	); err != nil {
		return nil, fmt.Errorf("failed to create connectionClosedMaxLifetimeTotal instrument, %v", err)
	}

	return &instruments, nil
}
