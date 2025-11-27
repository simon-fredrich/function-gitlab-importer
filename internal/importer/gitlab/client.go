package gitlab

import (
	"os"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/simon-fredrich/function-gitlab-importer/input/v1beta1"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type GitlabClient struct {
	Client *gitlab.Client
}

// LoadClient creates a new client for the gitlab api.
// The token must be provided in the environment and the BaseURL
// can be set in the input (searched first) or the environment
// (searched second).
func (g *GitlabClient) LoadClient(in *v1beta1.Input) error {
	// try to get BaseURL via input
	BaseURL := in.BaseURL

	// try to get token from environment
	token := os.Getenv("GITLAB_API_KEY")
	if token == "" {
		g.Client = nil
		return errors.New("token could not be retrieved from environment")
	}

	// either use BaseURL from input or from environment
	if BaseURL != "" {
		// create a new instance of the gitlab api "client-go" using BaseURL from input
		client, err := gitlab.NewClient(token, gitlab.WithBaseURL(BaseURL+"/api/v4"))
		if err != nil {
			g.Client = nil
			return errors.Errorf("creating new client for gitlab api using input: %w", err)
		}
		if client == nil {
			g.Client = nil
			return errors.New("gitlab client is nil (using BaseURL from input)")
		}
		g.Client = client
		return nil
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
		g.Client = nil
		return errors.Errorf("creating new client for gitlab api using env: %w", err)
	}
	if client == nil {
		g.Client = nil
		return errors.New("gitlab client is nil (using BaseURL from environment or default)")
	}
	g.Client = client
	return nil
}

func (g *GitlabClient) GetClient() (*gitlab.Client, error) {
	if g.Client != nil {
		return g.Client, nil
	}
	return nil, errors.New("gitlab client is empty")
}
