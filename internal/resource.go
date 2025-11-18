package internal

import (
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/meta"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
)

const externalNameAnnotationPath = `metadata.annotations["` + meta.AnnotationKeyExternalName + `"]`

type Resources struct {
	observedComposed map[resource.Name]resource.ObservedComposed
	desiredComposed  map[resource.Name]*resource.DesiredComposed
}

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

	resources := Resources{
		observedComposed: observed,
		desiredComposed:  desired,
	}

	return resources, nil

}

func (r Resources) GetObserved() map[resource.Name]resource.ObservedComposed {
	return r.observedComposed
}

func (r Resources) GetDesired() map[resource.Name]*resource.DesiredComposed {
	return r.desiredComposed
}

func (r Resources) GetNamespaceId(composedResourceName resource.Name) (int, error) {
	resourcePath := "spec.forProvider.namespaceId"
	namespaceId, err := r.observedComposed[composedResourceName].Resource.GetInteger(resourcePath)
	if err != nil {
		return -1, fmt.Errorf("cannot get namespaceId from resource: %v", err)
	}
	return int(namespaceId), nil
}

func (r Resources) GetPath(composedResourceName resource.Name) (string, error) {
	resourcePath := "spec.forProvider.path"
	pathString, err := r.observedComposed[composedResourceName].Resource.GetString(resourcePath)
	if err != nil {
		return "", fmt.Errorf("cannot get path from resource: %v", err)
	}
	return pathString, nil
}

func (r Resources) GetExternalName(composedResourceName resource.Name) (string, error) {
	externalName, err := r.observedComposed[composedResourceName].Resource.GetString(externalNameAnnotationPath)
	if err != nil {
		return "", fmt.Errorf("cannot get externalName from resource: %v", err)
	}
	return externalName, nil
}

func (r Resources) SetExternalName(composedResourceName resource.Name, externalName string) error {
	if externalName == "" {
		return nil
	}

	resource, ok := r.desiredComposed[composedResourceName]
	if !ok {
		return fmt.Errorf("composed name %q not found", composedResourceName)
	}
	resource.Resource.SetString(externalNameAnnotationPath, externalName)
	return nil
}
