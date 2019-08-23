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

// DatafileExperiment is the structure of the experiment within a datafile. This
// type is only used when deserializing the datafile.
type DatafileExperiment struct {
	ID                string                      `json:"id"`
	Key               string                      `json:"key"`
	LayerID           string                      `json:"layerId"`
	Status            string                      `json:"status"`
	Variations        []DatafileVariation         `json:"variations"`
	TrafficAllocation []DatafileTrafficAllocation `json:"trafficAllocation"`
	ForcedVariations  map[string]string           `json:"forcedVariations"`
}

// DatafileVariation is an experiment variation within a datafile used for deserialization.
type DatafileVariation struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

// DatafileTrafficAllocation is the structure of the traffic allocation with a datafile. This type
// is only used when deserializing the datafile.
type DatafileTrafficAllocation struct {
	EntityID   string `json:"entityId"`
	EndOfRange int    `json:"endOfRange"`
}

// Datafile used for loading the JSON datafile from Optimizely
type Datafile struct {
	Version     string               `json:"version"`
	Revision    string               `json:"revision"`
	ProjectID   string               `json:"projectId"`
	AccountID   string               `json:"accountId"`
	Experiments []DatafileExperiment `json:"experiments"`
}

// NewProjectFromDataFile creates a new Optimizely project given the raw JSON datafile
func NewProjectFromDataFile(datafileJSON []byte) (Project, error) {
	df := Datafile{}
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

// type used to place the project within context.Context
type ctxKey int

// the value used to place the project within context.Context
const projCtxKey ctxKey = iota

type projectContext struct {
	Project
	userID      string
	impressions []Impression
	mutex       sync.Mutex
}

// ToContext creates a context with the project as a value in the context for
// a specific user ID. By using GetVariation with the context returned from
// this method, not only will each Impression be returned to the caller, but
// each Impression will be recorded in the context. Once the lifecycle of the
// context is complete, use EventsFromContext to create a unified Events object
// containing every impression that occurred during the context's lifecycle.
// This provides simplified API for bucketing a user across multiple experiments
// and multiple code-paths.
func (p Project) ToContext(ctx context.Context, userID string) context.Context {
	projectCtx := &projectContext{
		Project:     p,
		userID:      userID,
		impressions: make([]Impression, 0),
	}
	return context.WithValue(ctx, projCtxKey, projectCtx)
}
