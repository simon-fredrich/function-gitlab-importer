// Package gvkimplementation provides a flexible registry that maps Kubernetes
// GroupVersionKinds (GVKs) to their corresponding handler and importer
// implementations.
//
// This design enables easy extension beyond GitLab resources. For example,
// additional GVKs for Azure AD Groups or other providers can be registered
// without modifying existing logic—simply add new entries to the registry.
//
// Functions:
//   - IsAllowed: checks if a GVK is supported.
//   - LookupByGVK: retrieves the implementation for a GVK.
//
// Extending for other providers:
//
//	To support Azure AD Groups or other resource types, define the new GVK
//	and register its handler/importer in ImplementationByGVK. No changes to
//	the processing logic are required.
package gvkimplementation
