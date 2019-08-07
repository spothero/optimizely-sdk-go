// Copyright 2019 SpotHero
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

package optimizely

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExperiment_getBucketValue(t *testing.T) {
	tests := []struct {
		experimentID  string
		bucketingID   string
		expectedValue int
	}{
		{
			"1886780721",
			"ppid1",
			5254,
		}, {
			"1886780721",
			"ppid2",
			4299,
		}, {
			"1886780722",
			"ppid2",
			2434,
		}, {
			"1886780721",
			"ppid3",
			5439,
		}, {
			"1886780721",
			"a very very very very very very very very very very very very very very very long ppd string",
			6128,
		},
	}
	for _, test := range tests {
		testName := fmt.Sprintf("experiment id %v, bucketing id %v", test.experimentID, test.bucketingID)
		t.Run(testName, func(t *testing.T) {
			assert.Equal(t, test.expectedValue, Experiment{id: test.experimentID}.getBucketValue(test.bucketingID))
		})
	}
}

func TestExperiment_findBucket(t *testing.T) {
	tests := []struct {
		name              string
		experiment        Experiment
		bucketValue       int
		expectedVariation *Variation
	}{
		{
			"variation is selected from traffic allocation",
			Experiment{trafficAllocation: []trafficAllocation{{
				endOfRange: 100,
				Variation: Variation{
					id:  "abc",
					Key: "abc",
				},
			}}},
			10,
			&Variation{
				id:  "abc",
				Key: "abc",
			},
		}, {
			"bucket value higher than allocation end of range returns no variation",
			Experiment{trafficAllocation: []trafficAllocation{{
				endOfRange: 100,
				Variation: Variation{
					id:  "abc",
					Key: "abc",
				},
			}}},
			101,
			nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedVariation, test.experiment.findBucket(test.bucketValue))
		})
	}
}

func TestProject_GetVariation(t *testing.T) {
	tests := []struct {
		name                   string
		project                Project
		experimentName, userID string
		expectedImpression     *Impression
		shouldCache            bool
	}{
		{
			"no experiment with name in project returns nil",
			Project{experiments: map[string]Experiment{
				"a": {},
			}},
			"b",
			"don't care",
			nil,
			false,
		}, {
			"experiment not running returns nil",
			Project{experiments: map[string]Experiment{
				"a": {status: "disabled"},
			}},
			"a",
			"don't care",
			nil,
			false,
		}, {
			"user in forced variation returns forced variation",
			Project{experiments: map[string]Experiment{
				"a": {
					status: runningStatus,
					forcedVariations: map[string]Variation{
						"user": {id: "abc", Key: "abc"},
					},
				},
			}},
			"a",
			"user",
			&Impression{Variation: Variation{id: "abc", Key: "abc"}, UserID: "user"},
			false,
		}, {
			"user found in cached variations returns cached variation",
			Project{experiments: map[string]Experiment{
				"a": {
					status:           runningStatus,
					forcedVariations: map[string]Variation{},
					cachedVariations: map[string]Variation{
						"user": {id: "abc", Key: "abc"},
					},
					mutex: &sync.RWMutex{},
				},
			}},
			"a",
			"user",
			&Impression{Variation: Variation{id: "abc", Key: "abc"}, UserID: "user"},
			true,
		}, {
			"user is bucketed into experiment",
			Project{experiments: map[string]Experiment{
				"a": {
					status:           runningStatus,
					forcedVariations: map[string]Variation{},
					trafficAllocation: []trafficAllocation{{
						endOfRange: maxTrafficValue,
						Variation:  Variation{id: "abc", Key: "abc"},
					}},
					cachedVariations: map[string]Variation{},
					mutex:            &sync.RWMutex{},
				},
			}},
			"a",
			"user",
			&Impression{Variation: Variation{id: "abc", Key: "abc"}, UserID: "user"},
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.project.GetVariation(test.experimentName, test.userID)
			if test.expectedImpression != nil {
				// make sure that the result timestamp is plausible, then overwrite with the zero time to
				// assert the rest of the result struct is valid
				now := time.Now()
				assert.InDelta(t, now.Nanosecond(), result.Timestamp.Nanosecond(), float64(100*time.Millisecond))
				test.expectedImpression.Timestamp = result.Timestamp
			}
			assert.Equal(t, test.expectedImpression, result)
			if test.shouldCache {
				assert.Contains(t, test.project.experiments[test.experimentName].cachedVariations, test.userID)
			}
		})
	}
}

func TestGetVariation(t *testing.T) {
	tests := []struct {
		name              string
		ctx               context.Context
		experimentName    string
		expectedVariation Variation
	}{
		{
			"impression returned from project stored in context",
			Project{
				experiments: map[string]Experiment{
					"a": {
						status: runningStatus,
						forcedVariations: map[string]Variation{
							"user": {id: "abc", Key: "abc"},
						},
					},
				}}.ToContext(context.Background(), "user"),
			"a",
			Variation{id: "abc", Key: "abc"},
		}, {
			"no project stored in context returns nil",
			context.Background(),
			"",
			Variation{},
		}, {
			"nil variation returns nil",
			Project{experiments: map[string]Experiment{
				"a": {status: "disabled"},
			}}.ToContext(context.Background(), "user"),
			"",
			Variation{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := GetVariation(test.ctx, test.experimentName)
			assert.Equal(t, test.expectedVariation, result)
			if result.Key != "" {
				assert.Len(t, test.ctx.Value(projCtxKey).(*projectContext).impressions, 1)
			}
		})
	}
}
