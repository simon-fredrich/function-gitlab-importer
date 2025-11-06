package internal

import (
	"errors"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// TODO: start function manually with set environment variables i.e. in the container
func LoadClientGitlab() (*gitlab.Client, error) {
	// get environment variables and check if they are initilized
	token := os.Getenv("GITLAB_API_KEY")
	if token == "" {
		return nil, errors.New("GITLAB_API_KEY is not set")
	}

	url := os.Getenv("GITLAB_URL")
	if url == "" {
		return nil, errors.New("GITLAB_URL is not set")
	}

	// create a new instance of the gitlab api "client-go"
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(url+"/api/v4"))
	if err != nil {
		return nil, fmt.Errorf("creating new client for gitlab api: %v", err)
	}

	return client, nil
}

// GetProject returns the `projectId` for a given `namespaceId` and `path`
func GetProject(client *gitlab.Client, namespaceId int, path string) (int, error) {
	// namespaceId is the ID of the group containing the desired project
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
	return -1, fmt.Errorf("there is no project with matching path in namespace with ID %+v", namespaceId)
}

// getGroup returns the `groupId` for a given `parentId` and `path`
func GetGroup(client *gitlab.Client, parentId int, path string) (int, error) {
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
	return -1, fmt.Errorf("there is no project with matching path in parent group with id: %+v", parentId)
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
