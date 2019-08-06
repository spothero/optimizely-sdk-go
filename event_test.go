package optimizely

import (
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
					Key:       "campaign_activated",
					Timestamp: int64(10 * time.Second / time.Millisecond),
				}},
			}},
		},
		impression.toVisitor(),
	)
}

func TestNewEvents(t *testing.T) {
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
				EnrichDecisions(true),
				ClientName("client"),
				ClientVersion("version"),
				AnonynmizeIP(true),
			},
			Events{
				AccountID:       "account",
				AnonymizeIP:     true,
				ClientName:      "client",
				ClientVersion:   "version",
				EnrichDecisions: true,
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
								Key:       "campaign_activated",
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
								Key:       "campaign_activated",
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
			assert.Equal(t, test.expectedEvents.AccountID, events.AccountID)
			assert.Equal(t, test.expectedEvents.AnonymizeIP, events.AnonymizeIP)
			assert.Equal(t, test.expectedEvents.ClientName, events.ClientName)
			assert.Equal(t, test.expectedEvents.ClientVersion, events.ClientVersion)
			assert.Equal(t, test.expectedEvents.EnrichDecisions, events.EnrichDecisions)
			assert.Equal(t, len(test.expectedEvents.Visitors), len(events.Visitors))
			for i := range test.expectedEvents.Visitors {
				assertVisitorEqual(t, test.expectedEvents.Visitors[i], events.Visitors[i])
			}
		})
	}
}
