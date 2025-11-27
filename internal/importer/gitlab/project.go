package gitlab

import (
	"github.com/rs/zerolog/log"
	"github.com/simon-fredrich/function-gitlab-importer/input/v1beta1"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

type GitlabProjectImport struct {
	Input *v1beta1.Input
}

// RequiresExternalName determines if a gitlab project needs to be supplied with
// an external-name.
func (g *GitlabProjectImport) RequiresExternalName(obs resource.ObservedComposed, des *resource.DesiredComposed) (bool, error) {
	return true, nil
}

// LoadExternalName searches gitlab projects for an external-name based on
// observed composed.
func (g *GitlabProjectImport) LoadExternalName(obs resource.ObservedComposed) error {
	return nil
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
