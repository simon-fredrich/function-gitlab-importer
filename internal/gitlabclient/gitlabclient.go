package gitlabclient

import (
	"os"

	"github.com/simon-fredrich/function-gitlab-importer/input/v1beta1"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/crossplane/function-sdk-go/errors"
)

func LoadClient(in *v1beta1.Input) (*gitlab.Client, error) {
	// try to get BaseURL via input
	BaseURL := in.BaseURL

	// try to get token from environment
	token := os.Getenv("GITLAB_API_KEY")
	if token == "" {
		return nil, errors.New("token could not be retrieved from environment")
	}

	// either use BaseURL from input or from environment
	if BaseURL != "" {
		// create a new instance of the gitlab api "client-go" using BaseURL from input
		client, err := gitlab.NewClient(token, gitlab.WithBaseURL(BaseURL+"/api/v4"))
		if err != nil {
			return nil, errors.Errorf("creating new client for gitlab api using input: %w", err)
		}
		if client == nil {
			return nil, errors.New("gitlab client is nil (using BaseURL from input)")
		}
		return client, nil
	}
	// try to get BaseURL from environment variables
	BaseURL = os.Getenv("GITLAB_URL")
	// if BaseURL not set in environment use default BaseURL
	if BaseURL == "" {
		BaseURL = "https://gitlab.com/"
	}

	// create a new instance of the gitlab api "client-go" using BaseURL from environment or default
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(BaseURL+"/api/v4"))
	if err != nil {
		return nil, errors.Errorf("creating new client for gitlab api using env: %w", err)
	}
	if client == nil {
		return nil, errors.New("gitlab client is nil (using BaseURL from environment or default)")
	}
	return client, nil
}
