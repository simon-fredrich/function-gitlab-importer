package handler

import "github.com/crossplane/function-sdk-go/resource"

// Handler defines a common contract for working with provider-specific resources in Crossplane functions.
//
// The interface ensures consistent operations across providers by specifying methods for:
//   - GetNamespaceID: Extracting the namespace or parent ID from a desired resource.
//   - GetPath: Retrieving the resource path from a desired resource.
//   - CheckResourceExists: Determining if the resource already exists based on observed conditions.
//
// Implementations of this interface (such as GitLab-specific handlers) provide provider-specific logic
type Handler interface {
	GetNamespaceID(des *resource.DesiredComposed) (int, error)
	GetPath(des *resource.DesiredComposed) (string, error)
	CheckResourceExists(obs resource.ObservedComposed) (string, bool)
}
