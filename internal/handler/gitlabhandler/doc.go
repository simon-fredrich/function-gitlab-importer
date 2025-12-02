// Package gitlabhandler provides helper types and methods for interacting with GitLab resources
// in the context of Crossplane compositions. It includes handlers for GitLab groups and projects,
// enabling extraction of key resource attributes and validation of resource existence.
//
// The package offers:
//   - GroupHandler: Retrieves GitLab group properties such as namespace ID and path,
//     and checks if a group already exists based on observed conditions.
//   - ProjectHandler: Retrieves GitLab project properties such as namespace ID and path,
//     and checks if a project already exists based on observed conditions.
//
// These handlers work with Crossplane's DesiredComposed and ObservedComposed resources,
// using the function-sdk-go for resource access and error handling.
//
// Typical usage:
//
//	groupHandler := &gitlabhandler.GroupHandler{}
//	namespaceID, err := groupHandler.GetNamespaceID(desired)
//	if err != nil {
//	    // handle error
//	}
//
//	existsMsg, exists := groupHandler.CheckResourceExists(observed)
//	if exists {
//	    // resource already exists
//	}
//
// This package is intended for Crossplane Composition Function implementations that manage GitLab resources.
package gitlabhandler
