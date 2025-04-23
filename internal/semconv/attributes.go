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

package semconv

import (
	"go.opentelemetry.io/otel/attribute"
	semconvlegacy "go.opentelemetry.io/otel/semconv/v1.24.0"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

// NewDBQueryTextAttributes returns a function that generates appropriate database query attributes
// based on the provided OTelSemConvStabilityOptInType.
//
//   - OTelSemConvStabilityOptInNone: Only legacy db.statement attribute
//   - OTelSemConvStabilityOptInDup: Both legacy db.statement and stable db.query.text attributes
//   - OTelSemConvStabilityOptInStable: Only stable db.query.text attribute
func NewDBQueryTextAttributes(optInType OTelSemConvStabilityOptInType) func(query string) []attribute.KeyValue {
	switch optInType {
	case OTelSemConvStabilityOptInDup:
		// Emit both legacy and stable attributes
		return func(query string) []attribute.KeyValue {
			return []attribute.KeyValue{
				semconvlegacy.DBStatementKey.String(query),
				semconv.DBQueryTextKey.String(query),
			}
		}
	case OTelSemConvStabilityOptInStable:
		// Only emit stable attribute
		return func(query string) []attribute.KeyValue {
			return []attribute.KeyValue{
				semconv.DBQueryTextKey.String(query),
			}
		}
	default:
		// OTelSemConvStabilityOptInNone or any unknown types
		// Only emit legacy attribute
		return func(query string) []attribute.KeyValue {
			return []attribute.KeyValue{
				semconvlegacy.DBStatementKey.String(query),
			}
		}
	}
}
