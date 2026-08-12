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
	"testing"
)

func BenchmarkConnQueryContext(b *testing.B) {
	for name, newCfg := range benchMarkConfigs {
		b.Run(name, func(b *testing.B) {
			cfg := newCfg()
			conn := newConn(benchConn{}, cfg)
			ctx := context.Background()

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					rows, err := conn.QueryContext(ctx, "SELECT 1", nil)
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

func BenchmarkConnExecContext(b *testing.B) {
	for name, newCfg := range benchMarkConfigs {
		b.Run(name, func(b *testing.B) {
			cfg := newCfg()
			conn := newConn(benchConn{}, cfg)
			ctx := context.Background()

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if _, err := conn.ExecContext(ctx, "UPDATE x SET y=1", nil); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

func BenchmarkConnResetSession(b *testing.B) {
	for name, newCfg := range benchMarkConfigs {
		b.Run(name, func(b *testing.B) {
			cfg := newCfg()
			conn := newConn(benchConn{}, cfg)
			ctx := context.Background()

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if err := conn.ResetSession(ctx); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

func BenchmarkConnBeginTx(b *testing.B) {
	for name, newCfg := range benchMarkConfigs {
		b.Run(name, func(b *testing.B) {
			cfg := newCfg()
			conn := newConn(benchConn{}, cfg)
			ctx := context.Background()

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					tx, err := conn.BeginTx(ctx, driver.TxOptions{})
					if err != nil {
						b.Fatal(err)
					}

					if err := tx.Commit(); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
