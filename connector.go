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
	"database/sql/driver"
	"io"

	"go.opentelemetry.io/otel/trace"
)

var _ driver.Connector = (*otConnector)(nil)
var _ io.Closer = (*otConnector)(nil)

// otConnector структура, описывающая connector
// для database/sql connector
type otConnector struct {
	driver.Connector
	otDriver *otDriver
	cfg      config
}

// newConnector иницилизирует otCollector с переданными настройками cfg,
// и драйвером otDriver.
func newConnector(connector driver.Connector, otDriver *otDriver) *otConnector {
	return &otConnector{
		Connector: connector,
		otDriver:  otDriver,
		cfg:       otDriver.cfg,
	}
}

// Connect метод структуры otConnector, осуществляющий подключение
// и возращающий интерфейс driver.Conn и ошибку
func (c *otConnector) Connect(ctx context.Context) (connection driver.Conn, err error) {
	method := MethodConnectorConnect
	onDefer := recordMetric(ctx, c.cfg.Instruments, c.cfg.Attributes, method)
	defer func() {
		onDefer(err)
	}()

	var span trace.Span
	if !c.cfg.SpanOptions.OmitConnectorConnect && filterSpan(ctx, c.cfg.SpanOptions, method, "", nil) {
		ctx, span = createSpan(ctx, c.cfg, method, false, "", nil)
		defer span.End()
	}

	connection, err = c.Connector.Connect(ctx)
	if err != nil {
		recordSpanError(span, c.cfg.SpanOptions, err)
		return nil, err
	}
	return newConn(connection, c.cfg), nil
}

// метод otConnector Driver возвращает
// поле структуры otConnector otDriver
func (c *otConnector) Driver() driver.Driver {
	return c.otDriver
}

func (c *otConnector) Close() error {
	// database/sql использует type assertion для проверки удоволетворяет ли connector io.Closer.
	if closer, ok := c.Connector.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// dsnConnector сорпирован с sql.dsnConnector.
type dsnConnector struct {
	dsn    string
	driver driver.Driver
}

// Connect метод структуры dsnConnector, осуществленяющий подклчение
// и возвращающий driver.Conn и ошибку.
func (t dsnConnector) Connect(_ context.Context) (driver.Conn, error) {
	return t.driver.Open(t.dsn)
}

// Driver метод структуры dsnConnector возвращает
// драйвер, хранящийся в структуре dsnConnector
func (t dsnConnector) Driver() driver.Driver {
	return t.driver
}
