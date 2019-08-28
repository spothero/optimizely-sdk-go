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

// Package API provides functionality for interacting with the Optimizely REST API. Current functionality includes
// reading projects, environments, and datafiles.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/xerrors"
)

const (
	baseURL        = "https://api.optimizely.com/v2"
	eventsEndpoint = "https://logx.optimizely.com/v1/events"
)

// Project is the API representation of an Optimizely project
type Project struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Status       string    `json:"status"`
	AccountID    int       `json:"account_id"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"last_modified"`
}

// Environment is the API representation of an Optimizely environment with a project
type Environment struct {
	ID                       int       `json:"id"`
	Key                      string    `json:"key"`
	Name                     string    `json:"name"`
	ProjectID                int       `json:"project_id"`
	Archived                 bool      `json:"archived"`
	Description              string    `json:"description"`
	HasRestrictedPermissions bool      `json:"has_restricted_permissions"`
	Created                  time.Time `json:"created"`
	LastModified             time.Time `json:"last_modified"`
	Datafile                 Datafile  `json:"datafile"`
	IsPrimary                bool      `json:"is_primary"`
}

// Datafile is the API representation of a datafile for an environment
type Datafile struct {
	ID             int      `json:"id"`
	LatestFileSize int      `json:"latest_file_size"`
	OtherURLs      []string `json:"other_urls"`
	Revision       int      `json:"revision"`
	SDKKey         string   `json:"sdk_key"`
	URL            string   `json:"url"`
}

// Client is the interface for interacting with the Optimizely API. NewClient returns a real implementation of this
// interface and the mocks package contains a version of this interface for testing purposes.
type Client interface {
	// GetDatafile returns the raw contents of the datafile for a given environment and project. This method will
	// return an error if the project cannot be found, the environment cannot be found in the project, or if there
	// is an error retrieving the datafile.
	GetDatafile(environmentName string, projectID int) ([]byte, error)
	// GetEnvironment returns a single environment with a given name within a Project with a given ID.
	// This method can return an error if the given project ID is not found or the environment with the specified name
	// is not found.
	GetEnvironmentByProjectID(name string, projectID int) (Environment, error)
	// GetEnvironmentByProjectName returns the a single environment with a given name within a Project with a given name.
	// This method can return an error if the given project is not found or the environment with the specified name
	// is not found.
	GetEnvironmentByProjectName(key, projectName string) (Environment, error)
	// GetEnvironmentsByProjectID returns a list of environments located in the project with the given ID.
	GetEnvironmentsByProjectID(projectID int) ([]Environment, error)
	// GetEnvironmentsByProjectName returns a list of environments located in the project with the given name.
	// If there is no project with the given name, an error is returned.
	GetEnvironmentsByProjectName(projectName string) ([]Environment, error)
	// GetProjects returns all Optimizely Projects within the Optimizely account that the client has access to.
	GetProjects() ([]Project, error)
	// ReportEvents sends serialized events to the Optimizely events API.
	ReportEvents(events []byte) error
}

func (c client) GetProjects() ([]Project, error) {
	responses, err := c.apiClient.sendPaginatedAPIRequest(
		http.MethodGet, fmt.Sprintf("%s/projects", baseURL), nil, nil, nil)
	if err != nil {
		return nil, err
	}
	projects := make([]Project, 0)
	for _, response := range responses {
		var projectsInResponse []Project
		err := json.NewDecoder(response.Body).Decode(&projectsInResponse)
		if err != nil {
			return nil, xerrors.Errorf("error decoding project response: %w", err)
		}
		projects = append(projects, projectsInResponse...)
	}
	return projects, nil
}

func (c client) GetEnvironmentsByProjectID(projectID int) ([]Environment, error) {
	query := url.Values{}
	query.Set("project_id", fmt.Sprintf("%d", projectID))
	responses, err := c.apiClient.sendPaginatedAPIRequest(
		http.MethodGet, fmt.Sprintf("%s/environments", baseURL), nil, query, nil)
	if err != nil {
		return nil, err
	}
	environments := make([]Environment, 0)
	for _, response := range responses {
		var environmentsInResponse []Environment
		err := json.NewDecoder(response.Body).Decode(&environmentsInResponse)
		if err != nil {
			return nil, xerrors.Errorf("error decoding environments in response: %w", err)
		}
		environments = append(environments, environmentsInResponse...)
	}
	return environments, nil
}

func (c client) GetEnvironmentsByProjectName(projectName string) ([]Environment, error) {
	projects, err := c.GetProjects()
	if err != nil {
		return nil, xerrors.Errorf("failed to get environments because failed to get projects: %w", err)
	}
	for _, proj := range projects {
		if proj.Name == projectName {
			return c.GetEnvironmentsByProjectID(proj.ID)
		}
	}
	return nil, fmt.Errorf("could not find project with name %s", projectName)
}

func (c client) GetEnvironmentByProjectName(name, projectName string) (Environment, error) {
	environments, err := c.GetEnvironmentsByProjectName(projectName)
	if err != nil {
		return Environment{}, err
	}
	for _, env := range environments {
		if env.Name == name {
			return env, nil
		}
	}
	return Environment{}, fmt.Errorf("could not find environment with name %s for project %s", name, projectName)
}

func (c client) GetEnvironmentByProjectID(key string, projectID int) (Environment, error) {
	environments, err := c.GetEnvironmentsByProjectID(projectID)
	if err != nil {
		return Environment{}, err
	}
	for _, env := range environments {
		if env.Key == key {
			return env, nil
		}
	}
	return Environment{}, fmt.Errorf("could not find environment with key %s for project %d", key, projectID)
}

func (c client) ReportEvents(events []byte) error {
	response, err := c.apiClient.httpClient().Post(
		eventsEndpoint, "application/json", bytes.NewBuffer(events))
	if err != nil {
		return xerrors.Errorf("error reporting events to Optimizely API: %w", err)
	}
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code (%d) received from events API", response.StatusCode)
	}
	return nil
}

func (c client) GetDatafile(environmentName string, projectID int) ([]byte, error) {
	environment, err := c.GetEnvironmentByProjectID(environmentName, projectID)
	if err != nil {
		return nil, err
	}
	response, err := c.apiClient.httpClient().Get(environment.Datafile.URL)
	if err != nil {
		return nil, xerrors.Errorf("failed to retrieve datafile from %s: %w", environment.Datafile.URL, err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, xerrors.Errorf(
			"invalid response (%d) received while retrieving datafile: %w", response.StatusCode, err)
	}
	return ioutil.ReadAll(response.Body)
}
