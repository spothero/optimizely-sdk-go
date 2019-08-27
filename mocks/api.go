package mocks

import (
	"github.com/spothero/optimizely-sdk-go/api"
	"github.com/stretchr/testify/mock"
)

// Client mocks out the OptimizelyAPI interface for use in testing
type Client struct {
	mock.Mock
}

func (c *Client) GetDatafile(environmentName string, projectID int) ([]byte, error) {
	call := c.Called(environmentName, projectID)
	return call.Get(0).([]byte), call.Error(1)
}

func (c *Client) GetEnvironmentByProjectID(name string, projectID int) (api.Environment, error) {
	call := c.Called(name, projectID)
	return call.Get(0).(api.Environment), call.Error(1)
}

func (c *Client) GetEnvironmentByProjectName(name, projectName string) (api.Environment, error) {
	call := c.Called(name, projectName)
	return call.Get(0).(api.Environment), call.Error(1)
}

func (c *Client) GetEnvironmentsByProjectID(projectID int) ([]api.Environment, error) {
	call := c.Called(projectID)
	return call.Get(0).([]api.Environment), call.Error(1)
}

func (c *Client) GetEnvironmentsByProjectName(projectName string) ([]api.Environment, error) {
	call := c.Called(projectName)
	return call.Get(0).([]api.Environment), call.Error(1)
}

func (c *Client) GetProjects() ([]api.Project, error) {
	call := c.Called()
	return call.Get(0).([]api.Project), call.Error(1)
}

func (c *Client) ReportEvents(events []byte) error {
	return c.Called(events).Error(0)
}
