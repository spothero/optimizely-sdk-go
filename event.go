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
	"time"

	"github.com/google/uuid"
	"github.com/spothero/optimizely-sdk-go/api"
	"golang.org/x/xerrors"
)

type event struct {
	EntityID  string `json:"entity_id"`
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
	ClientVersion   *string   `json:"client_version,omitempty"`
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

// Version of this library to report to Optimizely. If unset and the version
// cannot be pulled out of the Go module info, it will not be sent.
var clientVersion = ""

// NewEvents constructs a set of reportable events from the provided options.
func NewEvents(options ...func(*Events) error) (Events, error) {
	events := Events{
		ClientName:      packagePath,
		ClientVersion:   &clientVersion,
		AnonymizeIP:     true,
		EnrichDecisions: true,
	}
	for _, option := range options {
		if err := option(&events); err != nil {
			return Events{}, err
		}
	}
	if *events.ClientVersion == "" {
		events.ClientVersion = nil
	}
	if len(events.Visitors) == 0 {
		return Events{}, fmt.Errorf("cannot build event with no activated variations")
	}
	return events, nil
}

// ActivatedImpression adds the variation impression to the set of reported events. Note that
// while many impressions can be added as events, each impression must have originated from
// the same Optimizely account or an error will be returned while creating the events.
func ActivatedImpression(i Impression) func(*Events) error {
	return func(e *Events) error {
		if e.AccountID == "" {
			e.AccountID = i.experiment.project.AccountID
		} else if e.AccountID != i.experiment.project.AccountID {
			return fmt.Errorf("activated variations must all be in the same account")
		}
		e.Visitors = append(e.Visitors, i.toVisitor())
		return nil
	}
}

// EnrichDecisions sets the enrich decisions property on the events. Defaults to true.
func EnrichDecisions(enrich bool) func(*Events) error {
	return func(e *Events) error {
		e.EnrichDecisions = enrich
		return nil
	}
}

// ClientName sets the client name property on the events. By default,
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
// information. Otherwise, unless ClientVersion is set here, no version will
// be reported to Optimizely.
func ClientVersion(version string) func(*Events) error {
	return func(e *Events) error {
		e.ClientVersion = &version
		return nil
	}
}

// AnonymizeIP sets the anonymize IP flag on the events. Defaults to true.
func AnonymizeIP(anonymize bool) func(*Events) error {
	return func(e *Events) error {
		e.AnonymizeIP = anonymize
		return nil
	}
}

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
		Type:      "campaign_activated",
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

// EventsFromContext creates Events from all the impressions that were seen
// during the lifecycle of the provided context. If no impressions were seen
// or no project was found in the provided context, nil is returned.
// The options provided to this function match the options provided to
// NewEvents with the exception that the ActivatedImpression function
// should never be provided as an option and may result in a panic if
// the provided impression was created by a project in a different account from
// the project stored in the context.
func EventsFromContext(ctx context.Context, options ...func(*Events) error) *Events {
	projectCtx, ok := ctx.Value(projCtxKey).(*projectContext)
	if !ok {
		return nil
	}
	projectCtx.mutex.Lock()
	defer projectCtx.mutex.Unlock()
	if len(projectCtx.impressions) == 0 {
		return nil
	}
	for _, impression := range projectCtx.impressions {
		options = append(options, ActivatedImpression(impression))
	}
	// There can never be an error here when this API is used correctly because
	// there are only two cases that can cause an error: no impressions, and
	// impressions from different projects. We know that there are impressions
	// because the case of no impressions is handled above, and we know that all
	// impressions are from the same project because they had to be inserted
	// into the context by the same project. Thus, the only way an error
	// can occur here is if the API is misused and an impression from
	// a different project was passed as an additional option to this
	// function.
	events, err := NewEvents(options...)
	if err != nil {
		panic(err)
	}

	// reset impressions in case the project context gets reused
	projectCtx.impressions = make([]Impression, 0)

	return &events
}

// ReportEvents is a convenience wrapper for sending events to the Optimizely reporting API that marshals
// the events to JSON and calls the api package.
//
// Note: The provided client does not necessarily
// have to be instantiated with a token as the events endpoint does not require one.
func ReportEvents(client api.Client, events Events) error {
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return xerrors.Errorf("error marshaling events to JSON: %w", err)
	}
	// the events endpoint does not require auth nor take any other parameters so just use the empty API client
	return client.ReportEvents(eventsJSON)
}
