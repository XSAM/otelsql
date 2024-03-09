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
	"errors"
	"strconv"
	"sync"

	"go.opentelemetry.io/otel/metric"
)

var registerLock sync.Mutex

var maxDriverSlot = 1000

// Register инициализирует и регистрирует OTel обёрнутый database driver
// идентефициронный с помощью driverName,используя дополнительные настройки Option.
// Возможно заргестрировать multiple wrappers для одинаковых database driver если
// вы нуждаетесь в разных дополнительных настройках Option.
func Register(driverName string, options ...Option) (string, error) {
	// Извлечение реализации driver для последующей обёртки с помощью инстрементария.
	db, err := sql.Open(driverName, "")
	if err != nil {
		return "", err
	}
	dri := db.Driver()
	if err = db.Close(); err != nil {
		return "", err
	}

	registerLock.Lock()
	defer registerLock.Unlock()

	// Если вы завели multiple OTel drivers с разными 
	// конфигурациями, но имеем одинаковые database driver, мы
	// проходимся циклом, чтобы найти доступный driver.
	driverName = driverName + "-otelsql-"
	for i := 0; i < maxDriverSlot; i++ {
		var (
			found   = false
			regName = driverName + strconv.FormatInt(int64(i), 10)
		)
		for _, name := range sql.Drivers() {
			if name == regName {
				found = true
			}
		}
		if !found {
			sql.Register(regName, newDriver(dri, newConfig(options...)))
			return regName, nil
		}
	}
	return "", errors.New("unable to register driver, all slots have been taken")
}

// WrapDriver принимает SQL driver и обороачивает с помощью инструментария OTel.
func WrapDriver(dri driver.Driver, options ...Option) driver.Driver {
	return newDriver(dri, newConfig(options...))
}

// Open это обёртка над sql.Open, реализованная с помощью инструментария OTel.
func Open(driverName, dataSourceName string, options ...Option) (*sql.DB, error) {
  // Извлечение реализации driver для последующей обёртки с помощью инстрементария.
	db, err := sql.Open(driverName, "")
	if err != nil {
		return nil, err
	}
	d := db.Driver()
	if err = db.Close(); err != nil {
		return nil, err
	}

	otDriver := newOtDriver(d, newConfig(options...))

	if _, ok := d.(driver.DriverContext); ok {
		connector, err := otDriver.OpenConnector(dataSourceName)
		if err != nil {
			return nil, err
		}
		return sql.OpenDB(connector), nil
	}

	return sql.OpenDB(dsnConnector{dsn: dataSourceName, driver: otDriver}), nil
}

// OpenDB это обёртка над sql.OpenDB, реализованная с помощью инструментария OTel.
func OpenDB(c driver.Connector, options ...Option) *sql.DB {
	d := newOtDriver(c.Driver(), newConfig(options...))
	connector := newConnector(c, d)

	return sql.OpenDB(connector)
}

// RegisterDBStatsMetrics регистрирует sql.DBStats metrics с помощью инструментария OTel.
func RegisterDBStatsMetrics(db *sql.DB, opts ...Option) error {
	cfg := newConfig(opts...)
	meter := cfg.Meter

	instruments, err := newDBStatsInstruments(meter)
	if err != nil {
		return err
	}

	_, err = meter.RegisterCallback(func(_ context.Context, observer metric.Observer) error {
		dbStats := db.Stats()

		recordDBStatsMetrics(dbStats, instruments, cfg, observer)
		return nil
	}, instruments.connectionMaxOpen,
		instruments.connectionOpen,
		instruments.connectionWaitTotal,
		instruments.connectionWaitDurationTotal,
		instruments.connectionClosedMaxIdleTotal,
		instruments.connectionClosedMaxIdleTimeTotal,
		instruments.connectionClosedMaxLifetimeTotal)
	if err != nil {
		return err
	}
	return nil
}

func recordDBStatsMetrics(
	dbStats sql.DBStats, instruments *dbStatsInstruments, cfg config, observer metric.Observer,
) {
	observer.ObserveInt64(instruments.connectionMaxOpen,
		int64(dbStats.MaxOpenConnections),
		metric.WithAttributes(cfg.Attributes...),
	)

	observer.ObserveInt64(instruments.connectionOpen,
		int64(dbStats.InUse),
		metric.WithAttributes(append(cfg.Attributes, connectionStatusKey.String("inuse"))...),
	)
	observer.ObserveInt64(instruments.connectionOpen,
		int64(dbStats.Idle),
		metric.WithAttributes(append(cfg.Attributes, connectionStatusKey.String("idle"))...),
	)

	observer.ObserveInt64(instruments.connectionWaitTotal,
		dbStats.WaitCount,
		metric.WithAttributes(cfg.Attributes...),
	)
	observer.ObserveFloat64(instruments.connectionWaitDurationTotal,
		float64(dbStats.WaitDuration.Nanoseconds())/1e6,
		metric.WithAttributes(cfg.Attributes...),
	)
	observer.ObserveInt64(instruments.connectionClosedMaxIdleTotal,
		dbStats.MaxIdleClosed,
		metric.WithAttributes(cfg.Attributes...),
	)
	observer.ObserveInt64(instruments.connectionClosedMaxIdleTimeTotal,
		dbStats.MaxIdleTimeClosed,
		metric.WithAttributes(cfg.Attributes...),
	)
	observer.ObserveInt64(instruments.connectionClosedMaxLifetimeTotal,
		dbStats.MaxLifetimeClosed,
		metric.WithAttributes(cfg.Attributes...),
	)
}
