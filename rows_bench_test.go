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

func BenchmarkNewRows(b *testing.B) {
	cfg := newBenchConfigNeverSample()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = newRows(ctx, benchRows{}, cfg)
	}
}

func BenchmarkRows_Next(b *testing.B) {
	configs := make(map[string]*config, len(benchMarkConfigs))
	for name, cfg := range benchMarkConfigs {
		configs[name] = cfg()
	}

	configs["DefaultWithRowsNextEvent"] = func() *config {
		cfg := newBenchConfig()
		cfg.SpanOptions.RowsNext = true

		return cfg
	}()

	for name, cfg := range configs {
		b.Run(name, func(b *testing.B) {
			ctx := context.Background()

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				// Each goroutine works on its own rows handle, matching how a real
				// caller would iterate a result set. newRows is amortized over many
				// Next() calls, so its construction cost is not what's reported here.
				rows := newRows(ctx, benchRows{}, cfg)

				dest := make([]driver.Value, 0)
				for pb.Next() {
					if err := rows.Next(dest); err != nil {
						b.Fatal(err)
					}
				}

				_ = rows.Close()
			})
		})
	}
}

func BenchmarkRows_Close(b *testing.B) {
	for name, newCfg := range benchMarkConfigs {
		b.Run(name, func(b *testing.B) {
			cfg := newCfg()
			ctx := context.Background()

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					rows := newRows(ctx, benchRows{}, cfg)
					if err := rows.Close(); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
