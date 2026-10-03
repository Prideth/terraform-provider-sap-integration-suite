package features

// EvidenceRecord is the dated research record behind a feature that is not
// fully supported: what was checked, what it showed, and what would change
// the classification. docs/research/capability-evidence-2026.md is generated
// from these records, and a test requires one for every such feature.
type EvidenceRecord struct {
	// CheckedOn is the date of the latest check (YYYY-MM-DD).
	CheckedOn string
	// Sources are the checked sources, each with its own date or version.
	Sources []string
	// Finding is what the sources show, in one or two sentences.
	Finding string
	// NextStep is the concrete evidence or action that would change the
	// classification.
	NextStep string
}

// Classification is the single label a feature's status and reason reduce
// to in the evidence document: the status for experimental, partial,
// read-only and separate-provider features, otherwise the reason.
func Classification(f Feature) string {
	switch f.SupportStatus {
	case StatusUnsupported:
		return string(f.SupportReason)
	default:
		return string(f.SupportStatus)
	}
}

// Sources checked in the September 2026 re-audit. Each names its own date or
// version, so a record stays readable when the sources move on.
const (
	srcHubPackages = "Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)"
	srcHubCI       = "Hub package CloudIntegrationAPI (Integration Content, Security Content, Partner Directory, Message Stores, MPL, Log Files, B2B Scenarios; modified 2026-08-07)"
	srcHubAPIM     = "Hub package APIMgmt (API Portal, Developer Hub, Metering, Billing, Graph Configuration APIs; modified 2026-09-24)"
	srcHubEIC      = "Hub package sap-int-eic-eic-operations (Jobs, Components, Partner Directory, Message Stores, MPL; created 2026-04-30, modified 2026-09-21)"
	srcHubIA       = "Hub package SAPIntegrationAssessment (EntitiesAPI, ManagementAPI, OData; modified 2025-07-25)"
	srcHubDSI      = "Hub package dataspaceintegration (DSIAPI 2.0.0, REST; modified 2026-07-16)"
	srcHelp        = "SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)"
	srcWhatsNew    = "What's New for API Management, Cloud Foundry (entries up to 2026-09-20)"
	srcSDK         = "SAP API Management Client SDK 3.0.6 (Maven Central, published 2026-09-24; classes StandardAPIProxyClient, StandardAPIProductClient, StandardAPIKeyValueMapClient)"
	srcRecipes     = "SAP/apibusinesshub-api-recipes (commit 2668274, 2026-05-07)"
	srcCICD        = "SAP/cicd-actions-for-sap-integration-suite (2026-06-02)"
	srcMetaCI      = "Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)"
	srcMetaAPIM    = "API portal Management.svc $metadata snapshot testdata/api-metadata/classic-api-management.json (2026-09-26)"
	srcTenant      = "tenant probes and acceptance runs on a development tenant (September 2026)"
	srcExport      = "a package export of the development tenant (2026-09-26): API artifacts and data types travel as package content"
	srcCatalog     = "earlier catalog research, recorded in the feature's limitations"
	srcMetaIA      = "Integration Assessment Entities and Management $metadata, fetched live on 2026-09-27 (snapshots testdata/api-metadata/integration-assessment-*.json)"
	srcHubSearch   = "Business Accelerator Hub: the API artifacts of 173 Integration Suite, API Management, Edge, Graph and BTP packages (308 APIs) searched for API management, API artifact, Integration Cell, runtime profile, virtual host, MCP, API deployment, API policy, reusable API, proxy, transport and gateway (2026-09-27); the package list was unchanged since 2026-09-26"
	srcSDKCheck    = "Maven Central metadata of apim-client-sdk (2026-09-27): latest version still 3.0.6"
	srcGapProbe    = "tenant-probe -GapTests on a development tenant (2026-09-27), synthetic tfAccProbe* objects only"
	srcIAProbe     = "ia-probe -LandscapeTests on the development tenant's Integration Assessment (2026-09-27 17:29), synthetic tfacc-probe objects only"
	srcNavProbe    = "tenant-probe -GapTests on the development tenant (2026-09-29 07:51): IntegrationPackages('<id>')/IntegrationDesigntimeArtifacts for a package with an integration flow, an API artifact and an MCP server returned only the integration flow"
	srcBundles     = "artifact bundles downloaded from the development tenant (2026-09-27): an API artifact is a RESTAPI bundle and an MCP server an MCPSERVER bundle, both for runtime profile integrationcell; the MCP server requires the capability of its source API artifact"
	srcAccRun      = "acceptance run of 2026-09-27 17:29 on the development tenant"
	srcIAProbe2    = "ia-probe -LandscapeTests on the development tenant (2026-09-27 19:33), synthetic tfacc-probe objects only"
	srcIAProbe3    = "ia-probe -LandscapeTests, extended, on the development tenant (2026-09-28 07:54): link changes with PATCH and the technology association sets, synthetic tfacc-probe objects only"
	srcHelpIA2     = "SAP Help, Integration Assessment APIs (docs/ISuite_Integration_Assessment/integration-assessment-apis-47847b5.md in SAP-docs/btp-integration-suite, last changed 2026-07-02), read 2026-09-29, and the Hub catalog of package SAPIntegrationAssessment (EntitiesAPI 1.0.0, specification behind a login)"
	srcAccRun3     = "acceptance run of 2026-09-29 14:35 on the development tenant, including TestAccIntegrationAssessment_landscape with the in-place moves"
	srcAccRun2     = "acceptance run of 2026-09-27 21:30 on the development tenant, including TestAccIntegrationAssessment_landscape"
)

const checked = "2026-09-26"

func ev(finding, next string, sources ...string) EvidenceRecord {
	return EvidenceRecord{CheckedOn: checked, Sources: sources, Finding: finding, NextStep: next}
}

// evOn is ev with an explicit date of the latest check.
func evOn(date, finding, next string, sources ...string) EvidenceRecord {
	r := ev(finding, next, sources...)
	r.CheckedOn = date
	return r
}

// reaudited is ev for the current API Management family, checked again on
// 2026-09-27 with the Hub search and the SDK version check.
func reaudited(finding, next string, sources ...string) EvidenceRecord {
	r := ev(finding, next, append(sources, srcHubSearch, srcSDKCheck)...)
	r.CheckedOn = "2026-09-27"
	return r
}

// Evidence holds a record for every feature that is not StatusSupported.
var Evidence = map[string]EvidenceRecord{
	// --- Cloud Integration ---
	"cloud_integration.value_mapping": ev(
		"Create, read, deploy and delete are verified on a tenant; an in-place content update is not documented, so every change replaces the artifact.",
		"A tenant check of PUT on ValueMappingDesigntimeArtifacts, or SAP documentation of it, would allow in-place updates and save_as_version.",
		srcHelp, srcMetaCI, srcTenant),
	"cloud_integration.value_mapping_entry": evOn("2026-09-27",
		"On a synthetic value mapping, the first UpsertValMaps (IsConfigured=true) dropped the pair's design-time values, a second upsert of the same source value added a duplicate that SAP made the default, and DeleteValMaps answered 202 without removing anything. There is no safe update or destroy.",
		"None within Terraform unless SAP documents an entry update and a delete that works; revisit when the Integration Content API reference changes.",
		srcHelp, srcMetaCI, srcGapProbe),
	"cloud_integration.design_time_versioning": ev(
		"SaveAsVersion is documented for integration flows and in $metadata for every versioned design-time type; the provider uses it for flows, message mappings and script collections.",
		"Value mappings need an in-place update first (see cloud_integration.value_mapping); the other types need their own resources.",
		srcHelp, srcMetaCI, srcTenant),
	"cloud_integration.data_type": evOn("2026-10-03",
		"TestAccDataType_basic passed on a tenant: create, in-place change of schema and description, import and SaveAsVersion. Create, update (PUT), SaveAsVersion and delete work on a tenant with the bundle SAP stores for a data type (XSD, additionalAttributes.json, metainfo.prop); elements added on create and update are kept. SAP Help documents no request, so the contract comes from the $metadata and the probes.",
		"None within the provider; it becomes supported if SAP documents the Data Types API (the Hub and SAP Help list none today).",
		srcHelp, srcMetaCI, srcExport, srcGapProbe),
	"cloud_integration.message_type": evOn("2026-10-03",
		"TestAccMessageType_basic passed on a tenant for message types and fault message types. Create without content, update of Description and DataTypeUsed (PUT without Name, which SAP refuses), SaveAsVersion and delete work on a tenant; SAP generates the schema from DataTypeUsed. SAP Help documents neither message types nor fault message types.",
		"None within the provider; both become supported if SAP documents the message type APIs.",
		srcHelp, srcMetaCI, srcTenant),
	"cloud_integration.service_interface": ev(
		"$metadata has ServiceInterfaceDesigntimeArtifacts with SaveAsVersion and a Resources navigation; SAP Help documents only the UI and ESR import.",
		"Same as message types, after data types are settled.",
		srcHelp, srcMetaCI),
	"cloud_integration.service_endpoints": ev(
		"Endpoints are generated from deployed content; the read contract is verified against $metadata and a tenant.",
		"None; discovery only by design.",
		srcHelp, srcMetaCI, srcTenant),
	"cloud_integration.integration_adapter": ev(
		"SAP documents delete and deploy for IntegrationAdapterDesigntimeArtifacts, not create or read; the properties are confirmed by $metadata. No adapter content was available for a tenant test.",
		"An adapter ZIP for a tenant test (pending tenant check #12).",
		srcHelp, srcMetaCI),
	"cloud_integration.integration_adapter_deployment": ev(
		"Deploy is documented; status polling through IntegrationRuntimeArtifacts is by analogy with the other artifacts.",
		"The same tenant test as the adapter itself.",
		srcHelp, srcMetaCI),
	"cloud_integration.custom_tag_configuration": ev(
		"POST CustomTagConfigurations?Overwrite=true is documented for create and update; no delete or clear is documented.",
		"SAP documentation of a delete, or a tenant check that an empty overwrite clears the configuration (a destructive, tenant-wide test).",
		srcHelp, srcMetaCI),
	"cloud_integration.message_processing_logs": ev(
		"Public and documented, but runtime monitoring data.",
		"None; out of scope by the provider's scope rule.",
		srcHelp, srcHubCI),
	"cloud_integration.message_stores": ev(
		"Public and documented, but runtime messages and queues; the operations are support actions.",
		"None; out of scope.",
		srcHelp, srcHubCI),
	"cloud_integration.variable": ev(
		"Only GET Variables(...)/$value is documented; variables are written by deployed flows.",
		"None; out of scope.",
		srcHelp, srcMetaCI),
	"cloud_integration.data_store": ev(
		"Only a monitoring GET is documented; data stores come into existence through deployed flows.",
		"None; out of scope.",
		srcHelp, srcMetaCI),
	"cloud_integration.data_store_entry": ev(
		"Only GETs are documented; entries are business messages.",
		"None; out of scope.",
		srcHelp, srcMetaCI),
	"cloud_integration.archiving": ev(
		"activateArchivingConfiguration and activateB2BArchivingConfiguration exist without a deactivation; archiving settings cannot be changed once enabled.",
		"A documented deactivation. Without it, a resource could never implement destroy.",
		srcHelp, srcMetaCI),

	// --- Security ---
	"security.user_credential": ev(
		"Create, read, update and delete verified on a tenant; the password is write-only, and kinds other than default are corroborated only by example payloads.",
		"tenant-probe -GapTests lists the kinds that exist on the tenant; the deployment status property is still unknown.",
		srcHelp, srcMetaCI, srcTenant),
	"security.oauth2_client_credential": ev(
		"Lifecycle verified on a tenant; CustomParameters is a navigation without a documented write path.",
		"A UI-created custom parameter read back through the API (pending tenant check), then a deep-insert test.",
		srcHelp, srcMetaCI, srcTenant),
	"security.keystore_entry": ev(
		"Read contract verified against $metadata and a tenant; certificates and key pairs have their own resources.",
		"None; read-only by design.",
		srcMetaCI, srcTenant),
	"security.key_pair": ev(
		"Generation, read, SSH export and single-alias delete verified on a tenant; no update is documented.",
		"Documentation of which generation parameters KeystoreEntries returns.",
		srcHelp, srcMetaCI, srcTenant),
	"security.ssh_key": ev(
		"SSHKeyGenerationRequests and SSHKeyResources exist in $metadata without documented requests; key pairs cover SSH through the OpenSSH export.",
		"None needed; revisit only if SAP documents SSH keys as a separate artifact.",
		srcHelp, srcMetaCI),
	"security.certificate_chain": evOn("2026-09-27",
		"$metadata has the CertificateChainResources media entity and ChainCertificates; SAP Help documents chain import only in the UI. The tenant returns a key pair's chain through KeystoreEntries('<hexalias>')/ChainCertificates.",
		"A read-only chain attribute on key pairs or a data source; an upload needs a documented media type.",
		srcHelp, srcMetaCI, srcGapProbe),
	"security.certificate_user_mapping": ev(
		"Documented only for the Neo environment.",
		"A Cloud Foundry API from SAP.",
		srcHelp, srcHubPackages),
	"security.known_hosts": ev(
		"UI-only; no entity in $metadata and none in the Security Content resource table.",
		"A Known Hosts entity in $metadata (the discovery test fails on any new entity set).",
		srcHelp, srcMetaCI),
	"security.oauth2_password_credential": ev(
		"UI-only; no entity set in $metadata. It may be stored as a UserCredentials kind.",
		"tenant-probe -GapTests lists the UserCredentials kinds of a tenant that has such an artifact; a documented kind would make it a user_credential variant.",
		srcHelp, srcMetaCI),
	"security.oauth2_saml_bearer": ev(
		"UI-only; no entity set in $metadata.",
		"Same as OAuth2 password credentials.",
		srcHelp, srcMetaCI),
	"security.where_used": ev(
		"UI-only; no entity or function import in $metadata.",
		"A where-used entity in $metadata; it would become a data source.",
		srcHelp, srcMetaCI),
	"security.secure_parameter": evOn("2026-09-27",
		"SAP Help documents Secure Parameters only in the Monitor UI, and neither the Security Content resource table nor SAP Help's list of API resources names SecureParameters. The entity set comes from the $metadata; create, read, PUT update and delete were verified on a tenant, and the acceptance test passes.",
		"Supported as soon as SAP documents SecureParameters (Security Content API reference or its example requests).",
		srcHelp, srcMetaCI, srcTenant),
	"security.pgp_keyring": evOn("2026-09-27",
		"$metadata declares keyrings, keys, sub keys, user IDs and upload media entities; SAP Help documents no request. On the tenant PgpPublicKeyrings answers 200 but PgpKeyrings 404: not every declared set is addressable.",
		"A documented upload format; the secret keyring would be write-only.",
		srcHelp, srcMetaCI, srcGapProbe),

	// --- Partner Directory ---
	"partner_directory.partner": ev(
		"No create exists; a partner appears with its first entry, and delete removes every entry of it.",
		"None; read-only by design.",
		srcHelp, srcTenant),

	// --- Classic API Management ---
	"api_management.classic.api_provider": ev(
		"Create, read and delete documented; no update (SAP's Piper tooling says create only). A tenant create failed with DEST_CREATION_FAILED_AUTH_ISSUE.",
		"Whether the create works on the tenant through the UI (pending tenant check #6), and field mappings for the other connection types.",
		srcHelp, srcMetaAPIM, srcTenant),
	"api_management.classic.api_proxy": reaudited(
		"Upload through Transport.svc as SAP's Client SDK does it, read and delete through Management.svc; the update semantics of an import over an existing proxy are undocumented. The acceptance run of 2026-09-27 17:29 answered the import of SAP's sample bundle, renamed to a tfacc proxy, with 400 APIPROXY_ZIP_ERROR (\"Verify the directory structure inside the zip\"); the request itself matches the SDK.",
		"apim-probe -ProxyTests (2026-09-27 19:33): the tenant's own export of a proxy, renamed, was rejected the same way, so the request is at fault, not the bundle. The implementation is withheld from the provider; the official Transport API specification (APIPortal_Transport_CF) decides the request, and TestAccAPIProxy_sample passing registers the resource as experimental.",
		srcSDK, srcHubAPIM, srcMetaAPIM, srcRecipes, srcTenant, srcAccRun),
	"api_management.classic.api_proxy_deployment": reaudited(
		"Imported proxies are deployed by default; no public call deploys or undeploys an existing proxy.",
		"A documented deploy or undeploy call.",
		srcHelp, srcSDK),
	"api_management.classic.policy": ev(
		"Policies are XML inside the proxy bundle, not separate entities; the proxy resource manages them as content.",
		"None; policies stay part of the proxy bundle.",
		srcRecipes, srcMetaAPIM),
	"api_management.classic.virtual_host": ev(
		"Create, update and delete are documented through Configuration.svc/VirtualHostRequests; the read schema is confirmed. The write path needs the APIManagement.SelfService.Administrator role and answered 403 with an administrator key.",
		"A service key with APIManagement.SelfService.Administrator and a tenant test (a tenant-wide change, so a destructive gate).",
		srcHelp, srcMetaAPIM, srcHubAPIM, srcTenant),
	"api_management.classic.certificate_store": ev(
		"Schema confirmed; the Hub describes the KeyStore and TrustStore APIs as create and view only.",
		"Documented update and delete.",
		srcHubAPIM, srcMetaAPIM),
	"api_management.classic.application": ev(
		"Schema confirmed; the Hub describes the Applications API as view only, creation only through the Developer API.",
		"Documented create through the API portal; the app secret would need write-only handling.",
		srcHubAPIM, srcMetaAPIM),
	"api_management.classic.environment_key_value_map": ev(
		"KeyMapEntries and KeyMapEntryValues confirmed by $metadata; how they relate to the generic key value maps is not documented.",
		"SAP documentation of the difference, or a tenant test.",
		srcHubAPIM, srcMetaAPIM),
	"api_management.classic.cache_resource": ev(
		"Schema confirmed; the Hub lists a cache resource API only for Neo.",
		"A Cloud Foundry cache resource API.",
		srcHubAPIM, srcMetaAPIM),
	"api_management.classic.rate_plan": ev(
		"Schema confirmed; the Hub lists only billing and metering APIs for monetization.",
		"A rate plan API.",
		srcHubAPIM, srcMetaAPIM),
	"api_management.classic.policy_template": ev(
		"Schema confirmed; no policy template API on the Hub, SAP Help describes only the UI.",
		"A policy template API.",
		srcHubAPIM, srcMetaAPIM, srcHelp),
	"api_management.classic.product_access_control": ev(
		"Schema confirmed; the Hub describes the Access Control Service as view and create rules.",
		"Documented update and delete.",
		srcHubAPIM, srcMetaAPIM),
	"api_management.classic.key_value_map": ev(
		"Create documented verbatim and verified; entry updates are not documented, encrypted maps not readable safely.",
		"A documented entry update or a tenant test of it.",
		srcHelp, srcMetaAPIM, srcTenant),

	// --- Current API Management ---
	"api_gateway.api_artifact": reaudited(
		"No public API: the Hub's APIMgmt and CloudIntegrationAPI packages, the Client SDK and SAP Help describe API artifacts only in the UI. API artifacts travel as package content (resourceType API) and reach another tenant only through package export/import (POST IntegrationPackages with PackageContent) or transport. The documented package navigation of the Integration Content API lists only a package's integration flows, not its API artifacts or MCP servers.",
		"An API artifact API from SAP. Until then, whole-package import is the only public path; it is opaque to individual API artifacts and not modelled.",
		srcHubAPIM, srcHubCI, srcHubPackages, srcSDK, srcHelp, srcWhatsNew, srcExport, srcBundles, srcNavProbe, srcCICD),
	"api_gateway.api_artifact_deployment": reaudited(
		"Deployment is a UI action; IntegrationRuntimeArtifacts is not documented for API artifacts.",
		"tenant-probe -GapTests records whether deployed API artifacts appear among the runtime artifact types; a documented deploy call is still needed.",
		srcHelp, srcHubCI),
	"api_gateway.api_policy": reaudited(
		"Policies are edited inside the API artifact; no separate lifecycle.",
		"Depends on an API artifact API.",
		srcHelp),
	"api_gateway.reusable_api_artifact": reaudited(
		"Released 2026-07-05 as a UI feature; same blocker as API artifacts.",
		"Depends on an API artifact API.",
		srcWhatsNew, srcHelp),
	"api_gateway.mcp_server": reaudited(
		"MCP Gateway (2026-07-05) and remote MCP servers (2026-09-20) are UI features; MCP servers travel as package content (MCPSERVER bundles that depend on their source API artifact).",
		"An MCP server API from SAP.",
		srcWhatsNew, srcHelp, srcHubAPIM, srcSDK, srcBundles),
	"api_gateway.runtime_profile": reaudited(
		"Enabled under Settings > Integrations only; the Runtime Profiles page lists no Integration Cell row.",
		"An API; even then a weak data source.",
		srcHelp),
	"integration_cell.runtime": reaudited(
		"Activation, status and configuration are UI-only.",
		"An Integration Cell API.",
		srcHelp, srcHubPackages),
	"integration_cell.virtual_host": reaudited(
		"Managed in Monitor > Integrations and APIs > Virtual Host only.",
		"A virtual host API for Integration Cell.",
		srcHelp, srcHubPackages),

	// --- Edge Integration Cell ---
	"edge_integration_cell.registration": ev(
		"Edge nodes are added through Edge Lifecycle Management and a local bridge executable.",
		"None expected; a registration API from SAP would reopen it.",
		srcHelp),
	"edge_integration_cell.local_api": ev(
		"The local OData APIs are public (Hub package created 2026-04-30); they cover jobs, components, message stores, MPL and, new in the package, the Partner Directory. Monitoring stays out of scope, and the Partner Directory of an Edge Integration Cell is reachable through /location/<id>/api/v1 from the cloud.",
		"None for monitoring; the Partner Directory goes through edge_integration_cell.deployment_target.",
		srcHubEIC, srcHelp),
	"edge_integration_cell.runtime": ev(
		"Runtime parameters configure Kubernetes-level resources; components can be restarted, an imperative action.",
		"None; out of scope.",
		srcHubEIC, srcHelp),
	"edge_integration_cell.deployment_target": ev(
		"SAP Help (2026-09-18) documents https://<host>/location/<runtime location id>/api/v1 for Integration Content, Security Content, Partner Directory, MPL, message stores and number ranges, without per-operation examples. Not verified on a tenant with an Edge Integration Cell.",
		"TestAccEdgeIntegrationCell_securityAndPartnerDirectory on a tenant with an Edge Integration Cell; passing it makes runtime_location_id supported for those resources.",
		srcHelp, srcHubEIC),
	"edge_integration_cell.access_policy_replication": ev(
		"AccessPolicyRuntimeAssignments is readable through the access policy; no write is documented.",
		"Documented creation of assignments.",
		srcMetaCI, srcTenant),

	// --- Capability activation ---
	"capabilities.cloud_integration": ev(
		"Activated in the subscription wizard; no API.",
		"None expected.",
		srcHelp, srcHubPackages),
	"capabilities.api_management": ev(
		"Activated under Manage Capabilities; no API.",
		"None expected.",
		srcHelp, srcHubPackages),
	"capabilities.api_gateway": reaudited(
		"Activated with the API Management capability in the UI; no API.",
		"None expected.",
		srcHelp, srcHubPackages),
	"capabilities.integration_cell": reaudited(
		"Activated under Settings > Runtime; no API.",
		"None expected.",
		srcHelp, srcHubPackages),
	"capabilities.edge_integration_cell": ev(
		"Activated in the UI and through Edge Lifecycle Management; no API.",
		"None expected.",
		srcHelp, srcHubPackages),

	// --- Other capabilities ---
	"odata_provisioning": ev(
		"UI-only configuration; no OData Provisioning package on the Hub.",
		"A management API from SAP.",
		srcHelp, srcHubPackages),
	"developer_hub": ev(
		"Public APIs exist (APIMgmt package: Product, Application, Discovery, Registering Users, External Governance); a separate provider by design.",
		"None for this provider.",
		srcHubAPIM),
	"integration_advisor.design_time_content": ev(
		"No API page in the documentation; the Hub has only EDI integration templates (modified 2026-04-02).",
		"An Integration Advisor API from SAP.",
		srcHelp, srcHubPackages),
	"integration_advisor.runtime_artifact_injection": ev(
		"A UI wizard and a one-shot action.",
		"None; out of scope.",
		srcHelp),
	"trading_partner_management.company_profile": ev(
		"UI-only with a JSON download; the B2B Scenarios API covers monitoring only.",
		"A TPM configuration API.",
		srcHelp, srcHubCI, srcHubPackages, srcMetaCI),
	"trading_partner_management.partner_profile": ev(
		"UI-only, like company profiles.",
		"A TPM configuration API.",
		srcHelp, srcHubPackages),
	"trading_partner_management.agreement_template": ev(
		"UI-only, like company profiles.",
		"A TPM configuration API.",
		srcHelp, srcHubPackages),
	"trading_partner_management.agreement": ev(
		"UI-only, like company profiles.",
		"A TPM configuration API.",
		srcHelp, srcHubPackages),
	"trading_partner_management.partner_directory_generation": ev(
		"A side effect of activating an agreement.",
		"None; out of scope.",
		srcHelp),
	"event_mesh": ev(
		"A general BTP messaging service with its own management APIs (Hub package modified 2026-09-18); a separate provider by design.",
		"None for this provider.",
		srcHubPackages, srcHelp),
	"integration_assessment.master_data": evOn("2026-09-29",
		"The live Entities $metadata (27 entity sets) confirms the ISA-M taxonomy entities field by field, and every set was read on a tenant (7 domains, 6 styles, 24 use case patterns, 12 integration patterns, 14 key characteristics in 5 groups with 29 values, 4 recommendation degrees, 9 domain determinations). SAP Help's entity list (last changed 2026-07-02) describes each entity; its text for Domain Determination repeats the recommendation degree's, so that entity's meaning comes from the $metadata (a domain between a source and a target deployment model). The Hub's EntitiesAPI specification still ends at a login page (2026-09-29).",
		"Every taxonomy entity has a read-only data source; TestAccIntegrationAssessment_taxonomy and _technologyProfile passed on 2026-09-29. SAP publishing the Entities API specification without a login, or its download from the Hub into .specs/specs, would make the contract official.",
		srcHubIA, srcHelp, srcMetaIA, srcIAProbe, srcHelpIA2),
	"integration_assessment.landscape_configuration": evOn("2026-09-28",
		"A second landscape probe (2026-09-27 19:33) verified create (201 with a UUID Id), read, PATCH and PUT (204) and delete (204, then 404) for vendors, applications, application instances, technologies and technology instances, with links written as {\"Id\": ...}; setting, removing and resetting an application's vendor worked. The first probe had failed only on its own request format (charset parameter, __metadata links). TestAccIntegrationAssessment_landscape passed on 2026-09-27 21:30. A third probe (2026-09-28) changed an instance's application and deployment model, a technology's vendor and a technology instance's name and deployment model with PATCH, each confirmed by a read; TechnologyDomain, TechnologyStyle and TechnologyKeyCharacteristic were created, read and deleted, and a key characteristic's PATCH was refused (400 V101).",
		"TestAccIntegrationAssessment_landscape with the in-place moves passed on 2026-09-29; TestAccIntegrationAssessment_technologyProfile confirms the technology profile resources and lookups. The Hub specification (EntitiesAPI) would make the contract official and allow the status supported.",
		srcHubIA, srcHelp, srcMetaIA, srcIAProbe, srcIAProbe2, srcIAProbe3, srcAccRun2, srcAccRun3),
	"integration_assessment.assessment_workflow": ev(
		"A request status machine; workflow state.",
		"None; out of scope.",
		srcHelp),
	"migration_assessment.source_system": ev(
		"Migration Assessment consumes APIs; it exposes none.",
		"A Migration Assessment API from SAP.",
		srcHelp, srcHubPackages),
	"migration_assessment.extraction_and_evaluation": ev(
		"Action-triggered workflow and reports.",
		"None; out of scope.",
		srcHelp),
	"data_space_integration": ev(
		"DSIAPI 2.0.0 (REST) is on the Hub; SAP Help shows only consumer runtime requests. Assets, policies and contract definitions are documented only in the UI.",
		"The DSIAPI OpenAPI document, parsed with cmd/apidiscovery -from, to confirm the configuration objects; then tenant tests.",
		srcHubDSI, srcHelp),
	"open_connectors": ev(
		"A catalog of third-party connectors with connector-specific APIs.",
		"None; out of scope.",
		srcHelp, srcCatalog),
}
