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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOTelSemConvStabilityOptIn(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected OTelSemConvStabilityOptInType
	}{
		{
			name:     "unset env var",
			envValue: "",
			expected: OTelSemConvStabilityOptInNone,
		},
		{
			name:     "database/dup",
			envValue: "database/dup",
			expected: OTelSemConvStabilityOptInDup,
		},
		{
			name:     "database",
			envValue: "database",
			expected: OTelSemConvStabilityOptInStable,
		},
		{
			name:     "database/dup with other values",
			envValue: "foo,database/dup,bar",
			expected: OTelSemConvStabilityOptInDup,
		},
		{
			name:     "database with other values",
			envValue: "foo,database,bar",
			expected: OTelSemConvStabilityOptInStable,
		},
		{
			name:     "database/dup has precedence over database",
			envValue: "database,database/dup",
			expected: OTelSemConvStabilityOptInDup,
		},
		{
			name:     "irrelevant values",
			envValue: "foo,bar",
			expected: OTelSemConvStabilityOptInNone,
		},
		{
			name:     "whitespace handling",
			envValue: " database/dup , database ",
			expected: OTelSemConvStabilityOptInDup,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment variable before each test
			t.Cleanup(func() {
				os.Unsetenv(OTelSemConvStabilityOptIn)
			})

			// Always set environment variable for the test
			os.Setenv(OTelSemConvStabilityOptIn, tt.envValue)

			// Test the function
			result := ParseOTelSemConvStabilityOptIn()
			assert.Equal(t, tt.expected, result)
		})
	}
}
