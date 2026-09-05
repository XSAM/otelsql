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
	"errors"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	internalsemconv "github.com/XSAM/otelsql/internal/semconv"
)

var timeNow = time.Now

func recordSpanErrorDeferred(span trace.Span, cfg config, err *error) {
	recordSpanError(span, cfg, *err)
}

func recordSpanError(span trace.Span, cfg config, err error) {
	if span == nil || err == nil {
		return
	}

	if cfg.SpanErrorAttributesGetter != nil && span.IsRecording() {
		if attributes := cfg.SpanErrorAttributesGetter(err); len(attributes) > 0 {
			span.SetAttributes(attributes...)
		}
	}

	opts := cfg.SpanOptions
	if opts.RecordError != nil && !opts.RecordError(err) {
		return
	}

	if opts.DisableErrSkip && errors.Is(err, driver.ErrSkip) {
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, "")
}

func recordDuration(
	ctx context.Context,
	instruments *instruments,
	cfg config,
	duration time.Duration,
	attributes []attribute.KeyValue,
	method Method,
	err error,
) {
	attributes = append(attributes, semconv.DBOperationName(string(method)))
	if err != nil && (!cfg.DisableSkipErrMeasurement || !errors.Is(err, driver.ErrSkip)) {
		attributes = append(attributes, internalsemconv.ErrorTypeAttribute(err))
	}

	instruments.duration.RecordSet(
		ctx,
		duration.Seconds(),
		attribute.NewSet(attributes...),
	)
}

// TODO: remove instruments from arguments.
func recordMetric(
	ctx context.Context,
	instruments *instruments,
	cfg config,
	method Method,
	query string,
	args []driver.NamedValue,
) func(error) {
	startTime := timeNow()

	return func(err error) {
		duration := timeNow().Sub(startTime)

		var getterAttributes []attribute.KeyValue
		if cfg.InstrumentAttributesGetter != nil {
			getterAttributes = cfg.InstrumentAttributesGetter(ctx, method, query, args)
		}

		var errAttributes []attribute.KeyValue

		if err != nil {
			if cfg.InstrumentErrorAttributesGetter != nil {
				errAttributes = cfg.InstrumentErrorAttributesGetter(err)
			}
		}

		// number of attributes + InstrumentAttributesGetter + InstrumentErrorAttributesGetter + estimated 2 from recordDuration.
		attributes := make(
			[]attribute.KeyValue,
			len(cfg.Attributes),
			len(cfg.Attributes)+len(getterAttributes)+len(errAttributes)+2,
		)
		copy(attributes, cfg.Attributes)
		attributes = append(attributes, getterAttributes...)
		attributes = append(attributes, errAttributes...)

		recordDuration(ctx, instruments, cfg, duration, attributes, method, err)
	}
}

var spanKindClientOption = trace.WithSpanKind(trace.SpanKindClient)

func createSpan(
	ctx context.Context,
	cfg config,
	method Method,
	enableDBStatement bool,
	query string,
	args []driver.NamedValue,
) (context.Context, trace.Span) {
	spanCtx, span := cfg.Tracer.Start(ctx, cfg.SpanNameFormatter(ctx, method, query), spanKindClientOption)
	if span.IsRecording() {
		var dbStatementAttributes []attribute.KeyValue
		if enableDBStatement && !cfg.SpanOptions.DisableQuery {
			dbStatementAttributes = internalsemconv.DBQueryTextAttributes(query)
		}

		var getterAttributes []attribute.KeyValue
		if cfg.AttributesGetter != nil {
			getterAttributes = cfg.AttributesGetter(ctx, method, query, args)
		}

		// Allocate attributes slice (Attributes + AttributesGetter + DBQueryTextAttributes).
		attributes := make(
			[]attribute.KeyValue,
			len(cfg.Attributes),
			len(cfg.Attributes)+len(getterAttributes)+len(dbStatementAttributes),
		)
		copy(attributes, cfg.Attributes)
		attributes = append(attributes, dbStatementAttributes...)
		attributes = append(attributes, getterAttributes...)

		span.SetAttributes(attributes...)
	}

	return spanCtx, span
}

func filterSpan(
	ctx context.Context,
	spanOptions SpanOptions,
	method Method,
	query string,
	args []driver.NamedValue,
) bool {
	return spanOptions.SpanFilter == nil || spanOptions.SpanFilter(ctx, method, query, args)
}

// Copied from stdlib database/sql package: src/database/sql/ctxutil.go.
func namedValueToValue(named []driver.NamedValue) ([]driver.Value, error) {
	dargs := make([]driver.Value, len(named))

	for n, param := range named {
		if len(param.Name) > 0 {
			return nil, errors.New("sql: driver does not support the use of Named Parameters")
		}

		dargs[n] = param.Value
	}

	return dargs, nil
}
