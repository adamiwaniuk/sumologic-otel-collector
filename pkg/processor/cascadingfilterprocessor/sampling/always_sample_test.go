// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sampling

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/model/pdata"
	"go.uber.org/zap"
)

func newAlwaysSample() *policyEvaluator {
	return &policyEvaluator{
		logger:            zap.NewNop(),
		maxSpansPerSecond: math.MaxInt64,
	}
}

func TestEvaluate_AlwaysSample(t *testing.T) {
	filter := newAlwaysSample()
	decision := filter.Evaluate(
		pdata.NewTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}),
		newTraceStringAttrs(map[string]pdata.AttributeValue{}, "example", "value"),
	)
	assert.Equal(t, decision, Sampled)
}

func TestOnLateArrivingSpans_AlwaysSample(t *testing.T) {
	filter := newAlwaysSample()
	err := filter.OnLateArrivingSpans(NotSampled, nil)
	assert.Nil(t, err)
}
