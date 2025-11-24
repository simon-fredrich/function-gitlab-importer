package internal

import (
	"fmt"
	"os"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/rs/zerolog/log"
	"github.com/simon-fredrich/function-gitlab-importer/input/v1beta1"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// LoadClientGitlab creates a new client for the gitlab api.
// The token must be provided in the environment and the baseUrl
// can be set in the input (searched first) or the environment
// (searched second).
func LoadClientGitlab(in *v1beta1.Input) (*gitlab.Client, error) {
	// try to get baseUrl via input
	baseUrl := in.BaseUrl

	// try to get token from environment
	token := os.Getenv("GITLAB_API_KEY")
	if token == "" {
		return nil, errors.New("token could not be retrieved from environment")
	}

	// either use baseUrl from input or from environment
	if baseUrl != "" {
		// create a new instance of the gitlab api "client-go" using baseUrl from input
		client, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseUrl+"/api/v4"))
		if err != nil {
			return nil, errors.Errorf("creating new client for gitlab api using input: %w", err)
		}
		if client == nil {
			return nil, errors.New("gitlab client is nil (using baseUrl from input)")
		}
		return client, nil
	} else {
		// try to get baseUrl from environment variables
		baseUrl = os.Getenv("GITLAB_URL")
		// if baseUrl not set in environment use default baseUrl
		if baseUrl == "" {
			baseUrl = "https://gitlab.com/"
		}

		// create a new instance of the gitlab api "client-go" using baseUrl from environment or default
		client, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseUrl+"/api/v4"))
		if err != nil {
			return nil, errors.Errorf("creating new client for gitlab api using env: %w", err)
		}
		if client == nil {
			return nil, errors.New("gitlab client is nil (using baseUrl from environment or default)")
		}
		return client, nil
	}
}

// GetProject returns the `projectId` for a given `namespaceId` and `path`
func GetProject(client *gitlab.Client, namespaceId int, path string) (int, error) {
	// namespaceId is the ID of the parentgroup containing the desired project
	parentId := namespaceId

	// find project based on path
	projects, err := getProjects(client, parentId, "")
	if err != nil {
		log.Error().Err(err).Msgf("can't get projects")
		return -1, err
	}
	for _, project := range projects {
		if project.Path == path {
			return project.ID, nil
		}
	}
	return -1, errors.Errorf("there is no project with matching path in namespace with ID %+v", namespaceId)
}

// GetGroup returns the `groupId` for a given `parentId` and `path`
func GetGroup(client *gitlab.Client, namespaceId int, path string) (int, error) {
	// namespaceId is the ID of the parentgroup containing the desired subgroup
	parentId := namespaceId

	// find group based on path
	groups, err := getSubGroups(client, parentId)
	if err != nil {
		log.Error().Err(err).Msgf("can't get subgroups")
		return -1, err
	}
	for _, group := range groups {
		if group.Path == path {
			return group.ID, nil
		}
	}
	return -1, fmt.Errorf("there is no group with matching path in parent group with id: %+v", parentId)
}

// getSubGroups returns all groups of a given parent group
func getSubGroups(client *gitlab.Client, groupId int) ([]*gitlab.Group, error) {
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

		subgroups, resp, err := client.Groups.ListSubGroups(groupId, opt)
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

// getProjects returns all projects of a given parent group and has
func getProjects(client *gitlab.Client, groupId int, searchTerm string) ([]*gitlab.Project, error) {
	projectsTotal := []*gitlab.Project{}
	page := 1

	// iterate over all pages to retrieve all possible projects in group with the given groupId
	for {
		opt := &gitlab.ListGroupProjectsOptions{
			Search: gitlab.Ptr(searchTerm),
			ListOptions: gitlab.ListOptions{
				PerPage: 10,
				Page:    page,
			},
		}

		projects, resp, err := client.Groups.ListGroupProjects(groupId, opt)
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
