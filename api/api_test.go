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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockClient struct {
	mock.Mock
}

func (m *mockClient) sendAPIRequest(method, url string, body io.Reader, query url.Values, headers http.Header) (*http.Response, error) {
	call := m.Called(method, url, body, query, headers)
	return call.Get(0).(*http.Response), call.Error(1)
}

func (m *mockClient) sendPaginatedAPIRequest(method, url string, body io.Reader, query url.Values, headers http.Header) ([]*http.Response, error) {
	call := m.Called(method, url, body, query, headers)
	return call.Get(0).([]*http.Response), call.Error(1)
}

func createMockClient(projectResponses []string, projectErr error, environmentResponses []string, environmentErr error, environmentProjectID int) (*mockClient, *mock.Call, *mock.Call) {
	mc := &mockClient{}
	prs := make([]*http.Response, 0, len(projectResponses))
	for _, body := range projectResponses {
		prs = append(prs, &http.Response{Body: ioutil.NopCloser(strings.NewReader(body))})
	}
	ers := make([]*http.Response, 0, len(environmentResponses))
	for _, body := range environmentResponses {
		ers = append(ers, &http.Response{Body: ioutil.NopCloser(strings.NewReader(body))})
	}
	var projectAPICall, environmentAPICall *mock.Call
	if len(projectResponses) > 0 {
		projectAPICall = mc.On(
			"sendPaginatedAPIRequest",
			http.MethodGet,
			fmt.Sprintf("%s/projects", baseURL),
			nil,
			url.Values(nil),
			http.Header(nil),
		).Return(
			prs, projectErr,
		)
	}
	if len(environmentResponses) > 0 {
		environmentAPICall = mc.On(
			"sendPaginatedAPIRequest",
			http.MethodGet,
			fmt.Sprintf("%s/environments", baseURL),
			nil,
			url.Values{"project_id": []string{fmt.Sprintf("%d", environmentProjectID)}},
			http.Header(nil),
		).Return(
			ers, environmentErr,
		)
	}
	return mc, projectAPICall, environmentAPICall
}

func TestClient_GetProjects(t *testing.T) {
	tests := []struct {
		name             string
		responseBodies   []string
		apiErr           error
		expectedProjects []Project
		expectErr        bool
	}{
		{
			"projects are retrieved from the api",
			[]string{`
[
  {
    "name": "Project",
    "description": "project description",
    "status": "active",
    "account_id": 12345,
    "created": "2019-08-21T20:37:12Z",
    "id": 1000,
    "last_modified": "2019-08-21T20:37:12Z"
  },
  {
    "name": "Project 2",
    "description": "project 2 description",
    "status": "active",
    "account_id": 12345,
    "created": "2019-08-21T20:37:12Z",
    "id": 2000,
    "last_modified": "2019-08-21T20:37:12Z"
  }
]
`, `
[
  {
    "name": "Project 3",
    "description": "project 3 description",
    "status": "active",
    "account_id": 12345,
    "created": "2019-08-21T20:37:12Z",
    "id": 3000,
    "last_modified": "2019-08-21T20:37:12Z"
  }
]
`,
			},
			nil,
			[]Project{
				{
					ID:           1000,
					Name:         "Project",
					Description:  "project description",
					Status:       "active",
					AccountID:    12345,
					Created:      time.Date(2019, 8, 21, 20, 37, 12, 0, time.UTC),
					LastModified: time.Date(2019, 8, 21, 20, 37, 12, 0, time.UTC),
				}, {
					ID:           2000,
					Name:         "Project 2",
					Description:  "project 2 description",
					Status:       "active",
					AccountID:    12345,
					Created:      time.Date(2019, 8, 21, 20, 37, 12, 0, time.UTC),
					LastModified: time.Date(2019, 8, 21, 20, 37, 12, 0, time.UTC),
				}, {
					ID:           3000,
					Name:         "Project 3",
					Description:  "project 3 description",
					Status:       "active",
					AccountID:    12345,
					Created:      time.Date(2019, 8, 21, 20, 37, 12, 0, time.UTC),
					LastModified: time.Date(2019, 8, 21, 20, 37, 12, 0, time.UTC),
				},
			},
			false,
		}, {
			"api error returns an error",
			[]string{""},
			fmt.Errorf("api error"),
			nil,
			true,
		}, {
			"error decoding json returns an error",
			[]string{"{"},
			nil,
			nil,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mc, projectAPICall, _ := createMockClient(test.responseBodies, test.apiErr, nil, nil, 0)
			defer mc.AssertExpectations(t)
			if projectAPICall != nil {
				projectAPICall.Once()
			}
			c := Client{mc}
			projects, err := c.GetProjects()
			if test.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.expectedProjects, projects)
		})
	}
}

func TestClient_GetEnvironmentsByProjectID(t *testing.T) {
	const projectID = 1
	tests := []struct {
		name                 string
		responseBodies       []string
		apiErr               error
		expectedEnvironments []Environment
		expectErr            bool
	}{
		{
			"environments are retrieved from the api",
			[]string{
				`
[
  {
    "id": 1,
    "key": "key",
    "name": "Staging",
    "project_id": 1,
    "archived": true,
    "description": "staging environment",
    "has_restricted_permissions": false,
    "created": "2019-08-21T20:37:12Z",
    "is_primary": false,
    "last_modified": "2019-08-21T20:37:12Z",
    "datafile": {
      "id": 1,
      "latest_file_size": 100,
      "other_urls": [
        "https://datafile.url"
      ],
      "revision": 1,
      "sdk_key": "abc123",
      "url": "https://datafile.url"
    }
  }
]
`, `
[
  {
    "id": 2,
    "key": "key 2",
    "name": "Production",
    "project_id": 1,
    "archived": false,
    "description": "production environment",
    "has_restricted_permissions": false,
    "created": "2019-08-21T20:37:12Z",
    "is_primary": true,
    "last_modified": "2019-08-21T20:37:12Z",
    "datafile": {
      "id": 2,
      "latest_file_size": 200,
      "other_urls": [
        "https://datafile.url"
      ],
      "revision": 2,
      "sdk_key": "abc123",
      "url": "https://datafile.url"
    }
  }
]
`},
			nil,
			[]Environment{
				{
					ID:                       1,
					Key:                      "key",
					Name:                     "Staging",
					ProjectID:                1,
					Archived:                 true,
					Description:              "staging environment",
					HasRestrictedPermissions: false,
					Created:                  time.Date(2019, 8, 21, 20, 37, 12, 0, time.UTC),
					LastModified:             time.Date(2019, 8, 21, 20, 37, 12, 0, time.UTC),
					Datafile: Datafile{
						ID:             1,
						LatestFileSize: 100,
						OtherURLs:      []string{"https://datafile.url"},
						Revision:       1,
						SDKKey:         "abc123",
						URL:            "https://datafile.url",
					},
					IsPrimary: false,
				}, {
					ID:                       2,
					Key:                      "key 2",
					Name:                     "Production",
					ProjectID:                1,
					Archived:                 false,
					Description:              "production environment",
					HasRestrictedPermissions: false,
					Created:                  time.Date(2019, 8, 21, 20, 37, 12, 0, time.UTC),
					LastModified:             time.Date(2019, 8, 21, 20, 37, 12, 0, time.UTC),
					Datafile: Datafile{
						ID:             2,
						LatestFileSize: 200,
						OtherURLs:      []string{"https://datafile.url"},
						Revision:       2,
						SDKKey:         "abc123",
						URL:            "https://datafile.url",
					},
					IsPrimary: true,
				},
			},
			false,
		}, {
			"api error returns an error",
			[]string{""},
			fmt.Errorf("api error"),
			nil,
			true,
		}, {
			"error decoding json returns an error",
			[]string{"{"},
			nil,
			nil,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mc, _, environmentsAPICall := createMockClient(nil, nil, test.responseBodies, test.apiErr, projectID)
			if environmentsAPICall != nil {
				environmentsAPICall.Once()
			}
			defer mc.AssertExpectations(t)
			c := Client{apiClient: mc}
			environments, err := c.GetEnvironmentsByProjectID(projectID)
			if test.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.expectedEnvironments, environments)
		})
	}
}

func TestClient_GetEnvironmentsByProjectName(t *testing.T) {
	const projectBody = `
[
  {
    "name": "Project",
    "description": "project description",
    "status": "active",
    "account_id": 12345,
    "created": "2019-08-21T20:37:12Z",
    "id": 3000,
    "last_modified": "2019-08-21T20:37:12Z"
  }
]
`
	const environmentBody = `
[
  {
    "id": 1,
    "key": "key",
    "name": "Staging",
    "project_id": 3000,
    "archived": true,
    "description": "staging environment",
    "has_restricted_permissions": false,
    "created": "2019-08-21T20:37:12Z",
    "is_primary": false,
    "last_modified": "2019-08-21T20:37:12Z",
    "datafile": {
      "id": 1,
      "latest_file_size": 100,
      "other_urls": [
        "https://datafile.url"
      ],
      "revision": 1,
      "sdk_key": "abc123",
      "url": "https://datafile.url"
    }
  }
]
`
	tests := []struct {
		name                 string
		projectName          string
		projectApiErr        error
		environmentApiErr    error
		expectedEnvironments []Environment
		expectErr            bool
	}{
		{
			"environment is retrieved by project name",
			"Project",
			nil,
			nil,
			[]Environment{{
				ID:                       1,
				Key:                      "key",
				Name:                     "Staging",
				ProjectID:                3000,
				Archived:                 true,
				Description:              "staging environment",
				HasRestrictedPermissions: false,
				Created:                  time.Date(2019, 8, 21, 20, 37, 12, 0, time.UTC),
				LastModified:             time.Date(2019, 8, 21, 20, 37, 12, 0, time.UTC),
				Datafile: Datafile{
					ID:             1,
					LatestFileSize: 100,
					OtherURLs:      []string{"https://datafile.url"},
					Revision:       1,
					SDKKey:         "abc123",
					URL:            "https://datafile.url",
				},
				IsPrimary: false,
			}},
			false,
		}, {
			"project name not found returns error",
			"Project1234",
			nil,
			nil,
			nil,
			true,
		}, {
			"error getting projects returns error",
			"Project",
			fmt.Errorf("project error"),
			nil,
			nil,
			true,
		}, {
			"error getting environments returns error",
			"Project",
			nil,
			fmt.Errorf("environment error"),
			nil,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mc, projectsAPICall, environmentsAPICall := createMockClient(
				[]string{projectBody}, test.projectApiErr,
				[]string{environmentBody}, test.environmentApiErr, 3000)
			defer mc.AssertExpectations(t)
			projectsAPICall.Once()
			environmentsAPICall.Maybe()
			c := Client{apiClient: mc}
			environments, err := c.GetEnvironmentsByProjectName(test.projectName)
			if test.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.expectedEnvironments, environments)
		})
	}
}

func TestClient_GetEnvironment(t *testing.T) {
	const projectBody = `
[
  {
    "name": "Project",
    "description": "project description",
    "status": "active",
    "account_id": 12345,
    "created": "2019-08-21T20:37:12Z",
    "id": 3000,
    "last_modified": "2019-08-21T20:37:12Z"
  }
]
`
	const environmentBody = `
[
  {
    "id": 1,
    "key": "key",
    "name": "Staging",
    "project_id": 3000,
    "archived": true,
    "description": "staging environment",
    "has_restricted_permissions": false,
    "created": "2019-08-21T20:37:12Z",
    "is_primary": false,
    "last_modified": "2019-08-21T20:37:12Z",
    "datafile": {
      "id": 1,
      "latest_file_size": 100,
      "other_urls": [
        "https://datafile.url"
      ],
      "revision": 1,
      "sdk_key": "abc123",
      "url": "https://datafile.url"
    }
  }
]
`
	tests := []struct {
		name                string
		projectName         string
		environmentName     string
		environmentApiErr   error
		expectedEnvironment Environment
		expectErr           bool
	}{
		{
			"environment is retrieved by project name",
			"Project",
			"Staging",
			nil,
			Environment{
				ID:                       1,
				Key:                      "key",
				Name:                     "Staging",
				ProjectID:                3000,
				Archived:                 true,
				Description:              "staging environment",
				HasRestrictedPermissions: false,
				Created:                  time.Date(2019, 8, 21, 20, 37, 12, 0, time.UTC),
				LastModified:             time.Date(2019, 8, 21, 20, 37, 12, 0, time.UTC),
				Datafile: Datafile{
					ID:             1,
					LatestFileSize: 100,
					OtherURLs:      []string{"https://datafile.url"},
					Revision:       1,
					SDKKey:         "abc123",
					URL:            "https://datafile.url",
				},
				IsPrimary: false,
			},
			false,
		}, {
			"environment name not found returns error",
			"Project",
			"bad environment",
			nil,
			Environment{},
			true,
		}, {
			"error getting environments returns error",
			"Project",
			"",
			fmt.Errorf("envirnment error"),
			Environment{},
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mc, projectsAPICall, environmentsAPICall := createMockClient(
				[]string{projectBody}, nil, []string{environmentBody}, test.environmentApiErr, 3000)
			defer mc.AssertExpectations(t)
			projectsAPICall.Once()
			environmentsAPICall.Once()
			c := Client{apiClient: mc}
			environment, err := c.GetEnvironment(test.environmentName, test.projectName)
			if test.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.expectedEnvironment, environment)
		})
	}
}
