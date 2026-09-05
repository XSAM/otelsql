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

//go:build go1.27

package otelsql

import "database/sql/driver"

var _ driver.RowsColumnScanner = (*otRowsColumnScanner)(nil)

type otRowsColumnScanner struct {
	*otRows

	scanner driver.RowsColumnScanner
}

func wrapRowsColumnScanner(rows *otRows) driver.Rows {
	scanner, ok := rows.Rows.(driver.RowsColumnScanner)
	if !ok {
		return rows
	}

	return &otRowsColumnScanner{
		otRows:  rows,
		scanner: scanner,
	}
}

func (r otRowsColumnScanner) NextRow() (err error) {
	r.beforeNext()

	err = r.scanner.NextRow()
	r.afterNext(err)

	return
}

func (r otRowsColumnScanner) ScanColumn(scanCtx driver.ScanContext, index int, dest any) error {
	return r.scanner.ScanColumn(scanCtx, index, dest)
}
