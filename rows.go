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

var (
	_ driver.Rows                           = (*otRows)(nil)
	_ driver.RowsNextResultSet              = (*otRows)(nil)
	_ driver.RowsColumnTypeDatabaseTypeName = (*otRows)(nil)
	_ driver.RowsColumnTypeLength           = (*otRows)(nil)
	_ driver.RowsColumnTypeNullable         = (*otRows)(nil)
	_ driver.RowsColumnTypePrecisionScale   = (*otRows)(nil)
)

type otRows struct {
	driver.Rows

	span    trace.Span
	cfg     config
	onClose func(err error)
}

func newRows(ctx context.Context, rows driver.Rows, cfg config) *otRows {
	var span trace.Span

	method := MethodRows
	onClose := recordMetric(ctx, cfg.Instruments, cfg.Attributes, method)

	if !cfg.SpanOptions.OmitRows && filterSpan(ctx, cfg.SpanOptions, method, "", nil) {
		_, span = createSpan(ctx, cfg, method, false, "", nil)
	}

	return &otRows{
		Rows:    rows,
		span:    span,
		cfg:     cfg,
		onClose: onClose,
	}
}

// HasNextResultSet вызывает реализацию driver.RowsNextResultSet для otRows.
// Возвращает результат HasNextResultSet от otRows.parent
// если parent реализует driver.RowsNextResultSet.
func (r otRows) HasNextResultSet() bool {
	if v, ok := r.Rows.(driver.RowsNextResultSet); ok {
		return v.HasNextResultSet()
	}

	return false
}

// NextResultSet вызывает реализацию driver.RowsNextResultSet для otRows.
// Возвращает NextResultSet от otRows.parent
// если parent удовоетворяет driver.RowsNextResultSet.
func (r otRows) NextResultSet() error {
	if v, ok := r.Rows.(driver.RowsNextResultSet); ok {
		return v.NextResultSet()
	}

	return io.EOF
}

// ColumnTypeDatabaseTypeName вызывает реализацию driver.RowsColumnTypeDatabaseTypeName для otRows.
// Возвращает результат ColumnTypeDatabaseTypeName от otRows.Rows
// если Rows реализует driver.RowsColumnTypeDatabaseTypeName.
func (r otRows) ColumnTypeDatabaseTypeName(index int) string {
	if v, ok := r.Rows.(driver.RowsColumnTypeDatabaseTypeName); ok {
		return v.ColumnTypeDatabaseTypeName(index)
	}

	return ""
}

// ColumnTypeLength вызывает реализацию driver.RowsColumnTypeLength для otRows.
// Возвращает результат ColumnTypeLength от otRows.Rows
// если Rows реализует driver.RowsColumnTypeLength.
func (r otRows) ColumnTypeLength(index int) (length int64, ok bool) {
	if v, ok := r.Rows.(driver.RowsColumnTypeLength); ok {
		return v.ColumnTypeLength(index)
	}

	return 0, false
}

// ColumnTypeNullable вызывает реализацию driver.RowsColumnTypeNullable для otRows.
// Возвращает результат ColumnTypeNullable от otRows.Rows
// если Rows реализует driver.RowsColumnTypeNullable.
func (r otRows) ColumnTypeNullable(index int) (nullable, ok bool) {
	if v, ok := r.Rows.(driver.RowsColumnTypeNullable); ok {
		return v.ColumnTypeNullable(index)
	}

	return false, false
}

// ColumnTypePrecisionScale вызывает реализацию driver.RowsColumnTypePrecisionScale для otRows.
// Возвращает результат ColumnTypePrecisionScale от otRows.Rows
// если Rows реализует driver.RowsColumnTypePrecisionScale.
func (r otRows) ColumnTypePrecisionScale(index int) (precision, scale int64, ok bool) {
	if v, ok := r.Rows.(driver.RowsColumnTypePrecisionScale); ok {
		return v.ColumnTypePrecisionScale(index)
	}

	return 0, 0, false
}

func (r otRows) Close() (err error) {
	defer func() {
		if r.span != nil {
			r.span.End()
		}
		r.onClose(err)
	}()

	err = r.Rows.Close()
	if err != nil {
		recordSpanError(r.span, r.cfg.SpanOptions, err)
	}
	return
}

func (r otRows) Next(dest []driver.Value) (err error) {
	if r.cfg.SpanOptions.RowsNext && r.span != nil {
		r.span.AddEvent(string(EventRowsNext))
	}

	err = r.Rows.Next(dest)
	// io.EOF это не ошибка. Это непредвиденное обстоятельство во время итерации.
	if err != nil && err != io.EOF {
		recordSpanError(r.span, r.cfg.SpanOptions, err)
	}
	return
}
