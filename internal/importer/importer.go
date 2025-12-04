package importer

import (
	"github.com/crossplane/function-sdk-go/resource"
)

// Importer defines a contract for importing resources in Crossplane functions.
// The interface exists to support multiple providers, each with its own import logic,
// while maintaining a consistent method signature.
//
// Method:
//   - Import: Takes a desired resource and performs the import operation,
//     returning a string identifier (such as an external name) or an error.
//   - PassClient: Provides the underlying provider client to the importer.
//     The client must be of the expected type (e.g., *gitlab.Client), otherwise
//     an error is returned.
type Importer interface {
	Import(des *resource.DesiredComposed) (string, error)
	PassClient(client any) error
}
