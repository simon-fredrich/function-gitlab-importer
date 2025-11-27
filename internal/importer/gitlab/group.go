package gitlab

import (
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/simon-fredrich/function-gitlab-importer/internal"
	"github.com/simon-fredrich/function-gitlab-importer/internal/importer"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

type GitlabGroupImporter struct {
	ExternalName     string
	Client           importer.Client[*gitlab.Client]
	ObservedComposed resource.ObservedComposed
	DesiredComposed  *resource.DesiredComposed
}

// RequiresExternalName determines if a gitlab group needs to be supplied with
// an external-name because it is not set.
func (g *GitlabGroupImporter) RequiresExternalName() bool {
	return true
}

// LoadExternalName searches gitlab groups for an external-name based on
// observed composed.
func (g *GitlabGroupImporter) LoadExternalName() error {
	client, err := g.Client.GetClient()
	if err != nil {
		return errors.Errorf("cannot get client: %w", err)
	}

	// Get namespaceID from desired composed based on kind
	kind := g.DesiredComposed.Resource.GroupVersionKind().Kind
	namespaceID, err := internal.GetNamespaceID(g.DesiredComposed, kind)
	if err != nil {
		return errors.Errorf("cannot get namespace from %s: %w", kind, err)
	}

	path, err := internal.GetPath(g.DesiredComposed)
	if err != nil {
		return errors.Errorf("cannot get path from %s: %w", kind, err)
	}

	groupID, err := GetGroup(client, namespaceID, path)
	if err != nil {
		return errors.Errorf("cannot get external-name from %s: %w", kind, err)
	}

	g.ExternalName = strconv.Itoa(groupID)
	return nil
}

// ResourceAlreadyExists determines if group already exists externally
// based on synced condition of observed resource.
func (g *GitlabGroupImporter) ResourceAlreadyExists() (string, bool) {
	// TODO: custom url - maybe don't need url at all.
	// TODO: regex: what parts of errorMessage are important to determine if the project/group needs to be imported from gitlab.
	const errorMessage = `cannot create Gitlab Group: POST https://gitlab.com/api/v4/groups: 400 {message: Failed to save group {:name=>["has already been taken"], :path=>["has already been taken"]}}`
	const nameError = `name=>["has already been taken"]`
	const pathError = `path=>["has already been taken"]`

	// check if error message matches
	conditionSynced := g.ObservedComposed.Resource.GetCondition("Synced")
	switch conditionSynced.Message {
	case errorMessage:
		return errorMessage, true
	case nameError:
		return nameError, true
	case pathError:
		return pathError, true
	default:
		return conditionSynced.Message, false
	}
}

// Get external-name from gitlab group importer.
func (g *GitlabGroupImporter) GetExternalName() string {
	return g.ExternalName
}

// Get observed composed from gitlab group importer.
func (g *GitlabGroupImporter) GetObservedComposed() resource.ObservedComposed {
	return g.ObservedComposed
}

// Get desired composed from gitlab group importer.
func (g *GitlabGroupImporter) GetDesiredComposed() *resource.DesiredComposed {
	return g.DesiredComposed
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
