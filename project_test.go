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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProjectFromDataFile(t *testing.T) {
	tests := []struct {
		name               string
		datafile           []byte
		genExpectedProject func(datafile []byte) Project
		expectError        bool
	}{
		{
			"project is created from datafile",
			[]byte(`
{
  "version": "4",
  "projectId": "1234",
  "unused_key": "zzz",
  "accountId": "00001",
  "revision": "666",
  "experiments": [
    {
      "status": "Running",
      "variations": [
        {
          "id": "abc123",
          "key": "variation_1"
        },
        {
          "id": "def456",
          "key": "variation_2"
        }
      ],
      "id": "5678",
      "key": "an_experiment",
      "layerId": "layer",
      "trafficAllocation": [
        {
          "entityId": "abc123",
          "endOfRange": 3000
        },
        {
          "entityId": "def456",
          "endOfRange": 10000
        }
      ],
      "forcedVariations": {
        "xyz": "variation_1",
        "abc": "variation_2"
      }
    }
  ]
}
`),
			func(datafile []byte) Project {
				proj := Project{
					Version:     "4",
					Revision:    "666",
					ProjectID:   "1234",
					AccountID:   "00001",
					RawDataFile: datafile,
				}
				exp := Experiment{
					id:               "5678",
					Key:              "an_experiment",
					layerID:          "layer",
					status:           "Running",
					cachedVariations: map[string]Variation{},
					mutex:            &sync.RWMutex{},
					project:          &proj,
				}
				var1 := Variation{
					id:         "abc123",
					Key:        "variation_1",
					experiment: &exp,
				}
				var2 := Variation{
					id:         "def456",
					Key:        "variation_2",
					experiment: &exp,
				}
				exp.trafficAllocation = []trafficAllocation{
					{endOfRange: 3000, Variation: var1},
					{endOfRange: 10000, Variation: var2},
				}
				exp.forcedVariations = map[string]Variation{"xyz": var1, "abc": var2}
				proj.experiments = map[string]Experiment{"an_experiment": exp}
				return proj
			},
			false,
		},
		{
			"error on unsupported datafile version",
			[]byte(`{"version": "3"}`),
			func(_ []byte) Project { return Project{} },
			true,
		}, {
			"forced variation without variation present ignores forced variation",
			[]byte(`
{
  "version": "4",
  "experiments": [
    {
      "variations": [
        {
          "id": "abc123",
          "key": "variation_1"
        }
      ],
      "trafficAllocation": [],
      "forcedVariations": {
        "abc": "variation_2"
      }
    }
  ]
}
`),
			func(datafile []byte) Project {
				proj := Project{
					Version:     "4",
					RawDataFile: datafile,
				}
				exp := Experiment{
					forcedVariations:  map[string]Variation{},
					trafficAllocation: []trafficAllocation{},
					cachedVariations:  map[string]Variation{},
					mutex:             &sync.RWMutex{},
					project:           &proj,
				}
				proj.experiments = map[string]Experiment{"": exp}
				return proj
			},
			false,
		}, {
			"malformed JSON results in an error",
			[]byte("{"),
			func(_ []byte) Project { return Project{} },
			true,
		}, {
			"unknown variation in traffic allocation returns error",
			[]byte(`
{
  "version": "4",
  "experiments": [
    {
      "status": "Running",
      "variations": [],
      "id": "5678",
      "key": "an_experiment",
      "layerId": "layer",
      "trafficAllocation": [
        {
          "entityId": "abc123",
          "endOfRange": 3000
        }
      ],
      "forcedVariations": {}
    }
  ]
}
`),
			func(_ []byte) Project { return Project{} },
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			project, err := NewProjectFromDataFile(test.datafile)
			expectedProject := test.genExpectedProject(test.datafile)
			if test.expectError {
				assert.Error(t, err)
				return
			}
			assert.Equal(t, expectedProject, project)
			assert.NoError(t, err)
		})
	}
}
