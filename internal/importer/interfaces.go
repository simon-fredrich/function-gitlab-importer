package importer

import (
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/simon-fredrich/function-gitlab-importer/input/v1beta1"
)

type Importer interface {
	RequiresExternalName(obs resource.ObservedComposed) (bool, error)
	LoadExternalName(obs resource.ObservedComposed) error
}

type Client interface {
	LoadClient(in *v1beta1.Input) error
	GetClient() (any, error)
}
