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

	// get all resources from the request
	resources, err := internal.GetResources(req)
	if err != nil {
		f.log.Info("Failed to extract observed and desired composed resources.",
			"error", err,
		)
		response.Fatal(rsp, fmt.Errorf("cannot extract observed and desired composed resources: %v", err))
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
	desResourcesWithUpdate := f.processResources(resources, in, rsp)

	// Commit all changes once
	if err := response.SetDesiredComposedResources(rsp, desResourcesWithUpdate); err != nil {
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

// process gitlab related resources
func (f *Function) processResources(resources internal.Resources, in *v1beta1.Input, rsp *fnv1.RunFunctionResponse) map[resource.Name]*resource.DesiredComposed {
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

		obsGroup := obs.Resource.GroupVersionKind().Group
		obsKind := obs.Resource.GroupVersionKind().Kind
		// TODO: relocate code for project/group into function
		if obsGroup == "projects.gitlab.crossplane.io" && obsKind == "Project" {
			if f.handleProjectResource(name, obs, des, in, rsp, obsKind) {
				desResourcesWithUpdate[name] = des
			}
		} else if obsGroup == "groups.gitlab.crossplane.io" && obsKind == "Group" {
			f.log.Info("found group")
			if f.handleGroupResource(name, obs, des, in, rsp, obsKind) {
				desResourcesWithUpdate[name] = des
			}
		}
	}
	return desResourcesWithUpdate
}

// handle gitlab project resources to keep their external-name up-to-date
func (f *Function) handleProjectResource(name resource.Name, obs resource.ObservedComposed, des *resource.DesiredComposed, in *v1beta1.Input, rsp *fnv1.RunFunctionResponse, obsKind string) bool {
	f.log.Info("processing resource...", "kind", obsKind, "name", name)
	// check if external-name is already set in observed resource
	currentExternalName := internal.GetExternalNameFromObserved(obs)
	desiredExternalName := internal.GetExternalNameFromDesired(des)
	if currentExternalName != "" {
		f.log.Info("external-name already set in observed; copy external-name to desired resource", "name", name, "externalName", currentExternalName)
		internal.SetExternalNameOnDesired(des, currentExternalName)
	} else if desiredExternalName != "" {
		internal.SetExternalNameOnDesired(des, desiredExternalName)
	} else {
		if f.ifHasAlreadyBeenTaken(obs) {
			f.log.Info("could not create resource on gitlab, because it already exists; fetching external-name from gitlab")
			projectId, err := f.fetchExternalNameFromGitlab(des, in, rsp, obsKind)
			if err != nil {
				f.log.Info("external-name could not be fetched from gitlab", "err", err)
				return false
			}
			err = internal.SetExternalNameOnDesired(des, strconv.Itoa(projectId))
			if err != nil {
				response.Fatal(rsp, errors.New(fmt.Sprintf("cannot set externalName: %v", err)))
				return false
			}
			f.log.Info("external-name aquired on gitlab and written to desired resource", "name", name, "external-name", projectId)
		} else {
			f.log.Info(fmt.Sprintf("%v in transition...", obsKind))
		}
	}
	return true
}

// handle gitlab group resources to keep their external-name up-to-date
func (f *Function) handleGroupResource(name resource.Name, obs resource.ObservedComposed, des *resource.DesiredComposed, in *v1beta1.Input, rsp *fnv1.RunFunctionResponse, obsKind string) bool {
	f.log.Info("processing resource...", "kind", obsKind, "name", name)
	// check if external-name is already set in observed resource
	currentExternalName := internal.GetExternalNameFromObserved(obs)
	desiredExternalName := internal.GetExternalNameFromDesired(des)
	if currentExternalName != "" {
		f.log.Info("external-name already set in observed; copy external-name to desired resource", "name", name, "externalName", currentExternalName)
		internal.SetExternalNameOnDesired(des, currentExternalName)
	} else if desiredExternalName != "" {
		internal.SetExternalNameOnDesired(des, desiredExternalName)
	} else {
		if f.ifHasAlreadyBeenTaken(obs) {
			f.log.Info("could not create resource on gitlab, because it already exists; fetching external-name from gitlab")
			groupId, err := f.fetchExternalNameFromGitlab(des, in, rsp, obsKind)
			if err != nil {
				f.log.Info("external-name could not be fetched from gitlab", "err", err)
				return false
			}
			err = internal.SetExternalNameOnDesired(des, strconv.Itoa(groupId))
			if err != nil {
				response.Fatal(rsp, errors.New(fmt.Sprintf("cannot set externalName: %v", err)))
				return false
			}
			f.log.Info("external-name aquired on gitlab and written to desired resource", "name", name, "external-name", groupId)
		} else {
			f.log.Info(fmt.Sprintf("%v in transition...", obsKind))
		}
	}
	return true
}

// determine if a gitlab project or group already exsists externally
func (f *Function) ifHasAlreadyBeenTaken(obs resource.ObservedComposed) bool {
	// check if error message matches
	f.log.Info("check condition 'Synced'")
	conditionSynced := obs.Resource.GetCondition("Synced")
	// return conditionSynced.Status == "False" &&
	// 	(strings.Contains(conditionSynced.Message, nameError) ||
	// 		strings.Contains(conditionSynced.Message, pathError) ||
	// 		strings.Contains(conditionSynced.Message, namespaceError))
	return conditionSynced.Status == "False" && strings.Contains(conditionSynced.Message, "has already been taken")
}

// find a gitlab project or group based on clientGitlab, namespace and path
func (f *Function) fetchExternalNameFromGitlab(des *resource.DesiredComposed, in *v1beta1.Input, rsp *fnv1.RunFunctionResponse, obsKind string) (int, error) {
	clientGitlab, err := internal.LoadClientGitlab(in)
	if err != nil {
		f.log.Debug("cannot init gitlab-client", "err", err)
		f.log.Info("cannot init gitlab-client", "err", err)
		response.Warning(rsp, errors.Wrap(err, "gitlab lookup failed")).TargetCompositeAndClaim()
		return -1, errors.Errorf("cannot init gitlab-client: %v", err)
	}

	namespace, err := internal.GetNamespaceId(des, obsKind)
	if err != nil {
		return -1, errors.Errorf(fmt.Sprintf("cannot get namespace from %v: %v", obsKind, err))
	}

	path, err := internal.GetPath(des)
	if err != nil {
		return -1, errors.Errorf("cannot get path from %v: %v", obsKind, err)
	}

	switch obsKind {
	case "Project":
		externalName, err := internal.GetProject(clientGitlab, namespace, path)
		if err != nil {
			return -1, errors.Errorf("cannot get externalName from %v: %v", obsKind, err)
		}
		f.log.Info(fmt.Sprintf("Found %v on gitlab!", obsKind), "namespace", namespace, "path", path, "external-name", externalName)
		return externalName, nil
	case "Group":
		externalName, err := internal.GetProject(clientGitlab, namespace, path)
		if err != nil {
			return -1, errors.Errorf("cannot get externalName from %v: %v", obsKind, err)
		}
		f.log.Info(fmt.Sprintf("Found %v on gitlab!", obsKind), "namespace", namespace, "path", path, "external-name", externalName)
		return externalName, nil
	default:
		return -1, errors.Errorf("cannot handle resource of kind %v", obsKind)
	}
}
