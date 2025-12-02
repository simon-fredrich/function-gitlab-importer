// Package handler defines a common interface for handling provider-specific resources
// in Crossplane functions. This abstraction exists because multiple providers (e.g., GitLab,
// Azure, GitHub) may require different logic for resource operations, but share the same
// high-level tasks such as retrieving identifiers, paths, and checking resource existence.
package handler
