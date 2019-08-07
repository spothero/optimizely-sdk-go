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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockTransport struct {
	mock.Mock
}

func (m *mockTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	call := m.Called(request)
	return call.Get(0).(*http.Response), call.Error(1)
}

func TestReporter_reportEvents(t *testing.T) {
	version := "version"
	tests := []struct {
		name         string
		events       Events
		expectedBody []byte
		response     *http.Response
		httpErr      error
		expectErr    bool
	}{
		{
			"events are sent to the Optimizely API",
			Events{
				AccountID:       "account",
				AnonymizeIP:     true,
				ClientName:      "client",
				ClientVersion:   &version,
				EnrichDecisions: true,
				Visitors: []visitor{{
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
							Timestamp: 10,
							UUID:      "uuid",
						}},
					}},
				}},
			},
			[]byte(`
{
  "account_id": "account",
  "anonymize_ip": true,
  "client_name": "client",
  "client_version": "version",
  "enrich_decisions": true,
  "visitors": [
    {
      "visitor_id": "user",
      "snapshots": [
        {
          "decisions": [
            {
              "campaign_id": "layer",
              "experiment_id": "experiment",
              "variation_id": "variation"
            }
          ],
          "events": [
            {
              "entity_id": "layer",
              "type": "campaign_activated",
              "timestamp": 10,
              "uuid": "uuid"
            }
          ]
        }
      ]
    }
  ]
}
`),
			&http.Response{StatusCode: http.StatusNoContent},
			nil,
			false,
		}, {
			"error POSTing to Optimizely returns error",
			Events{},
			[]byte{},
			nil,
			fmt.Errorf("something bad happened"),
			true,
		}, {
			"non-204 status code from Optimizely returns error",
			Events{},
			[]byte{},
			&http.Response{StatusCode: http.StatusBadRequest},
			nil,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mt := &mockTransport{}
			mt.On("RoundTrip", mock.Anything).Return(test.response, test.httpErr).Once()
			defer mt.AssertExpectations(t)
			err := reporter{http.Client{Transport: mt}}.reportEvents(test.events)
			if test.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			actualJSONBuf := bytes.Buffer{}
			_, err = actualJSONBuf.ReadFrom(mt.Calls[0].Arguments[0].(*http.Request).Body)
			require.NoError(t, err)
			// load expected and actual JSON and assert that they are equal so that
			// whitespace and key ordering doesn't matter
			var expectedJSONIface, actualJSONIface interface{}
			require.NoError(t, json.Unmarshal(test.expectedBody, &expectedJSONIface))
			require.NoError(t, json.Unmarshal(actualJSONBuf.Bytes(), &actualJSONIface))
			assert.Equal(t, expectedJSONIface, actualJSONIface)
		})
	}
}
