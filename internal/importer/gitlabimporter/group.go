package gitlabimporter

import (
	"strconv"

	"github.com/simon-fredrich/function-gitlab-importer/internal"
	"github.com/simon-fredrich/function-gitlab-importer/internal/handler/gitlabhandler"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

// GroupImporter implements the Importer interface for GitLab groups.
// It uses the GitLab API client to locate an existing subgroup within a parent group
// based on the desired resource specification, and sets its ID as the external-name
// in the Crossplane composition.
//
// This type is intended for Crossplane functions that need to import existing GitLab groups
// rather than creating new ones.
type GroupImporter struct {
	Client *gitlab.Client
}

// Import locates an existing GitLab group based on the desired resource specification
// and sets its ID as the external-name in the Crossplane composition.
//
// It performs the following steps:
//  1. Retrieves the parent group ID (namespaceID) and path from the desired resource.
//  2. Uses the GitLab API client to find the subgroup within the parent group.
//  3. Converts the group ID to a string and sets it as the external-name.
//
// Returns:
//   - The external-name (group ID as a string) if successful.
//   - An error if the resource cannot be imported or the group cannot be found.
func (g *GroupImporter) Import(des *resource.DesiredComposed) (string, error) {
	handler := &gitlabhandler.GroupHandler{}
	namespaceID, err := handler.GetNamespaceID(des)
	if err != nil {
		return "", errors.Errorf("cannot import resource: %w", err)
	}
	path, err := handler.GetPath(des)
	if err != nil {
		return "", errors.Errorf("cannot import resource: %w", err)
	}
	groupID, err := GetGroup(g.Client, namespaceID, path)
	if err != nil {
		return "", errors.Errorf("cannot import resource: %w", err)
	}

	externalName := strconv.Itoa(groupID)
	err = internal.SetExternalNameOnDesired(des, externalName)
	if err != nil {
		return "", errors.Errorf("cannot import resource: %w", err)
	}

	return externalName, nil
}

// GetGroup returns the ID of a GitLab subgroup given its namespace ID and path.
// It retrieves all subgroups under the specified parent group and searches for a match.
//
// Returns:
//   - The subgroup ID if found.
//   - An error if the subgroup cannot be found or the API call fails.
func GetGroup(client *gitlab.Client, namespaceID int, path string) (int, error) {
	// namespaceID is the ID of the parentgroup containing the desired subgroup
	parentID := namespaceID

	// find group based on path
	groups, err := getSubGroups(client, parentID)
	if err != nil {
		return -1, errors.Errorf("cannot get subgroups: %w", err)
	}
	for _, group := range groups {
		if group.Path == path {
			return group.ID, nil
		}
	}
	return -1, errors.Errorf("there is no group with matching path in parent group with id: %+v", parentID)
}

// getSubGroups returns all groups of a given parent group.
func getSubGroups(client *gitlab.Client, groupID int) ([]*gitlab.Group, error) {
	subgroupsTotal := []*gitlab.Group{}
	page := 1

	// Iterate over all pages to retrieve all possible subgroups.
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
			return nil, errors.Errorf("cannot get list of subgroups: %w; gitlab resp: %+v", err, resp)
		}
		subgroupsTotal = append(subgroupsTotal, subgroups...)

		if resp.CurrentPage >= resp.TotalPages {
			break
		}
		page++
	}

	return subgroupsTotal, nil
}
