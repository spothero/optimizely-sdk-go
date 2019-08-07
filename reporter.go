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
)

const eventsEndpoint = "https://logx.optimizely.com/v1/events"

// reporter contains an http client used for making API calls to Optimizely so
// that the http client can be stubbed out for testing.
type reporter struct {
	http.Client
}

func (r reporter) reportEvents(events Events) error {
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return err
	}
	response, err := r.Post(eventsEndpoint, "application/json", bytes.NewBuffer(eventsJSON))
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code (%d) received from events API", response.StatusCode)
	}
	return nil
}

// ReportEvents synchronously sends events to the Optimizely API for processing.
func ReportEvents(events Events) error {
	return reporter{}.reportEvents(events)
}
