package testutils

import (
	"log"
	"os"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// LoadDataFromFile imports a file for processing in tests.
func LoadDataFromFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return string(data)
}

// LoadDesiredComposedFromFile loads a DesiredComposed resource from a JSON file.
func LoadDesiredComposedFromFile(filepath string) (*resource.DesiredComposed, error) {
	// Read the JSON file
	data, err := os.ReadFile(filepath)
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
