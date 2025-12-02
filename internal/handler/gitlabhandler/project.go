package gitlabhandler

import (
	"strings"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

// ProjectHandler provides helper methods for working with GitLab project resources
// in the context of Crossplane compositions. It enables extracting key attributes
// from desired resources and checking if a project already exists based on observed
// conditions.
//
// The handler offers:
//   - GetNamespaceID: Retrieves the namespace ID of the GitLab project from the desired resource.
//   - GetPath: Retrieves the path of the GitLab project from the desired resource.
//   - CheckResourceExists: Determines if a GitLab project already exists by inspecting
//     the "Synced" condition message for duplication errors.
//
// This type is intended for use in Crossplane function implementations that manage GitLab projects.
type ProjectHandler struct {
	Name string
}

// GetNamespaceID retrieves the namespace ID of the GitLab project from the desired resource.
// It looks up the value at the path "spec.forProvider.namespaceId" in the resource.
// Returns:
//   - The namespace ID as an int if successful.
//   - An error if the value cannot be retrieved or converted.
func (p *ProjectHandler) GetNamespaceID(des *resource.DesiredComposed) (int, error) {
	resourcePath := "spec.forProvider.namespaceId"
	namespaceID, err := des.Resource.GetInteger(resourcePath)
	if err != nil {
		return -1, errors.Errorf("cannot get namespaceId from resource: %w", err)
	}
	return int(namespaceID), nil
}

// GetPath retrieves the path of the GitLab project from the desired resource.
// It looks up the value at the path "spec.forProvider.path" in the resource.
// Returns:
//   - The path as a string if successful.
//   - An error if the value cannot be retrieved.
func (p *ProjectHandler) GetPath(des *resource.DesiredComposed) (string, error) {
	resourcePath := "spec.forProvider.path"
	pathString, err := des.Resource.GetString(resourcePath)
	if err != nil {
		return "", errors.Errorf("cannot get path from resource: %w", err)
	}
	return pathString, nil
}

// CheckResourceExists determines if a GitLab project already exists based on the observed resource.
// It inspects the "Synced" condition message and checks if it contains the phrase "has already been taken".
// Returns:
//   - The condition message.
//   - A boolean indicating whether the resource already exists.
func (p *ProjectHandler) CheckResourceExists(obs resource.ObservedComposed) (string, bool) {
	const errorMessage = "has already been taken"

	// check if error message matches
	conditionSynced := obs.Resource.GetCondition("Synced")
	return conditionSynced.Message, strings.Contains(conditionSynced.Message, errorMessage)
}
