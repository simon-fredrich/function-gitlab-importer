package gitlabhandler

import (
	"strings"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

type GroupHandler struct{}

func (g *GroupHandler) GetNamespaceID(des *resource.DesiredComposed) (int, error) {
	resourcePath := "spec.forProvider.parentId"
	namespaceID, err := des.Resource.GetInteger(resourcePath)
	if err != nil {
		return -1, errors.Errorf("cannot get parentId from resource: %w", err)
	}
	return int(namespaceID), nil
}

func (g *GroupHandler) GetPath(des *resource.DesiredComposed) (string, error) {
	resourcePath := "spec.forProvider.path"
	pathString, err := des.Resource.GetString(resourcePath)
	if err != nil {
		return "", errors.Errorf("cannot get path from resource: %w", err)
	}
	return pathString, nil
}

func (g *GroupHandler) CheckResourceExists(obs resource.ObservedComposed) (string, bool) {
	const errorMessage = "has already been taken"

	// check if error message matches
	conditionSynced := obs.Resource.GetCondition("Synced")
	return conditionSynced.Message, strings.Contains(conditionSynced.Message, errorMessage)
}
