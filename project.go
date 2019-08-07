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
	"encoding/json"
	"fmt"
	"sync"
)

// only version 4 of the datafile is currently supported
const supportedDatafileVersion = "4"

// Project is an Optimizely project containing a set of experiments. Project also includes
// the raw JSON datafile which was used to generate the Project.
type Project struct {
	Version     string
	Revision    string
	ProjectID   string
	AccountID   string
	experiments map[string]Experiment
	RawDataFile json.RawMessage
}

// Experiment represents a single Optimizely experiment. It contains metadata
// as well as the traffic allocation for the experiment and any forced variations.
type Experiment struct {
	Key               string
	id                string
	layerID           string
	status            string
	trafficAllocation []trafficAllocation
	forcedVariations  map[string]Variation
	mutex             *sync.RWMutex
	cachedVariations  map[string]Variation
	project           *Project // backref to the owning project
}

// Variation represents a variation of an Optimizely experiment.
type Variation struct {
	id         string
	Key        string
	experiment *Experiment // backref to the owning experiment
}

// trafficAllocation defines the value of traffic to direct to a particular experiment variation.
type trafficAllocation struct {
	endOfRange int
	Variation  Variation
}

// datafileExperiment is the structure of the experiment within a datafile. This
// type is only used when deserializing the datafile.
type datafileExperiment struct {
	ID                string                      `json:"id"`
	Key               string                      `json:"key"`
	LayerID           string                      `json:"layerId"`
	Status            string                      `json:"status"`
	Variations        []datafileVariation         `json:"variations"`
	TrafficAllocation []datafileTrafficAllocation `json:"trafficAllocation"`
	ForcedVariations  map[string]string           `json:"forcedVariations"`
}

// datafileVariation is an experiment variation within a datafile used for deserialization.
type datafileVariation struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

// datafileTrafficAllocation is the structure of the traffic allocation with a datafile. This type
// is only used when deserializing the datafile.
type datafileTrafficAllocation struct {
	EntityID   string `json:"entityId"`
	EndOfRange int    `json:"endOfRange"`
}

// datafile used for loading the JSON datafile from Optimizely
type datafile struct {
	Version     string               `json:"version"`
	Revision    string               `json:"revision"`
	ProjectID   string               `json:"projectId"`
	AccountID   string               `json:"accountId"`
	Experiments []datafileExperiment `json:"experiments"`
}

// NewProjectFromDataFile creates a new Optimizely project given the raw JSON datafile
func NewProjectFromDataFile(datafileJSON []byte) (Project, error) {
	df := datafile{}
	if err := json.Unmarshal(datafileJSON, &df); err != nil {
		return Project{}, err
	}
	if df.Version != supportedDatafileVersion {
		return Project{}, fmt.Errorf("could not create project from unsupported datafile version %v", df.Version)
	}

	project := Project{
		Version:     df.Version,
		Revision:    df.Revision,
		ProjectID:   df.ProjectID,
		AccountID:   df.AccountID,
		RawDataFile: datafileJSON,
	}

	// convert list of experiments in the datafile to a map of experiments for faster lookup
	experiments := make(map[string]Experiment, len(df.Experiments))
	for _, exp := range df.Experiments {
		experiment := Experiment{
			id:               exp.ID,
			Key:              exp.Key,
			layerID:          exp.LayerID,
			status:           exp.Status,
			cachedVariations: make(map[string]Variation),
			mutex:            &sync.RWMutex{},
			project:          &project,
		}
		// store variations by their ID, but keep track by key for constructing the force variations map later
		variationsByID := make(map[string]Variation, len(exp.Variations))
		variationsByKey := make(map[string]Variation, len(exp.Variations))
		for _, v := range exp.Variations {
			variation := Variation{
				id:         v.ID,
				Key:        v.Key,
				experiment: &experiment,
			}
			variationsByID[v.ID] = variation
			variationsByKey[v.Key] = variation
		}

		ta := make([]trafficAllocation, 0, len(exp.TrafficAllocation))
		for _, a := range exp.TrafficAllocation {
			variation, ok := variationsByID[a.EntityID]
			if !ok {
				return Project{}, fmt.Errorf("unknown variation ID %v found in traffic allocation", a.EntityID)
			}
			ta = append(
				ta,
				trafficAllocation{
					endOfRange: a.EndOfRange,
					Variation:  variation,
				},
			)
		}
		experiment.trafficAllocation = ta

		forcedVariations := make(map[string]Variation, len(exp.ForcedVariations))
		for userID, variationName := range exp.ForcedVariations {
			variation, ok := variationsByKey[variationName]
			if !ok {
				continue
			}
			forcedVariations[userID] = variation
		}
		experiment.forcedVariations = forcedVariations
		experiments[experiment.Key] = experiment
	}
	project.experiments = experiments

	return project, nil
}
