package importer

import "github.com/crossplane/function-sdk-go/resource"

type Importer interface {
	Import(des *resource.DesiredComposed) (string, error)
}
