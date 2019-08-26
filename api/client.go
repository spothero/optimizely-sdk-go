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

package api

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/tomnomnom/linkheader"
	"golang.org/x/xerrors"
)

// client is the structure used for interacting with the Optimizely API. This type fulfills both the
// apiClient and Client interfaces.
type client struct {
	httpClient http.Client
	apiClient  apiClient
	token      string
	perPage    int
}

type apiClient interface {
	sendAPIRequest(method, url string, body io.Reader, query url.Values, headers http.Header) (*http.Response, error)
	sendPaginatedAPIRequest(method, url string, body io.Reader, query url.Values, headers http.Header) ([]*http.Response, error)
}

// NewClient constructs a new Optimizely API client from optional provided options.
func NewClient(options ...func(*client)) Client {
	c := client{perPage: 25}
	for _, option := range options {
		option(&c)
	}
	return c
}

// Token provides the Optimizely API token as an option when building a new Client.
func Token(t string) func(*client) {
	return func(c *client) {
		c.token = t
	}
}

// PerPage sets the requested number of items to return on each request to the optimizely API as an option when
// building a new Client. If this option is not provided to NewClient, the default value is 25 items per page.
func PerPage(i int) func(*client) {
	return func(c *client) {
		c.perPage = i
	}
}

// sends a single API request to the Optimizely API and returns the response or error. If the response is a non-200
// level response, an error is also returned.
func (c client) sendAPIRequest(method, uri string, body io.Reader, query url.Values, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, xerrors.Errorf("error creating Optimizely API request: %w", err)
	}
	// merge the provided query into the request's query
	q := req.URL.Query()
	for k, v := range query {
		for _, s := range v {
			q.Add(k, s)
		}
	}
	// append per_page to the query
	if c.perPage > 0 {
		q.Set("per_page", fmt.Sprintf("%d", c.perPage))
	}
	req.URL.RawQuery = q.Encode()

	// merge provided headers into the request's headers
	for k, v := range headers {
		for _, s := range v {
			req.Header.Add(k, s)
		}
	}
	// append authorization header if token is not empty
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, xerrors.Errorf("error making Optimizely API request: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, xerrors.Errorf("received %d status from Optimizely API", resp.StatusCode)
	}
	return resp, nil
}

// sends a request to the Optimizely API and follows all pagination links and aggregates the responses.
func (c client) sendPaginatedAPIRequest(method, uri string, body io.Reader, query url.Values, headers http.Header) ([]*http.Response, error) {
	responses := make([]*http.Response, 0, 1)
	curURL := uri
	for {
		resp, err := c.sendAPIRequest(method, curURL, body, query, headers)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
		links := linkheader.Parse(resp.Header.Get("link"))
		next := links.FilterByRel("next")
		if len(next) == 0 {
			return responses, nil
		}
		curURL = next[0].URL
	}
}
