package main

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/response"
	"k8s.io/apimachinery/pkg/runtime"
	// "github.com/simon-fredrich/function-gitlab-importer/internal"
)

// Function returns whatever response you ask it to.
type Function struct {
	fnv1.UnimplementedFunctionRunnerServiceServer

	log logging.Logger
}

const externalNameAnnotationPath = `metadata.annotations["` + meta.AnnotationKeyExternalName + `"]`

// RunFunction runs the Function.
func (f *Function) RunFunction(_ context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
	f.log.Info("Running function", "tag", req.GetMeta().GetTag())
	rsp := response.To(req, response.DefaultTTL)

	// make observed composed resource available
	observed, err := request.GetObservedComposedResources(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get observed composed resources from %T", req))
		return rsp, nil
	}

	// end function if no observed resource found
	if len(observed) == 0 {
		return rsp, nil
	}

	// steps to implement in a loop over observed resources

	// 1.1 if APIVersion and Kind of observed resource relates to Gitlab-Project/-Group check its status.message
	// 1.2 continue if status.message == 'create failed: cannot create Gitlab project: POST https://gitlab.com/api/v4/projects:
	//       400 {message: {name: [has already been taken]}, {path: [has already been taken]},
	//       {project_namespace.name: [has already been taken]}}'
	// 1.3 use gitlab-import-test functions to find projectId and/or groupId depending on Kind
	// 1.4 annotate external-name of observed resource

	for name, obs := range observed {
		f.log.Debug("Information about observed resource",
			"composition-resource-name", name,
			"APIVersion", obs.Resource.GetAPIVersion(),
			"Kind", obs.Resource.GetKind())

		obs.Resource.SetString(externalNameAnnotationPath, "test")

		f.log.Debug("With external annotation path")

		// resourceUnstructured := value.Resource.UnstructuredContent()
		// f.log.Debug("Observed resource found!",
		// 	"composition-resource-name", key)

		// var resource Resource
		// err := runtime.DefaultUnstructuredConverter.FromUnstructured(resourceUnstructured, &resource)
		// checkUnmarshal(err, rsp, resourceUnstructured)

		// compareMessage := "create failed: cannot create Gitlab project: POST https://gitlab.com/api/v4/projects: 400 {message: {name: [has already been taken]}, {path: [has already been taken]}, {project_namespace.name: [has already been taken]}}"
		// if resource.APIVersion == "groups.gitlab.crossplane.io/v1alpha1" && resource.Kind == "Group" {
		// 	if checkMessage(f, rsp, resource, compareMessage) {
		// 		forProvider := getForProvider(f, rsp, resource)
		// 		f.log.Debug("Got group details!", "parentId", forProvider.ParentId, "path", forProvider.Path)
		// 	}
		// } else if resource.APIVersion == "projects.gitlab.crossplane.io/v1alpha1" && resource.Kind == "Project" {
		// 	if checkMessage(f, rsp, resource, compareMessage) {
		// 		forProvider := getForProvider(f, rsp, resource)
		// 		f.log.Debug("Got project details!", "namespaceId", forProvider.NamespaceId, "path", forProvider.Path)
		// 	}
		// }
	}

	// You can set a custom status condition on the claim. This allows you to
	// communicate with the user. See the link below for status condition
	// guidance.
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
	response.ConditionTrue(rsp, "FunctionSuccess", "Success").
		TargetCompositeAndClaim()

	return rsp, nil
}

// check if status.condition[i].message of resource is the same as the message provided as `compareMessage`
func checkMessage(f *Function, rsp *fnv1.RunFunctionResponse, resource Resource, compareMessage string) bool {
	var status Status
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Status, &status)
	checkUnmarshal(err, rsp, resource.Status)

	f.log.Debug("Resource has been unmarschalled",
		"APIVersion", resource.APIVersion,
		"Kind", resource.Kind)

	for key, value := range status.Conditions {
		var condition Condition
		f.log.Debug("Condition!", "key", key, "value", value)
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(value, &condition)
		checkUnmarshal(err, rsp, value)
		if condition.Message == "create failed: cannot create Gitlab project: POST https://gitlab.com/api/v4/projects: 400 {message: {name: [has already been taken]}, {path: [has already been taken]}, {project_namespace.name: [has already been taken]}}" {
			f.log.Debug("found error message")
			return true
		}
	}
	return false
}

// get forProvider details for provided resource
func getForProvider(f *Function, rsp *fnv1.RunFunctionResponse, resource Resource) ForProvider {
	var spec Spec
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Spec, &spec)
	checkUnmarshal(err, rsp, resource.Spec)
	var forProvider ForProvider
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(spec.ForProvider, &forProvider)
	checkUnmarshal(err, rsp, spec.ForProvider)
	return forProvider
}

func checkUnmarshal(err error, rsp *fnv1.RunFunctionResponse, value map[string]interface{}) {
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot unmarschal unstructured content from %T", value))
	}
}
