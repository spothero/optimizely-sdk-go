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

// GetProjects returns all Optimizely Projects within the Optimizely account that the client has access to.
func (c Client) GetProjects() ([]Project, error) {
	responses, err := c.sendPaginatedAPIRequest(
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

// GetEnvironmentsByProjectID returns a list of environments located in the project with the given ID.
func (c Client) GetEnvironmentsByProjectID(projectID int) ([]Environment, error) {
	query := url.Values{}
	query.Set("project_id", fmt.Sprintf("%d", projectID))
	responses, err := c.sendPaginatedAPIRequest(
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

// GetEnvironmentsByProjectName returns a list of environments located in the project with the given name.
// If there is no project with the given name, an error is returned.
func (c Client) GetEnvironmentsByProjectName(projectName string) ([]Environment, error) {
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

// GetEnvironment returns the a single environment with a given name within a Project with a given name.
// This method can return an error if the given project is not found or the environment with the specified is not found.
func (c Client) GetEnvironment(name, projectName string) (Environment, error) {
	environments, err := c.GetEnvironmentsByProjectName(projectName)
	if err != nil {
		return Environment{}, err
	}
	for _, env := range environments {
		if env.Name == name {
			return env, nil
		}
	}
	return Environment{}, fmt.Errorf("could not find environment with name %s", name)
}

// ReportEvents sends serialized events to the Optimizely events API.
func (c Client) ReportEvents(events []byte) error {
	response, err := c.apiClient.(*client).Post(eventsEndpoint, "application/json", bytes.NewBuffer(events))
	if err != nil {
		return xerrors.Errorf("error reporting events to Optimizely API: %w", err)
	}
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code (%d) received from events API", response.StatusCode)
	}
	return nil
}
