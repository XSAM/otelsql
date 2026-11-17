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
	"testing"
)

func BenchmarkNewStmt(b *testing.B) {
	cfg := newBenchConfigNeverSample()
	conn := newConn(benchConn{}, cfg)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = newStmt(benchStmt{}, cfg, "SELECT 1", conn)
	}
}

func BenchmarkStmtExecContext(b *testing.B) {
	for name, newCfg := range benchMarkConfigs {
		b.Run(name, func(b *testing.B) {
			cfg := newCfg()
			conn := newConn(benchConn{}, cfg)
			stmt := newStmt(benchStmt{}, cfg, "INSERT INTO x VALUES (?)", conn)
			ctx := context.Background()

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if _, err := stmt.ExecContext(ctx, nil); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

func BenchmarkStmtQueryContext(b *testing.B) {
	for name, newCfg := range benchMarkConfigs {
		b.Run(name, func(b *testing.B) {
			cfg := newCfg()
			conn := newConn(benchConn{}, cfg)
			stmt := newStmt(benchStmt{}, cfg, "SELECT * FROM x WHERE id = ?", conn)
			ctx := context.Background()

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					rows, err := stmt.QueryContext(ctx, nil)
					if err != nil {
						b.Fatal(err)
					}

					if err := rows.Close(); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
