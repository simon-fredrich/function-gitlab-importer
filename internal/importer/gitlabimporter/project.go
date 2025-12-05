package gitlabimporter

import (
	"strconv"

	"github.com/simon-fredrich/function-gitlab-importer/internal"
	"github.com/simon-fredrich/function-gitlab-importer/internal/handler/gitlabhandler"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

// ProjectImporter implements the Importer interface for GitLab projects.
// It uses the GitLab API client to locate an existing project within a namespace
// based on the desired resource specification, and sets its ID as the external-name
// in the Crossplane composition.
//
// This type is intended for Crossplane functions that need to import existing GitLab projects
// rather than creating new ones.
type ProjectImporter struct {
	Client    *gitlab.Client
	projectID *int
}

// Import locates an existing GitLab project based on the desired resource specification
// and sets its ID as the external-name in the Crossplane composition.
//
// It performs the following steps:
//  1. Retrieves the namespace ID and path from the desired resource.
//  2. Uses the GitLab API client to find the project within the namespace.
//  3. Converts the project ID to a string and sets it as the external-name.
//
// Returns:
//   - The external-name (project ID as a string) if successful.
//   - An error if the resource cannot be imported or the project cannot be found.
func (p *ProjectImporter) Import(des *resource.DesiredComposed) (string, error) {
	handler := &gitlabhandler.ProjectHandler{}
	namespaceID, err := handler.GetNamespaceID(des)
	if err != nil {
		return "", errors.Errorf("cannot import resource: %w", err)
	}
	path, err := handler.GetPath(des)
	if err != nil {
		return "", errors.Errorf("cannot import resource: %w", err)
	}
	projectID, err := GetProject(p.Client, namespaceID, path)
	if err != nil {
		return "", errors.Errorf("cannot import resource: %w", err)
	}
	p.projectID = &projectID

	externalName := strconv.Itoa(projectID)
	err = internal.SetExternalNameOnDesired(des, externalName)
	if err != nil {
		return "", errors.Errorf("cannot import resource: %w", err)
	}

	return externalName, nil
}

// PassClient assigns a GitLab client to the ProjectImporter.
//
// It expects the provided client to be of type *gitlab.Client. If the type
// assertion fails, an error is returned indicating the expected and actual
// types.
func (p *ProjectImporter) PassClient(client any) error {
	c, ok := client.(*gitlab.Client)
	if !ok {
		return errors.Errorf("tried to pass client with wrong type: expected *gitlab.Client, got %T", c)
	}
	p.Client = c
	return nil
}

// GetFullPath returns the full path of the external project resource.
//
// It expects the following values to be available:
//   - ProjectImporter.projectID: Needs this value to search project on gitlab.
//   - ProjectImporter.Client: Needs the client to interact with the gitlab-API.
func (p *ProjectImporter) GetFullPath() (string, error) {
	if p.projectID != nil {
		if p.Client != nil {
			project, rsp, err := p.Client.Projects.GetProject(*p.projectID, &gitlab.GetProjectOptions{})
			if err != nil {
				return "", errors.Errorf("cannot get project, rsp: %v, err: %v", rsp, err)
			}
			return project.PathWithNamespace, nil
		}
		return "", errors.New("client has not been initialized jet, use importer.PassClient first")
	}
	return "", errors.New("projectID has not been initialized jet, use importer.Import first")
}

// GetProject returns the ID of a GitLab project given its namespace ID and path.
// It retrieves all projects under the specified namespace and searches for a match.
//
// Returns:
//   - The project ID if found.
//   - An error if the project cannot be found or the API call fails.
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
