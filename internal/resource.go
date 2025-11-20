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

func GetNamespaceId(des *resource.DesiredComposed) (int, error) {
	resourcePath := "spec.forProvider.namespaceId"
	namespaceId, err := des.Resource.GetInteger(resourcePath)
	if err != nil {
		return -1, fmt.Errorf("cannot get namespaceId from resource: %v", err)
	}
	return int(namespaceId), nil
}

func GetPath(des *resource.DesiredComposed) (string, error) {
	resourcePath := "spec.forProvider.path"
	pathString, err := des.Resource.GetString(resourcePath)
	if err != nil {
		return "", fmt.Errorf("cannot get path from resource: %v", err)
	}
	return pathString, nil
}

func GetExternalNameFromDesired(des *resource.DesiredComposed) (string, error) {
	externalName := des.Resource.GetAnnotations()[externalNameAnnotationPath]
	if externalName == "" {
		return "", fmt.Errorf("external-name from desired resource is empty")
	}
	return externalName, nil
}

func GetExternalNameFromObserved(obs resource.ObservedComposed) (string, error) {
	externalName := obs.Resource.GetAnnotations()[externalNameAnnotationPath]
	if externalName == "" {
		return "", fmt.Errorf("external-name from observed resource is empty")
	}
	return externalName, nil
}

func SetExternalNameOnDesired(des *resource.DesiredComposed, externalName string) error {
	if externalName == "" {
		return fmt.Errorf("external-name is empty")
	}

	annotations := make(map[string]string)
	annotations[externalNameAnnotationPath] = externalName
	meta.AddAnnotations(des.Resource, annotations)

	return nil
}
