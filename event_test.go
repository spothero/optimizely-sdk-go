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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ensure that the visitor objects are equal by checking that the UUID
// on each event of the actual visitor is valid, then copying the actual
// UUID to the expected UUID and checking for equality.
func assertVisitorEqual(t *testing.T, expected, actual visitor) {
	require.Equal(t, len(expected.Snapshots), len(actual.Snapshots))
	for i := range expected.Snapshots {
		expectedVisitor := expected.Snapshots[i]
		actualVisitor := actual.Snapshots[i]
		for j := range expectedVisitor.Events {
			actualEvent := actualVisitor.Events[j]
			_, err := uuid.Parse(actualEvent.UUID)
			assert.NoError(t, err)
			expected.Snapshots[i].Events[j].UUID = actualEvent.UUID
		}
	}
	assert.Equal(t, expected, actual)
}

func assertEventsEqual(t *testing.T, expected, actual Events) {
	assert.Equal(t, expected.AccountID, actual.AccountID)
	assert.Equal(t, expected.AnonymizeIP, actual.AnonymizeIP)
	assert.Equal(t, expected.ClientName, actual.ClientName)
	assert.Equal(t, expected.ClientVersion, actual.ClientVersion)
	assert.Equal(t, expected.EnrichDecisions, actual.EnrichDecisions)
	assert.Equal(t, len(expected.Visitors), len(actual.Visitors))
	for i := range expected.Visitors {
		assertVisitorEqual(t, expected.Visitors[i], actual.Visitors[i])
	}
}

func TestImpression_toVisitor(t *testing.T) {
	impression := Impression{
		Variation: Variation{
			id:  "variation",
			Key: "key",
			experiment: &Experiment{
				layerID: "layer",
				id:      "experiment",
			},
		},
		UserID:    "user",
		Timestamp: time.Unix(10, 0),
	}

	assertVisitorEqual(
		t,
		visitor{
			ID: "user",
			Snapshots: []snapshot{{
				Decisions: []decision{{
					CampaignID:   "layer",
					ExperimentID: "experiment",
					VariationID:  "variation",
				}},
				Events: []event{{
					EntityID:  "layer",
					Type:      "campaign_activated",
					Timestamp: int64(10 * time.Second / time.Millisecond),
				}},
			}},
		},
		impression.toVisitor(),
	)
}

func TestNewEvents(t *testing.T) {
	version := "version"
	tests := []struct {
		name           string
		options        []func(*Events) error
		expectedEvents Events
		expectError    bool
	}{
		{
			"events are created",
			[]func(*Events) error{
				ActivatedImpression(
					Impression{
						Variation: Variation{
							id:  "variation_id_1",
							Key: "variation_key_1",
							experiment: &Experiment{
								layerID: "layer_1",
								id:      "experiment_1",
								project: &Project{AccountID: "account"},
							},
						},
						UserID:    "user_1",
						Timestamp: time.Unix(10, 0),
					},
				),
				ActivatedImpression(
					Impression{
						Variation: Variation{
							id:  "variation_id_2",
							Key: "variation_key_2",
							experiment: &Experiment{
								layerID: "layer_2",
								id:      "experiment_2",
								project: &Project{AccountID: "account"},
							},
						},
						UserID:    "user_2",
						Timestamp: time.Unix(20, 0),
					},
				),
				EnrichDecisions(false),
				ClientName("client"),
				ClientVersion(version),
				AnonynmizeIP(false),
			},
			Events{
				AccountID:       "account",
				AnonymizeIP:     false,
				ClientName:      "client",
				ClientVersion:   &version,
				EnrichDecisions: false,
				Visitors: []visitor{
					{
						ID: "user_1",
						Snapshots: []snapshot{{
							Decisions: []decision{{
								CampaignID:   "layer_1",
								ExperimentID: "experiment_1",
								VariationID:  "variation_id_1",
							}},
							Events: []event{{
								EntityID:  "layer_1",
								Type:      "campaign_activated",
								Timestamp: int64(10 * time.Second / time.Millisecond),
							}},
						}},
					}, {
						ID: "user_2",
						Snapshots: []snapshot{{
							Decisions: []decision{{
								CampaignID:   "layer_2",
								ExperimentID: "experiment_2",
								VariationID:  "variation_id_2",
							}},
							Events: []event{{
								EntityID:  "layer_2",
								Type:      "campaign_activated",
								Timestamp: int64(20 * time.Second / time.Millisecond),
							}},
						}},
					},
				},
			},
			false,
		}, {
			"error returned when impressions are from different projects",
			[]func(*Events) error{
				ActivatedImpression(
					Impression{
						Variation: Variation{
							experiment: &Experiment{
								project: &Project{AccountID: "account"},
							},
						},
					},
				),
				ActivatedImpression(
					Impression{
						Variation: Variation{
							experiment: &Experiment{
								project: &Project{AccountID: "other account"},
							},
						},
					},
				),
			},
			Events{},
			true,
		}, {
			"error returned when there are no visitors",
			[]func(*Events) error{},
			Events{},
			true,
		}, {
			"unset client version sets version to nil",
			[]func(*Events) error{
				ActivatedImpression(
					Impression{
						Variation: Variation{
							experiment: &Experiment{
								project: &Project{AccountID: "account"},
							},
						},
						Timestamp: time.Unix(0, 0),
					},
				),
			},
			Events{
				ClientVersion:   nil,
				AccountID:       "account",
				ClientName:      "github.com/spothero/optimizely-sdk-go",
				AnonymizeIP:     true,
				EnrichDecisions: true,
				Visitors: []visitor{
					{
						Snapshots: []snapshot{{
							Decisions: []decision{{}},
							Events:    []event{{Type: "campaign_activated"}},
						}},
					},
				},
			},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events, err := NewEvents(test.options...)
			if test.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assertEventsEqual(t, test.expectedEvents, events)
		})
	}
}

func TestEventsFromContext(t *testing.T) {
	tests := []struct {
		name           string
		projectCtx     *projectContext
		options        []func(*Events) error
		expectedEvents *Events
		expectPanic    bool
	}{
		{
			"events pulled from impressions in context",
			&projectContext{
				impressions: []Impression{{
					Variation: Variation{experiment: &Experiment{project: &Project{}}},
					Timestamp: time.Unix(0, 0),
				}},
			},
			[]func(*Events) error{ClientName(""), AnonynmizeIP(false), EnrichDecisions(false)},
			&Events{
				Visitors: []visitor{{
					Snapshots: []snapshot{{
						Decisions: []decision{{}},
						Events:    []event{{Type: "campaign_activated"}},
					}},
				}},
			},
			false,
		}, {
			"no impressions returns nil",
			&projectContext{impressions: []Impression{}},
			[]func(*Events) error{},
			nil,
			false,
		}, {
			"improper usage with additional recorded impression from another account panics",
			&projectContext{
				impressions: []Impression{{
					Variation: Variation{experiment: &Experiment{project: &Project{AccountID: "account"}}},
				}},
			},
			[]func(*Events) error{
				ActivatedImpression(
					Impression{Variation: Variation{experiment: &Experiment{project: &Project{AccountID: "account_2"}}}},
				),
			},
			nil,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), projCtxKey, test.projectCtx)
			if test.expectPanic {
				assert.Panics(t, func() { EventsFromContext(ctx, test.options...) })
				return
			}
			result := EventsFromContext(ctx, test.options...)
			if test.expectedEvents == nil {
				assert.Nil(t, result)
				return
			}
			assertEventsEqual(t, *test.expectedEvents, *result)
			assert.Len(t, test.projectCtx.impressions, 0)
		})
	}
}
