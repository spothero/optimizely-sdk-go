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
	"math"
	"time"

	"github.com/spaolacci/murmur3"
)

// status of an experiment that is in the running state
const runningStatus = "Running"

// max value of a traffic allocation; used as an upper bound for the bucketing hash
const maxTrafficValue = 10000

// value to seed the murmur hash algorithm with
const hashSeed = 1

// Impression is the outcome of bucketing a user into a specific variation. This type
// holds the variation that the user was bucketed into, the user ID that generated
// the outcome, and the timestamp at which the variation was generated.
type Impression struct {
	Variation
	UserID    string
	Timestamp time.Time
}

// GetVariation returns an impression, if applicable, for a given experiment
// and a given user id. If no variation is applicable, nil is returned. The
// Impression returned by this method can be used later to generate events
// for reporting to the Optimizely API.
func (p Project) GetVariation(experimentName, userID string) *Impression {
	experiment, ok := p.experiments[experimentName]
	if !ok {
		return nil
	}
	if experiment.status != runningStatus {
		return nil
	}
	timestamp := time.Now()
	forcedVariation, ok := experiment.forcedVariations[userID]
	if ok {
		return &Impression{
			Variation: forcedVariation,
			UserID:    userID,
			Timestamp: timestamp,
		}
	}
	experiment.mutex.RLock()
	cachedVariation, ok := experiment.cachedVariations[userID]
	experiment.mutex.RUnlock()
	if ok {
		return &Impression{
			Variation: cachedVariation,
			UserID:    userID,
			Timestamp: timestamp,
		}
	}
	variation := experiment.findBucket(experiment.getBucketValue(userID))
	experiment.mutex.Lock()
	defer experiment.mutex.Unlock()
	experiment.cachedVariations[userID] = *variation
	return &Impression{
		Variation: *variation,
		UserID:    userID,
		Timestamp: timestamp,
	}
}

// getBucketValue finds the value of the bucket given a unique ID (should be the user ID)
// using the murmur hash algorithm.
func (e Experiment) getBucketValue(bucketingID string) int {
	bucketingKey := fmt.Sprintf("%v%v", bucketingID, e.id)
	hashCode := murmur3.Sum32WithSeed([]byte(bucketingKey), hashSeed)
	ratio := float64(hashCode) / math.MaxUint32
	return int(math.Floor(ratio * maxTrafficValue))
}

// findBucket finds the variation from the experiment's traffic allocation given a bucketing value.
func (e Experiment) findBucket(bucketValue int) *Variation {
	for _, allocation := range e.trafficAllocation {
		if bucketValue < allocation.endOfRange {
			return &allocation.Variation
		}
	}
	return nil
}

// GetVariation returns the variation, if applicable, for the given experiment
// name from the project and user ID stored in the context. See
// Project.ToContext for more details.
func GetVariation(ctx context.Context, experimentName string) Variation {
	projectCtx, ok := ctx.Value(projCtxKey).(*projectContext)
	if !ok {
		return Variation{}
	}
	impression := projectCtx.GetVariation(experimentName, projectCtx.userID)
	if impression == nil {
		return Variation{}
	}
	projectCtx.mutex.Lock()
	defer projectCtx.mutex.Unlock()
	projectCtx.impressions = append(projectCtx.impressions, *impression)
	return impression.Variation
}
