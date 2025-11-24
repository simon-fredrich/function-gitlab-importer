package internal

import (
	"fmt"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
)

// Type to store both observed and desired composed resources for ease of access.
type Resources struct {
	observedComposed map[resource.Name]resource.ObservedComposed
	desiredComposed  map[resource.Name]*resource.DesiredComposed
}

// GetResources fetches both the observed and desired resources from a request
// and stores them in a type Resources for ease of access.
func GetResources(req *fnv1.RunFunctionRequest) (Resources, error) {
	// get observed composed resources
	observed, err := request.GetObservedComposedResources(req)
	if err != nil {
		return Resources{}, fmt.Errorf("getting observed composed resources from request: %v", err)
	}

	// get desired composed resources
	desired, err := request.GetDesiredComposedResources(req)
	if err != nil {
		return Resources{}, fmt.Errorf("getting desired composed resources from request: %v", err)
	}

	// store observed and desired composed resources
	resources := Resources{
		observedComposed: observed,
		desiredComposed:  desired,
	}

	return resources, nil
}

// GetObserved returns observed composed resources.
func (r Resources) GetObserved() map[resource.Name]resource.ObservedComposed {
	return r.observedComposed
}

// GetDesired returns desired composed resources.
func (r Resources) GetDesired() map[resource.Name]*resource.DesiredComposed {
	return r.desiredComposed
}

// GetNamespaceId returns the namespaceId of a desired composed resource of kind Project
// as well as Group.
func GetNamespaceId(des *resource.DesiredComposed, obsKind string) (int, error) {
	var resourcePath string
	switch obsKind {
	case "Project":
		resourcePath = "spec.forProvider.namespaceId"
	case "Group":
		resourcePath = "spec.forProvider.parentId"
	}
	namespaceId, err := des.Resource.GetInteger(resourcePath)
	if err != nil {
		return -1, fmt.Errorf("cannot get namespaceId from resource: %v", err)
	}
	return int(namespaceId), nil
}

// GetPath returns the path of a desired composed resource.
func GetPath(des *resource.DesiredComposed) (string, error) {
	resourcePath := "spec.forProvider.path"
	pathString, err := des.Resource.GetString(resourcePath)
	if err != nil {
		return "", fmt.Errorf("cannot get path from resource: %v", err)
	}
	return pathString, nil
}

// GetExternalNameFromDesired returns the external-name from a desired composed resource.
func GetExternalNameFromDesired(des *resource.DesiredComposed) string {
	return des.Resource.GetAnnotations()["crossplane.io/external-name"]
}

// GetExternalNameFromObserved returns the external-name from a observed composed resource.
func GetExternalNameFromObserved(obs resource.ObservedComposed) string {
	return obs.Resource.GetAnnotations()["crossplane.io/external-name"]
}

// SetExternalNameOnDesired sets the external-name of a desired composed resource.
func SetExternalNameOnDesired(des *resource.DesiredComposed, externalName string) error {
	if externalName == "" {
		return fmt.Errorf("external-name is empty")
	}

	annotations := des.Resource.GetAnnotations()

	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations["crossplane.io/external-name"] = externalName
	des.Resource.SetAnnotations(annotations)

	return nil
}
