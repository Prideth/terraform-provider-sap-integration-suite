package features

// Operations records which lifecycle operations this provider implements
// for a feature. It is deliberately more detailed than SupportStatus alone:
// two features can both be StatusPartial for entirely different reasons,
// and a single boolean cannot say whether the gap is Update, Delete,
// Deploy, or something else. A feature with no Terraform resource or data
// source at all (for example a capability-activation concern with no
// public API) has every field false.
type Operations struct {
	Create   bool
	Read     bool
	Update   bool
	Delete   bool
	Import   bool
	Deploy   bool
	Undeploy bool
}

// Feature is one entry in the canonical support catalog: a single SAP
// Integration Suite object or capability area, and exactly how (or whether)
// this provider version manages it.
type Feature struct {
	// Key is a stable, hierarchical, machine-readable identifier, for
	// example "cloud_integration.value_mapping". Keys are never reused
	// for a different feature and never renamed once shipped in a
	// release, since practitioners' Terraform configurations reference
	// them via sapintegrationsuite_provider_feature.key.
	Key string
	// Domain groups related features, for example "cloud_integration" or
	// "security". It is the same as Key's first dot-separated segment
	// for every hierarchical key; flat, ungrouped keys (broad SAP
	// capability areas this project has not yet placed in a domain, such
	// as "integration_advisor") use "other_capability".
	Domain string
	// Name is a short, human-readable label.
	Name string
	// Description explains what the feature is and, briefly, its support
	// status in prose.
	Description   string
	SupportStatus SupportStatus
	// SupportReason explains SupportStatus when it is not StatusSupported.
	// Empty for StatusSupported features.
	SupportReason SupportReason
	// ResourceTypes lists the full Terraform resource type names (for
	// example "sapintegrationsuite_value_mapping") this provider
	// registers for this feature. Empty if none.
	ResourceTypes []string
	// DataSourceTypes lists the full Terraform data source type names
	// this provider registers for this feature. Empty if none.
	DataSourceTypes []string
	// PublicAPI reports whether SAP publishes a public, supported API for
	// this feature at all — independent of whether this provider
	// implements it. A feature can have PublicAPI true and
	// SupportStatus "unsupported" (a documented API this provider simply
	// has not gotten to yet), or PublicAPI false (no known public API
	// exists, so this provider cannot implement it no matter how much
	// engineering time is spent).
	PublicAPI bool
	// APIProtocol names the wire protocol of the public API when
	// PublicAPI is true and a single protocol applies, for example
	// "OData V2". Left empty when PublicAPI is false or the protocol is
	// not yet confirmed.
	APIProtocol string
	// Planned reports whether this feature has a concrete place on
	// ROADMAP.md, as opposed to being an open-ended possibility.
	Planned bool
	// Limitations lists specific, concrete caveats a practitioner should
	// know before relying on this feature, for example an unconfirmed
	// delete scope or a missing in-place update path. Empty for a
	// feature with no caveats beyond its SupportStatus/SupportReason.
	Limitations []string
	Operations  Operations
	// ContractSource says where the contract of the implemented part comes
	// from; required for every feature with a resource or data source.
	ContractSource ContractSource
	// UndocumentedOperations names operations the provider implements that
	// only $metadata and tenant tests describe, while the rest of the
	// feature is officially documented, for example "update (PUT)". They
	// work but are unofficial: SAP may change them without notice.
	UndocumentedOperations []string
}

// Catalog is the complete, canonical list of SAP Integration Suite features
// this project has evaluated. It intentionally includes unsupported and
// out-of-scope features, not just what is implemented: the whole point of
// this catalog is to let a practitioner see gaps, not just capabilities.
//
// This is derived from, and must be kept in sync with, docs/resource-design.md,
// docs/sap-api-references.md, docs/api-capability-matrix.md,
// docs/provisioning-capability-matrix.md, docs/provider-scope.md, and
// ROADMAP.md — see CONTRIBUTING.md for the rule that every feature change
// updates this catalog in the same change.
var Catalog = []Feature{
	// --- Cloud Integration ---
	{
		Key:             "cloud_integration.integration_package",
		Domain:          "cloud_integration",
		Name:            "Integration Package",
		Description:     "A Cloud Integration content package that groups integration flows and other design-time artifacts.",
		SupportStatus:   StatusSupported,
		ContractSource:  SourceSAPDocumentation,
		ResourceTypes:   []string{"sapintegrationsuite_integration_package"},
		DataSourceTypes: []string{"sapintegrationsuite_integration_package"},
		PublicAPI:       true,
		APIProtocol:     "OData V2",
		Limitations: []string{
			"Metadata update only applies to customer-created packages; SAP-provided packages are read-only by SAP's own design, not a provider limitation.",
			"Verified on a tenant (September 2026): create needs ShortText (\"Property 'ShortText' " +
				"cannot be empty\"), so short_text is required; an update is a PUT, because PATCH " +
				"answers 501, and it carries the current Version, Vendor and tag fields, since a PUT " +
				"without them resets them; SAP stores the description as HTML and wraps plain text in <p>...</p>, " +
				"which the provider removes when reading.",
		},
		Operations: Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},
	{
		Key:             "cloud_integration.integration_flow",
		Domain:          "cloud_integration",
		Name:            "Integration Flow",
		Description:     "An integration flow's design-time content, managed as file-based (ZIP) content.",
		SupportStatus:   StatusSupported,
		ContractSource:  SourceSAPDocumentation,
		ResourceTypes:   []string{"sapintegrationsuite_integration_flow"},
		DataSourceTypes: []string{},
		PublicAPI:       true,
		APIProtocol:     "OData V2",
		Operations:      Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},
	{
		Key:             "cloud_integration.integration_flow_deployment",
		Domain:          "cloud_integration",
		Name:            "Integration Flow Deployment",
		Description:     "The runtime deployment state of an integration flow, managed independently of its design-time content.",
		SupportStatus:   StatusSupported,
		ContractSource:  SourceSAPDocumentation,
		ResourceTypes:   []string{"sapintegrationsuite_integration_flow_deployment"},
		DataSourceTypes: []string{},
		PublicAPI:       true,
		APIProtocol:     "OData V2",
		Operations:      Operations{Create: true, Read: true, Update: true, Delete: true, Import: true, Deploy: true, Undeploy: true},
	},
	{
		Key:    "cloud_integration.integration_flow_configuration",
		Domain: "cloud_integration",
		Name:   "Integration Flow Configuration",
		Description: "Externalized parameters of an integration flow (receiver hosts, endpoint " +
			"addresses, credential names and similar values set per environment).",
		SupportStatus:  StatusSupported,
		ContractSource: SourceSAPDocumentation,
		ResourceTypes:  []string{"sapintegrationsuite_integration_flow_configuration"},
		PublicAPI:      true,
		APIProtocol:    "OData V2",
		Limitations: []string{
			"Only the keys listed in parameters are managed; other parameters keep their values. " +
				"SAP documents reading and updating parameters (PUT " +
				"IntegrationDesigntimeArtifacts(Id,Version)/$links/Configurations('<key>')), not " +
				"creating or deleting them: parameters come from the flow model. Destroying the " +
				"resource therefore leaves the values in place and only stops managing them.",
			"New values take effect at runtime only after a redeploy. Reference the parameters in " +
				"redeploy_triggers of sapintegrationsuite_integration_flow_deployment to redeploy " +
				"automatically.",
			"SAP does not document whether uploading new flow content keeps parameter values. If " +
				"it resets them, the next plan shows the managed keys as drift and applying writes " +
				"them again.",
		},
		Operations: Operations{Create: true, Read: true, Update: true, Import: true},
	},
	{
		Key:            "cloud_integration.value_mapping",
		Domain:         "cloud_integration",
		Name:           "Value Mapping",
		Description:    "A value mapping design-time artifact's content, managed as file-based content.",
		SupportStatus:  StatusPartial,
		ContractSource: SourceSAPDocumentation,
		SupportReason:  ReasonUnsafeTerraformLifecycle,
		ResourceTypes:  []string{"sapintegrationsuite_value_mapping"},
		DataSourceTypes: []string{
			"sapintegrationsuite_value_mapping",
		},
		PublicAPI:   true,
		APIProtocol: "OData V2",
		Limitations: []string{
			"No confirmed in-place update: changing name, content, or content_hash replaces the resource (Terraform deletes the artifact and uploads it again) instead of calling an unverified PUT.",
			"Explicit versions (ValueMappingDesigntimeArtifactSaveAsVersion) are not used yet; see cloud_integration.design_time_versioning.",
			"Whether Delete removes only the active version or every version of the artifact is unconfirmed against a primary source.",
		},
		Operations: Operations{Create: true, Read: true, Update: false, Delete: true, Import: true},
	},
	{
		Key:             "cloud_integration.value_mapping_deployment",
		Domain:          "cloud_integration",
		Name:            "Value Mapping Deployment",
		Description:     "The runtime deployment state of a value mapping, managed independently of its design-time content.",
		SupportStatus:   StatusSupported,
		ContractSource:  SourceSAPDocumentation,
		ResourceTypes:   []string{"sapintegrationsuite_value_mapping_deployment"},
		DataSourceTypes: []string{},
		PublicAPI:       true,
		APIProtocol:     "OData V2",
		Operations:      Operations{Create: true, Read: true, Update: true, Delete: true, Import: true, Deploy: true, Undeploy: true},
	},
	{
		Key:    "cloud_integration.value_mapping_entry",
		Domain: "cloud_integration",
		Name:   "Value Mapping Entry",
		Description: "Individual source/target value pairs inside a value mapping's " +
			"agency/identifier pair, managed through UpsertValMaps, UpdateDefaultValMap and " +
			"DeleteValMaps.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonUnsafeTerraformLifecycle,
		PublicAPI:     true,
		APIProtocol:   "OData V2 (function imports)",
		Limitations: []string{
			"Tenant check of 2026-09-27 on a synthetic value mapping: the first UpsertValMaps with " +
				"IsConfigured=true switched the agency pair to State Configured and dropped the two " +
				"values the design-time content had defined; a second UpsertValMaps with the same " +
				"source value added a second entry instead of changing the first, and SAP made the " +
				"newest one the default; DeleteValMaps for the pair answered 202 and removed nothing; " +
				"every call left the artifact in version Draft. A resource could neither update an " +
				"entry in place nor destroy what it created, so entries stay unmanaged.",
			"Create and update are documented (POST /UpsertValMaps?Id=&Version=&SrcAgency=&SrcId=" +
				"&TgtAgency=&TgtId=&SrcValue=&TgtValue=&IsConfigured=, returning a ValMap with Id " +
				"and Value{SrcValue, TgtValue}), and reading goes through ValMapSchema(...)/ValMaps. " +
				"Deletion is not documented per entry: DeleteValMaps takes only Id, Version and the " +
				"agency/identifier pair, and the tenant $metadata has no ValMapId parameter for it.",
			"A resource per entry could therefore not implement destroy. A resource owning a whole " +
				"agency/identifier pair could, but SAP does not document whether UpsertValMaps creates " +
				"a missing pair, what IsConfigured=false does (SAP only says it should always be true " +
				"once configured), or whether DeleteValMaps removes the pair or only its entries. " +
				"Those need a check against a tenant before business data is written.",
			"Entries set through the API also compete with the value mapping's own content: " +
				"uploading content through sapintegrationsuite_value_mapping replaces them.",
		},
	},
	{
		Key:    "cloud_integration.design_time_versioning",
		Domain: "cloud_integration",
		Name:   "Design-Time Artifact Versioning",
		Description: "Saving a design-time artifact under an explicit version number (for example " +
			"1.0.3) instead of working only on the active draft.",
		SupportStatus:          StatusPartial,
		ContractSource:         SourceSAPDocumentation,
		UndocumentedOperations: []string{"save_as_version on script collections (ScriptCollectionDesigntimeArtifactSaveAsVersion is only in $metadata; integration flows and message mappings are documented)"},
		SupportReason:          ReasonNotImplemented,
		ResourceTypes: []string{
			"sapintegrationsuite_integration_flow",
			"sapintegrationsuite_message_mapping",
			"sapintegrationsuite_script_collection",
		},
		PublicAPI:   true,
		APIProtocol: "OData V2 (function imports)",
		Limitations: []string{
			"save_as_version calls <Artifact>SaveAsVersion?Id=''&SaveAsVersion='' after the content " +
				"upload, as SAP Help documents for IntegrationDesigntimeArtifactSaveAsVersion; the " +
				"tenant $metadata confirms the same function import for message mappings and script " +
				"collections. A new version is saved only when save_as_version changes.",
			"Not yet available for value mappings: that resource replaces the artifact on every " +
				"change, and a version bump must not recreate it. Data types, message types, fault " +
				"message types and service interfaces have the function import too but no resource.",
			"SAP does not document what happens when the version already exists or is lower than " +
				"the current one; SAP's error is passed through unchanged.",
		},
		Operations: Operations{Create: true, Update: true},
	},
	{
		Key:    "cloud_integration.data_type",
		Domain: "cloud_integration",
		Name:   "Data Type",
		Description: "A reusable XSD data type artifact (simple or complex) used by message types " +
			"and mappings.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonPublicAPIIncomplete,
		PublicAPI:     true,
		APIProtocol:   "OData V2",
		Limitations: []string{
			"The tenant $metadata defines DataTypeDesigntimeArtifacts (Id and Version as key, " +
				"PackageId, Name, Namespace, Description, IsSimpleType, ArtifactContent) and a " +
				"DataTypeDesigntimeArtifactSaveAsVersion function import. SAP Help documents only the " +
				"UI and lists no API resource or example request for data types, so create, update " +
				"and delete are unverified.",
			"Tenant check of 2026-09-27: reading works (the collection and " +
				"IntegrationPackages('<id>')/DataTypeDesigntimeArtifacts return the package's data " +
				"type). A create with the message mapping's body (Id, Name, PackageId, ArtifactContent " +
				"as the exported bundle) failed with 500 \"Cannot invoke java.util.Map.containsKey(Object) " +
				"because map is null\". The next probe adds Namespace, Description and IsSimpleType; " +
				"until a create succeeds, there is no resource.",
		},
	},
	{
		Key:           "cloud_integration.message_type",
		Domain:        "cloud_integration",
		Name:          "Message Type",
		Description:   "A message type artifact that wraps a data type as a message root element.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonPublicAPIIncomplete,
		PublicAPI:     true,
		APIProtocol:   "OData V2",
		Limitations: []string{
			"The tenant $metadata defines MessageTypeDesigntimeArtifacts (Id and Version as key, " +
				"PackageId, Name, Namespace, Description, DataTypeUsed, ArtifactContent) and a " +
				"SaveAsVersion function import; FaultMessageTypeDesigntimeArtifacts has the same shape " +
				"for fault messages. SAP Help documents only the UI for both.",
		},
	},
	{
		Key:    "cloud_integration.service_interface",
		Domain: "cloud_integration",
		Name:   "Service Interface",
		Description: "A service interface artifact describing operations and their request, " +
			"response and fault message types.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonPublicAPIIncomplete,
		PublicAPI:     true,
		APIProtocol:   "OData V2",
		Limitations: []string{
			"The tenant $metadata defines ServiceInterfaceDesigntimeArtifacts (Id and Version as " +
				"key, PackageId, Name, Namespace, Description, ArtifactContent, Resources navigation) " +
				"and a SaveAsVersion function import. SAP Help documents creating, editing and " +
				"importing service interfaces from the Enterprise Services Repository only in the UI.",
		},
	},
	{
		Key:                    "cloud_integration.script_collection",
		Domain:                 "cloud_integration",
		Name:                   "Script Collection",
		Description:            "A reusable collection of Groovy/JavaScript scripts shared across multiple integration flows.",
		SupportStatus:          StatusSupported,
		ContractSource:         SourceSAPDocumentation,
		UndocumentedOperations: []string{"update of the content (PUT; SAP documents create, upload of resources and deploy)"},
		ResourceTypes:          []string{"sapintegrationsuite_script_collection"},
		DataSourceTypes:        []string{"sapintegrationsuite_script_collection"},
		PublicAPI:              true,
		APIProtocol:            "OData V2",
		Operations:             Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},
	{
		Key:             "cloud_integration.script_collection_deployment",
		Domain:          "cloud_integration",
		Name:            "Script Collection Deployment",
		Description:     "The runtime deployment state of a script collection, managed independently of its design-time content.",
		SupportStatus:   StatusSupported,
		ContractSource:  SourceSAPDocumentation,
		ResourceTypes:   []string{"sapintegrationsuite_script_collection_deployment"},
		DataSourceTypes: []string{},
		PublicAPI:       true,
		APIProtocol:     "OData V2",
		Operations:      Operations{Create: true, Read: true, Update: true, Delete: true, Import: true, Deploy: true, Undeploy: true},
	},
	{
		Key:                    "cloud_integration.message_mapping",
		Domain:                 "cloud_integration",
		Name:                   "Message Mapping",
		Description:            "A reusable, package-level message mapping artifact's design-time content — not the inline/local message mapping step an integration flow can also define directly inside its own content.",
		SupportStatus:          StatusSupported,
		ContractSource:         SourceSAPDocumentation,
		UndocumentedOperations: []string{"update of the content (PUT; SAP documents read, create and delete)"},
		ResourceTypes:          []string{"sapintegrationsuite_message_mapping"},
		DataSourceTypes:        []string{"sapintegrationsuite_message_mapping"},
		PublicAPI:              true,
		APIProtocol:            "OData V2",
		Operations:             Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},
	{
		Key:             "cloud_integration.message_mapping_deployment",
		Domain:          "cloud_integration",
		Name:            "Message Mapping Deployment",
		Description:     "The runtime deployment state of a message mapping, managed independently of its design-time content.",
		SupportStatus:   StatusSupported,
		ContractSource:  SourceSAPDocumentation,
		ResourceTypes:   []string{"sapintegrationsuite_message_mapping_deployment"},
		DataSourceTypes: []string{},
		PublicAPI:       true,
		APIProtocol:     "OData V2",
		Operations:      Operations{Create: true, Read: true, Update: true, Delete: true, Import: true, Deploy: true, Undeploy: true},
	},
	{
		Key:    "cloud_integration.service_endpoints",
		Domain: "cloud_integration",
		Name:   "Service Endpoints",
		Description: "Read-only discovery of the runtime service endpoints (entry point URLs and " +
			"API definition links) SAP generates for deployed Cloud Integration content.",
		SupportStatus:   StatusReadOnly,
		ContractSource:  SourceSAPDocumentation,
		SupportReason:   ReasonUnsafeTerraformLifecycle,
		DataSourceTypes: []string{"sapintegrationsuite_service_endpoints"},
		PublicAPI:       true,
		APIProtocol:     "OData V2",
		Limitations: []string{
			"Discovery only, by design: SAP generates service endpoints from deployed content and " +
				"there is no create/update/delete API for them, so this provider intentionally has " +
				"no matching resource type — see docs/guides/service-endpoints.md.",
			"No single-endpoint (sapintegrationsuite_service_endpoint) data source exists: this " +
				"project could not confirm that Name uniquely and stably identifies exactly one " +
				"service endpoint, so only the collection data source " +
				"(sapintegrationsuite_service_endpoints, with optional name/protocol filters) is " +
				"implemented, to avoid a lookup data source that silently returns the wrong result " +
				"if more than one endpoint ever matches.",
			"All exposed fields are checked against a tenant $metadata document. API definition " +
				"links carry a url and a name; the format \"type\" (oas-json, edmx, ...) that SAP's " +
				"documentation mentions is not a property of the Definition entity, and earlier " +
				"releases that exposed it always returned an empty value.",
		},
		Operations: Operations{Read: true},
	},
	{
		Key:    "cloud_integration.integration_adapter",
		Domain: "cloud_integration",
		Name:   "Integration Adapter",
		Description: "A custom Integration Adapter design-time artifact (a *.esa archive built with " +
			"the SAP Adapter SDK), imported into a Cloud Integration package. Cloud Foundry " +
			"environment only.",
		SupportStatus:   StatusPartial,
		ContractSource:  SourceSAPDocumentation,
		SupportReason:   ReasonPublicAPIIncomplete,
		ResourceTypes:   []string{"sapintegrationsuite_integration_adapter"},
		DataSourceTypes: []string{"sapintegrationsuite_integration_adapter"},
		PublicAPI:       true,
		APIProtocol:     "OData V2",
		Planned:         true,
		Limitations: []string{
			"This provider's evidence base for this entity is thinner than for the sibling " +
				"design-time artifact types it manages: SAP's own \"Integration Adapter Example " +
				"Requests, Cloud Foundry Environment\" documentation shows only Delete (confirming " +
				"the entity is keyed by Id alone, not the composite (Id, Version) key every other " +
				"design-time artifact type in this API uses) and the Deploy action — no Create or " +
				"Read example was found. The property names and the single Id key are confirmed by " +
				"a tenant $metadata document, but no SAP example shows a Create request for this " +
				"entity. See docs/guides/integration-adapters.md.",
			"No in-place update: SAP documents that importing an ID that already exists on the " +
				"tenant is rejected as an error, which is positive evidence against a working " +
				"reimport-to-update flow, so every attribute is RequiresReplace rather than an " +
				"unverified PUT/PATCH.",
			"The type and application shown in SAP's import dialog cannot be set: the " +
				"IntegrationAdapterDesigntimeArtifact entity has only Id, Version, PackageId, Name, " +
				"ArtifactContent and Description (tenant $metadata). Earlier releases sent Type and " +
				"Application anyway; both attributes were removed. description is exposed read-only.",
			"Distinct from SAP Business Accelerator Hub prebundled adapters (a different import/" +
				"auto-deploy lifecycle reached from inside the integration flow editor) and from " +
				"Integration Suite capability activation — see docs/guides/integration-adapters.md " +
				"for why these are not the same feature.",
			"There is no sapintegrationsuite_integration_adapters collection data source: no " +
				"confirmed list/filter contract for this entity set was found, unlike " +
				"ServiceEndpoints' documented Name/Protocol filters.",
		},
		Operations: Operations{Create: true, Read: true, Delete: true, Import: true},
	},
	{
		Key:    "cloud_integration.integration_adapter_deployment",
		Domain: "cloud_integration",
		Name:   "Integration Adapter Deployment",
		Description: "The runtime deployment state of a custom Integration Adapter, independent of " +
			"its design-time content lifecycle.",
		SupportStatus:   StatusPartial,
		ContractSource:  SourceSAPDocumentation,
		SupportReason:   ReasonPublicAPIIncomplete,
		ResourceTypes:   []string{"sapintegrationsuite_integration_adapter_deployment"},
		DataSourceTypes: []string{},
		PublicAPI:       true,
		APIProtocol:     "OData V2",
		Planned:         true,
		Limitations: []string{
			"Deploy is confirmed directly from SAP's own example request: POST " +
				"DeployIntegrationAdapterDesigntimeArtifact?Id='...', singular \"Artifact\" (matching " +
				"every sibling deploy action in this API), with no Version query parameter (unlike " +
				"every sibling deploy action, consistent with Id being this entity's only confirmed " +
				"key).",
			"Runtime status polling and undeploy reuse the same shared IntegrationRuntimeArtifacts " +
				"entity every other *_deployment resource in this provider polls/undeploys through, " +
				"by analogy — this project could not independently confirm that a deployed custom " +
				"adapter surfaces through that same shared entity as opposed to an adapter-specific " +
				"status/undeploy mechanism (for example BuildAndDeployStatus). See " +
				"docs/guides/integration-adapters.md.",
			"Whether a fresh deployment's runtime status becomes visible immediately or after a " +
				"build/deploy delay specific to adapters (as opposed to ordinary content deployment) " +
				"was not confirmed.",
		},
		Operations: Operations{Create: true, Read: true, Delete: true, Deploy: true, Undeploy: true},
	},
	{
		Key:    "cloud_integration.custom_tag_configuration",
		Domain: "cloud_integration",
		Name:   "Custom Tag Configuration",
		Description: "The tenant-wide set of custom tags integration package owners are asked, or " +
			"required, to classify their packages with.",
		SupportStatus:   StatusPartial,
		ContractSource:  SourceSAPDocumentation,
		SupportReason:   ReasonUnsafeTerraformLifecycle,
		ResourceTypes:   []string{"sapintegrationsuite_custom_tag_configuration"},
		DataSourceTypes: []string{"sapintegrationsuite_custom_tag_configuration"},
		PublicAPI:       true,
		APIProtocol:     "OData V2",
		Planned:         true,
		Limitations: []string{
			"No confirmed delete or clear operation exists for this entity anywhere in SAP's public " +
				"documentation. Destroying this resource in Terraform returns an explicit error " +
				"rather than guessing that an empty overwrite means delete, or silently dropping " +
				"Terraform state while leaving the tenant's configuration untouched — see " +
				"docs/guides/custom-tag-configurations.md.",
			"Create and Update both use the same confirmed POST .../CustomTagConfigurations?" +
				"Overwrite=true operation (SAP documents no separate plain-POST-without-Overwrite " +
				"path this provider relies on), sending the complete desired tag list every time. " +
				"Whether Overwrite=true is a full replace (removing tags not present in the new " +
				"list) is strongly implied by the word \"Overwrite\" and by the fact the documented " +
				"payload is the complete configuration, not a delta, but SAP's documentation never " +
				"uses the word \"replace\" explicitly.",
			"Whether tag names must be unique, whether permitted values are case-sensitive, and " +
				"whether SAP preserves submitted ordering are all unconfirmed by SAP's " +
				"documentation. This provider enforces tag-name uniqueness itself and treats " +
				"ordering (of both tags and permitted values) as not semantically meaningful, " +
				"modeling both as unordered Terraform sets so a reordered response never produces " +
				"a spurious diff.",
			"One documented example response shows a single-element permittedValues array " +
				"containing a comma-separated string (\"Mr. Bean, Ms. Bean\") rather than two " +
				"separate array elements; this is treated as a documentation artifact, not a " +
				"confirmed wire format, since every other array-typed field in SAP's own examples " +
				"(and everywhere else in this provider) uses one array element per value.",
		},
		Operations: Operations{Create: true, Read: true, Update: true, Delete: false, Import: true},
	},
	{
		Key:           "cloud_integration.message_processing_logs",
		Domain:        "cloud_integration",
		Name:          "Message Processing Logs",
		Description:   "Runtime message processing log records for deployed integration flows.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonOutOfScope,
		PublicAPI:     true,
		APIProtocol:   "OData V2",
		Limitations: []string{
			"Monitoring/operational data, not infrastructure state this provider manages — see docs/provider-scope.md.",
		},
	},
	{
		Key:    "cloud_integration.message_stores",
		Domain: "cloud_integration",
		Name:   "Message Store Entries / JMS Resources",
		Description: "Runtime persisted-message-store entries (created by the Persist step) and " +
			"JMS queue resource metadata used by deployed integration flows. Data Stores, Data " +
			"Store Entries, Variables, and Number Ranges — all part of the same broader Message " +
			"Stores API family — each have their own dedicated catalog entry; see " +
			"cloud_integration.data_store, cloud_integration.data_store_entry, " +
			"cloud_integration.variable, and cloud_integration.number_range.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonOutOfScope,
		PublicAPI:     true,
		APIProtocol:   "OData V2",
		Limitations: []string{
			"Payload/queue content is operational data, not desired state this provider manages.",
		},
	},
	{
		Key:    "cloud_integration.number_range",
		Domain: "cloud_integration",
		Name:   "Number Range",
		Description: "A Number Ranges object: generates unique interchange numbers for outbound " +
			"EDI/EDIFACT documents, with a static configuration (min/max/description/rotate/" +
			"field length) and a live runtime counter (CurrentValue, the UI's \"Next Value\") " +
			"that advances as deployed content consumes numbers.",
		SupportStatus:          StatusSupported,
		ContractSource:         SourceSAPDocumentation,
		UndocumentedOperations: []string{"read by name", "delete", "import"},
		ResourceTypes:          []string{"sapintegrationsuite_number_range"},
		PublicAPI:              true,
		APIProtocol:            "OData V2",
		Limitations: []string{
			"SAP documents only POST and PUT. GET by name and DELETE were verified on a tenant " +
				"(September 2026: GET returned every field as sent, DELETE answered 202 and a read " +
				"afterwards 404), so with enable_unofficial the resource reads, deletes and imports. " +
				"The collection rejects $top with 501, and every write answered 202 without a body.",
			"Without enable_unofficial the resource uses only POST and PUT: a refresh keeps the state " +
				"(no drift detection, and deployed_by/deployed_on stay empty), an update must change " +
				"current_value_wo_version because SAP needs a counter in the PUT and the live one " +
				"cannot be read, and delete and import are refused. Create then cannot check whether " +
				"the name already exists.",
			"The runtime counter is a write-only, version-gated attribute (current_value_wo / " +
				"current_value_wo_version), sent on create and only when the version marker changes; " +
				"current_value reports the live value. After an import the first apply records the " +
				"marker without touching the counter.",
			"SAP rejects a PUT without CurrentValue (500, object unchanged), so every other update " +
				"reads the live counter right before the PUT and sends it back; a number consumed " +
				"during that round trip would be handed out again.",
			"With enable_unofficial, create stops when the name already exists, because SAP does not " +
				"document what a create on an existing name does. Names may only contain letters, digits, " +
				"spaces and underscores; SAP rejects anything else with a 500 and says so in its message.",
			"SAP documents an Edge Integration Cell path (/location/<id>/api/v1/NumberRanges); " +
				"runtime_location_id is not offered; Edge Integration Cell targeting is not supported.",
		},
		Operations: Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},
	{
		Key:    "cloud_integration.variable",
		Domain: "cloud_integration",
		Name:   "Variable",
		Description: "A tenant-persisted runtime value written by an integration flow's \"Write " +
			"Variables\" step, shared across steps of the same flow (local) or across every flow " +
			"deployed on the tenant (global).",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonOutOfScope,
		PublicAPI:     true,
		APIProtocol:   "OData V2",
		Limitations: []string{
			"SAP documents exactly one public operation for this entity: GET .../Variables(...)/" +
				"$value, which downloads the raw value with no structured metadata (no Visibility/ " +
				"UpdatedAt/RetainUntil fields are returned by this endpoint). There is no collection " +
				"GET, no POST, no PUT, and no confirmed DELETE — Variables are created and updated " +
				"exclusively by deployed integration flow content, an entirely different ownership " +
				"domain than Terraform-managed infrastructure.",
			"A read-only data source was deliberately not implemented: the only confirmed read " +
				"operation returns nothing but the raw runtime value itself, with no safer " +
				"metadata-only alternative available, and this provider does not place arbitrary " +
				"runtime business values into Terraform state merely because an API can return " +
				"them — see docs/provider-scope.md.",
		},
	},
	{
		Key:    "cloud_integration.data_store",
		Domain: "cloud_integration",
		Name:   "Data Store",
		Description: "A tenant-persisted runtime container of Data Store Entries, created " +
			"implicitly by an integration flow's Data Store Write step (or an XI adapter's " +
			"Temporary Storage option) the first time it writes an entry.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonOutOfScope,
		PublicAPI:     true,
		APIProtocol:   "OData V2",
		Limitations: []string{
			"SAP documents exactly one public operation for this entity: GET .../DataStores?" +
				"overdueonly=true, an aggregate monitoring endpoint (NumberOfMessages/" +
				"NumberOfOverdueMessages per store) — the same class of runtime monitoring data as " +
				"cloud_integration.message_processing_logs, not configuration. There is no " +
				"independent declarative creation API: a Data Store comes into existence only as a " +
				"side effect of deployed integration flow content.",
			"The DataStores API does not support $filter, $inlinecount, $orderby, $skip, $top, " +
				"$expand, or $select (confirmed directly from SAP's own documentation).",
		},
	},
	{
		Key:    "cloud_integration.data_store_entry",
		Domain: "cloud_integration",
		Name:   "Data Store Entry",
		Description: "A single runtime message (payload and headers) persisted inside a Data " +
			"Store by an integration flow's Data Store Write step, read back by a Data Store Get " +
			"or Select step, and deleted only by a Data Store Delete step.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonOutOfScope,
		PublicAPI:     true,
		APIProtocol:   "OData V2",
		Limitations: []string{
			"SAP documents only GET operations for this entity (a single entry by composite key, " +
				"all entries for a store, and all entries for a message ID) — every field (Status, " +
				"MessageId, DueAt, CreatedAt, RetainUntil) is runtime business-message state, not " +
				"infrastructure desired state. Delete exists only as a design-time integration flow " +
				"step (entry-by-entry or bulk via an XPath-derived ID list at runtime), never as a " +
				"REST call this provider could wrap in a Terraform destroy.",
			"Deliberately out_of_scope rather than not_implemented: this provider does not manage " +
				"business message payloads or place them into Terraform state, and 'terraform " +
				"destroy' semantics are not an excuse to expose operational message deletion as " +
				"desired infrastructure state — see docs/provider-scope.md.",
		},
	},
	{
		Key:    "cloud_integration.archiving",
		Domain: "cloud_integration",
		Name:   "Archiving Configuration",
		Description: "Archiving of message processing logs (and, for Trading Partner Management, " +
			"B2B interchange payloads) to an external CMIS repository.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonUnsafeTerraformLifecycle,
		PublicAPI:     true,
		APIProtocol:   "OData V2 (function imports)",
		Limitations: []string{
			"Tenant-wide activation is a one-way switch: the tenant $metadata has " +
				"activateArchivingConfiguration and activateB2BArchivingConfiguration (POST, no " +
				"parameters) and read-only ArchivingConfigurations/B2BArchivingConfigurations (Id, " +
				"Active), but no deactivation. SAP's \"Enable Archiving\" page adds that the CMS " +
				"metadata properties cannot be changed once archiving is enabled. A resource could " +
				"never implement destroy, so none is offered.",
			"Per-integration-flow settings (sender and receiver channel messages, persisted " +
				"messages, log attachments) are documented only in the Integration Content Monitor; the " +
				"Archiving* flags in the $metadata sit on MessageProcessingLog and record what was " +
				"active when a message ran.",
			"The destination (CloudIntegration_Archive / CloudIntegration_B2BArchive) is a BTP " +
				"destination, outside this provider. The KPI entity sets are monitoring data.",
		},
	},

	// --- Security ---
	{
		Key:                    "security.access_policy",
		Domain:                 "security",
		Name:                   "Access Policy",
		Description:            "An access policy restricting which artifacts a role can access.",
		SupportStatus:          StatusSupported,
		ContractSource:         SourceSAPTooling,
		UndocumentedOperations: []string{"description update (PATCH, verified on a tenant)"},
		ResourceTypes:          []string{"sapintegrationsuite_access_policy"},
		DataSourceTypes: []string{
			"sapintegrationsuite_access_policy",
		},
		PublicAPI:   true,
		APIProtocol: "OData V2",
		Limitations: []string{
			"Runtime targeting is read-only. The runtimes a policy is replicated to and their " +
				"replication state can be read with " +
				"sapintegrationsuite_access_policy_runtime_assignments; whether they can be written " +
				"through the API is not documented, so they are chosen in the UI. Earlier releases " +
				"exposed a reconciliation_status attribute; it was removed because the AccessPolicies " +
				"entity has no such property.",
			"role_name is matched by SAP against the Values attribute of a BTP custom role. That role " +
				"and the role collection granting it belong to the SAP/btp provider; this provider only " +
				"passes the string through.",
			"The full lifecycle was verified on a tenant (September 2026): create (201 with the new " +
				"ID), read by Int64 key, lookup by role name with $filter, a description change with " +
				"PATCH (PUT answers 501), reference create with the AccessPolicy link (201), reference " +
				"and policy delete (204).",
			"role_name forces replacement: it is the policy's identity, and renaming in place was not " +
				"tested. Replacing a policy also deletes its artifact references on SAP's side.",
			"description is limited to 200 characters. SAP's documentation and the tenant $metadata " +
				"declare no limit, but a tenant accepts a longer description on create and stores only " +
				"its first 200 characters (observed on a tenant). The provider rejects a longer " +
				"description at plan time instead of letting the apply fail after the policy exists.",
			"role_name is limited to 200 characters for the same reason: a tenant probe " +
				"(2026-10-03) found that SAP stores only the first 200 characters of a longer role " +
				"name without an error. The provider rejects a longer name at plan time.",
		},
		Operations: Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},
	{
		Key:            "security.access_policy_reference",
		Domain:         "security",
		Name:           "Access Policy Reference",
		Description:    "A single artifact reference (attribute/operator/value match rule) on an access policy.",
		SupportStatus:  StatusSupported,
		ContractSource: SourceSAPTooling,
		UndocumentedOperations: []string{"create a reference to an artifact type SAP does not document for " +
			"access policies (eight types SAP's API lists, verified on a tenant)"},
		ResourceTypes: []string{"sapintegrationsuite_access_policy_reference"},
		DataSourceTypes: []string{
			"sapintegrationsuite_access_policy_reference",
		},
		PublicAPI:   true,
		APIProtocol: "OData V2",
		Limitations: []string{
			"No in-place update: SAP's UI can edit a reference, but the only public contract found " +
				"(SAP's own CI/CD tooling) creates and deletes references and never updates one, so every " +
				"attribute forces replacement. Replacement deletes the old reference before creating the " +
				"new one, which briefly leaves the matched artifacts unprotected by it.",
			"artifact_type, attribute and operator are checked at plan time against the wire values " +
				"and combinations SAP allows: 14 artifact types, Name or ID, exactString or " +
				"regularExpression; Integration Package only with exactString, message queues, global " +
				"variables and global data stores only by Name. SAP Help documents the types by UI label " +
				"only; the constants come from SAP's audit log documentation and from the tenant, which " +
				"names them in its 400 answers and accepted every listed combination (October 2026). " +
				"Eight further types that SAP accepts but does not document for access policies " +
				"(credentials, secure parameters, adapters, service interfaces, fault message types) need " +
				"enable_unofficial.",
			"Releases before this correction sent invented field names (ArtifactType, Attribute, " +
				"Operator, Value) and could not have worked against a real tenant.",
			"name is limited to 50 characters, description to 200 and value to 150. SAP declares no " +
				"limit, but a tenant probe (2026-10-03) found that it stores only the first characters " +
				"of a longer value without an error. The provider rejects longer values at plan time.",
		},
		Operations: Operations{Create: true, Read: true, Update: false, Delete: true, Import: true},
	},
	{
		Key:    "security.user_credential",
		Domain: "security",
		Name:   "User Credential",
		Description: "A \"User Credentials\" security material artifact: a username/password credential " +
			"integration flow adapters use for outbound basic or username-token authentication.",
		SupportStatus:   StatusPartial,
		ContractSource:  SourceSAPDocumentation,
		SupportReason:   ReasonPublicAPIIncomplete,
		ResourceTypes:   []string{"sapintegrationsuite_user_credential"},
		DataSourceTypes: []string{"sapintegrationsuite_user_credential"},
		PublicAPI:       true,
		APIProtocol:     "OData V2",
		Planned:         true,
		Limitations: []string{
			"The password is never returned by SAP's read API; password_wo/password_wo_version are " +
				"write-only attributes (Terraform CLI 1.11+ required) and drift on the password value " +
				"itself cannot be detected — only readable metadata (user, description, kind, " +
				"company_id) is compared on Read.",
			"Update is implemented as a full PUT redeploy, matching SAP's documented \"Edit\" action " +
				"for Credentials artifacts, and resends password_wo on every apply that touches this " +
				"resource (SAP documents re-entering the secret on every edit for the sibling OAuth2 " +
				"Client Credentials artifact; this provider assumes the same requirement here since it " +
				"could not find a documented exception for User Credentials).",
			"Kind and CompanyId are properties of UserCredential in the tenant $metadata, and a tenant " +
				"rejected a create without Kind. SAP does not document the accepted Kind values beyond " +
				"the UI's types; kind=\"SuccessFactors\" and \"OpenConnectors\" were not tested against those " +
				"systems. See docs/guides/security-content.md.",
			"Deployment status (SAP's UI shows Stored/Deployed/Error) is not exposed: this project could " +
				"not confirm the OData property name for it, and would rather omit a computed attribute " +
				"than expose one that is silently always empty.",
			"Verified on a tenant (September 2026): SAP rejects a create without Kind (\"must not " +
				"be empty or null\") or with Description null, and reports a generic credential's kind " +
				"as \"default\". kind therefore defaults to \"default\", and Kind, Description and " +
				"CompanyId are always sent. Create, read, update (PUT) and delete were exercised.",
		},
		Operations: Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},
	{
		Key:    "security.oauth2_client_credential",
		Domain: "security",
		Name:   "OAuth2 Client Credential",
		Description: "An \"OAuth2 Client Credentials\" security material artifact: the client ID, " +
			"client secret, and token service URL an integration flow adapter uses for the OAuth2 " +
			"client credentials grant (RFC 6749) on outbound requests.",
		SupportStatus:   StatusPartial,
		ContractSource:  SourceSAPDocumentation,
		SupportReason:   ReasonPublicAPIIncomplete,
		ResourceTypes:   []string{"sapintegrationsuite_oauth2_client_credential"},
		DataSourceTypes: []string{"sapintegrationsuite_oauth2_client_credential"},
		PublicAPI:       true,
		APIProtocol:     "OData V2",
		Planned:         true,
		Limitations: []string{
			"The client secret is never returned by SAP's read API; client_secret_wo/" +
				"client_secret_wo_version are write-only attributes (Terraform CLI 1.11+ required) and " +
				"drift on the secret value itself cannot be detected.",
			"client_authentication, scope_content_type, resource and audience map to the " +
				"ClientAuthentication, ScopeContentType, Resource and Audience properties confirmed by " +
				"a tenant $metadata. Their accepted constants are undocumented, so they are passed " +
				"through; they are Optional+Computed so every PUT resends values set in the UI.",
			"Custom parameters are not managed: $metadata shows them as a CustomParameters " +
				"navigation (Key, Value, SendAsPartOf, all three forming the key), but not whether they " +
				"are written by deep insert or separately. The UI's grant-type placement (URL or body) " +
				"has no API property at all. Because a PUT replaces the entity, custom parameters set " +
				"in the UI may not survive an update through Terraform; this has not been verified.",
			"Update is implemented as a full PUT redeploy and resends client_secret_wo on every apply " +
				"that touches this resource, matching SAP's documented requirement to re-enter the " +
				"client secret on every edit.",
			"OAuth2 Authorization Code and OAuth2 SAML Bearer Assertion are separate SAP artifact types " +
				"this provider does not implement: Authorization Code requires interactive human " +
				"authorization (see security.oauth2_authorization_code note in " +
				"docs/guides/security-content.md). OAuth2 SAML Bearer Assertion and the 2026 OAuth2 " +
				"Password Credentials artifact have no entity set in the tenant $metadata of /api/v1, " +
				"so there is no public API to manage them.",
		},
		Operations: Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},
	{
		Key:    "security.keystore_entry",
		Domain: "security",
		Name:   "Keystore Entry",
		Description: "Any entry (certificate, SAP-generated key pair, or other RSA/DSA/EC-keyed " +
			"entry) in the tenant's keystore, read-only.",
		SupportStatus:  StatusReadOnly,
		ContractSource: SourceSAPDocumentation,
		SupportReason:  ReasonUnsafeTerraformLifecycle,
		DataSourceTypes: []string{
			"sapintegrationsuite_keystore_entry",
			"sapintegrationsuite_keystore_entries",
		},
		PublicAPI:   true,
		APIProtocol: "OData V2",
		Limitations: []string{
			"Fields follow the tenant $metadata: besides alias, key type and size and validity, " +
				"the data sources expose entry type, owner, status, subject and issuer DN, serial " +
				"number, signature algorithm, elliptic curve, certificate version, SHA-1/256/512 " +
				"fingerprints and creation/modification details. Dates are converted from OData V2 " +
				"literals to RFC 3339.",
			"owner shows who owns an entry, but its values are not documented. The certificate and " +
				"key pair resources therefore still rely on SAP's own server-side protection when an " +
				"Update or Delete targets an SAP-owned entry, rather than trying to detect it in " +
				"advance — see docs/guides/security-content.md.",
			"No resource: this entity represents fundamentally different object types (plain " +
				"certificates, generated key pairs) with different lifecycles, so a single mutable " +
				"sapintegrationsuite_keystore_entry resource was deliberately not created — see " +
				"security.certificate and security.key_pair instead.",
		},
	},
	{
		Key:    "security.certificate",
		Domain: "security",
		Name:   "Certificate",
		Description: "A standalone X.509 certificate keystore entry (as opposed to a key pair), " +
			"typically an uploaded root or intermediate CA certificate.",
		SupportStatus:  StatusSupported,
		ContractSource: SourceSAPDocumentation,
		ResourceTypes:  []string{"sapintegrationsuite_certificate"},
		PublicAPI:      true,
		APIProtocol:    "OData V2",
		Planned:        true,
		Limitations: []string{
			"Create and update use PUT CertificateResources('<hexalias>')/$value with a plain PEM " +
				"body (Content-Type application/pkix-cert), verified on a tenant in September 2026. " +
				"Both send fingerprintVerified=true: without it SAP answered a self-signed " +
				"certificate with 409 and did not import it. Update adds update=true: without it SAP " +
				"answered 400 \"Entry with alias ... already exists\". Configuring a certificate is " +
				"therefore the decision to trust it; compare certificate_sha256 with the fingerprint " +
				"you expect.",
			"Delete uses the documented keystore mass-deletion operation (PUT " +
				"KeystoreResources('system')?deleteEntries=true) with exactly the one alias this " +
				"resource owns — there is no documented per-entry DELETE for this entity.",
			"Drift detection compares a locally-computed SHA-256 fingerprint of the certificate's DER " +
				"bytes, not raw PEM text, so line-wrapping/line-ending differences between your " +
				"configuration and SAP's own re-serialization never produce a spurious diff; a " +
				"genuinely different certificate does surface as drift.",
			"certificate is intentionally not Sensitive: public X.509 certificate content is not " +
				"confidential. This resource never handles a private key.",
		},
		Operations: Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},
	{
		Key:    "security.key_pair",
		Domain: "security",
		Name:   "Key Pair",
		Description: "An SAP-generated key pair keystore entry (private key plus X.509 certificate), " +
			"as opposed to one uploaded from outside the tenant.",
		SupportStatus:  StatusPartial,
		ContractSource: SourceSAPDocumentation,
		SupportReason:  ReasonUnsafeTerraformLifecycle,
		ResourceTypes:  []string{"sapintegrationsuite_key_pair"},
		PublicAPI:      true,
		APIProtocol:    "OData V2",
		Planned:        true,
		Limitations: []string{
			"Create confirmed field-for-field via SAP's own \"Generate a Key Pair\" documentation: " +
				"POST KeyPairGenerationRequests. The private key never leaves SAP — this resource has " +
				"no field for it and never requests one.",
			"No update operation is documented for a generated key pair: every attribute that defines " +
				"the generated key material is RequiresReplace.",
			"Only the subset SAP confirms KeystoreEntries returns (key_type, key_size, " +
				"valid_not_before, valid_not_after) is read back and refreshed on every plan; " +
				"generation-only parameters SAP does not confirm returning (signature_algorithm, " +
				"key_algorithm_parameter, the subject DN fields) are trusted from the last successful " +
				"write, not re-verified — the same reason this is `partial`, not `supported`.",
			"Delete uses the same documented keystore mass-deletion operation as " +
				"sapintegrationsuite_certificate, with exactly the one alias this resource owns.",
		},
		Operations: Operations{Create: true, Read: true, Delete: true, Import: true},
	},
	{
		Key:           "security.ssh_key",
		Domain:        "security",
		Name:          "SSH Key",
		Description:   "An SSH-capable key pair used for SFTP public-key authentication.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonOutOfScope,
		PublicAPI:     true,
		APIProtocol:   "OData V2",
		Limitations: []string{
			"Reverified for this feature family and corrected: SAP's Security Content API overview " +
				"lists no independent \"SSH Key\" resource, and the tenant keystore's own \"Creating a " +
				"Key Pair/SSH Key Pair\" UI documentation uses the identical Key Pair attribute set " +
				"(alias, key type, key size, signature algorithm, subject DN fields, validity) for " +
				"both — \"Create > Key Pair\" and \"Create > SSH Key\" are the same underlying " +
				"mechanism with a different label. The tenant $metadata does define " +
				"SSHKeyGenerationRequests (with SSHFile and Password) and SSHKeyResources, but SAP " +
				"documents no request for either, so no separate resource is built on them.",
			"An RSA or DSA sapintegrationsuite_key_pair's public key can be exported in OpenSSH " +
				"format via public_key_openssh, backed by SAP's confirmed " +
				"KeystoreEntries('<hexalias>')/Sshkey/$value — this covers the SSH use case without a " +
				"separate resource. EC key pairs are documented as unsupported for this export.",
		},
	},
	{
		Key:           "security.certificate_chain",
		Domain:        "security",
		Name:          "Certificate Chain",
		Description:   "A certificate chain associated with a key pair.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonPublicAPIIncomplete,
		PublicAPI:     true,
		APIProtocol:   "OData V2",
		Planned:       true,
		Limitations: []string{
			"The tenant $metadata defines CertificateChainResources, a media entity keyed by the " +
				"key pair's Hexalias with a KeystoreEntry navigation, and a read-only ChainCertificates " +
				"set (Hexalias, Index and certificate details). SAP Help describes chain import and " +
				"export only as a capability of the Key Pair resource and documents neither the media " +
				"type nor the request that uploads a chain, so nothing is implemented yet.",
			"Once the upload contract is confirmed, the intended shape is a resource scoped to one " +
				"key pair alias (for example sapintegrationsuite_key_pair_certificate_chain), not a " +
				"standalone global resource.",
			"Tenant check of 2026-09-27: KeystoreEntries('<hexalias>')/ChainCertificates returns the " +
				"chain of an SAP key pair (Hexalias, Index, subject, issuer, validity, fingerprints), so " +
				"reading is confirmed and a read-only chain attribute or data source is possible; " +
				"uploading a chain still has no documented media type.",
		},
	},
	{
		Key:    "security.certificate_user_mapping",
		Domain: "security",
		Name:   "Certificate-User Mapping",
		Description: "A mapping from a client certificate to an inbound user identity, used for " +
			"inbound client certificate authentication.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Planned:       false,
		Limitations: []string{
			"Reverified for this feature family: SAP's own documentation (\"Managing Certificate-to-" +
				"User Mappings\", \"Client Certificate Authentication and Certificate-to-User Mapping " +
				"(Inbound)\", \"Setting Up Inbound HTTP Connections with Certificate-to-User Mapping\") " +
				"exists only under the Neo environment, with no Cloud Foundry equivalent found in SAP's " +
				"published documentation set. This provider targets the Cloud Foundry environment (its " +
				"other Security Content resources use the Cloud Foundry \"/api/v1\" OData host), so this " +
				"catalog entry is corrected from its previous \"not_implemented\"/PublicAPI:true state " +
				"to \"no_public_api\": the feature cannot be implemented for this provider's target " +
				"environment, not merely unimplemented yet. If SAP publishes a Cloud Foundry " +
				"certificate-to-user-mapping API in the future, re-open this entry.",
		},
	},
	{
		Key:    "security.secure_parameter",
		Domain: "security",
		Name:   "Secure Parameter",
		Description: "A \"Secure Parameter\" security material artifact: an opaque confidential value " +
			"(for example for a custom adapter) deployed without an associated username.",
		SupportStatus:  StatusUnofficial,
		SupportReason:  ReasonPublicAPIIncomplete,
		ContractSource: SourceMetadataOnly,
		ResourceTypes:  []string{"sapintegrationsuite_secure_parameter"},
		PublicAPI:      true,
		APIProtocol:    "OData V2",
		Limitations: []string{
			"SAP Help documents the artifact only in the Monitor UI. The entity set comes from the " +
				"tenant $metadata (key Name; Description, SecureParam, DeployedBy, DeployedOn, Status), " +
				"and a tenant test in September 2026 verified create (POST), read by name, update " +
				"(PUT) and delete, each write answering 202 without a body.",
			"The value is write-only (secure_param_wo / secure_param_wo_version) and sent on create " +
				"and every update; SAP returns SecureParam as null. Import recovers the name and " +
				"description only, so the first apply after an import sends the configured value.",
			"Create stops when the name already exists, because SAP does not document what a create " +
				"on an existing name does. Edge Integration Cell targeting is not supported and not " +
				"offered for this resource.",
		},
		Operations: Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},
	{
		Key:    "security.known_hosts",
		Domain: "security",
		Name:   "Known Hosts (SSH)",
		Description: "The SSH \"known_hosts\" file artifact used to validate SFTP server host keys for " +
			"outbound SFTP connections.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Planned:       false,
		Limitations: []string{
			"Reverified for this feature family and strengthened from research_required to " +
				"no_public_api: unlike Secure Parameter (at least conceptually listed in SAP's " +
				"Security Content API overview's resource table), Known Hosts does not appear in " +
				"that table at all. SAP's \"Deploying an SSH Known Hosts Artifact\" documentation " +
				"describes only the Manage Security Material UI (Create > Known Hosts (SSH), " +
				"Browse/Add/Deploy), with no REST endpoint mentioned anywhere. The tenant $metadata of " +
				"/api/v1 (September 2026) has no known-hosts entity either.",
		},
	},
	{
		Key:    "security.oauth2_password_credential",
		Domain: "security",
		Name:   "OAuth2 Password Credentials",
		Description: "An OAuth2 resource owner password credentials artifact (new in 2026): user " +
			"name, password and optional client authentication for the OAuth2 password grant.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"Documented only as a Security Material UI procedure (Create > OAuth2 Password " +
				"Credentials). The tenant $metadata of /api/v1 has no entity set for it, and " +
				"OAuth2ClientCredential has no user or password property it could be stored in.",
		},
	},
	{
		Key:    "security.oauth2_saml_bearer",
		Domain: "security",
		Name:   "OAuth2 SAML Bearer Assertion",
		Description: "An OAuth2 SAML bearer assertion artifact for principal propagation to " +
			"OAuth-protected receivers.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"Documented only as a Security Material UI procedure. The tenant $metadata of /api/v1 " +
				"has no SAML bearer entity set.",
		},
	},
	{
		Key:    "security.where_used",
		Domain: "security",
		Name:   "Security Material Where-Used",
		Description: "The list of integration artifacts that reference a security material " +
			"artifact.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"Shown in the Security Material UI only; the tenant $metadata of /api/v1 has no " +
				"where-used entity or function import. If one appears it would be a read-only data " +
				"source, never mutable state.",
		},
	},
	{
		Key:    "security.pgp_keyring",
		Domain: "security",
		Name:   "PGP Keyrings",
		Description: "The tenant's PGP public and secret keyrings used by the PGP encryptor and " +
			"decryptor steps.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonPublicAPIIncomplete,
		PublicAPI:     true,
		APIProtocol:   "OData V2",
		Limitations: []string{
			"The tenant $metadata defines PgpKeyrings, PgpPublicKeyrings, PgpSecretKeyrings, " +
				"PgpKeyEntries, PgpSubKeys, PgpUserIds and keyring upload resources, but SAP Help " +
				"documents no request for any of them. Keyrings are whole-file objects, and the secret " +
				"keyring holds private keys, so a Terraform design would need a confirmed upload format " +
				"and write-only handling of the secret keyring before anything is implemented.",
			"Tenant check of 2026-09-27: PgpPublicKeyrings answers 200, but PgpKeyrings answers 404 " +
				"\"Could not find an entity set or function import\" although $metadata declares it. " +
				"Neither $metadata nor the service document decides whether a set can be addressed " +
				"(the provider uses several sets the service document does not list); only a tenant " +
				"check does.",
		},
	},

	// --- Partner Directory ---
	{
		Key:            "partner_directory.partner",
		Domain:         "partner_directory",
		Name:           "Partner",
		Description:    "A Partner ID (Pid) known to the tenant's Partner Directory.",
		SupportStatus:  StatusReadOnly,
		ContractSource: SourceSAPDocumentation,
		SupportReason:  ReasonUnsafeTerraformLifecycle,
		DataSourceTypes: []string{
			"sapintegrationsuite_partner",
			"sapintegrationsuite_partners",
		},
		PublicAPI:   true,
		APIProtocol: "OData V2",
		Limitations: []string{
			"No resource: SAP's API reads all partners and deletes a partner, but has no create " +
				"operation. A Pid comes into existence implicitly the first time a StringParameter, " +
				"BinaryParameter, AlternativePartner, AuthorizedUser, or UserCredentialParameter " +
				"references it.",
			"Deleting a partner removes the partner and all its entities (SAP's Delete Partner " +
				"description). A Partner resource's destroy could therefore erase content owned by " +
				"other Terraform resources or modules, which is the other reason this stays read-only.",
			"SAP refuses to read a single partner by key (\"Reading of single partner entities is " +
				"not supported\", tenant test September 2026); the data source filters the collection " +
				"by Pid instead. A partner exists only while it has entries: after its last entry is " +
				"deleted, DELETE Partners('<pid>') answers 404.",
		},
		Operations: Operations{Read: true},
	},
	{
		Key:            "partner_directory.string_parameter",
		Domain:         "partner_directory",
		Name:           "Partner Directory String Parameter",
		Description:    "A named text value scoped to a Partner ID (Pid).",
		SupportStatus:  StatusSupported,
		ContractSource: SourceSAPDocumentation,
		ResourceTypes:  []string{"sapintegrationsuite_partner_string_parameter"},
		DataSourceTypes: []string{
			"sapintegrationsuite_partner_string_parameter",
			"sapintegrationsuite_partner_string_parameters",
		},
		PublicAPI:   true,
		APIProtocol: "OData V2",
		Limitations: []string{
			"Partner Directory data is stored unencrypted; do not store passwords, secrets, or " +
				"other sensitive values in a string parameter — see " +
				"docs/guides/partner-directory.md.",
		},
		Operations: Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},
	{
		Key:            "partner_directory.binary_parameter",
		Domain:         "partner_directory",
		Name:           "Partner Directory Binary Parameter",
		Description:    "A named binary value (for example an XSD schema or certificate) scoped to a Partner ID (Pid).",
		SupportStatus:  StatusSupported,
		ContractSource: SourceSAPDocumentation,
		ResourceTypes:  []string{"sapintegrationsuite_partner_binary_parameter"},
		DataSourceTypes: []string{
			"sapintegrationsuite_partner_binary_parameter",
		},
		PublicAPI:   true,
		APIProtocol: "OData V2",
		Limitations: []string{
			"Partner Directory data is stored unencrypted; do not store secrets, private keys, or " +
				"other sensitive content — see docs/guides/partner-directory.md.",
			"Values can be up to 1.5 MiB (1572864 bytes), the MaxLength of BinaryParameter.Value in " +
				"the tenant $metadata and the size SAP's entity types page gives; some SAP pages still " +
				"say 260 KB. The provider checks the limit before uploading.",
			"content_type takes SAP's values (xml, xsl, xsd, json, text, zip, gz, zlib, crt), for " +
				"the text types optionally with an encoding such as \"xml;encoding=UTF-8\".",
		},
		Operations: Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},
	{
		Key:            "partner_directory.alternative_partner",
		Domain:         "partner_directory",
		Name:           "Alternative Partner",
		Description:    "A mapping from an external identity tuple (agency, scheme, external_id) to an internal Partner ID (Pid).",
		SupportStatus:  StatusSupported,
		ContractSource: SourceSAPDocumentation,
		ResourceTypes:  []string{"sapintegrationsuite_alternative_partner"},
		DataSourceTypes: []string{
			"sapintegrationsuite_alternative_partner",
		},
		PublicAPI:   true,
		APIProtocol: "OData V2",
		Limitations: []string{
			"SAP's actual OData key is a hex encoding of agency/scheme/external_id, not the plain " +
				"strings; this provider computes and hides that transform, and uses the three hex " +
				"segments as the Terraform import ID so any character combination round-trips " +
				"unambiguously — see docs/guides/partner-directory.md.",
		},
		Operations: Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},
	{
		Key:            "partner_directory.authorized_user",
		Domain:         "partner_directory",
		Name:           "Partner Directory Authorized User",
		Description:    "A mapping from a communication user to the Partner ID (Pid) that user is authorized to act as.",
		SupportStatus:  StatusSupported,
		ContractSource: SourceSAPDocumentation,
		ResourceTypes:  []string{"sapintegrationsuite_partner_authorized_user"},
		DataSourceTypes: []string{
			"sapintegrationsuite_partner_authorized_user",
		},
		PublicAPI:   true,
		APIProtocol: "OData V2",
		Limitations: []string{
			"SAP stores users lowercased (its example creates \"MyUser\" and returns \"myuser\"), " +
				"so the provider rejects uppercase letters in user at plan time.",
			"Manages only the Partner Directory mapping, never the underlying BTP user, OAuth " +
				"client, or communication user credential itself.",
		},
		Operations: Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},
	{
		Key:            "partner_directory.user_credential_parameter",
		Domain:         "partner_directory",
		Name:           "Partner Directory User Credential Parameter",
		Description:    "A communication username/password credential scoped to a Partner ID (Pid).",
		SupportStatus:  StatusSupported,
		ContractSource: SourceSAPDocumentation,
		ResourceTypes:  []string{"sapintegrationsuite_partner_user_credential_parameter"},
		PublicAPI:      true,
		APIProtocol:    "OData V2",
		Limitations: []string{
			"The password is a write-only attribute (password_wo): Terraform never stores it in " +
				"plan or state, and this provider never requests or reads a password back from " +
				"SAP, which does not document returning one. Requires Terraform CLI 1.11 or later.",
			"User and password change in place: SAP documents POST with the same Pid and Id as " +
				"the update (PUT is not supported). Changing password_wo_version sends the new " +
				"password. Because that POST overwrites, create stops when the credential already " +
				"exists and asks for an import instead.",
			"UserCredentialParameter cannot be combined with other Partner Directory entity types " +
				"in a single OData batch (ChangeSet) request; this provider always issues it " +
				"standalone.",
			"Import recovers partner_id, parameter_id, and user, but never the password. The " +
				"first apply after import is an in-place update that sets the configured password.",
		},
		Operations: Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},

	// --- Classic API Management ---
	{
		Key:    "api_management.classic.api_provider",
		Domain: "api_management_classic",
		Name:   "API Provider (classic API Management)",
		Description: "A classic API Management backend/API provider system definition — the " +
			"connection an API Proxy targets.",
		SupportStatus:   StatusPartial,
		ContractSource:  SourceSAPDocumentation,
		SupportReason:   ReasonUnsafeTerraformLifecycle,
		ResourceTypes:   []string{"sapintegrationsuite_api_provider"},
		DataSourceTypes: []string{"sapintegrationsuite_api_provider", "sapintegrationsuite_api_providers"},
		PublicAPI:       true,
		APIProtocol:     "OData V2 (Management.svc)",
		Limitations: []string{
			"Only the \"Internet\" connection type is supported (direct host/port, optionally over " +
				"SSL). SAP documents three further connection types (On Premise via Cloud Connector, " +
				"Open Connectors, Cloud Integration) with distinct field sets this provider could not " +
				"confirm a field-level JSON mapping for from a reachable primary source.",
			"No Update operation: SAP's own official Piper apiProviderUpload tooling documents that " +
				"only Create is supported through this API; every attribute is RequiresReplace.",
			"Eventual consistency: SAP documents up to approximately 20 seconds of caching before a " +
				"just-created or just-deleted provider is reliably visible to GET; Create polls with " +
				"bounded, jittered backoff to reduce (not eliminate) this window.",
		},
		Operations: Operations{Create: true, Read: true, Delete: true, Import: true},
	},
	{
		Key:    "api_management.classic.api_proxy",
		Domain: "api_management_classic",
		Name:   "API Proxy (classic API Management)",
		Description: "A classic API Management API proxy definition: the ZIP-bundled design-time " +
			"content (proxy endpoint, target endpoint, policies, resources) deployed as a callable API.",
		SupportStatus:   StatusUnsupported,
		SupportReason:   ReasonPublicAPIIncomplete,
		DataSourceTypes: []string{},
		PublicAPI:       true,
		APIProtocol:     "REST (Transport.svc) for upload, OData V2 (Management.svc/APIProxies) for read and delete",
		Planned:         true,
		Limitations: []string{
			"An implementation exists in the repository but is not part of the provider yet: the API " +
				"portal answered its import, the request SAP's Client SDK 3.0.6 sends, with 400 " +
				"APIPROXY_ZIP_ERROR (\"Verify the directory structure inside the zip\"), even for a bundle " +
				"the same API portal had exported (tenant test, 2026-09-27). It is added once the request " +
				"documented in the official Transport API specification (APIPortal_Transport_CF) works.",
			"Experimental until the acceptance test (TestAccAPIProxy_sample) passes on a tenant. " +
				"Upload follows SAP's API Management Client SDK 3.0.6 (published 2026-09-24) byte for " +
				"byte: POST /apiportal/api/1.0/Transport.svc/APIProxies with the SDK's own query and " +
				"the raw ZIP as application/octet-stream. The Business Accelerator Hub lists \"API " +
				"Portal - Transport (CF)\" (\"Export and Import API Proxy via zip bundle\") as the " +
				"official API; its specification needs an SAP login.",
			"The APIProxy entity is confirmed by the Management.svc $metadata (key name; " +
				"provider_name, state, status_code, version, isPublished and navigations to endpoints, " +
				"policies, resources and the API provider), and its GET returned 200 on a tenant. The " +
				"read and delete paths Management.svc/APIProxies('<name>') are the ones SAP's " +
				"documentation and worked examples use.",
			"Replace-only: nothing public says whether importing a changed bundle over an existing " +
				"proxy replaces it cleanly, so a different content_hash deletes the proxy and imports " +
				"it again. SAP's documentation says an imported proxy is deployed by default, so there " +
				"is no separate deployment step (see api_management.classic.api_proxy_deployment).",
			"API providers a bundle's target endpoint references must already exist on the tenant: " +
				"SAP's sample repository documents that the import fails otherwise. Bundles with a " +
				"target URL (provider_id NONE) have no such dependency.",
			"The SDK's JSON create path (/api/1.0/apis/ with isFromCli) is an internal endpoint and " +
				"not used.",
		},
		Operations: Operations{Create: true, Read: true, Delete: true, Import: true},
	},
	{
		Key:    "api_management.classic.api_proxy_deployment",
		Domain: "api_management_classic",
		Name:   "API Proxy Deployment (classic API Management)",
		Description: "The runtime deployment state of a classic API Proxy, potentially independent " +
			"of its design-time content.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonPublicAPIIncomplete,
		PublicAPI:     true,
		Limitations: []string{
			"SAP's own documentation states that a proxy transported or exported, individually or " +
				"as part of a product, \"by default gets imported to the target in the deployed " +
				"state\", so an import deploys the proxy, and " +
				"exposes the resulting state. No documented API deploys or undeploys an existing proxy " +
				"on its own, so there is no separate deployment resource.",
		},
	},
	{
		Key:    "api_management.classic.policy",
		Domain: "api_management_classic",
		Name:   "Policy (classic API Management)",
		Description: "An individual mediation policy (for example VerifyAPIKey, Quota, " +
			"AssignMessage) attached to a classic API Proxy's proxy or target endpoint flow.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonPublicAPIIncomplete,
		PublicAPI:     true,
		Limitations: []string{
			"Confirmed to be XML content embedded inside the API Proxy ZIP bundle (a <policies> " +
				"element in the proxy's root XML, referencing named files under a Policy/ folder), not " +
				"an independently addressable OData entity with its own Create/Read/Update/Delete — so " +
				"individual policies are not a separate resource candidate; they are managed as part " +
				"of a proxy's bundle content.",
		},
	},
	{
		Key:    "api_management.classic.virtual_host",
		Domain: "api_management_classic",
		Name:   "API Management Virtual Host (Classic)",
		Description: "A virtual host of the Classic API Portal: the default-domain alias or custom " +
			"domain (with one-way or mutual TLS) under which API proxies are exposed.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonPublicAPIIncomplete,
		PublicAPI:     true,
		APIProtocol:   "OData V2 (Configuration.svc/VirtualHostRequests, Management.svc/VirtualHosts)",
		Planned:       true,
		Limitations: []string{
			"Create, update and delete are documented in SAP Help (Configuring a Default Domain / " +
				"Custom Domain / Mutual TLS for a Virtual Host): POST " +
				"/apiportal/operations/1.0/Configuration.svc/VirtualHostRequests with operation " +
				"CREATE, UPDATE or DELETE and the fields accountId, virtualHostUrl (max 63 characters " +
				"for an alias), isDefaultVirtualHostRequest, isForCustomDomain, keyStoreName, " +
				"keyStoreAlias, trustStore, isClientAuthEnabled and virtualHostId. The response carries " +
				"virtualHostId and allocationStatus.",
			"Reading is confirmed by an API Portal tenant's Management.svc $metadata (September " +
				"2026): VirtualHosts has the key id and the properties name, virtual_host, " +
				"virtual_port, isDefault, isSSL, isForCustomDomain, isClientAuthEnabled, keyStoreName, " +
				"keyStoreAlias, trustStore and projectPath, which covers every field the write request " +
				"sets. A GET with an APIPortal.Administrator key returned 200. Whether " +
				"allocationStatus can be anything other than COMPLETE is still not documented, and the " +
				"write path has not been exercised.",
			"Needs a service key with the APIManagement.SelfService.Administrator role, separate from " +
				"APIPortal.Administrator. Deletion is refused while proxies (deployed, draft or in a " +
				"revision) reference the host or while it is the default.",
		},
	},
	{
		Key:    "api_management.classic.api_product",
		Domain: "api_management_classic",
		Name:   "API Product (classic API Management)",
		Description: "A classic API Management API product bundling one or more API proxies for " +
			"subscription, with optional custom attributes and request quotas. Replaced, never " +
			"updated in place.",
		SupportStatus:   StatusSupported,
		ContractSource:  SourceSAPDocumentation,
		ResourceTypes:   []string{"sapintegrationsuite_api_product"},
		DataSourceTypes: []string{"sapintegrationsuite_api_product"},
		PublicAPI:       true,
		APIProtocol:     "OData V2 (Management.svc)",
		Limitations: []string{
			"No update: a tenant test in September 2026 answered PUT, PATCH and MERGE on an " +
				"existing product with 405 \"UPDATE operation not supported on APIProduct entity\". " +
				"Every attribute forces a new product, which drops the subscriptions of the old one.",
			"At least one existing API proxy is required (SAP: \"At least one API Proxy should be " +
				"linked to an API Product\"). Proxies come from the UI or a transport; see " +
				"api_management.classic.api_proxy.",
			"status_code is required by SAP on create and defaults to PUBLISHED; DRAFT creates an " +
				"unpublished product. Both were confirmed on a tenant.",
			"Additional properties can only be sent inside the create request, each with the " +
				"product's name as entityId; a separate POST APIProductAdditionalProperties answers 405.",
			"Linked proxies and additional properties are read through the apiProxies and " +
				"additionalProperties navigation properties, since the product itself only returns " +
				"__deferred links.",
		},
		Operations: Operations{Create: true, Read: true, Update: false, Delete: true, Import: true},
	},
	{
		Key:    "api_management.classic.certificate_store_reference",
		Domain: "api_management_classic",
		Name:   "Certificate Store Reference (classic API Management)",
		Description: "A named pointer to an already-existing API Management keystore or " +
			"truststore, used so a virtual host's TLS configuration can be repointed at a new store " +
			"(for certificate rotation) without editing the virtual host itself.",
		SupportStatus:  StatusSupported,
		ContractSource: SourceSAPDocumentation,
		ResourceTypes:  []string{"sapintegrationsuite_api_management_certificate_store_reference"},
		DataSourceTypes: []string{
			"sapintegrationsuite_api_management_certificate_store_reference",
		},
		PublicAPI:   true,
		APIProtocol: "OData V2 (Management.svc)",
		Limitations: []string{
			"Manages only the reference (a name pointing at a store name); the keystore/truststore " +
				"and its certificate content are confirmed to be a UI-only upload with no accompanying " +
				"REST API, so this provider requires the referenced store to already exist.",
			"This is the best-confirmed Classic API Management resource in this provider: full " +
				"Create/Read/Update/Delete, request and response bodies, and error codes are all " +
				"confirmed verbatim from SAP's own official documentation.",
		},
		Operations: Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},
	{
		Key:    "api_management.classic.key_value_map",
		Domain: "api_management_classic",
		Name:   "Key Value Map (classic API Management)",
		Description: "A classic API Management key-value map used for runtime configuration " +
			"lookups, readable through the Key Value Map Operations policy.",
		SupportStatus:   StatusPartial,
		ContractSource:  SourceSAPDocumentation,
		SupportReason:   ReasonUnsafeTerraformLifecycle,
		ResourceTypes:   []string{"sapintegrationsuite_api_key_value_map"},
		DataSourceTypes: []string{"sapintegrationsuite_api_key_value_map"},
		PublicAPI:       true,
		APIProtocol:     "OData V2 (Management.svc/GenericKeyMapEntries)",
		Limitations: []string{
			"No Update: SAP's documentation confirms a full Create/Update-entries/Delete UI " +
				"lifecycle exists, but only Create's REST payload is shown verbatim anywhere " +
				"reachable; every attribute, including entries, is RequiresReplace.",
			"Encrypted maps are not supported: SAP's isEncrypted field is real and documented, but " +
				"this provider could not confirm whether GET returns an encrypted entry's plaintext " +
				"value back, a masked placeholder, or nothing at all. This resource always sends " +
				"isEncrypted = false and rejects a configuration that sets encrypted = true.",
		},
		Operations: Operations{Create: true, Read: true, Delete: true, Import: true},
	},
	{
		Key:    "api_management.classic.certificate_store",
		Domain: "api_management_classic",
		Name:   "API Management Certificate Store and Certificate (Classic)",
		Description: "Key stores and trust stores of the API Portal and the certificates in them, " +
			"used for TLS towards backends and for virtual hosts.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonPublicAPIIncomplete,
		PublicAPI:     true,
		APIProtocol:   "OData V2 (Management.svc/CertificateStores, Certificates)",
		Limitations: []string{
			"Management.svc $metadata: CertificateStores (key name; storeType) and Certificates " +
				"(key name and storeName; content as Edm.Binary, format, password, validity and " +
				"issuer fields). The Business Accelerator Hub describes the KeyStore and TrustStore " +
				"APIs as \"create and view\"; update and delete are not described, so a Terraform " +
				"lifecycle cannot be confirmed yet. certificate_store_reference covers pointing at an " +
				"existing store.",
		},
	},
	{
		Key:    "api_management.classic.application",
		Domain: "api_management_classic",
		Name:   "API Management Application and Developer (Classic)",
		Description: "Consumer applications subscribed to API products, with their generated " +
			"application key and secret, and the developers who own them.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonPublicAPIIncomplete,
		PublicAPI:     true,
		APIProtocol:   "OData V2 (Management.svc/Applications, Developers)",
		Limitations: []string{
			"Management.svc $metadata: Applications (key id; app_key, app_secret, callbackurl, " +
				"status_code, validity, subscribedRatePlan, navigation to apiProducts and developer) " +
				"and Developers (key id; emailId, firstName, lastName, country). The Hub describes the " +
				"CF Applications API as \"view all available applications\"; creating applications " +
				"through the API is described only for the Developer API. The generated app_secret " +
				"would have to be kept out of state or treated as sensitive.",
		},
	},
	{
		Key:    "api_management.classic.environment_key_value_map",
		Domain: "api_management_classic",
		Name:   "API Management Key Value Map across API Proxies (Classic)",
		Description: "Key value maps shared across API proxies (KeyMapEntries), as opposed to the " +
			"generic, scoped key value maps sapintegrationsuite_api_key_value_map manages.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonPublicAPIIncomplete,
		PublicAPI:     true,
		APIProtocol:   "OData V2 (Management.svc/KeyMapEntries, KeyMapEntryValues)",
		Limitations: []string{
			"Management.svc $metadata: KeyMapEntries (key name; encrypted, scope) with " +
				"KeyMapEntryValues (key map_name and name; value). The Hub lists \"Key Value Maps " +
				"(CF)\" (\"create key value pairs across the API proxies\") next to the generic key " +
				"value maps this provider implements. How the two relate, and which one SAP " +
				"recommends, is not documented.",
		},
	},
	{
		Key:           "api_management.classic.cache_resource",
		Domain:        "api_management_classic",
		Name:          "API Management Cache Resource (Classic)",
		Description:   "A named cache used by response cache and lookup cache policies.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonPublicAPIIncomplete,
		PublicAPI:     true,
		APIProtocol:   "OData V2 (Management.svc/CacheResources)",
		Limitations: []string{
			"Management.svc $metadata: CacheResources (key name; sizes, compression, overflow and " +
				"expiry settings). The Hub documents create, view, update and delete only for the Neo " +
				"version of this API; no Cloud Foundry version is listed.",
		},
	},
	{
		Key:           "api_management.classic.rate_plan",
		Domain:        "api_management_classic",
		Name:          "API Management Rate Plan (Classic)",
		Description:   "Monetization rate plans attached to API products.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonPublicAPIIncomplete,
		PublicAPI:     true,
		APIProtocol:   "OData V2 (Management.svc/RatePlans)",
		Limitations: []string{
			"Management.svc $metadata: RatePlans (key id; rate, currency, frequency, type, " +
				"validity, isActive, isPublished). The Hub lists only billing and metering APIs for " +
				"monetization, no rate plan API.",
		},
	},
	{
		Key:           "api_management.classic.policy_template",
		Domain:        "api_management_classic",
		Name:          "API Management Policy Template (Classic)",
		Description:   "Reusable policy templates that can be applied to API proxies.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonPublicAPIIncomplete,
		PublicAPI:     true,
		APIProtocol:   "OData V2 (Management.svc/PolicyTemplateContainers)",
		Limitations: []string{
			"Management.svc $metadata: PolicyTemplateContainers (key name; proxy and target " +
				"endpoint XML, navigations to policies and file resources). The token scopes include " +
				"import, export and apply for policy templates, but the Hub lists no policy template " +
				"API and SAP Help describes only the UI.",
		},
	},
	{
		Key:           "api_management.classic.product_access_control",
		Domain:        "api_management_classic",
		Name:          "API Management Product Access Control (Classic)",
		Description:   "Rules that grant user groups access to API products in the Developer Hub.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonPublicAPIIncomplete,
		PublicAPI:     true,
		APIProtocol:   "OData V2 (Management.svc/ACLProductLinkages)",
		Limitations: []string{
			"Management.svc $metadata: ACLProductLinkages (key ruleId; entityId, entityType, " +
				"permissionSet, operation, isPublished). The Hub describes the Access Control Service " +
				"(CF) as \"view and create rules\"; update and delete are not described.",
		},
	},

	// --- Current API Management (API Artifacts / MCP Servers / Integration Cell) — distinct
	// from Classic API Management (API Providers/Proxies/Products). Keys keep the "api_gateway"
	// prefix from earlier research passes (never renamed once shipped); names and descriptions
	// use SAP's current terminology. Re-audited September 2026: see
	// docs/guides/current-api-management.md and docs/sap-api-references.md.
	{
		Key:    "api_gateway.api_artifact",
		Domain: "api_gateway",
		Name:   "API Artifact — Current API Management",
		Description: "A design-time API (REST, SOAP or OData) in SAP's API-centric integration " +
			"model: endpoints, policies and security, created inside an integration package under " +
			"Design > Integrations and APIs and deployed to Integration Cell or Edge Integration Cell.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Planned:       true,
		Limitations: []string{
			"SAP has not published an API for API artifacts. The September 2026 re-audit checked " +
				"the Integration Content API's resource table (Help version of 2026-07-10; no API " +
				"artifact resource), both SAP API Documentation index pages (only the " +
				"CloudIntegrationAPI and Classic APIMgmt packages), every current creation, " +
				"versioning, copy, deletion and deployment page (UI procedures only), SAP's " +
				"API Management Client SDK 3.0.6 (Classic API Portal endpoints only) and SAP's own " +
				"2026 CI/CD tooling (no API artifact automation).",
			"The 2026 features around API artifacts, such as API-centric integration, simplified " +
				"creation from a URL or specification, OpenAPI specifications drafted by SAP's assistant and " +
				"product subscriptions, are all UI features.",
			"Re-checked 2026-09-26 (Hub package APIMgmt of 2026-09-24, SAP Help mirror of " +
				"2026-09-18, Client SDK 3.0.6 of 2026-09-24): still no API. API artifacts and MCP " +
				"servers are integration package content (resourceType API in a package export) and " +
				"move between tenants with the package: export, POST IntegrationPackages with " +
				"PackageContent (optionally ?Overwrite=true), SAP Cloud Transport Management or CTS+. " +
				"That whole-package import is the only public path and is opaque to individual API " +
				"artifacts, so it is not modelled as an API artifact resource.",
			"A downloaded API artifact (2026-09-27) is a bundle of type RESTAPI for the Integration " +
				"Cell runtime profile. It holds an OpenAPI 3.0 definition and an integration flow " +
				"that SAP generates from it. The format is SAP-internal, so building such bundles is " +
				"not a way around the missing API.",
			"The documented Integration Content API does not see them: for a package holding an " +
				"integration flow, an API artifact and an MCP server, " +
				"IntegrationPackages('<id>')/IntegrationDesigntimeArtifacts listed only the integration " +
				"flow (2026-09-29).",
		},
	},
	{
		Key:    "api_gateway.api_artifact_deployment",
		Domain: "api_gateway",
		Name:   "API Artifact Deployment — Integration Cell",
		Description: "The runtime deployment of an API Artifact or MCP Server on Integration Cell " +
			"or Edge Integration Cell, including the virtual host chosen at deployment time.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Planned:       true,
		Limitations: []string{
			"SAP documents deployment only as a UI action; no deploy, undeploy or status API " +
				"exists for API artifacts or MCP servers.",
			"Semantics to preserve if an API appears: the runtime profile is fixed once the " +
				"artifact exists (except for choosing the target Edge Integration Cell), and the " +
				"deployment-time virtual host may differ from the design-time one, with the deployed " +
				"endpoint URL following the deployment-time choice. Terraform would need to keep the " +
				"two virtual hosts as separate attributes.",
		},
	},
	{
		Key:    "api_gateway.api_policy",
		Domain: "api_gateway",
		Name:   "API Artifact Policy",
		Description: "A policy or mediation step (authentication, quota, rate limiting, " +
			"transformation, external callout and so on) inside an API Artifact.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Planned:       true,
		Limitations: []string{
			"Policies are edited inside the API artifact's policy editor and have no separate " +
				"lifecycle in SAP Help. Without an API for the artifact itself there is nothing to " +
				"show whether policies would be nested content or addressable entities; a separate " +
				"resource would only make sense for the latter.",
		},
	},
	{
		Key:    "api_gateway.reusable_api_artifact",
		Domain: "api_gateway",
		Name:   "Reusable API Artifact",
		Description: "An internal-only API Artifact with no external endpoint, invoked by other " +
			"API Artifacts through the API Direct adapter to share logic such as authentication " +
			"or transformation.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"A variant of the API artifact (unique base path, not reachable over HTTP, cannot call " +
				"another reusable API), created through the same UI. It shares the API artifact's " +
				"blocker. If an API appears, the intended model is a type discriminator on the API " +
				"artifact resource rather than a second resource.",
		},
	},
	{
		Key:    "api_gateway.mcp_server",
		Domain: "api_gateway",
		Name:   "MCP Server",
		Description: "A Model Context Protocol server artifact that exposes APIs as tools for AI " +
			"agents, created from an API artifact, an HTTP endpoint with an OpenAPI specification, " +
			"an RFC-enabled backend, a remote MCP server or a Classic API proxy, and deployed to " +
			"Integration Cell.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"New in 2026 (MCP Gateway, July 2026; remote MCP servers, September 2026). Creation, " +
				"tool selection, authentication and deployment are documented only as UI procedures, " +
				"and no MCP server resource appears in any public API or in SAP's Client SDK.",
			"Like API artifacts, MCP servers are integration package content: a downloaded MCP " +
				"server (2026-09-27) is a bundle of type MCPSERVER for the Integration Cell runtime " +
				"profile that requires the bundle of its source API artifact. It can only be moved " +
				"with the whole package.",
			"Publishing an MCP server as a Developer Hub product belongs to the separate Developer " +
				"Hub provider, not to this one.",
		},
	},
	{
		Key:    "api_gateway.runtime_profile",
		Domain: "api_gateway",
		Name:   "Runtime Profile",
		Description: "The target platform (Cloud Integration, Integration Cell, Edge Integration " +
			"Cell, SAP Process Orchestration) an API Artifact or integration flow is designed and " +
			"deployed for.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"Runtime profiles are enabled and disabled under Settings > Integrations only. Even " +
				"with an API, the list would be a weak data source: small, platform-defined and better " +
				"served by documentation.",
			"SAP's Runtime Profiles reference page still (September 2026) lists no Integration " +
				"Cell row, although Integration Cell is offered as a runtime profile when creating API " +
				"artifacts and MCP servers.",
		},
	},

	// --- Integration Cell / Edge Integration Cell (content/config, once active) ---
	{
		Key:           "integration_cell.runtime",
		Domain:        "integration_cell",
		Name:          "Integration Cell Runtime",
		Description:   "Runtime status and configuration of an already-activated Integration Cell, distinct from activating the capability itself.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"Split by concern, none has a public API: activation is a Settings > Runtime UI " +
				"step; runtime discovery and status are shown only in Monitor > Integrations and " +
				"APIs with the Integration Cell runtime selected; configuration such as trace log " +
				"level (new for Integration Cell APIs in August 2026) is UI-only; and the only " +
				"deployment targeting is the runtime profile chosen in the UI.",
		},
	},
	{
		Key:    "integration_cell.virtual_host",
		Domain: "integration_cell",
		Name:   "Integration Cell Virtual Host",
		Description: "A host name under the Integration Cell default domain (or a custom domain) " +
			"through which API Artifacts and MCP Servers are exposed.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"Virtual hosts are added, edited and deleted in Monitor > Integrations and APIs > " +
				"Virtual Host (PI_Administrator); SAP documents no API for any of these steps.",
			"Rules a future resource would have to respect: at most 11 virtual hosts per tenant, " +
				"the name Default is reserved, the host alias is at most 22 characters, and deletion " +
				"is blocked for the default virtual host and for hosts used by deployed APIs. APIs that " +
				"referenced a deleted host fall back to the default one. The default host would have " +
				"to be read-only in Terraform.",
		},
	},
	{
		Key:           "edge_integration_cell.registration",
		Domain:        "edge_integration_cell",
		Name:          "Edge Integration Cell Registration",
		Description:   "SAP-side registration and runtime association of an Edge Integration Cell, never the customer-managed Kubernetes workloads themselves.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"Reconfirmed: an Edge Node is added and removed exclusively through the Edge Lifecycle " +
				"Management (ELM) UI, and onboarding is completed by running the standalone Edge " +
				"Lifecycle Management Bridge executable against the target Kubernetes cluster's " +
				"kubeconfig — a local CLI/Kubernetes handshake, not an HTTP API this provider could call. " +
				"No public registration, runtime-association, or deregistration API was found.",
		},
	},
	{
		Key:    "edge_integration_cell.local_api",
		Domain: "edge_integration_cell",
		Name:   "Edge Integration Cell Local API Access",
		Description: "SAP's public /local/api/v1 and /local/api/eic/v1 REST endpoints, reachable directly " +
			"against a running Edge Integration Cell node without going through the cloud UI: Message " +
			"Processing Logs, Message Stores/JMS, DataStores/Variables (OData V2), and the Operations " +
			"Cockpit's Component/Job/RuntimeParameter resources (OData V4).",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonOutOfScope,
		PublicAPI:     true,
		APIProtocol:   "OData V2 (Message Processing Logs, Message Stores) / OData V4 (Operations Cockpit)",
		Limitations: []string{
			"Confirmed reachable and documented (SAP Business Accelerator Hub package " +
				"sap-int-eic-eic-operations, created 2026-04-30, with the APIs Jobs, Components, " +
				"Partner Directory, Message Stores and Message Processing Logs), authenticated with the same certificate or " +
				"clientId/clientsecret mechanisms as the rest of this provider, and CSRF-token gated for " +
				"modifying calls — but every entity behind it is monitoring/operational runtime data " +
				"(message processing records, message store/JMS contents, data store/variable values), " +
				"the same category this provider already excludes for Cloud Integration's own " +
				"MessageProcessingLogs/DataStores/Variables. See edge_integration_cell.runtime for the " +
				"separate Operations Cockpit control-plane objects reachable through this same /local " +
				"prefix.",
			"The package's Partner Directory API is configuration, not monitoring, but it is the same " +
				"Partner Directory the provider reaches from the cloud through /location/<runtime " +
				"location id>/api/v1 (see edge_integration_cell.deployment_target), so a second, " +
				"network-local path is not needed.",
		},
	},
	{
		Key:    "edge_integration_cell.runtime",
		Domain: "edge_integration_cell",
		Name:   "Edge Integration Cell Runtime Operations",
		Description: "The Operations Cockpit API's Component, Pod, RuntimeParameter, and Job/JobSchedule " +
			"entities: per-component status, pod resource limits/replica counts, log levels, and " +
			"scheduled-job configuration for an Edge Integration Cell's own SAP-operator-managed pods.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonOutOfScope,
		PublicAPI:     true,
		APIProtocol:   "OData V4",
		Limitations: []string{
			"Confirmed reachable at /local/api/eic/v1 (also documented on SAP Business Accelerator Hub " +
				"as the Edge Integration Cell package's OData V4 API), and RuntimeParameter is explicitly " +
				"described as changeable (\"Save to run the changes in the back end\") — a genuinely " +
				"writable-looking configuration object, not read-only monitoring data.",
			"Every RuntimeParameter documented (LOG_LEVEL, MIN_REPLICAS, MAX_REPLICAS, CPU_LIMIT, " +
				"MEMORY_LIMIT, EPHEMERAL_STORAGE_LIMIT) configures Kubernetes-level pod resource requests, " +
				"replica counts, or log verbosity for SAP-operator-managed components — the same category " +
				"of concern docs/provider-scope.md already excludes as Kubernetes/Helm infrastructure, " +
				"just fronted by an EIC-specific API instead of the raw Kubernetes API.",
			"The API's own documentation states the Component resource lets a caller \"restart " +
				"components\" — an imperative runtime operation this provider's standing policy refuses " +
				"to model as a Terraform resource (see the retry/restart/cancel/purge rule this provider " +
				"already applies elsewhere).",
			"The exact entity keys and PATCH/POST payload shapes for RuntimeParameter and JobSchedule are " +
				"not confirmed from a reachable primary source: the package's $metadata/EDMX and worked " +
				"examples live behind SAP Business Accelerator Hub's authenticated catalog pages, which " +
				"redirect unauthenticated requests to a login page, the same access limitation this " +
				"project has documented repeatedly for other api.sap.com packages.",
		},
	},
	{
		Key:    "edge_integration_cell.deployment_target",
		Domain: "edge_integration_cell",
		Name:   "Edge Integration Cell Runtime Targeting",
		Description: "Addressing an Edge Integration Cell instead of the cloud runtime: deploying " +
			"content to it and managing its security material, selected with runtime_location_id.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonPublicAPIIncomplete,
		PublicAPI:     true,
		APIProtocol:   "OData V2",
		Limitations: []string{
			"Not supported yet. SAP Help's Integration Content, Security Content, Partner Directory, " +
				"message processing log, message store and number range pages (mirror of 2026-09-18) " +
				"document https://<host>/location/<runtime location id>/api/v1/<path> for calling the " +
				"same APIs against an Edge Integration Cell, once for all operations and without " +
				"per-operation examples, and SAP's own CI/CD tooling still called the path unpublished " +
				"in May 2026. It has not been verified against a tenant with an Edge Integration Cell.",
			"The deployment, credential, certificate, key pair and Partner Directory resources and " +
				"their data sources carry an optional runtime_location_id that sends requests to that " +
				"path, and import IDs accept a location:<id>/ prefix. " +
				"TestAccEdgeIntegrationCell_securityAndPartnerDirectory (gate " +
				"SAP_INTEGRATION_SUITE_ACC_EDGE_INTEGRATION_CELL) checks both; until it passes on a " +
				"tenant, leave runtime_location_id unset.",
		},
	},
	{
		Key:    "edge_integration_cell.access_policy_replication",
		Domain: "edge_integration_cell",
		Name:   "Edge Integration Cell Access Policy Replication",
		Description: "The runtimes (Cloud Integration runtime, Integration Cell, specific Edge " +
			"Integration Cells) an Access Policy is replicated to, and the replication state of each.",
		SupportStatus:   StatusUnofficial,
		ContractSource:  SourceMetadataOnly,
		SupportReason:   ReasonPublicAPIIncomplete,
		DataSourceTypes: []string{"sapintegrationsuite_access_policy_runtime_assignments"},
		PublicAPI:       true,
		APIProtocol:     "OData V2",
		Limitations: []string{
			"Readable, not writable: the tenant $metadata defines AccessPolicyRuntimeAssignments " +
				"(Id, RuntimeLocationId, TransferStatus, TransferErrors, StatusUpdatedAt) as a " +
				"navigation property of AccessPolicies, which the data source reads. Whether " +
				"assignments can be created or deleted through the API is not documented, and " +
				"$metadata carries no creatable/updatable flags, so choosing runtimes stays a UI step.",
			"transfer_status is passed through as SAP returns it: the UI shows Fail, Success and " +
				"Pending, but the API values are plain strings with no documented enumeration.",
		},
		Operations: Operations{Read: true},
	},

	// --- Capability activation/provisioning (subscription-level toggles) ---
	{
		Key:           "capabilities.cloud_integration",
		Domain:        "capability_provisioning",
		Name:          "Cloud Integration Capability Activation",
		Description:   "Activating the Cloud Integration capability itself within an Integration Suite tenant.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"Activated as part of the Integration Suite subscription/onboarding wizard; stays a manual, one-time bootstrap step.",
		},
	},
	{
		Key:           "capabilities.api_management",
		Domain:        "capability_provisioning",
		Name:          "API Management Capability Activation",
		Description:   "Activating the classic API Management capability itself within an Integration Suite tenant.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"Documented as a UI step (Integration Suite → Manage Capabilities → activate API Management); no public API found.",
		},
	},
	{
		Key:           "capabilities.api_gateway",
		Domain:        "capability_provisioning",
		Name:          "Current API Management Capability Activation",
		Description:   "Activating SAP's current, API-centric API Management capability (API Artifacts, Integration Cell) itself within an Integration Suite tenant.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
	},
	{
		Key:           "capabilities.integration_cell",
		Domain:        "capability_provisioning",
		Name:          "Integration Cell Capability Activation",
		Description:   "Activating the Integration Cell capability itself within an Integration Suite tenant.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"SAP's own documentation describes activation as choosing Activate in Integration Suite → Settings → Runtime; no public API was found.",
		},
	},
	{
		Key:           "capabilities.edge_integration_cell",
		Domain:        "capability_provisioning",
		Name:          "Edge Integration Cell Capability Activation",
		Description:   "Activating the Edge Integration Cell capability itself within an Integration Suite tenant.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
	},

	// --- API Composition ---
	{
		Key:    "api_composition.business_data_graph",
		Domain: "api_composition",
		Name:   "API Composition Business Data Graph",
		Description: "A business data graph combines the business systems of a landscape (S/4HANA, " +
			"SAP Sales Cloud, custom OData services and others) into one connected API. Managed " +
			"through API Composition's Configuration API.",
		SupportStatus:          StatusExperimental,
		ContractSource:         SourceSAPDocumentation,
		UndocumentedOperations: []string{"delete (DELETE on the graph)", "update body (PATCH with the writable properties)"},
		SupportReason:          ReasonPublicAPIIncomplete,
		ResourceTypes:          []string{"sapintegrationsuite_business_data_graph"},
		DataSourceTypes:        []string{"sapintegrationsuite_business_data_graph"},
		PublicAPI:              true,
		APIProtocol:            "OData V4 (Configuration API)",
		Limitations: []string{
			"Needs its own credentials in provider.api_composition: a service key of an API " +
				"Composition service instance with plan \"configuration\". SAP does not document that " +
				"service key field by field, so the four values are entered as they are.",
			"SAP documents the Create body, GET and PATCH on GraphConfiguration/{id}, and the status " +
				"model. It gives no PATCH body and no delete request. The provider sends the writable " +
				"properties as the PATCH body and DELETE to the graph's URL. Not yet verified against " +
				"a live system. Both need enable_unofficial in addition to enable_experimental, so " +
				"without it a graph can be created and read, but not changed in place or destroyed.",
			"SAP processes graphs asynchronously. Create and Update wait until the status leaves " +
				"PROCESSING (20 minutes by default, configurable with timeouts). A graph that ends in " +
				"FAILED is kept in state and marked tainted.",
			"Extensions cannot be managed through the Configuration API, according to SAP; " +
				"extensions is read-only and left alone on update.",
			"Cue-scoped key mappings and the OData containment setting are described by SAP without " +
				"a property name and cannot be set.",
			"SAP does not describe a logMessages entry, so each is exposed as the JSON text SAP " +
				"returned.",
		},
		Operations: Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
	},
	{
		Key:    "odata_provisioning",
		Domain: "other_capability",
		Name:   "OData Provisioning",
		Description: "A capability that exposes SAP Business Suite backend OData services " +
			"(SAP Gateway back-end-enablement) through SAP Integration Suite, without requiring an " +
			"on-premise SAP Gateway hub.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"Re-audit September 2026: registering OData services, adding destinations, switching a " +
				"service on or off, error tolerance for multi-origin composition, and metadata " +
				"validation and cache settings are documented only in the UI (Configure > OData " +
				"Services). The complete Business Accelerator Hub package list has no OData " +
				"Provisioning package.",
			"ODPAPIAccess, earlier read as a sign of a management API, grants access to the service " +
				"document of registered services; APIFullAccess grants runtime access and ODPManage " +
				"the UI. The service key (Serverless Runtime, plan odpruntime) is for calling the " +
				"registered services, not for configuring them.",
			"Registered services would suit Terraform (named configuration with destinations and " +
				"settings) if SAP publishes a management API.",
		},
	},
	{
		Key:           "developer_hub",
		Domain:        "other_capability",
		Name:          "Developer Hub",
		Description:   "SAP's API/Event/MCP Server catalog, publication, and subscription capability for Integration Suite, reachable through its own /api/1.0 REST API and its own devportal-apiaccess OAuth credentials, separate from every Cloud Integration and current API Management endpoint this provider otherwise talks to.",
		SupportStatus: StatusSeparateProvider,
		SupportReason: ReasonOutOfScope,
		PublicAPI:     true,
		Limitations: []string{
			"Developer Hub has its own API boundary, its own OAuth client credentials, and a consumer/catalog object lifecycle (Products, Applications, Subscriptions) distinct in shape from this provider's Integration Suite content and capability model.",
			"Planned as a separate, independently versioned Terraform provider (working name Prideth/terraform-provider-sap-developer-hub) rather than a domain inside this one, so its release cadence and credential surface never entangle with this provider's.",
			"This entry intentionally represents the whole Developer Hub capability as a single scope statement; individual Developer Hub objects (Product, Application, Subscription, and so on) are not separately cataloged here.",
		},
	},
	{
		Key:    "integration_advisor.design_time_content",
		Domain: "other_capability",
		Name:   "Integration Advisor Design-Time Content",
		Description: "Message Implementation Guidelines (MIGs), Mapping Guidelines (MAGs, " +
			"standard/overlay/XSLT), custom Type Systems, Codelists, Shared Code, and Global " +
			"Parameters — SAP's collaborative B2B interface-content-design objects.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"No API-access documentation page (the kind that, when present, confirmed a real public " +
				"API for Classic API Management and Integration Assessment) exists anywhere in " +
				"Integration Advisor's roughly eighty-five-page documentation tree. The one OAuth-" +
				"credential page found in this area (\"Creating OAuth Client Credentials for Cloud " +
				"Foundry Environment\") turned out, on full reading, to describe authenticating " +
				"against the destination Cloud Integration tenant for artifact injection (Process " +
				"Integration Runtime service, plan api), not a credential for Integration Advisor's " +
				"own design-time content management.",
			"Re-audit September 2026: all 80 pages of the current documentation checked again, and " +
				"the full Business Accelerator Hub package list (1971 packages) contains only " +
				"integration content for Integration Advisor (EDI Integration Templates), no API.",
		},
	},
	{
		Key:    "integration_advisor.runtime_artifact_injection",
		Domain: "other_capability",
		Name:   "Integration Advisor Runtime Artifact Injection",
		Description: "Injecting generated runtime artifacts (from a Mapping Guideline) directly " +
			"into an integration flow's resources on a target Cloud Integration tenant.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonOutOfScope,
		PublicAPI:     false,
		Limitations: []string{
			"Confirmed as a UI wizard (Mapping Guideline > Inject > SAP Cloud Integration Flow " +
				"Resources > choose tenant/package/integration flow > Inject), not a documented REST " +
				"call, and — independent of that — an imperative one-shot action rather than " +
				"desired-state configuration this provider's plan/apply model could represent even if " +
				"an API for it were confirmed.",
		},
	},
	{
		Key:    "trading_partner_management.company_profile",
		Domain: "other_capability",
		Name:   "Trading Partner Management Company Profile",
		Description: "The tenant's own company profile and its subsidiaries, the initiator side " +
			"of every trading partner agreement.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"Reconfirmed: no \"accessing APIs programmatically\", API reference, or service-instance/" +
				"service-key page exists anywhere in SAP's entire Trading Partner Management " +
				"documentation tree (roughly 90 pages checked) — the same kind of page that, when " +
				"present, confirmed a real public API for Classic API Management and Integration " +
				"Assessment. Content is downloadable as JSON through a UI Download button only " +
				"(company.json), never through a documented REST endpoint.",
			"Re-audit September 2026: all 93 pages checked again. The Business Accelerator Hub's " +
				"Cloud Integration package has a \"B2B Scenarios\" OData API, but the tenant $metadata " +
				"shows what it covers: BusinessDocuments, interchanges, payloads, events and the " +
				"reprocessing functions singleInterchangeProcess/massInterchangeProcess, all B2B " +
				"monitoring data. Profiles, agreement templates and agreements still have no API. " +
				"The only TPM configuration call SAP documents is B2B archiving activation, see " +
				"cloud_integration.archiving.",
		},
	},
	{
		Key:    "trading_partner_management.partner_profile",
		Domain: "other_capability",
		Name:   "Trading Partner Management Partner Profile",
		Description: "Trading partner profiles and communication partner profiles: the " +
			"counterparty side of a trading partner agreement.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"Same evidence as trading_partner_management.company_profile: UI-only (Design > B2B " +
				"Scenarios), downloadable as JSON, no documented REST/OData endpoint found.",
		},
	},
	{
		Key:           "trading_partner_management.agreement_template",
		Domain:        "other_capability",
		Name:          "Trading Partner Management Agreement Template",
		Description:   "A reusable template defining the shape of trading partner agreements created from it.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
	},
	{
		Key:    "trading_partner_management.agreement",
		Domain: "other_capability",
		Name:   "Trading Partner Management Agreement",
		Description: "A trading partner agreement: business transaction activities, identifiers, " +
			"and integration/message flow configuration between the tenant's company profile and a " +
			"trading partner.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
	},
	{
		Key:    "trading_partner_management.partner_directory_generation",
		Domain: "other_capability",
		Name:   "Trading Partner Management Partner Directory Generation",
		Description: "The generated Partner Directory entries (String/Binary Parameters, prefixed " +
			"\"SAP_TPM\") an agreement's activation pushes into Partner Directory for runtime use by " +
			"generic integration flows.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonOutOfScope,
		PublicAPI:     false,
		Limitations: []string{
			"Confirmed, verbatim: \"When a trading partner agreement gets activated, the complete " +
				"agreement information gets pushed into the partner directory.\" This is a UI-triggered " +
				"(Activate action), one-way generation side effect, not an independent create/update " +
				"operation this provider's already-implemented Partner Directory resources " +
				"(sapintegrationsuite_partner_string_parameter and siblings) could safely manage or " +
				"even represent — out of scope for the same reason this provider does not model other " +
				"imperative generation/replication actions as resources, regardless of whether a " +
				"public API for the trigger itself is ever confirmed.",
		},
	},
	{
		Key:    "event_mesh",
		Domain: "other_capability",
		Name:   "Event Mesh",
		Description: "SAP's Solace PubSub+-based event broker service for asynchronous, " +
			"event-driven integration: queues, topic subscriptions, and webhook subscriptions.",
		SupportStatus: StatusSeparateProvider,
		SupportReason: ReasonOutOfScope,
		PublicAPI:     true,
		Limitations: []string{
			"Reconfirmed, not just assumed: Event Mesh is activated through the same generic " +
				"\"Activating and Managing Capabilities\" mechanism as every other Integration Suite " +
				"capability (no dedicated public activation API, consistent with every other " +
				"capability audited), but once active it exposes a genuine, well-documented broker " +
				"management surface (service-key-based channel/queue/subscription creation, " +
				"AMQP/MQTT/REST messaging APIs) confirmed across roughly forty-five documentation " +
				"pages.",
			"Event Mesh predates, and is usable entirely independently of, SAP Integration Suite — " +
				"it is a general-purpose BTP messaging service consumed by many unrelated SAP " +
				"products, with its own Solace PubSub+-derived API family fundamentally different in " +
				"shape from the OData-centric model this provider is built around. Community " +
				"Terraform support for Solace PubSub+ already exists independently. Rather than " +
				"absorbing a broker-management surface into this Integration-Suite-scoped provider, " +
				"this belongs in a separate, independently versioned provider, the same reasoning " +
				"already applied to Developer Hub.",
		},
	},
	{
		Key:    "integration_assessment.master_data",
		Domain: "integration_assessment",
		Name:   "Integration Assessment Master Data",
		Description: "SAP Integration Solution Advisory Methodology (ISA-M) taxonomy: Domain, " +
			"Style, Use Case Pattern, Integration Pattern, Key Characteristic (and its Group/Value/" +
			"Recommendation), Deployment Model, Domain Determination — largely SAP-maintained " +
			"reference content a tenant can review and adjust.",
		SupportStatus:   StatusUnofficial,
		SupportReason:   ReasonPublicAPIIncomplete,
		ContractSource:  SourceMetadataOnly,
		DataSourceTypes: []string{"sapintegrationsuite_integration_assessment_deployment_model"},
		PublicAPI:       true,
		APIProtocol:     "OData V2",
		Operations:      Operations{Read: true},
		Limitations: []string{
			"SAP documents the Entities API and lists these entities with a description each; the " +
				"field-level specification on the Business Accelerator Hub needs an SAP login. The " +
				"contract comes from the live $metadata (snapshot testdata/api-metadata/integration-" +
				"assessment-entities.json), and every entity set was read on a tenant in September 2026.",
			"Only deployment models have a data source, looked up by name, because the landscape " +
				"instances link to them. The rest of the taxonomy (domains, styles, patterns, key " +
				"characteristics) is not exposed yet.",
			"Needs provider.integration_assessment: the service key of an \"Integration Assessment " +
				"APIs\" service instance; the other credential sets do not work there.",
		},
	},
	{
		Key:    "integration_assessment.landscape_configuration",
		Domain: "integration_assessment",
		Name:   "Integration Assessment Landscape Configuration",
		Description: "Tenant-owned integration landscape inventory: Application, Application " +
			"Instance, Technology, Technology Instance, Vendor, and their association entities " +
			"(Technology Domain, Technology Style, Technology Key Characteristic).",
		SupportStatus:  StatusUnofficial,
		SupportReason:  ReasonPublicAPIIncomplete,
		ContractSource: SourceMetadataOnly,
		ResourceTypes: []string{
			"sapintegrationsuite_integration_assessment_vendor",
			"sapintegrationsuite_integration_assessment_application",
			"sapintegrationsuite_integration_assessment_application_instance",
			"sapintegrationsuite_integration_assessment_technology",
			"sapintegrationsuite_integration_assessment_technology_instance",
		},
		DataSourceTypes: []string{
			"sapintegrationsuite_integration_assessment_vendor",
			"sapintegrationsuite_integration_assessment_technology",
		},
		PublicAPI:   true,
		APIProtocol: "OData V2",
		Operations:  Operations{Create: true, Read: true, Update: true, Delete: true, Import: true},
		Limitations: []string{
			"SAP documents the Entities API, these entities and their per-tenant limits (20,000 " +
				"applications, 20,000 application instances, 50 technologies, 150 technology instances, " +
				"10,000 vendors), but the field-level specification on the Business Accelerator Hub needs " +
				"an SAP login. The requests follow the live $metadata and were verified on a tenant on " +
				"2026-09-27: create (201 with a UUID Id), read, PATCH and PUT (204), delete (204) for all " +
				"five objects, and links written as {\"Id\": ...}; the service rejects links written as " +
				"__metadata URIs (V124) and a Content-Type with a charset parameter (V122).",
			"Names, descriptions and links change in place (PATCH, verified on 2026-09-28): an " +
				"application's vendor, an application instance's application and deployment model, a " +
				"technology's vendor and a technology instance's deployment model. Only moving a technology " +
				"instance to another technology was not tested, so it replaces the instance.",
			"The association sets of a technology (TechnologyDomain, TechnologyStyle, " +
				"TechnologyKeyCharacteristic) are not managed yet. Create, read and delete work on a " +
				"tenant (2026-09-28); a key characteristic cannot be changed (PATCH: 400 V101).",
			"Needs provider.integration_assessment: the service key of an \"Integration Assessment " +
				"APIs\" service instance; the other credential sets do not work there.",
		},
	},
	{
		Key:    "integration_assessment.assessment_workflow",
		Domain: "integration_assessment",
		Name:   "Integration Assessment Requests and Assessment Workflow",
		Description: "Business solution/interface Request and Request Line Item workflow objects, " +
			"the Integration Flow/Message Flow content they reference, and Request Line Item " +
			"Technology Instance Decision.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonOutOfScope,
		PublicAPI:     true,
		Limitations: []string{
			"Confirmed to be workflow/project state, not desired-state configuration: SAP documents " +
				"an explicit Request status machine (draft -> new -> in progress -> completed, with a " +
				"Reopen action), matching this provider's established Message Processing Log/" +
				"Developer Hub Subscription category of exclusion — out of scope regardless of whether " +
				"a field-level API contract is ever confirmed for it.",
		},
	},
	{
		Key:    "migration_assessment.source_system",
		Domain: "other_capability",
		Name:   "Migration Assessment Source System",
		Description: "A registered SAP Process Orchestration system (7.31 SP28+, 7.40 SP23+, or " +
			"7.50 SP06+) Migration Assessment extracts integration scenario data from.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonNoPublicAPI,
		PublicAPI:     false,
		Limitations: []string{
			"No API-access or service-key documentation page exists anywhere in Migration " +
				"Assessment's documentation tree (a small, roughly fifteen-page tree, entirely " +
				"checked). Its own documentation instead describes Migration Assessment as an API " +
				"*consumer*: it reaches into a registered source system's own SAP Process " +
				"Orchestration APIs (via Cloud Connector/Destination service) to extract data — the " +
				"opposite direction from a public API this provider could manage Migration " +
				"Assessment's own objects through.",
			"Re-audit September 2026: the 11 current pages and the full Business Accelerator Hub " +
				"package list show no Migration Assessment API.",
		},
	},
	{
		Key:    "migration_assessment.extraction_and_evaluation",
		Domain: "other_capability",
		Name:   "Migration Assessment Extraction and Scenario Evaluation",
		Description: "Data Extraction Requests, Scenario Evaluation Requests, and their resulting " +
			"assessment-category/migration-readiness/effort-estimate reports.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonOutOfScope,
		PublicAPI:     false,
		Limitations: []string{
			"Confirmed to be action-triggered workflow and reporting data, not desired-state " +
				"configuration: SAP's own documentation describes choosing \"Create\" on a Data " +
				"Extraction Request as starting an extraction process with a resulting Completed/" +
				"Completed with warnings/Completed with errors status, and Scenario Evaluation " +
				"results are assessment-category classifications, migration-readiness scores, and " +
				"effort estimates — reporting output, the same category this provider already " +
				"excludes for Message Processing Logs. Out of scope regardless of whether a public " +
				"API is ever confirmed for triggering these actions.",
		},
	},
	{
		Key:    "data_space_integration",
		Domain: "other_capability",
		Name:   "Data Space Integration",
		Description: "SAP's Dataspace-Protocol-based data space connectivity capability " +
			"(Connectors, Assets, Policies, Contract Definitions, Contract Negotiations/" +
			"Agreements) within Integration Suite, initially scoped to the Catena-X data space.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonResearchRequired,
		PublicAPI:     true,
		Limitations: []string{
			"A dedicated \"Data Space Integration API Access\" service instance (plan api, roles " +
				"AuthGroup_DataspaceConsumer/AuthGroup_DataspaceProvider, client_credentials grant, " +
				"one instance per connector) gives API access. The Business Accelerator Hub lists one " +
				"REST API, DSIAPI 2.0.0; its specification needs an SAP login.",
			"Re-audit September 2026: SAP's Help shows requests only for consumer runtime flows " +
				"under /api/dsi/v1 (catalog, contract negotiation, transfer process). Assets, " +
				"policies, contract definitions, company policies and contract references, the " +
				"objects Terraform could manage, are documented only through the UI, so their API " +
				"contract is not confirmed.",
			"Being a REST API, it has no OData $metadata to read the contract from; the Hub " +
				"specification or new SAP documentation is needed. See the Data Space Integration " +
				"guide.",
		},
	},
	{
		Key:    "open_connectors",
		Domain: "other_capability",
		Name:   "Open Connectors",
		Description: "SAP's third-party SaaS connectivity capability within Integration Suite: a " +
			"catalog of 170+ third-party connector types (each with its own normalized REST API and " +
			"OpenAPI-documented instance configuration), originally the standalone Cloud Elements " +
			"product.",
		SupportStatus: StatusUnsupported,
		SupportReason: ReasonOutOfScope,
		PublicAPI:     true,
		Limitations: []string{
			"A deliberate suitability judgment, not a research gap: Open Connectors is not one " +
				"coherent API this provider could model with a handful of resources, the pattern " +
				"every other capability in this catalog follows. It is a catalog of 170+ independent " +
				"third-party connector types, each with its own authentication scheme, configuration " +
				"schema, and normalized-but-still-connector-specific REST surface. Implementing even " +
				"connector-instance management generically would mean either an unbounded, " +
				"per-connector-type schema explosion, or an opaque untyped-JSON resource that gives " +
				"up the type safety and validation this provider's schema-first design otherwise " +
				"provides everywhere else.",
		},
	},
}

// Lookup returns the Feature registered under key, and whether it was
// found.
func Lookup(key string) (Feature, bool) {
	for _, f := range Catalog {
		if f.Key == key {
			return f, true
		}
	}
	return Feature{}, false
}
