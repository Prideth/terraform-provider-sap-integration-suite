// Package features is the canonical, hand-maintained catalog of every SAP
// Integration Suite feature this provider knows about, whether implemented
// or not. It exists so that "what does this provider support" has exactly
// one place to answer from — the Terraform data sources in
// internal/provider, docs/feature-support.md, and the consistency tests in
// internal/provider all derive from Catalog in catalog.go rather than
// restating support information independently.
//
// This catalog is static provider metadata: it describes what this
// provider *binary* implements, never what capabilities happen to be
// active in any particular SAP tenant. It requires no SAP connection and
// makes no network call — see docs/feature-support.md for the distinction
// from a possible future tenant-capability-discovery data source.
package features

// SupportStatus is the small, stable set of values this provider uses to
// describe how well a feature is supported. Deliberately kept short:
// finer-grained nuance belongs in Feature.Limitations and Feature.Operations,
// not in additional status values.
type SupportStatus string

const (
	// StatusSupported means every operation this provider claims for the
	// feature (see Feature.Operations) is implemented and has been
	// verified against SAP's public API behavior.
	StatusSupported SupportStatus = "supported"
	// StatusPartial means the feature is usable but this provider
	// deliberately does not implement the full lifecycle SAP's API
	// might otherwise allow — for example, an update path exists on
	// SAP's side but this provider does not trust it enough to expose
	// as an in-place Terraform update. See Feature.SupportReason and
	// Feature.Limitations for why.
	StatusPartial SupportStatus = "partial"
	// StatusReadOnly means this provider only reads the feature (for
	// example as a data source), never creates, updates, or deletes it.
	StatusReadOnly SupportStatus = "read_only"
	// StatusExperimental means the feature is implemented but its
	// contract might still change because the underlying SAP API
	// behavior is not yet fully confirmed.
	StatusExperimental SupportStatus = "experimental"
	// StatusUnsupported means this provider does not implement the
	// feature at all today. Feature.SupportReason explains why.
	StatusUnsupported SupportStatus = "unsupported"
	// StatusUnofficial means the feature works and was verified on a tenant,
	// but its contract comes only from the service's published $metadata
	// and those tests: neither SAP's documentation, an official API
	// specification nor SAP's SDK or tooling describes it. SAP may change it
	// without notice. As soon as an official source describes it, it moves
	// to StatusSupported (or StatusPartial).
	StatusUnofficial SupportStatus = "unofficial"
	// StatusSeparateProvider means this feature is deliberately excluded
	// from this provider not because of a missing API or an unsafe
	// lifecycle, but because it belongs to a different, independently
	// versioned Terraform provider (existing or planned) with its own
	// API boundary, credentials, and release cadence. This is distinct
	// from a generic StatusUnsupported: the gap is not something this
	// provider's own engineering effort would ever close, by design.
	StatusSeparateProvider SupportStatus = "separate_provider"
)

// Valid reports whether s is one of the enumerated SupportStatus values.
func (s SupportStatus) Valid() bool {
	switch s {
	case StatusSupported, StatusPartial, StatusReadOnly, StatusExperimental, StatusUnofficial, StatusUnsupported, StatusSeparateProvider:
		return true
	}
	return false
}

// ContractSource says where the contract a feature is built on comes from.
// Only an official source can make a feature StatusSupported, StatusPartial
// or StatusReadOnly; a contract known only from $metadata makes it
// StatusUnofficial once verified on a tenant, StatusExperimental before.
type ContractSource string

const (
	// SourceSAPDocumentation: SAP Help describes the operations, for example
	// in the API's example requests or its list of API resources.
	SourceSAPDocumentation ContractSource = "sap_documentation"
	// SourceAPISpecification: the official API specification published on
	// the SAP Business Accelerator Hub (OpenAPI or EDMX download).
	SourceAPISpecification ContractSource = "api_specification"
	// SourceSAPTooling: SAP's own SDK or tooling (the API Management Client
	// SDK, SAP's CI/CD actions, Project Piper) sends these requests.
	SourceSAPTooling ContractSource = "sap_tooling"
	// SourceMetadataOnly: only the service's $metadata and tests against a
	// tenant; no official source describes it.
	SourceMetadataOnly ContractSource = "metadata_only"
)

// Official reports whether the source is SAP documentation, a specification
// or SAP tooling.
func (c ContractSource) Official() bool {
	switch c {
	case SourceSAPDocumentation, SourceAPISpecification, SourceSAPTooling:
		return true
	}
	return false
}

// Valid reports whether c is one of the enumerated sources, or empty for a
// feature that has no resource or data source.
func (c ContractSource) Valid() bool {
	return c == "" || c.Official() || c == SourceMetadataOnly
}

// SupportReason is the small, stable set of reasons a feature is not fully
// StatusSupported. Left empty for StatusSupported features.
type SupportReason string

const (
	// ReasonNotImplemented means a confirmed, sufficiently documented
	// public SAP API exists, but this provider has not yet implemented
	// a resource or data source for it.
	ReasonNotImplemented SupportReason = "not_implemented"
	// ReasonPublicAPIIncomplete means a public API exists but this
	// project could not confirm enough of its contract (exact payload
	// shapes, delete granularity, and similar details) to implement it
	// safely.
	ReasonPublicAPIIncomplete SupportReason = "public_api_incomplete"
	// ReasonNoPublicAPI means no public, SAP-supported API was found for
	// the feature at all — only internal/UI-only endpoints, which this
	// project's standing policy refuses to depend on.
	ReasonNoPublicAPI SupportReason = "no_public_api"
	// ReasonResearchRequired means this feature has not yet been
	// investigated in enough depth to say whether a usable public API
	// exists at all.
	ReasonResearchRequired SupportReason = "research_required"
	// ReasonOutOfScope means the feature is deliberately excluded from
	// this provider's boundaries — see docs/provider-scope.md — even
	// though a public API might exist for it.
	ReasonOutOfScope SupportReason = "out_of_scope"
	// ReasonUnsafeTerraformLifecycle means a public API exists and is
	// even partly implemented, but part of its lifecycle (typically
	// Update) is not implemented because doing so safely under
	// Terraform's plan/apply model could not be confirmed.
	ReasonUnsafeTerraformLifecycle SupportReason = "unsafe_terraform_lifecycle"
)

// Valid reports whether r is one of the enumerated SupportReason values, or
// empty (meaning "not applicable", the expected value for StatusSupported).
func (r SupportReason) Valid() bool {
	switch r {
	case "", ReasonNotImplemented, ReasonPublicAPIIncomplete, ReasonNoPublicAPI,
		ReasonResearchRequired, ReasonOutOfScope, ReasonUnsafeTerraformLifecycle:
		return true
	}
	return false
}
