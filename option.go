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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Option это интерфейс, который инициализирует опции.
type Option interface {
	// Apply выставяет значение Option необходимых config.
	Apply(*config)
}

var _ Option = OptionFunc(nil)

// OptionFunc функция, удоволетворяющая интерфейсу Option.
type OptionFunc func(*config)

func (f OptionFunc) Apply(c *config) {
	f(c)
}

// WithTracerProvider инициализирует tracer provider для создания tracer.
// По умолчанию используется global provider.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return OptionFunc(func(cfg *config) {
		cfg.TracerProvider = provider
	})
}

// WithAttributes инициализирует attributes, которые будут применены к каждому span.
func WithAttributes(attributes ...attribute.KeyValue) Option {
	return OptionFunc(func(cfg *config) {
		cfg.Attributes = attributes
	})
}

// WithSpanNameFormatter принимает функцию, которая будет вызвана на всякую
// со span и строка, которая вернётся, станет именем span.
func WithSpanNameFormatter(spanNameFormatter SpanNameFormatter) Option {
	return OptionFunc(func(cfg *config) {
		cfg.SpanNameFormatter = spanNameFormatter
	})
}

// WithSpanOptions иницилизирует некоторые опции для span.
func WithSpanOptions(opts SpanOptions) Option {
	return OptionFunc(func(cfg *config) {
		cfg.SpanOptions = opts
	})
}

// WithMeterProvider инициализирует tracer provider для создания tracer.
// По умолчанию используется global provider.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return OptionFunc(func(cfg *config) {
		cfg.MeterProvider = provider
	})
}

// WithSQLCommenter включает или отключает проброс context для database
// посредством включения комментария в SQL statements.
//
// e.g., a SQL query
//
//	SELECT * from FOO
//
// will become
//
//	SELECT * from FOO /*traceparent='00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01',tracestate='congo%3Dt61rcWkgMzE%2Crojo%3D00f067aa0ba902b7'*/
//
// Данная опцияя по умолчанию отключена.
//
// Notice: Эта опция ЭКСПЕРЕМЕНТАЛЬНА и, возможно, будет изменена или удалена
// в более поздних релизах.
func WithSQLCommenter(enabled bool) Option {
	return OptionFunc(func(cfg *config) {
		cfg.SQLCommenterEnabled = enabled
	})
}

// WithAttributesGetter принимает AttributesGetter которая будет вызвана при
// создании span.
func WithAttributesGetter(attributesGetter AttributesGetter) Option {
	return OptionFunc(func(cfg *config) {
		cfg.AttributesGetter = attributesGetter
	})
}
