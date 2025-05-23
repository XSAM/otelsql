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

	internalsemconv "github.com/XSAM/otelsql/internal/semconv"
	"go.opentelemetry.io/otel/attribute"
)

var (
	attrs10 = []attribute.KeyValue{
		attribute.Bool("b1", true),
		attribute.Int("i1", 324),
		attribute.Float64("f1", -230.213),
		attribute.String("s1", "value1"),
		attribute.String("s2", "value2"),
		attribute.Bool("b2", false),
		attribute.Int("i2", 39847),
		attribute.Float64("f2", 0.382964329),
		attribute.String("s3", "value3"),
		attribute.String("s4", "value4"),
	}
	attrs5 = attrs10[:5]
)

func BenchmarkRecordMetric(b *testing.B) {
	cfg := newConfig()
	cfg.SemConvStabilityOptIn = internalsemconv.OTelSemConvStabilityOptInStable
	// Prevent reallocation of Attributes slice, which increase the chance to detect data races.
	cfg.Attributes = make([]attribute.KeyValue, 0, 10)

	b.Run("InstrumentAttributesGetter", func(b *testing.B) {
		b.Run("5", func(b *testing.B) {
			cfg := cfg
			cfg.InstrumentAttributesGetter = func(ctx context.Context, method Method, query string, args []driver.NamedValue) []attribute.KeyValue {
				return attrs5
			}

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					recordFunc := recordMetric(context.Background(), cfg.Instruments, cfg, MethodStmtQuery, "SELECT 1", nil)
					recordFunc(nil)
				}
			})
		})

		b.Run("10", func(b *testing.B) {
			cfg := cfg
			cfg.InstrumentAttributesGetter = func(ctx context.Context, method Method, query string, args []driver.NamedValue) []attribute.KeyValue {
				return attrs10
			}

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					recordFunc := recordMetric(context.Background(), cfg.Instruments, cfg, MethodStmtQuery, "SELECT 1", nil)
					recordFunc(nil)
				}
			})
		})
	})
}

func BenchmarkCreateSpan(b *testing.B) {
	cfg := newConfig()
	cfg.SemConvStabilityOptIn = internalsemconv.OTelSemConvStabilityOptInStable
	// Prevent reallocation of Attributes slice, which increase the chance to detect data races.
	cfg.Attributes = make([]attribute.KeyValue, 0, 10)

	ctx := context.Background()

	b.Run("AttributesGetter", func(b *testing.B) {
		b.Run("5", func(b *testing.B) {
			cfg := cfg
			cfg.AttributesGetter = func(ctx context.Context, method Method, query string, args []driver.NamedValue) []attribute.KeyValue {
				return attrs5
			}

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, _ = createSpan(ctx, cfg, MethodStmtQuery, true, "SELECT 1", nil)
				}
			})
		})

		b.Run("10", func(b *testing.B) {
			cfg := cfg
			cfg.AttributesGetter = func(ctx context.Context, method Method, query string, args []driver.NamedValue) []attribute.KeyValue {
				return attrs10
			}

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, _ = createSpan(ctx, cfg, MethodStmtQuery, true, "SELECT 1", nil)
				}
			})
		})
	})
}
