package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/response"
	"github.com/simon-fredrich/function-gitlab-importer/input/v1beta1"
	"github.com/simon-fredrich/function-gitlab-importer/internal"
)

// Function returns whatever response you ask it to.
type Function struct {
	fnv1.UnimplementedFunctionRunnerServiceServer

	log logging.Logger
}

// TODO: custom url - maybe don't need url at all
// TODO: regex: what parts of errorMessage are important to determine if the project/group needs to be imported from gitlab
const errorMessage = "create failed: cannot create Gitlab project: POST https://gitlab.com/api/v4/projects: 400 {message: {name: [has already been taken]}, {path: [has already been taken]}, {project_namespace.name: [has already been taken]}}"
const nameError = "name: [has already been taken]"
const pathError = "path: [has already been taken]"
const namespaceError = "project_namespace.name: [has already been taken]"

// RunFunction runs the Function.
func (f *Function) RunFunction(_ context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
	f.log.Info("Running function", "tag", req.GetMeta().GetTag())
	rsp := response.To(req, response.DefaultTTL)
	in := &v1beta1.Input{}
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

	resources, err := internal.GetResources(req)
	if err != nil {
		f.log.Info("Failed to extract observed and desired composed resources.",
			"error", err,
		)
		response.Fatal(rsp, fmt.Errorf("cannot extract observed and desired composed resources: %v", err))
		return rsp, nil
	}

	// end function if no observed resource found
	if len(resources.GetDesired()) == 0 {
		f.log.Info("No desired resources found")
		return rsp, nil
	}

	f.log.Debug("resources found", "des", resources.GetDesired(), "obs", resources.GetObserved())
	f.log.Info("Observed resources found")
	for name, _ := range resources.GetObserved() {
		f.log.Info("obs", "name", name)
	}
	f.log.Info("Desired resources found")
	for name, _ := range resources.GetDesired() {
		f.log.Info("des", "name", name)
	}

	for name, obs := range resources.GetObserved() {
		f.log.Debug("Information about observed resource",
			"composition-resource-name", name,
			"APIVersion", obs.Resource.GetAPIVersion(),
			"Kind", obs.Resource.GetKind())

		// ensure there is a matching desired resource we can update
		_, ok := resources.GetDesired()[name]
		if !ok {
			f.log.Info("no corresponding desired resource found; skipping", "name", name)
			continue
		}

		// check if error message matches
		conditionSynced := obs.Resource.GetCondition("Synced")
		if strings.Contains(conditionSynced.Message, nameError) || strings.Contains(conditionSynced.Message, pathError) || strings.Contains(conditionSynced.Message, namespaceError) {
			obsGroup := obs.Resource.GroupVersionKind().Group
			obsKind := obs.Resource.GroupVersionKind().Kind
			// TODO: relocate code for project/group into function
			if obsGroup == "projects.gitlab.crossplane.io" && obsKind == "Project" {
				f.log.Info("Annotations before processing", "annotations", resources.GetDesired()[name].Resource.GetAnnotations())
				f.log.Info("Processing Project.", "name", name)
				clientGitlab, err := internal.LoadClientGitlab(in)
				if err != nil {
					f.log.Debug("cannot init gitlab-client", "err", err)
					f.log.Info("cannot init gitlab-client", "err", err)
					response.Fatal(rsp, errors.New(fmt.Sprintf("cannot get client: %v", err)))
					continue
				}

				projectNamespace, err := resources.GetNamespaceId(name)
				if err != nil {
					response.Fatal(rsp, errors.New(fmt.Sprintf("cannot get projectNamespace: %v", err)))
					continue
				}

				projectPath, err := resources.GetPath(name)
				if err != nil {
					response.Fatal(rsp, errors.New(fmt.Sprintf("cannot get projectPath: %v", err)))
					continue
				}
				projectId, err := internal.GetProject(clientGitlab, projectNamespace, projectPath)
				if err != nil {
					response.Fatal(rsp, errors.New(fmt.Sprintf("cannot get projectId: %v", err)))
					continue
				}

				f.log.Debug("Found projectId!", "projectNamespace", projectNamespace, "projectPath", projectPath, "projectId", projectId)
				f.log.Info("Found projectId!", "projectNamespace", projectNamespace, "projectPath", projectPath, "projectId", projectId)

				err = resources.SetExternalName(name, strconv.Itoa(projectId))
				if err != nil {
					response.Fatal(rsp, errors.New(fmt.Sprintf("cannot set externalName: %v", err)))
					continue
				}
				f.log.Debug("ExternalName set successfully", "name", name, "projectId", projectId)
				f.log.Info("ExternalName set successfully", "name", name, "projectId", projectId)
				f.log.Info("Annotations after processing", "annotations", resources.GetDesired()[name].Resource.GetAnnotations())
			} else if obsGroup == "groups.gitlab.crossplane.io" && obsKind == "Group" {
				f.log.Info("found group")
			}
		}
	}

	// Commit all changes once
	if err := response.SetDesiredComposedResources(rsp, resources.GetDesired()); err != nil {
		f.log.Info("Failed to set desired composed resources", "err", err)
		response.Fatal(rsp, fmt.Errorf("cannot set desired composed resources: %v", err))
	}

	// You can set a custom status condition on the claim. This allows you to
	// communicate with the user. See the link below for status condition
	// guidance.
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
	response.ConditionTrue(rsp, "FunctionSuccess", "Success").
		TargetCompositeAndClaim()

	return rsp, nil
}
