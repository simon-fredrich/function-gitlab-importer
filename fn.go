package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/response"
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

// RunFunction runs the Function.
func (f *Function) RunFunction(_ context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
	f.log.Info("Running function", "tag", req.GetMeta().GetTag())
	rsp := response.To(req, response.DefaultTTL)

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

	for name, obs := range resources.GetObserved() {
		f.log.Debug("Information about observed resource",
			"composition-resource-name", name,
			"APIVersion", obs.Resource.GetAPIVersion(),
			"Kind", obs.Resource.GetKind())

		conditionSynced := obs.Resource.GetCondition("Synced")
		if conditionSynced.Message == errorMessage {
			f.log.Info("found error message")
			obsGroup := obs.Resource.GroupVersionKind().Group
			obsKind := obs.Resource.GroupVersionKind().Kind
			// TODO: relocate code for project/group into function
			if obsGroup == "projects.gitlab.crossplane.io" && obsKind == "Project" {
				clientGitlab, err := internal.LoadClientGitlab()
				if err != nil {
					f.log.Debug("cannot get gitlab-client", "err", err)
				}
				f.log.Info("found project")

				// TODO: Status an der composite resource angeben
				// if clientGitlab == nil {

				// }

				projectNamespace, err := resources.GetNamespaceId(name)
				if err != nil || projectNamespace == -1 {
					f.log.Debug("cannot get projectNamespace from resource", "err", err)
				}
				projectPath, err := resources.GetPath(name)
				if err != nil || projectPath == "" {
					f.log.Debug("cannog get projectPath from resource", "err", err)
				}
				projectId, err := internal.GetProject(clientGitlab, projectNamespace, projectPath)
				if err != nil || projectId == -1 {
					f.log.Debug("cannot get projectId from resource", "err", err)
				}

				f.log.Debug("Found projectId!", "projectId", projectId)

				err = resources.SetExternalName(name, strconv.Itoa(projectId))
				if err != nil {
					f.log.Debug("external name cannot be set", "err", err)
				}
				f.log.Debug("external name has been set", "desired resource", resources.GetDesired()[name].Resource)

				err = response.SetDesiredComposedResources(rsp, resources.GetDesired())
				if err != nil {
					f.log.Info("Failed to set desired composed resources.",
						"error", err,
						"desired", resources.GetDesired(),
					)
					response.Fatal(rsp, fmt.Errorf("cannot set desired composed resources in %v", err))
					return rsp, nil
				}
				return rsp, nil
			} else if obsGroup == "groups.gitlab.crossplane.io" && obsKind == "Group" {
				f.log.Info("found group")
			}
		} else {
			return rsp, nil
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
