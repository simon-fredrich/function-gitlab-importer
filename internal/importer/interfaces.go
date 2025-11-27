package importer

import (
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/simon-fredrich/function-gitlab-importer/input/v1beta1"
)

type Client[T any] interface {
	LoadClient(in *v1beta1.Input) error
	GetClient() (T, error)
}

type Importer interface {
	RequiresExternalName() bool
	LoadExternalName() error
	GetExternalName() string
	GetObservedComposed() resource.ObservedComposed
	GetDesiredComposed() *resource.DesiredComposed
	ResourceAlreadyExists() (string, bool)
}
