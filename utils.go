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
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"

	internalsemconv "github.com/XSAM/otelsql/internal/semconv"
)

// estimatedAttributesOfGettersCount is the estimated number of attributes from getter methods.
// This value 5 is borrowed from slog which
// performed a quantitative survey of log library use and found this value to
// cover 95% of all use-cases (https://go.dev/blog/slog#performance).
// This may not be accurate for metrics or traces, but it's a good starting point.
const estimatedAttributesOfGettersCount = 5

var timeNow = time.Now

func recordSpanErrorDeferred(span trace.Span, opts SpanOptions, err *error) {
	recordSpanError(span, opts, *err)
}

func recordSpanError(span trace.Span, opts SpanOptions, err error) {
	if span == nil {
		return
	}
	if opts.RecordError != nil && !opts.RecordError(err) {
		return
	}

	switch {
	case err == nil:
		return
	case errors.Is(err, driver.ErrSkip):
		if !opts.DisableErrSkip {
			span.RecordError(err)
			span.SetStatus(codes.Error, "")
		}
	default:
		span.RecordError(err)
		span.SetStatus(codes.Error, "")
	}
}

func recordLegacyLatency(
	ctx context.Context,
	instruments *instruments,
	cfg config,
	duration time.Duration,
	attributes []attribute.KeyValue,
	method Method,
	err error,
) {
	attributes = append(attributes, queryMethodKey.String(string(method)))

	if err != nil {
		if cfg.DisableSkipErrMeasurement && errors.Is(err, driver.ErrSkip) {
			attributes = append(attributes, queryStatusKey.String("ok"))
		} else {
			attributes = append(attributes, queryStatusKey.String("error"))
		}
	} else {
		attributes = append(attributes, queryStatusKey.String("ok"))
	}

	instruments.legacyLatency.Record(
		ctx,
		float64(duration.Nanoseconds())/1e6,
		metric.WithAttributes(attributes...),
	)
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
		attributes = append(attributes, internalsemconv.ErrorTypeAttributes(err)...)
	}

	instruments.duration.Record(
		ctx,
		duration.Seconds(),
		metric.WithAttributes(attributes...),
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

		// number of attributes + estimated 5 from InstrumentAttributesGetter and
		// InstrumentErrorAttributesGetter + estimated 2 from recordDuration.
		attributes := make(
			[]attribute.KeyValue,
			len(cfg.Attributes),
			len(cfg.Attributes)+estimatedAttributesOfGettersCount+2,
		)
		copy(attributes, cfg.Attributes)

		if cfg.InstrumentAttributesGetter != nil {
			attributes = append(attributes, cfg.InstrumentAttributesGetter(ctx, method, query, args)...)
		}
		if err != nil {
			if cfg.InstrumentErrorAttributesGetter != nil {
				attributes = append(attributes, cfg.InstrumentErrorAttributesGetter(err)...)
			}
		}

		switch cfg.SemConvStabilityOptIn {
		case internalsemconv.OTelSemConvStabilityOptInStable:
			recordDuration(ctx, instruments, cfg, duration, attributes, method, err)
		case internalsemconv.OTelSemConvStabilityOptInDup:
			// Intentionally emit both legacy and new metrics for backward compatibility.
			recordLegacyLatency(ctx, instruments, cfg, duration, attributes, method, err)
			recordDuration(ctx, instruments, cfg, duration, attributes, method, err)
		case internalsemconv.OTelSemConvStabilityOptInNone:
			recordLegacyLatency(ctx, instruments, cfg, duration, attributes, method, err)
		}
	}
}

func createSpan(
	ctx context.Context,
	cfg config,
	method Method,
	enableDBStatement bool,
	query string,
	args []driver.NamedValue,
) (context.Context, trace.Span) {
	// number of attributes + estimated 5 from AttributesGetter + estimated 2 from DBQueryTextAttributes.
	attributes := make(
		[]attribute.KeyValue,
		len(cfg.Attributes),
		len(cfg.Attributes)+estimatedAttributesOfGettersCount+2,
	)
	copy(attributes, cfg.Attributes)

	if enableDBStatement && !cfg.SpanOptions.DisableQuery {
		attributes = append(attributes, cfg.DBQueryTextAttributes(query)...)
	}
	if cfg.AttributesGetter != nil {
		attributes = append(attributes, cfg.AttributesGetter(ctx, method, query, args)...)
	}

	return cfg.Tracer.Start(ctx, cfg.SpanNameFormatter(ctx, method, query),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attributes...),
	)
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
