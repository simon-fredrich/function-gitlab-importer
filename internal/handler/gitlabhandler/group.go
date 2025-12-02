package gitlabhandler

import (
	"strings"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

// GroupHandler provides helper methods for working with GitLab group resources
// in the context of Crossplane compositions. It enables extracting key attributes
// from desired resources and checking if a group already exists based on observed
// conditions.
//
// The handler offers:
//   - GetNamespaceID: Retrieves the parent group ID (namespaceID) from the desired resource.
//   - GetPath: Retrieves the path of the GitLab group from the desired resource.
//   - CheckResourceExists: Determines if a GitLab group already exists by inspecting
//     the "Synced" condition message for duplication errors.
//
// This type is intended for use in Crossplane functions that manage GitLab groups.
type GroupHandler struct{}

// GetNamespaceID retrieves the parent group ID (namespaceID) from the desired resource.
// It looks up the value at the path "spec.forProvider.parentId" in the resource.
// Returns:
//   - The namespace ID as an int if successful.
//   - An error if the value cannot be retrieved or converted.
func (g *GroupHandler) GetNamespaceID(des *resource.DesiredComposed) (int, error) {
	resourcePath := "spec.forProvider.parentId"
	namespaceID, err := des.Resource.GetInteger(resourcePath)
	if err != nil {
		return -1, errors.Errorf("cannot get parentId from resource: %w", err)
	}
	return int(namespaceID), nil
}

// GetPath retrieves the path of the GitLab group from the desired resource.
// It looks up the value at the path "spec.forProvider.path" in the resource.
// Returns:
//   - The path as a string if successful.
//   - An error if the value cannot be retrieved.
func (g *GroupHandler) GetPath(des *resource.DesiredComposed) (string, error) {
	resourcePath := "spec.forProvider.path"
	pathString, err := des.Resource.GetString(resourcePath)
	if err != nil {
		return "", errors.Errorf("cannot get path from resource: %w", err)
	}
	return pathString, nil
}

// CheckResourceExists determines if a GitLab group already exists based on the observed resource.
// It inspects the "Synced" condition message and checks if it contains the phrase "has already been taken".
// Returns:
//   - The condition message.
//   - A boolean indicating whether the resource already exists.
func (g *GroupHandler) CheckResourceExists(obs resource.ObservedComposed) (string, bool) {
	const errorMessage = "has already been taken"

	// check if error message matches
	conditionSynced := obs.Resource.GetCondition("Synced")
	return conditionSynced.Message, strings.Contains(conditionSynced.Message, errorMessage)
}
