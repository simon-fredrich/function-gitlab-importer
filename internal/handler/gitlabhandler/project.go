package gitlabhandler

import (
	"strings"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

type ProjectHandler struct {
	Name string
}

func (p *ProjectHandler) GetNamespaceID(des *resource.DesiredComposed) (int, error) {
	resourcePath := "spec.forProvider.namespaceId"
	namespaceID, err := des.Resource.GetInteger(resourcePath)
	if err != nil {
		return -1, errors.Errorf("cannot get namespaceId from resource: %w", err)
	}
	return int(namespaceID), nil
}

func (p *ProjectHandler) GetPath(des *resource.DesiredComposed) (string, error) {
	resourcePath := "spec.forProvider.path"
	pathString, err := des.Resource.GetString(resourcePath)
	if err != nil {
		return "", errors.Errorf("cannot get path from resource: %w", err)
	}
	return pathString, nil
}

func (p *ProjectHandler) Exists(obs resource.ObservedComposed) bool {
	const errorMessage = "has already been taken"

	// check if error message matches
	conditionSynced := obs.Resource.GetCondition("Synced")
	return strings.Contains(conditionSynced.Message, errorMessage)
}
