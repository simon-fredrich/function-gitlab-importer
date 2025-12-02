package gitlabimporter

import (
	"strconv"

	"github.com/simon-fredrich/function-gitlab-importer/internal"
	"github.com/simon-fredrich/function-gitlab-importer/internal/handler/gitlabhandler"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

type ProjectImporter struct {
	Client *gitlab.Client
}

func (g *ProjectImporter) Import(des *resource.DesiredComposed) (string, error) {
	handler := &gitlabhandler.ProjectHandler{}
	namespaceID, err := handler.GetNamespaceID(des)
	if err != nil {
		return "", errors.Errorf("cannot import resource: %w", err)
	}
	path, err := handler.GetPath(des)
	if err != nil {
		return "", errors.Errorf("cannot import resource: %w", err)
	}
	projectID, err := GetProject(g.Client, namespaceID, path)
	if err != nil {
		return "", errors.Errorf("cannot import resource: %w", err)
	}

	externalName := strconv.Itoa(projectID)
	err = internal.SetExternalNameOnDesired(des, externalName)
	if err != nil {
		return "", errors.Errorf("cannot import resource: %w", err)
	}

	return externalName, nil
}

// GetProject returns the `projectID` for a given `namespaceID` and `path`.
func GetProject(client *gitlab.Client, namespaceID int, path string) (int, error) {
	// namespaceID is the ID of the parentgroup containing the desired project
	parentID := namespaceID

	// find project based on path
	projects, err := getProjects(client, parentID, "")
	if err != nil {
		return -1, errors.Errorf("cannot get projects: %w", err)
	}
	for _, project := range projects {
		if project.Path == path {
			return project.ID, nil
		}
	}
	return -1, errors.Errorf("there is no project with matching path in namespace with ID %+v", namespaceID)
}

// getProjects returns all projects of a given parent group.
func getProjects(client *gitlab.Client, groupID int, searchTerm string) ([]*gitlab.Project, error) {
	projectsTotal := []*gitlab.Project{}
	page := 1

	// Iterate over all pages to retrieve all possible projects in group with the given groupID.
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
			return nil, errors.Errorf("cannot get list of projects: %w; gitlab resp: %+v", err, resp)
		}

		projectsTotal = append(projectsTotal, projects...)

		if resp.CurrentPage >= resp.TotalPages {
			break
		}
		page++
	}

	return projectsTotal, nil
}
