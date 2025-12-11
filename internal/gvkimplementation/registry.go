package gvkimplementation

import (
	providergroupsv1alpha1 "github.com/crossplane-contrib/provider-gitlab/apis/cluster/groups/v1alpha1"
	providerprojectsv1alpha1 "github.com/crossplane-contrib/provider-gitlab/apis/cluster/projects/v1alpha1"
	"github.com/simon-fredrich/function-gitlab-importer/internal/handler"
	"github.com/simon-fredrich/function-gitlab-importer/internal/handler/gitlabhandler"
	"github.com/simon-fredrich/function-gitlab-importer/internal/importer"
	"github.com/simon-fredrich/function-gitlab-importer/internal/importer/gitlabimporter"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Implementation contains Handler and Importer for specific resource implementation.
type Implementation struct {
	Handler  handler.Handler
	Importer importer.Importer
}

var implementationByGVK = map[schema.GroupVersionKind]Implementation{
	providergroupsv1alpha1.GroupKubernetesGroupVersionKind: {
		Handler:  &gitlabhandler.GroupHandler{},
		Importer: &gitlabimporter.GroupImporter{},
	},
	providerprojectsv1alpha1.ProjectGroupVersionKind: {
		Handler:  &gitlabhandler.ProjectHandler{},
		Importer: &gitlabimporter.ProjectImporter{},
	},
}

var allowedGVKs = map[schema.GroupVersionKind]struct{}{
	providergroupsv1alpha1.GroupKubernetesGroupVersionKind: {},
	providerprojectsv1alpha1.ProjectGroupVersionKind:       {},
}

// IsAllowed checks whether the given GroupVersionKind (GVK) is present
// in the allowedGVKs set. It returns true if the GVK is permitted for
// processing, and false otherwise.
func IsAllowed(gvk schema.GroupVersionKind) bool {
	_, ok := allowedGVKs[gvk]
	return ok
}

// LookupByGKV retrieves the implementation (handler and importer) associated
// with the given GroupVersionKind (GVK) from the implementationByGVK registry.
// It returns the implementation and a boolean indicating whether the GVK was found.
func LookupByGKV(gkv schema.GroupVersionKind) (Implementation, bool) {
	i, ok := implementationByGVK[gkv]
	return i, ok
}
