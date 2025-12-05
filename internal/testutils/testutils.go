package testutils

import (
	"os"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
)

// LoadDataFromFile imports a file for processing in tests.
//
//nolint:gosec // using this function just for testing purposes.
func LoadDataFromFile(filename string) ([]byte, error) {
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return nil, errors.Errorf("filename contains invalid characters, could be a data breach")
	}

	// Define base directory for loading files
	baseDir := "testdata/"
	data, err := os.ReadFile(baseDir + filename)
	if err != nil {
		return []byte{}, errors.Errorf("could not load data from file: %w", err)
	}
	return data, nil
}

// LoadDesiredComposedFromFile loads a DesiredComposed resource from a JSON file.
func LoadDesiredComposedFromFile(filename string) (*resource.DesiredComposed, error) {
	// Read the JSON file
	data, err := LoadDataFromFile(filename)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON into structpb.Struct
	s := &structpb.Struct{}
	if err := protojson.Unmarshal(data, s); err != nil {
		return nil, err
	}

	// Create a composed.Unstructured from the Struct
	comp := composed.New()
	comp.SetUnstructuredContent(s.AsMap())

	// Return as DesiredComposed
	return &resource.DesiredComposed{
		Resource: comp,
		Ready:    resource.ReadyUnspecified,
	}, nil
}
