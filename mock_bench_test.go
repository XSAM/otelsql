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

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Stateless driver mocks used only for benchmarks. They allocate nothing and are
// safe for concurrent use.
// This allows benchmarks measure otelsql's wrapper overhead.

type benchConn struct{}

func (benchConn) Begin() (driver.Tx, error)           { return benchTx{}, nil }
func (benchConn) Close() error                        { return nil }
func (benchConn) Prepare(string) (driver.Stmt, error) { return benchStmt{}, nil }
func (benchConn) Ping(context.Context) error          { return nil }
func (benchConn) ResetSession(context.Context) error  { return nil }

func (benchConn) PrepareContext(context.Context, string) (driver.Stmt, error) {
	return benchStmt{}, nil
}

func (benchConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return benchResult{}, nil
}

func (benchConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return benchRows{}, nil
}

func (benchConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return benchTx{}, nil
}

type benchStmt struct{}

func (benchStmt) Close() error                               { return nil }
func (benchStmt) NumInput() int                              { return 0 }
func (benchStmt) Exec([]driver.Value) (driver.Result, error) { return benchResult{}, nil }
func (benchStmt) Query([]driver.Value) (driver.Rows, error)  { return benchRows{}, nil }
func (benchStmt) CheckNamedValue(*driver.NamedValue) error   { return nil }
func (benchStmt) ColumnConverter(int) driver.ValueConverter  { return driver.DefaultParameterConverter }

func (benchStmt) ExecContext(context.Context, []driver.NamedValue) (driver.Result, error) {
	return benchResult{}, nil
}

func (benchStmt) QueryContext(context.Context, []driver.NamedValue) (driver.Rows, error) {
	return benchRows{}, nil
}

type benchRows struct{}

func (benchRows) Columns() []string           { return nil }
func (benchRows) Close() error                { return nil }
func (benchRows) Next(_ []driver.Value) error { return nil }

type benchTx struct{}

func (benchTx) Commit() error   { return nil }
func (benchTx) Rollback() error { return nil }

type benchResult struct{}

func (benchResult) LastInsertId() (int64, error) { return 0, nil }
func (benchResult) RowsAffected() (int64, error) { return 0, nil }

var (
	_ driver.Conn               = benchConn{}
	_ driver.Pinger             = benchConn{}
	_ driver.ExecerContext      = benchConn{}
	_ driver.QueryerContext     = benchConn{}
	_ driver.ConnPrepareContext = benchConn{}
	_ driver.ConnBeginTx        = benchConn{}
	_ driver.SessionResetter    = benchConn{}

	_ driver.Stmt              = benchStmt{}
	_ driver.StmtExecContext   = benchStmt{}
	_ driver.StmtQueryContext  = benchStmt{}
	_ driver.NamedValueChecker = benchStmt{}

	_ driver.Rows   = benchRows{}
	_ driver.Tx     = benchTx{}
	_ driver.Result = benchResult{}
)

// benchAttrs mimics a small base attribute set typical in production
// (service identity + db identity). With 3 attributes the per-call cost of
// copying cfg.Attributes is visible without dominating the benchmark.
var benchAttrs = []attribute.KeyValue{
	attribute.String("service.name", "bench"),
	attribute.String("db.system.name", "mysql"),
	attribute.String("db.namespace", "test"),
}

func newBenchConfig() config {
	cfg := newConfig()
	cfg.Tracer = sdktrace.NewTracerProvider().Tracer("benchmark")
	// Use a fixed-capacity slice so cfg.Attributes header stays stable across
	// the benchmark and matches the production pattern (set once at startup).
	cfg.Attributes = make([]attribute.KeyValue, 0, len(benchAttrs))
	cfg.Attributes = append(cfg.Attributes, benchAttrs...)

	return cfg
}

func newBenchConfigNeverSample() config {
	cfg := newBenchConfig()
	cfg.Tracer = sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.NeverSample())).
		Tracer("benchmark")

	return cfg
}

var benchMarkConfigs = map[string]func() config{
	"Default":     newBenchConfig,
	"NeverSample": newBenchConfigNeverSample,
}
