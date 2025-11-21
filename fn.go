package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
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

	// f.log.Debug("resources found", "des", resources.GetDesired(), "obs", resources.GetObserved())
	// f.log.Info("Observed resources found")
	// for name, obs := range resources.GetObserved() {
	// 	f.log.Info("obs", "name", name, "external-name", obs.Resource.GetAnnotations()["crossplane.io/external-name"])
	// }
	f.log.Info("Desired resources found")
	for name, des := range resources.GetDesired() {
		f.log.Info("des", "name", name, "external-name", internal.GetExternalNameFromDesired(des))
		f.log.Info("desired resources from response", "name", name, "external-name", req.Desired.Resources[string(name)])
	}

	desResourcesWithUpdate := make(map[resource.Name]*resource.DesiredComposed)

	for name, obs := range resources.GetObserved() {
		f.log.Debug("Information about observed resource",
			"composition-resource-name", name,
			"APIVersion", obs.Resource.GetAPIVersion(),
			"Kind", obs.Resource.GetKind())

		// ensure there is a matching desired resource we can update
		des, ok := resources.GetDesired()[name]
		if !ok {
			f.log.Info("no corresponding desired resource found; skipping", "name", name)
			continue
		}

		// check if error message matches
		// conditionSynced := obs.Resource.GetCondition("Synced")
		// conditionReady := obs.Resource.GetCondition("Ready")
		// if conditionSynced.Status == "True" && conditionReady.Status == "True" {
		// 	f.log.Info("'Synced' and 'Ready' both 'True' -> skipping resource", "name", name)
		// 	continue
		// }
		// if conditionSynced.Status == "False" &&
		// 	(strings.Contains(conditionSynced.Message, nameError) ||
		// 		strings.Contains(conditionSynced.Message, pathError) ||
		// 		strings.Contains(conditionSynced.Message, namespaceError)) {
		obsGroup := obs.Resource.GroupVersionKind().Group
		obsKind := obs.Resource.GroupVersionKind().Kind
		// TODO: relocate code for project/group into function
		if obsGroup == "projects.gitlab.crossplane.io" && obsKind == "Project" {
			f.log.Info("Processing Project.", "name", name)
			// check if external-name is already set in observed resource
			currentExternalName := internal.GetExternalNameFromObserved(obs)
			desiredExternalName := internal.GetExternalNameFromDesired(des)
			if currentExternalName != "" {
				f.log.Info("External name already set in observed; copy external-name to desired resource", "name", name, "externalName", currentExternalName)
				internal.SetExternalNameOnDesired(des, currentExternalName)
			} else if desiredExternalName != "" {
				internal.SetExternalNameOnDesired(des, desiredExternalName)
			} else {
				projectId, err := f.fetchExternalNameFromGitlab(des, in, rsp)
				if err != nil {
					f.log.Info("external name could not be fetched from gitlab", "err", err)
					continue
				}

				err = internal.SetExternalNameOnDesired(des, strconv.Itoa(projectId))
				if err != nil {
					response.Fatal(rsp, errors.New(fmt.Sprintf("cannot set externalName: %v", err)))
					continue
				}
				f.log.Info("ExternalName set successfully", "name", name, "projectId", projectId)
			}
			desResourcesWithUpdate[name] = des
			f.log.Info("Annotations after processing", "annotations", des.Resource.GetAnnotations())
		} else if obsGroup == "groups.gitlab.crossplane.io" && obsKind == "Group" {
			f.log.Info("found group")
		}
		// }
	}

	f.log.Info("rsp BEFORE update", "rsp.Desired.Resources", rsp.Desired.Resources, "desResourcesWithUpdate", desResourcesWithUpdate)

	// Commit all changes once
	if err := response.SetDesiredComposedResources(rsp, desResourcesWithUpdate); err != nil {
		f.log.Info("Failed to set desired composed resources", "err", err)
		response.Fatal(rsp, fmt.Errorf("cannot set desired composed resources: %v", err))
	}

	f.log.Info("rsp AFTER update", "rsp.Desired.Resources", rsp.Desired.Resources, "desResourcesWithUpdate", desResourcesWithUpdate)

	// You can set a custom status condition on the claim. This allows you to
	// communicate with the user. See the link below for status condition
	// guidance.
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
	response.ConditionTrue(rsp, "FunctionSuccess", "Success").
		TargetCompositeAndClaim()

	return rsp, nil
}

func (f *Function) fetchExternalNameFromGitlab(des *resource.DesiredComposed, in *v1beta1.Input, rsp *fnv1.RunFunctionResponse) (int, error) {
	clientGitlab, err := internal.LoadClientGitlab(in)
	if err != nil {
		f.log.Debug("cannot init gitlab-client", "err", err)
		f.log.Info("cannot init gitlab-client", "err", err)
		response.Warning(rsp, errors.Wrap(err, "gitlab lookup failed")).TargetCompositeAndClaim()
		return -1, errors.Errorf("cannot init gitlab-client: %v", err)
	}

	projectNamespace, err := internal.GetNamespaceId(des)
	if err != nil {
		return -1, errors.Errorf(fmt.Sprintf("cannot get projectNamespace: %v", err))
	}

	projectPath, err := internal.GetPath(des)
	if err != nil {
		return -1, errors.Errorf("cannot get projectPath: %v", err)
	}
	projectId, err := internal.GetProject(clientGitlab, projectNamespace, projectPath)
	if err != nil {
		return -1, errors.Errorf("cannot get projectId: %v", err)
	}

	f.log.Debug("Found projectId!", "projectNamespace", projectNamespace, "projectPath", projectPath, "projectId", projectId)
	f.log.Info("Found projectId!", "projectNamespace", projectNamespace, "projectPath", projectPath, "projectId", projectId)
	return projectId, nil
}
