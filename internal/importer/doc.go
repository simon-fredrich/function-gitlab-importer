// Package importer defines a common interface for importing provider-specific resources
// into Crossplane compositions. This abstraction exists because multiple providers
// (e.g., GitLab, Azure, GitHub) may require different logic for importing resources,
// but share the same high-level operation: taking a desired resource and returning
// an identifier or status.
package importer
