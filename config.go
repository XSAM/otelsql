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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName = "github.com/XSAM/otelsql"
)

var (
	connectionStatusKey = attribute.Key("status")
	queryStatusKey      = attribute.Key("status")
	queryMethodKey      = attribute.Key("method")
)

// SpanNameFormatter помогает иницилизировать имена для spans.
type SpanNameFormatter func(ctx context.Context, method Method, query string) string

// AttributesGetter помогает иницилизировать spans при их создании.
type AttributesGetter func(ctx context.Context, method Method, query string, args []driver.NamedValue) []attribute.KeyValue

type SpanFilter func(ctx context.Context, method Method, query string, args []driver.NamedValue) bool

// config структура, содержащее все необходимое для трассировки базы данных.
type config struct {
	TracerProvider trace.TracerProvider
	Tracer         trace.Tracer

	MeterProvider metric.MeterProvider
	Meter         metric.Meter

	Instruments *instruments

	SpanOptions SpanOptions

	// Attributes, которые будут добавлены во все spans.
	Attributes []attribute.KeyValue

	// SpanNameFormatter будет вызван для присваивания имени span.
	// По умолчанию использует название метод как имя span.
	SpanNameFormatter SpanNameFormatter

	// SQLCommenterEnabled включает проброс context для database
	// при помощи включения комментария в SQL statements.
	//
	// Эксперементально!
	//
	// Notice: Эта опция ЭКСПЕРЕМЕНТАЛЬНА и, возможно, будет изменена или удалена в
	// более поздник релизах.
	SQLCommenterEnabled bool
	SQLCommenter        *commenter

	// AttributesGetter функция, которая будет вызвана для  инициализации дополнительных attributes 
  // во время создания spans.
	// По умолчанию возвращает nil
	AttributesGetter AttributesGetter
}

// SpanOptions структура, содержащая некоторые опции для тонко настройки tracing spans.
// По умолчанию все опции отключены.
type SpanOptions struct {
	// Ping, если выставлено значение true, включит создание spans по Ping requests.
	Ping bool

	// RowsNext, если высталено значение true, включит создание  events в spans на вызов RowsNext
	RowsNext bool

	// DisableErrSkip,если выставлено значение true, будет подавлять driver.ErrSkip errors в spans.
	DisableErrSkip bool

	// DisableQuery если выставлено значение true, будет подавлено db.statement в spans.
	DisableQuery bool

	// RecordError, если включено, будет вызвана с текущей ошибкой, если функция возвращает true
	// то запись будет записана в текущий span.
	//
	// В противном случае будет записывать все ошибки в текущий span (possible not ErrSkip, see option
	// DisableErrSkip).
	RecordError func(err error) bool

	// OmitConnResetSession, если выставлено значение true, будет подавлять sql.conn.reset_session spans
	OmitConnResetSession bool

	// OmitConnPrepare, если выставлено true, будет подавлять sql.conn.prepare spans
	OmitConnPrepare bool

	// OmitConnQuery, если выставлено true, будет подавлять sql.conn.query spans
	OmitConnQuery bool

	// OmitRows, если выставлено true, будет подавлять sql.rows spans
	OmitRows bool

	// OmitConnectorConnect, если выставлено true, будет подавлять sql.connector.connect spans
	OmitConnectorConnect bool

	// SpanFilter, функция, которая будет вызвана перед каждым вызовом span. Если функция возвращает
	// false, span will не будет создан.
	SpanFilter SpanFilter
}

func defaultSpanNameFormatter(_ context.Context, method Method, _ string) string {
	return string(method)
}

// newConfig функция, возвращающая config, иницилизированный переданными опциями options.
func newConfig(options ...Option) config {
	cfg := config{
		TracerProvider:    otel.GetTracerProvider(),
		MeterProvider:     otel.GetMeterProvider(),
		SpanNameFormatter: defaultSpanNameFormatter,
	}
	for _, opt := range options {
		opt.Apply(&cfg)
	}

	cfg.Tracer = cfg.TracerProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(Version()),
	)
	cfg.Meter = cfg.MeterProvider.Meter(
		instrumentationName,
		metric.WithInstrumentationVersion(Version()),
	)

	cfg.SQLCommenter = newCommenter(cfg.SQLCommenterEnabled)

	var err error
	if cfg.Instruments, err = newInstruments(cfg.Meter); err != nil {
		otel.Handle(err)
	}

	return cfg
}
