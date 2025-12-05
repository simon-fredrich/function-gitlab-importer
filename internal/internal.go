package internal

import (
	"fmt"
	"strconv"

	"github.com/crossplane/function-sdk-go/errors"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
)

// Resources stores both observed and desired composed resources for ease of access.
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
		return Resources{}, fmt.Errorf("getting observed composed resources from request: %w", err)
	}

	// get desired composed resources
	desired, err := request.GetDesiredComposedResources(req)
	if err != nil {
		return Resources{}, fmt.Errorf("getting desired composed resources from request: %w", err)
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

// GetExternalNameFromDesired returns the external-name from a desired composed resource.
func GetExternalNameFromDesired(des *resource.DesiredComposed) string {
	return des.Resource.GetAnnotations()["crossplane.io/external-name"]
}

// GetExternalNameFromObserved returns the external-name from a observed composed resource.
func GetExternalNameFromObserved(obs resource.ObservedComposed) string {
	return obs.Resource.GetAnnotations()["crossplane.io/external-name"]
}

// AddAnnotationOnDesired adds a custom annotation to a desired composed resource.
// If the resource has no existing annotations, a new map is created.
// The annotation is added using the provided key and value.
//
// This helper is useful in Crossplane functions for dynamically adding metadata
// to desired resources during function execution.
func AddAnnotationOnDesired(des *resource.DesiredComposed, key string, value string) {
	annotations := des.Resource.GetAnnotations()

	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[key] = value
	des.Resource.SetAnnotations(annotations)
}

// SetExternalNameOnDesired sets the external-name of a desired composed resource.
func SetExternalNameOnDesired(des *resource.DesiredComposed, externalName string) error {
	if externalName == "" {
		return fmt.Errorf("external-name is empty")
	}

	AddAnnotationOnDesired(des, "crossplane.io/external-name", externalName)
	return nil
}

// SetBoolAnnotation appends a boolean value as "true"/"false" to annotations.
func SetBoolAnnotation(des *resource.DesiredComposed, key string, value bool) {
	boolAsString := strconv.FormatBool(value)
	AddAnnotationOnDesired(des, key, boolAsString)
}

// SetManagedValues edits the desired composite resource to display that it
// has been imported and therefore is being managed and sets managementPolicies
// to only observe external resources.
func SetManagedValues(des *resource.DesiredComposed) error {
	// Mark resource to have its external-name managed.
	SetBoolAnnotation(des, "crossplane.io/managed-external-name", true)

	// Configure managementPolicies
	observeOnly := []string{"Observe"}
	err := des.Resource.SetValue("spec.managementPolicies", observeOnly)
	if err != nil {
		return errors.Errorf("cannot set managed values on resource: %w", err)
	}

	return nil
}
