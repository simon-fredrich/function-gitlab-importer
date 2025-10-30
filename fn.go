package main

import (
	"context"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/response"
	"k8s.io/apimachinery/pkg/runtime"
)

// Function returns whatever response you ask it to.
type Function struct {
	fnv1.UnimplementedFunctionRunnerServiceServer

	log logging.Logger
}

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

	type Resource struct {
		APIVersion string                 `json:"apiVersion"`
		Kind       string                 `json:"kind"`
		Status     map[string]interface{} `json:"status"`
	}

	type Status struct {
		Conditions []map[string]interface{} `json:"conditions,omitempty"`
	}

	type Condition struct {
		Message string `json:"message,omitempty"`
	}

	// steps to implement in a loop over observed resources

	// 1.1 if APIVersion and Kind of observed resource relates to Gitlab-Project/-Group check its status.message
	// 1.2 continue if status.message == 'create failed: cannot create Gitlab project: POST https://gitlab.com/api/v4/projects:
	//       400 {message: {name: [has already been taken]}, {path: [has already been taken]},
	//       {project_namespace.name: [has already been taken]}}'
	// 1.3 use gitlab-import-test functions to find projectId and/or groupId depending on Kind
	// 1.4 annotate external-name of observed resource

	// TODOS:
	// * type Group struct
	// * type Project struct

	for key, value := range observed {
		resourceUnstructured := value.Resource.UnstructuredContent()
		f.log.Debug("Observed resource found!",
			"composition-resource-name", key)

		var resource Resource
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(resourceUnstructured, &resource)
		cannotUnmarschal(err, rsp, resourceUnstructured)

		var status Status
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Status, &status)
		cannotUnmarschal(err, rsp, resource.Status)

		f.log.Debug("Resource has been unmarschalled",
			"APIVersion", resource.APIVersion,
			"Kind", resource.Kind)

		for key, value := range status.Conditions {
			var condition Condition
			f.log.Debug("Condition!", "key", key, "value", value)
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(value, &condition)
			cannotUnmarschal(err, rsp, value)
			if condition.Message == "create failed: cannot create Gitlab project: POST https://gitlab.com/api/v4/projects: 400 {message: {name: [has already been taken]}, {path: [has already been taken]}, {project_namespace.name: [has already been taken]}}" {
				f.log.Debug("found error message")
				break
			}
		}
	}

	// You can set a custom status condition on the claim. This allows you to
	// communicate with the user. See the link below for status condition
	// guidance.
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
	response.ConditionTrue(rsp, "FunctionSuccess", "Success").
		TargetCompositeAndClaim()

	return rsp, nil
}

func cannotUnmarschal(err error, rsp *fnv1.RunFunctionResponse, value map[string]interface{}) {
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot unmarschal unstructured content from %T", value))
	}
}
