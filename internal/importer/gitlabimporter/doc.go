// Package gitlabimporter provides implementations of the Importer interface for GitLab resources
// in Crossplane functions. It enables importing existing GitLab resources (such as groups and projects)
// into Crossplane compositions by resolving their IDs and setting them as external-names.
//
// The package includes:
//   - GroupImporter: Handles importing GitLab groups by locating a subgroup within a parent group.
//   - ProjectImporter: Handles importing GitLab projects by locating a project within a parent group.
//
// These importers use the GitLab API client to query resources and support pagination for large datasets.
package gitlabimporter
