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

func recordSpanError(span trace.Span, cfg *config, err error) {
	if span == nil || err == nil || !span.IsRecording() {
		return
	}

	if cfg.SpanErrorAttributesGetter != nil {
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

// recordSpanErrorDeferred is used when using `defer` to record a span error.
// It takes a pointer to the caller's error so the value of the error is read correctly.
func recordSpanErrorDeferred(span trace.Span, cfg *config, errp *error) {
	recordSpanError(span, cfg, *errp)
}

type durationMetric struct {
	ctx       context.Context
	startTime time.Time
}

func startDurationMetric(ctx context.Context) durationMetric {
	return durationMetric{
		ctx:       ctx,
		startTime: timeNow(),
	}
}

// Record records the duration since the metric was started.
// It takes a pointer to the caller's error so it can be used with `defer`
// and still read the final value of the error.
func (s *durationMetric) Record(cfg *config, method Method, query string, args []driver.NamedValue, errp *error) {
	var getterAttributes []attribute.KeyValue
	if cfg.InstrumentAttributesGetter != nil {
		getterAttributes = cfg.InstrumentAttributesGetter(s.ctx, method, query, args)
	}

	s.record(cfg, method, getterAttributes, errp)
}

func (s *durationMetric) record(cfg *config, method Method, getterAttributes []attribute.KeyValue, errp *error) {
	duration := timeNow().Sub(s.startTime)

	var err error
	if errp != nil {
		err = *errp
	}

	// Fast path: no error and no getter attributes, so the attributes are
	// exactly the precomputed set of this method.
	if err == nil && len(getterAttributes) == 0 {
		if set, ok := cfg.metricMethodSets[method]; ok {
			cfg.Instruments.duration.RecordSet(s.ctx, duration.Seconds(), set)

			return
		}
	}

	var (
		recordErrorType     bool
		getterErrAttributes []attribute.KeyValue
	)

	if err != nil {
		recordErrorType = !cfg.DisableSkipErrMeasurement || !errors.Is(err, driver.ErrSkip)

		if cfg.InstrumentErrorAttributesGetter != nil {
			getterErrAttributes = cfg.InstrumentErrorAttributesGetter(err)
		}
	}

	// Attributes + InstrumentAttributesGetter + InstrumentErrorAttributesGetter + db.operation.name + error.type.
	// The order matters: attribute.NewSet keeps the last value of a duplicated key,
	// so db.operation.name and error.type set here win over the ones from getters.
	attributes := make(
		[]attribute.KeyValue,
		len(cfg.Attributes),
		len(cfg.Attributes)+len(getterAttributes)+len(getterErrAttributes)+2,
	)
	copy(attributes, cfg.Attributes)
	attributes = append(attributes, getterAttributes...)
	attributes = append(attributes, getterErrAttributes...)
	attributes = append(attributes, semconv.DBOperationName(string(method)))

	if recordErrorType {
		attributes = append(attributes, internalsemconv.ErrorTypeAttribute(err))
	}

	cfg.Instruments.duration.RecordSet(
		s.ctx,
		duration.Seconds(),
		attribute.NewSet(attributes...),
	)
}

var spanKindClientOption = trace.WithSpanKind(trace.SpanKindClient)

func createSpan(
	ctx context.Context,
	cfg *config,
	method Method,
	enableDBStatement bool,
	query string,
	args []driver.NamedValue,
) (context.Context, trace.Span) {
	spanCtx, span := cfg.Tracer.Start(ctx, cfg.SpanNameFormatter(ctx, method, query), spanKindClientOption)
	if !span.IsRecording() {
		return spanCtx, span
	}

	addDBStatement := enableDBStatement && !cfg.SpanOptions.DisableQuery

	// Fast path when we only have to add config attributes
	if cfg.AttributesGetter == nil && !addDBStatement {
		if len(cfg.Attributes) > 0 {
			span.SetAttributes(cfg.Attributes...)
		}

		return spanCtx, span
	}

	var dbStatementAttributes []attribute.KeyValue
	if addDBStatement {
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
