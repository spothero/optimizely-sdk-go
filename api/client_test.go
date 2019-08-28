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
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		options  []func(*client)
		expected Client
	}{
		{
			"default client has no token and requests 25 records per page",
			[]func(*client){},
			client{apiClient: optimizelyAPIClient{perPage: 25}},
		}, {
			"token and per page are set when provided as options",
			[]func(*client){Token("abc"), PerPage(10)},
			client{apiClient: optimizelyAPIClient{token: "abc", perPage: 10}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, NewClient(test.options...))
		})
	}
}

type mockTransport struct{ mock.Mock }

func (m *mockTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	call := m.Called(request)
	return call.Get(0).(*http.Response), call.Error(1)
}

func TestOptimizelyAPIClient_sendAPIRequest(t *testing.T) {
	tests := []struct {
		name                  string
		method, path          string
		body                  io.Reader
		additionalQueryParams url.Values
		additionalHeaders     http.Header
		response              *http.Response
		httpErr               error
		expectErr             bool
		expectRequestSent     bool
	}{
		{
			"api request is sent",
			http.MethodPost,
			"https://fake.url",
			strings.NewReader("request body"),
			url.Values{"query": []string{"abc"}},
			http.Header{"header": []string{"abc"}},
			&http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(strings.NewReader("response body"))},
			nil,
			false,
			true,
		}, {
			"error creating request returns error",
			string(rune(2)), // a rune is an invalid method that will fail creating a request
			"https://fake.url",
			nil,
			nil,
			nil,
			nil,
			nil,
			true,
			false,
		}, {
			"error making request returns error",
			http.MethodGet,
			"https://fake.url",
			nil,
			nil,
			nil,
			nil,
			fmt.Errorf("http error"),
			true,
			true,
		}, {
			"non-200 status code returns error",
			http.MethodGet,
			"https://fake.url",
			nil,
			nil,
			nil,
			&http.Response{StatusCode: http.StatusInternalServerError},
			nil,
			true,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mt := &mockTransport{}
			client := optimizelyAPIClient{
				Client:  http.Client{Transport: mt},
				token:   "token",
				perPage: 5,
			}
			if test.expectRequestSent {
				mt.On("RoundTrip", mock.Anything).Return(test.response, test.httpErr).Once()
				defer mt.AssertExpectations(t)
				defer func() {
					sentRequest := mt.Calls[0].Arguments[0].(*http.Request)
					requestedPerPage, err := strconv.Atoi(sentRequest.URL.Query().Get("per_page"))
					require.NoError(t, err)
					assert.Equal(t, client.perPage, requestedPerPage)
					assert.Equal(t, fmt.Sprintf("Bearer %s", client.token), sentRequest.Header.Get("Authorization"))
					for queryName, queryVal := range test.additionalQueryParams {
						assert.Equal(t, queryVal[0], sentRequest.URL.Query().Get(queryName))
					}
					for headerName, headerVal := range test.additionalHeaders {
						assert.Equal(t, headerVal[0], sentRequest.Header.Get(headerName))
					}
				}()
			}
			response, err := client.sendAPIRequest(
				test.method, test.path, test.body, test.additionalQueryParams, test.additionalHeaders)
			if test.expectErr {
				assert.Error(t, err)
				assert.Nil(t, response)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.response, response)
		})
	}
}

func TestOptimizelyAPIClient_sendPaginatedAPIRequest(t *testing.T) {
	type mockApiResponse struct {
		requestURL string
		response   *http.Response
		err        error
	}
	tests := []struct {
		name      string
		responses []mockApiResponse
		expectErr bool
	}{
		{
			"multiple api pages are requested and all responses returned",
			[]mockApiResponse{
				{
					"https://fake.url",
					&http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Link": []string{"<https://fake.url?page=2>; rel=\"next\""}},
					},
					nil,
				}, {
					"https://fake.url?page=2",
					&http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Link": []string{"<https://fake.url?page=3>; rel=\"next\""}},
					},
					nil,
				}, {
					"https://fake.url?page=3",
					&http.Response{
						StatusCode: http.StatusOK,
					},
					nil,
				},
			},
			false,
		}, {
			"error with one response returns error",
			[]mockApiResponse{
				{
					"https://fake.url",
					&http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Link": []string{"<https://fake.url?page=2>; rel=\"next\""}},
					},
					nil,
				}, {
					"https://fake.url?page=2",
					nil,
					fmt.Errorf("http error"),
				},
			},
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mt := &mockTransport{}
			expectedResponses := make([]*http.Response, 0, len(test.responses))
			for _, resp := range test.responses {
				req, err := http.NewRequest(http.MethodGet, resp.requestURL, nil)
				require.NoError(t, err)
				mt.On("RoundTrip", req).Return(resp.response, resp.err).Once()
				expectedResponses = append(expectedResponses, resp.response)
			}
			defer mt.AssertExpectations(t)
			client := optimizelyAPIClient{Client: http.Client{Transport: mt}}
			responses, err := client.sendPaginatedAPIRequest(http.MethodGet, test.responses[0].requestURL, nil, nil, nil)
			if test.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, expectedResponses, responses)
		})
	}
}
