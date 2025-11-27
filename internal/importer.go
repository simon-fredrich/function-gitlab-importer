package internal

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/simon-fredrich/function-gitlab-importer/input/v1beta1"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/crossplane/function-sdk-go/errors"
)

// LoadClientGitlab creates a new client for the gitlab api.
// The token must be provided in the environment and the BaseURL
// can be set in the input (searched first) or the environment
// (searched second).
func LoadClientGitlab(in *v1beta1.Input) (*gitlab.Client, error) {
	// try to get BaseURL via input
	BaseURL := in.BaseURL

	// try to get token from environment
	token := os.Getenv("GITLAB_API_KEY")
	if token == "" {
		return nil, errors.New("token could not be retrieved from environment")
	}

	// either use BaseURL from input or from environment
	if BaseURL != "" {
		// create a new instance of the gitlab api "client-go" using BaseURL from input
		client, err := gitlab.NewClient(token, gitlab.WithBaseURL(BaseURL+"/api/v4"))
		if err != nil {
			return nil, errors.Errorf("creating new client for gitlab api using input: %w", err)
		}
		if client == nil {
			return nil, errors.New("gitlab client is nil (using BaseURL from input)")
		}
		return client, nil
	}
	// try to get BaseURL from environment variables
	BaseURL = os.Getenv("GITLAB_URL")
	// if BaseURL not set in environment use default BaseURL
	if BaseURL == "" {
		BaseURL = "https://gitlab.com/"
	}

	// create a new instance of the gitlab api "client-go" using BaseURL from environment or default
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(BaseURL+"/api/v4"))
	if err != nil {
		return nil, errors.Errorf("creating new client for gitlab api using env: %w", err)
	}
	if client == nil {
		return nil, errors.New("gitlab client is nil (using BaseURL from environment or default)")
	}
	return client, nil
}

// GetProject returns the `projectID` for a given `namespaceID` and `path`.
func GetProject(client *gitlab.Client, namespaceID int, path string) (int, error) {
	// namespaceID is the ID of the parentgroup containing the desired project
	parentID := namespaceID

	// find project based on path
	projects, err := getProjects(client, parentID, "")
	if err != nil {
		log.Error().Err(err).Msgf("can't get projects")
		return -1, err
	}
	for _, project := range projects {
		if project.Path == path {
			return project.ID, nil
		}
	}
	return -1, errors.Errorf("there is no project with matching path in namespace with ID %+v", namespaceID)
}

// GetGroup returns the `groupID` for a given `parentID` and `path`.
func GetGroup(client *gitlab.Client, namespaceID int, path string) (int, error) {
	// namespaceID is the ID of the parentgroup containing the desired subgroup
	parentID := namespaceID

	// find group based on path
	groups, err := getSubGroups(client, parentID)
	if err != nil {
		log.Error().Err(err).Msgf("can't get subgroups")
		return -1, err
	}
	for _, group := range groups {
		if group.Path == path {
			return group.ID, nil
		}
	}
	return -1, fmt.Errorf("there is no group with matching path in parent group with id: %+v", parentID)
}

// getSubGroups returns all groups of a given parent group.
func getSubGroups(client *gitlab.Client, groupID int) ([]*gitlab.Group, error) {
	subgroupsTotal := []*gitlab.Group{}
	page := 1

	// iterate over all pages to retrieve all possible subgroups
	for {
		opt := &gitlab.ListSubGroupsOptions{
			AllAvailable: gitlab.Ptr(true),
			ListOptions: gitlab.ListOptions{
				PerPage: 10,
				Page:    page,
			},
		}

		subgroups, resp, err := client.Groups.ListSubGroups(groupID, opt)
		if err != nil {
			log.Error().Err(err).Msgf("gitlab resp: %+v", resp)
			return nil, err
		}
		subgroupsTotal = append(subgroupsTotal, subgroups...)

		if resp.CurrentPage >= resp.TotalPages {
			break
		}
		page++
	}

	return subgroupsTotal, nil
}

// getProjects returns all projects of a given parent group.
func getProjects(client *gitlab.Client, groupID int, searchTerm string) ([]*gitlab.Project, error) {
	projectsTotal := []*gitlab.Project{}
	page := 1

	// iterate over all pages to retrieve all possible projects in group with the given groupID
	for {
		opt := &gitlab.ListGroupProjectsOptions{
			Search: gitlab.Ptr(searchTerm),
			ListOptions: gitlab.ListOptions{
				PerPage: 10,
				Page:    page,
			},
		}

		projects, resp, err := client.Groups.ListGroupProjects(groupID, opt)
		if err != nil {
			log.Error().Err(err).Msgf("gitlab resp: %+v", resp)
			return nil, err
		}

		projectsTotal = append(projectsTotal, projects...)

		if resp.CurrentPage >= resp.TotalPages {
			break
		}
		page++
	}

	return projectsTotal, nil
}
