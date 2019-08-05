package optimizely

import (
	"fmt"
	"sync"
	"testing"

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
				Variation: &Variation{
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
				Variation: &Variation{
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
		expectedVariation      *Variation
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
					forcedVariations: map[string]*Variation{
						"user": {id: "abc", Key: "abc"},
					},
				},
			}},
			"a",
			"user",
			&Variation{id: "abc", Key: "abc"},
			false,
		}, {
			"user found in cached variations returns cached variation",
			Project{experiments: map[string]Experiment{
				"a": {
					status:           runningStatus,
					forcedVariations: map[string]*Variation{},
					cachedVariations: map[string]*Variation{
						"user": {id: "abc", Key: "abc"},
					},
					mutex: &sync.RWMutex{},
				},
			}},
			"a",
			"user",
			&Variation{id: "abc", Key: "abc"},
			true,
		}, {
			"user is bucketed into experiment",
			Project{experiments: map[string]Experiment{
				"a": {
					status:           runningStatus,
					forcedVariations: map[string]*Variation{},
					trafficAllocation: []trafficAllocation{{
						endOfRange: maxTrafficValue,
						Variation:  &Variation{id: "abc", Key: "abc"},
					}},
					cachedVariations: map[string]*Variation{},
					mutex:            &sync.RWMutex{},
				},
			}},
			"a",
			"user",
			&Variation{id: "abc", Key: "abc"},
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedVariation, test.project.GetVariation(test.experimentName, test.userID))
			if test.shouldCache {
				assert.Contains(t, test.project.experiments[test.experimentName].cachedVariations, test.userID)
			}
		})
	}
}
