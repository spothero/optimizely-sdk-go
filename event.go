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
	"fmt"
	"time"

	"github.com/google/uuid"
)

type event struct {
	EntityID  string `json:"entity_id"`
	Key       string `json:"key"`
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	UUID      string `json:"uuid"`
}

type decision struct {
	CampaignID   string `json:"campaign_id"`
	ExperimentID string `json:"experiment_id"`
	VariationID  string `json:"variation_id"`
}

type snapshot struct {
	Decisions []decision `json:"decisions"`
	Events    []event    `json:"events"`
}

type visitor struct {
	ID        string     `json:"visitor_id"`
	Snapshots []snapshot `json:"snapshots"`
}

type eventBatch struct {
	AccountID       string    `json:"account_id"`
	AnonymizeIP     bool      `json:"anonymize_ip"`
	ClientName      string    `json:"client_name"`
	ClientVersion   string    `json:"client_version"`
	EnrichDecisions bool      `json:"enrich_decisions"`
	Visitors        []visitor `json:"visitors"`
}

// Events are reportable actions back to the Optimizely API. Currently only
// impression events are supported.
type Events eventBatch

// the default client name to report to Optimizely as well as
// the path of this package that will be searched for in the importing
// module's dependencies.
const packagePath = "github.com/spothero/optimizely-sdk-go"

// default version of this library to report to Optimizely. This will be set
// to the version of this library by default
var clientVersion = "unset"

// NewEvents constructs a set of reportable events from the provided options.
func NewEvents(options ...func(*Events) error) (Events, error) {
	events := Events{ClientName: packagePath, ClientVersion: clientVersion}
	for _, option := range options {
		if err := option(&events); err != nil {
			return Events{}, err
		}
	}
	if len(events.Visitors) == 0 {
		return Events{}, fmt.Errorf("cannot build event with no activated variations")
	}
	return events, nil
}

// ActivatedImpression adds the variation impression to the set of reported events. Note that
// while many impressions can be added as events, each impression must have originated from
// the same Optimizely account or an error will be returned while creating the events.
func ActivatedImpression(v Impression) func(*Events) error {
	return func(e *Events) error {
		if e.AccountID == "" {
			e.AccountID = v.experiment.project.AccountID
		} else if e.AccountID != v.experiment.project.AccountID {
			return fmt.Errorf("activated variations must all be in the same account")
		}
		e.Visitors = append(e.Visitors, v.toVisitor())
		return nil
	}
}

// EnrichDecisions sets the enrich decisions property on the events.
func EnrichDecisions(enrich bool) func(*Events) error {
	return func(e *Events) error {
		e.EnrichDecisions = enrich
		return nil
	}
}

// EnrichDecisions sets the client name property on the events. By default,
// the client name will be set to the path of this library, i.e.
// github.com/spothero/optimizely-sdk-go.
func ClientName(name string) func(*Events) error {
	return func(e *Events) error {
		e.ClientName = name
		return nil
	}
}

// ClientVersion overrides the client version of this library. If using Go 1.12+
// and Go modules, the version of this library will be extracted from the build
// information. Otherwise, unless ClientVersion is set, the version reported
// to Optimizely will be "unset".
func ClientVersion(version string) func(*Events) error {
	return func(e *Events) error {
		e.ClientVersion = version
		return nil
	}
}

// AnonymizeIP sets the anonymize IP flag on the events.
func AnonynmizeIP(anonymize bool) func(*Events) error {
	return func(e *Events) error {
		e.AnonymizeIP = anonymize
		return nil
	}
}

const impressionEvent = "campaign_activated"

// toVisitor converts an impression to the visitor data structure for sending
// to the Optimizely API.
func (v Impression) toVisitor() visitor {
	dec := decision{
		CampaignID:   v.experiment.layerID,
		ExperimentID: v.experiment.id,
		VariationID:  v.id,
	}
	ev := event{
		EntityID:  v.experiment.layerID,
		Type:      impressionEvent,
		Key:       impressionEvent,
		Timestamp: v.Timestamp.UTC().UnixNano() / int64(time.Millisecond/time.Nanosecond),
		UUID:      uuid.New().String(),
	}
	return visitor{
		ID: v.UserID,
		Snapshots: []snapshot{{
			Decisions: []decision{dec},
			Events:    []event{ev},
		}},
	}
}
