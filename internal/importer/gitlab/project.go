package gitlab

import (
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/simon-fredrich/function-gitlab-importer/internal"
	"github.com/simon-fredrich/function-gitlab-importer/internal/importer"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

type GitlabProjectImporter struct {
	ExternalName     string
	Client           importer.Client[*gitlab.Client]
	ObservedComposed resource.ObservedComposed
	DesiredComposed  *resource.DesiredComposed
}

// RequiresExternalName determines if a gitlab project needs to be supplied with
// an external-name because it is not set.
func (g *GitlabProjectImporter) RequiresExternalName() bool {
	currentExternalName := internal.GetExternalNameFromObserved(g.ObservedComposed)
	return currentExternalName == ""
}

// LoadExternalName searches gitlab projects for an external-name based on
// observed composed.
func (g *GitlabProjectImporter) LoadExternalName() error {
	client, err := g.Client.GetClient()
	if err != nil {
		return errors.Errorf("cannot get client: %w", err)
	}

	obsKind := g.DesiredComposed.Resource.GroupVersionKind().Kind
	namespaceID, err := internal.GetNamespaceID(g.DesiredComposed, obsKind)
	if err != nil {
		return errors.Errorf("cannot get namespace from %s: %w", obsKind, err)
	}

	path, err := internal.GetPath(g.DesiredComposed)
	if err != nil {
		return errors.Errorf("cannot get path from %s: %w", obsKind, err)
	}

	projectID, err := GetProject(client, namespaceID, path)
	if err != nil {
		return errors.Errorf("cannot get external-name from %s: %w", obsKind, err)
	}

	g.ExternalName = strconv.Itoa(projectID)
	return nil
}

// ResourceAlreadyExists determines if project already exists externally
// based on synced condition of observed resource.
func (g *GitlabProjectImporter) ResourceAlreadyExists() (string, bool) {
	// TODO: custom url - maybe don't need url at all.
	// TODO: regex: what parts of errorMessage are important to determine if the project/group needs to be imported from gitlab.
	const errorMessage = "create failed: cannot create Gitlab project: POST https://gitlab.com/api/v4/projects: 400 {message: {name: [has already been taken]}, {path: [has already been taken]}, {project_namespace.name: [has already been taken]}}"
	const nameError = "name: [has already been taken]"
	const pathError = "path: [has already been taken]"
	const namespaceError = "project_namespace.name: [has already been taken]"

	// check if error message matches
	conditionSynced := g.ObservedComposed.Resource.GetCondition("Synced")
	switch conditionSynced.Message {
	case errorMessage:
		return errorMessage, true
	case nameError:
		return nameError, true
	case pathError:
		return pathError, true
	case namespaceError:
		return namespaceError, true
	default:
		return conditionSynced.Message, false
	}
}

// Get external-name from gitlab project importer.
func (g *GitlabProjectImporter) GetExternalName() string {
	return g.ExternalName
}

// Get observed composed from gitlab project importer.
func (g *GitlabProjectImporter) GetObservedComposed() resource.ObservedComposed {
	return g.ObservedComposed
}

// Get desired composed from gitlab project importer.
func (g *GitlabProjectImporter) GetDesiredComposed() *resource.DesiredComposed {
	return g.DesiredComposed
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
