package main

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/response"
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
		ext, err := obs.Resource.GetString(externalNameAnnotationPath)
		if err != nil {
			response.Fatal(rsp, errors.Wrapf(err, "cannot get external name from %T", ext))
		}

		f.log.Debug("with external annotation path", "external-name", ext)
	}

	// You can set a custom status condition on the claim. This allows you to
	// communicate with the user. See the link below for status condition
	// guidance.
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
	response.ConditionTrue(rsp, "FunctionSuccess", "Success").
		TargetCompositeAndClaim()

	return rsp, nil
}

func checkUnmarshal(err error, rsp *fnv1.RunFunctionResponse, value map[string]interface{}) {
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot unmarschal unstructured content from %T", value))
	}
}
