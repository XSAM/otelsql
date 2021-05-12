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

	"go.opentelemetry.io/otel/trace"
)

var _ driver.Tx = (*otTx)(nil)

type otTx struct {
	tx  driver.Tx
	ctx context.Context
	cfg config
}

func newTx(ctx context.Context, tx driver.Tx, cfg config) *otTx {
	return &otTx{
		tx:  tx,
		ctx: ctx,
		cfg: cfg,
	}
}

func (t *otTx) Commit() (err error) {
	var span trace.Span
	if t.cfg.SpanOptions.AllowRoot || trace.SpanContextFromContext(t.ctx).IsValid() {
		_, span = t.cfg.Tracer.Start(t.ctx, t.cfg.SpanNameFormatter.Format(t.ctx, MethodTxCommit, ""),
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(t.cfg.Attributes...),
		)
		defer span.End()
	}

	err = t.tx.Commit()
	if err != nil {
		recordSpanError(span, t.cfg.SpanOptions, err)
		return err
	}
	return nil
}

func (t *otTx) Rollback() (err error) {
	var span trace.Span
	if t.cfg.SpanOptions.AllowRoot || trace.SpanContextFromContext(t.ctx).IsValid() {
		_, span = t.cfg.Tracer.Start(t.ctx, t.cfg.SpanNameFormatter.Format(t.ctx, MethodTxRollback, ""),
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(t.cfg.Attributes...),
		)
		defer span.End()
	}

	err = t.tx.Rollback()
	if err != nil {
		recordSpanError(span, t.cfg.SpanOptions, err)
		return err
	}
	return nil
}
