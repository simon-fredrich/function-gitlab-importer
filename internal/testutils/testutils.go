package testutils

import (
	"embed"
	"io/fs"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
)

// LoadDataFromFile imports a file for processing in tests.
func LoadDataFromFile(filename string) ([]byte, error) {
	var testdataFS embed.FS
	data, err := fs.ReadFile(testdataFS, "./testdata/"+filename)
	if err != nil {
		return []byte{}, err
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
