package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/simon-fredrich/function-gitlab-importer/input/v1beta1"
	"github.com/simon-fredrich/function-gitlab-importer/internal"
	"github.com/simon-fredrich/function-gitlab-importer/internal/gitlabclient"
	"github.com/simon-fredrich/function-gitlab-importer/internal/handler"
	"github.com/simon-fredrich/function-gitlab-importer/internal/handler/gitlabhandler"
	"github.com/simon-fredrich/function-gitlab-importer/internal/importer"
	"github.com/simon-fredrich/function-gitlab-importer/internal/importer/gitlabimporter"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/response"
)

// Function returns whatever response you ask it to.
type Function struct {
	fnv1.UnimplementedFunctionRunnerServiceServer

	Input  *v1beta1.Input
	Client *gitlab.Client

	log logging.Logger
}

// RunFunction runs the Function.
func (f *Function) RunFunction(_ context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
	f.log.Info("Running function", "tag", req.GetMeta().GetTag())
	rsp := response.To(req, response.DefaultTTL)
	in := &v1beta1.Input{}
	f.Input = in
	if err := request.GetInput(req, in); err != nil {
		// You can set a custom status condition on the claim. This allows you to
		// communicate with the user. See the link below for status condition
		// guidance.
		// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
		response.ConditionFalse(rsp, "FunctionSuccess", "InternalError").
			WithMessage("Something went wrong.").
			TargetCompositeAndClaim()

		// You can emit an event regarding the claim. This allows you to communicate
		// with the user. Note that events should be used sparingly and are subject
		// to throttling; see the issue below for more information.
		// https://github.com/crossplane/crossplane/issues/5802
		response.Warning(rsp, errors.New("something went wrong")).
			TargetCompositeAndClaim()

		response.Fatal(rsp, errors.Wrapf(err, "cannot get Function input from %T", req))
		return rsp, nil
	}

	// get all resources from the request
	resources, err := internal.GetResources(req)
	if err != nil {
		f.log.Info("Failed to extract observed and desired composed resources.",
			"error", err,
		)
		response.Fatal(rsp, fmt.Errorf("cannot extract observed and desired composed resources: %w", err))
		return rsp, nil
	}

	// end function if no observed resource found
	if len(resources.GetObserved()) == 0 {
		f.log.Info("No observed resources found")
		return rsp, nil
	}

	// end function if no desired resource found
	if len(resources.GetDesired()) == 0 {
		f.log.Info("No desired resources found")
		return rsp, nil
	}

	// process all resources and return those that need update
	desResourcesWithUpdate := f.processResources(resources)

	// Commit all changes once
	if err := response.SetDesiredComposedResources(rsp, desResourcesWithUpdate); err != nil {
		f.log.Info("Failed to set desired composed resources", "err", err)
		response.Fatal(rsp, fmt.Errorf("cannot set desired composed resources: %w", err))
	}

	// You can set a custom status condition on the claim. This allows you to
	// communicate with the user. See the link below for status condition
	// guidance.
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
	response.ConditionTrue(rsp, "FunctionSuccess", "Success").
		TargetCompositeAndClaim()

	return rsp, nil
}

// processRecources processes gitlab related resources.
func (f *Function) processResources(resources internal.Resources) map[resource.Name]*resource.DesiredComposed {
	// define map to hold desired resources that need an update
	desResourcesWithUpdate := make(map[resource.Name]*resource.DesiredComposed)

	// iterate through observed resources and filter out gitlab related ones
	for name, obs := range resources.GetObserved() {
		f.log.Info("Processing resource", "name", name)

		obsGroup := obs.Resource.GetObjectKind().GroupVersionKind().Group
		// only process resources related to gitlab
		if !strings.Contains(obsGroup, "gitlab") {
			continue
		}

		// ensure there is a matching desired resource we can update
		des, ok := resources.GetDesired()[name]
		if !ok {
			f.log.Info("no corresponding desired resource found; skipping", "name", name)
			continue
		}

		if err := f.ensureExternalName(obs, des); err != nil {
			f.log.Info("Failed to ensure external-name", "name", name, "err", err)
			continue
		}

		// Mark resource to have its external-name managed.
		internal.AddAnnotationToDesired(des, "crossplane.io/managed-external-name", "true")

		// TODO: Configure managementPolicies
		des.Resource.SetString("spec.managementPolicies", `["Observe"]`)

		desResourcesWithUpdate[name] = des
	}
	return desResourcesWithUpdate
}

func (f *Function) ensureExternalName(obs resource.ObservedComposed, des *resource.DesiredComposed) error {
	externalName := internal.GetExternalNameFromObserved(obs)
	// Test if external-name already present on observed.
	if externalName != "" {
		f.log.Info("Copy external-name from observed to desired composed resource...")
		if err := internal.SetExternalNameOnDesired(des, externalName); err != nil {
			return err
		}
		return nil
	}

	// If external-name not present try to import it using a fitting importer implementation.
	obsGroup := obs.Resource.GetObjectKind().GroupVersionKind().Group
	var resourceImporter importer.Importer
	var handler handler.Handler
	switch obsGroup {
	case "projects.gitlab.crossplane.io":
		handler = &gitlabhandler.ProjectHandler{}
		resourceImporter = &gitlabimporter.ProjectImporter{}
	case "groups.gitlab.crossplane.io":
		handler = &gitlabhandler.GroupHandler{}
		resourceImporter = &gitlabimporter.GroupImporter{}
	default:
		f.log.Debug("group does not have an importer", "observed group", obsGroup)
		return nil
	}
	msg, value := handler.CheckResourceExists(obs)
	if value {
		f.log.Info("Resource already exists; importing external-name", "msg", msg)
		if f.Client == nil {
			// supply function with gitlab client
			client, err := gitlabclient.LoadClient(f.Input)
			if err != nil {
				f.log.Debug("cannot supply function with gitlab client", "err", err)
			}
			f.Client = client
		}
		// supply importer with client
		err := resourceImporter.PassClient(f.Client)
		if err != nil {
			return err
		}
		externalName, err := resourceImporter.Import(des)
		if err != nil {
			return err
		}
		f.log.Info("Resource successfully imported!", "external-name", externalName)
		if err := internal.SetExternalNameOnDesired(des, externalName); err != nil {
			return err
		}
		return nil
	}

	return errors.Errorf("external-name could not be set: %s", msg)
}
