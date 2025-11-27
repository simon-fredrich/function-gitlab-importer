package gitlab

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/simon-fredrich/function-gitlab-importer/input/v1beta1"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/crossplane/function-sdk-go/resource"
)

type GitlabGroupImport struct {
	Input  *v1beta1.Input
	Client *gitlab.Client
}

// RequiresExternalName determines if a gitlab group needs to be supplied with
// an external-name.
func (g *GitlabGroupImport) RequiresExternalName(obs resource.ObservedComposed, des *resource.DesiredComposed) (bool, error) {
	return true, nil
}

// LoadExternalName searches gitlab groups for an external-name based on
// observed composed.
func (g *GitlabGroupImport) LoadExternalName(obs resource.ObservedComposed) error {
	return nil
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
