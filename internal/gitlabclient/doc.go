// Package gitlabclient provides utilities for creating and configuring a GitLab API client.
// It retrieves the GitLab personal access token from the environment variable `GITLAB_API_KEY`
// and determines the GitLab BaseURL from either the provided input or environment variables.
// If no BaseURL is specified, it defaults to `https://gitlab.com/`.
//
// Typical usage:
//
//	client, err := gitlabclient.LoadClient(input)
//	if err != nil {
//	    // handle error
//	}
//
// The package relies on:
//   - github.com/simon-fredrich/function-gitlab-importer/input/v1beta1 for input structure
//   - gitlab.com/gitlab-org/api/client-go for GitLab API interactions
//   - github.com/crossplane/function-sdk-go/errors for error handling
//
// This package is intended for use in a Crossplane Composition Function.
package gitlabclient
