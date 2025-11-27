package main

import (
	"context"
	"fmt"

	"github.com/simon-fredrich/function-gitlab-importer/input/v1beta1"
	"github.com/simon-fredrich/function-gitlab-importer/internal"
	"github.com/simon-fredrich/function-gitlab-importer/internal/importer"
	gitlabimpl "github.com/simon-fredrich/function-gitlab-importer/internal/importer/gitlab"
	gitlabapi "gitlab.com/gitlab-org/api/client-go"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/response"
)

// Function returns whatever response you ask it to.
// - Input: The v1beta1.Input object containing observed and desired resources for reconciliation.
// - GitlabClient: A generic client wrapper implementing importer.Client for GitLab API interactions.
type Function struct {
	Input        *v1beta1.Input
	GitlabClient importer.Client[*gitlabapi.Client]
	fnv1.UnimplementedFunctionRunnerServiceServer

	log logging.Logger
}

// RunFunction runs the Function.
func (f *Function) RunFunction(_ context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
	f.log.Info("Running function", "tag", req.GetMeta().GetTag())
	rsp := response.To(req, response.DefaultTTL)
	f.Input = &v1beta1.Input{}
	if err := request.GetInput(req, f.Input); err != nil {
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

	// supply function with gitlab client
	err = f.supplyGitlabClient()
	if err != nil {
		response.Fatal(rsp, fmt.Errorf("cannot supply function with gitlab client: %w", err))
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
		// ensure there is a matching desired resource we can update
		des, ok := resources.GetDesired()[name]
		if !ok {
			f.log.Info("no corresponding desired resource found; skipping", "name", name)
			continue
		}

		// Get group and kind to determine which importer is needed.
		obsGroup := obs.Resource.GroupVersionKind().Group
		obsKind := obs.Resource.GroupVersionKind().Kind

		// Leaves room for other importers with different clients
		var resourceImporter importer.Importer
		switch obsGroup {
		case "projects.gitlab.crossplane.io":
			if obsKind != "Project" {
				continue
			}
			resourceImporter = &gitlabimpl.GitlabProjectImporter{
				Client:           f.GitlabClient,
				ObservedComposed: obs,
				DesiredComposed:  des,
			}
		case "groups.gitlab.crossplane.io":
			if obsKind != "Group" {
				continue
			}
			resourceImporter = &gitlabimpl.GitlabGroupImporter{
				Client:           f.GitlabClient,
				ObservedComposed: obs,
				DesiredComposed:  des,
			}
		}
		if resourceImporter.RequiresExternalName() {
			// If external-name not present on the observed
			// composed and resource already exists
			// then import it from external location.
			msg, resourceExists := resourceImporter.ResourceAlreadyExists()
			if resourceExists {
				f.log.Info("Resource already exists", "msg", msg)
				err := f.executeImport(resourceImporter)
				if err != nil {
					f.log.Info("Failed to execute import", "err", err)
					continue
				}
				desResourcesWithUpdate[name] = resourceImporter.GetDesiredComposed()
			} else {
				f.log.Info("Resource in transition", "msg", msg)
			}
		} else {
			// If external-name already present on observed composed
			// then copy it to the desired composed.
			err := f.executeCopy(resourceImporter)
			if err != nil {
				f.log.Info("Failed to execute copy", "err", err)
			}
			desResourcesWithUpdate[name] = resourceImporter.GetDesiredComposed()
		}
	}
	return desResourcesWithUpdate
}

// Execute import pipeline to import external-name and set it on desired composed.
func (f *Function) executeImport(resourceImporter importer.Importer) error {
	// Load external-name from external source.
	err := resourceImporter.LoadExternalName()
	if err != nil {
		f.log.Info("Failed to load external-name", "err", err)
		return err
	}
	// Set external-name on desired composed.
	importedExternalName := resourceImporter.GetExternalName()
	err = internal.SetExternalNameOnDesired(resourceImporter.GetDesiredComposed(), importedExternalName)
	if err != nil {
		f.log.Info("Failed to set external-name on desired", "err", err)
		return err
	}
	return nil
}

// Copy external-name from observed composed to desired composed.
func (f *Function) executeCopy(resourceImporter importer.Importer) error {
	// Set external-name on desired composed.
	currentExternalName := internal.GetExternalNameFromObserved(resourceImporter.GetObservedComposed())
	err := internal.SetExternalNameOnDesired(resourceImporter.GetDesiredComposed(), currentExternalName)
	if err != nil {
		f.log.Info("Failed to set external-name on desired", "err", err)
		return err
	}
	return nil
}

// Supply function with gitlab client interface.
func (f *Function) supplyGitlabClient() error {
	f.GitlabClient = &gitlabimpl.GitlabClient{}
	err := f.GitlabClient.LoadClient(f.Input)
	if err != nil {
		f.log.Info("Failed to load gitlab client", "err", err)
		return err
	}
	return nil
}
