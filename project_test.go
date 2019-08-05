package optimizely

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProjectFromDataFile(t *testing.T) {
	tests := []struct {
		name            string
		datafile        []byte
		expectedProject Project
		expectError     bool
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
			Project{
				Version:   "4",
				Revision:  "666",
				ProjectID: "1234",
				AccountID: "00001",
				experiments: map[string]Experiment{
					"an_experiment": {
						id:      "5678",
						Key:     "an_experiment",
						layerID: "layer",
						status:  "Running",
						trafficAllocation: []trafficAllocation{
							{
								endOfRange: 3000,
								Variation: &Variation{
									id:  "abc123",
									Key: "variation_1",
								},
							}, {
								endOfRange: 10000,
								Variation: &Variation{
									id:  "def456",
									Key: "variation_2",
								},
							},
						},
						forcedVariations: map[string]*Variation{
							"xyz": {
								id:  "abc123",
								Key: "variation_1",
							},
							"abc": {
								id:  "def456",
								Key: "variation_2",
							},
						},
						cachedVariations: map[string]*Variation{},
						mutex:            &sync.RWMutex{},
					},
				},
			},
			false,
		}, {
			"error on unsupported datafile version",
			[]byte(`{"version": "3"}`),
			Project{},
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
			Project{
				Version: "4",
				experiments: map[string]Experiment{
					"": {
						forcedVariations:  map[string]*Variation{},
						trafficAllocation: []trafficAllocation{},
						cachedVariations:  map[string]*Variation{},
						mutex:             &sync.RWMutex{},
					},
				},
			},
			false,
		}, {
			"malformed JSON results in an error",
			[]byte("{"),
			Project{},
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
			Project{},
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			project, err := NewProjectFromDataFile(test.datafile)
			test.expectedProject.RawDataFile = test.datafile
			if test.expectError {
				assert.Error(t, err)
				return
			}
			assert.Equal(t, test.expectedProject, project)
			assert.NoError(t, err)
		})
	}
}
