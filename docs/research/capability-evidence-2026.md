# Capability evidence, September 2026

<!-- Generated from internal/features (catalog.go and evidence.go) by "go run ./cmd/gendocs". Do not edit by hand. -->

This document records, for every catalog entry that is not fully supported,
what was checked, what it showed, and what would change its classification.
It is the reference behind the statuses in [feature-support.md](../feature-support.md);
the rationale for the method is in
[sap-2026-public-api-gap-closure.md](sap-2026-public-api-gap-closure.md), and the
entity-level view of each service contract is in
[api-discovery-report.md](../api-discovery-report.md).

Two rules hold throughout. An entity set in a service's `$metadata` shows that the
entity exists, not that SAP supports writing it: a mutable resource additionally
needs SAP documentation, SAP's own tooling, or a safe verification on a tenant.
And the provider never uses browser endpoints of SAP's UIs.

| Classification | Features |
|---|---:|
| 🟡 Partial (`partial`) | 10 |
| 👁️ Read-only (`read_only`) | 3 |
| 🧭 Unofficial (`unofficial`) | 5 |
| 🔬 Research required (`research_required`) | 9 |
| ❌ Public API incomplete (`public_api_incomplete`) | 8 |
| ❌ Unsafe Terraform lifecycle (`unsafe_terraform_lifecycle`) | 2 |
| ❌ No public API (`no_public_api`) | 26 |
| ❌ Out of scope (`out_of_scope`) | 13 |
| ↗️ Separate provider (`separate_provider`) | 2 |

## 🟡 Partial

Usable, but the provider deliberately leaves out part of the lifecycle; the limitations say which part and why.

### API Provider (classic API Management)

`api_management.classic.api_provider` · checked 2026-09-26

- **Finding:** Create, read and delete documented; no update (SAP's Piper tooling says create only). A tenant create failed with DEST_CREATION_FAILED_AUTH_ISSUE.
- **Next step:** Whether the create works on the tenant through the UI (pending tenant check #6), and field mappings for the other connection types.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - API portal Management.svc $metadata snapshot testdata/api-metadata/classic-api-management.json (2026-09-26)
  - tenant probes and acceptance runs on a development tenant (September 2026)

### Key Value Map (classic API Management)

`api_management.classic.key_value_map` · checked 2026-09-26

- **Finding:** Create documented verbatim and verified; entry updates are not documented, encrypted maps not readable safely.
- **Next step:** A documented entry update or a tenant test of it.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - API portal Management.svc $metadata snapshot testdata/api-metadata/classic-api-management.json (2026-09-26)
  - tenant probes and acceptance runs on a development tenant (September 2026)

### Custom Tag Configuration

`cloud_integration.custom_tag_configuration` · checked 2026-09-26

- **Finding:** POST CustomTagConfigurations?Overwrite=true is documented for create and update; no delete or clear is documented.
- **Next step:** SAP documentation of a delete, or a tenant check that an empty overwrite clears the configuration (a destructive, tenant-wide test).
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)

### Design-Time Artifact Versioning

`cloud_integration.design_time_versioning` · checked 2026-09-26

- **Finding:** SaveAsVersion is documented for integration flows and in $metadata for every versioned design-time type; the provider uses it for flows, message mappings and script collections.
- **Next step:** Value mappings need an in-place update first (see cloud_integration.value_mapping); the other types need their own resources.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)
  - tenant probes and acceptance runs on a development tenant (September 2026)

### Integration Adapter

`cloud_integration.integration_adapter` · checked 2026-09-26

- **Finding:** SAP documents delete and deploy for IntegrationAdapterDesigntimeArtifacts, not create or read; the properties are confirmed by $metadata. No adapter content was available for a tenant test.
- **Next step:** An adapter ZIP for a tenant test (pending tenant check #12).
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)

### Integration Adapter Deployment

`cloud_integration.integration_adapter_deployment` · checked 2026-09-26

- **Finding:** Deploy is documented; status polling through IntegrationRuntimeArtifacts is by analogy with the other artifacts.
- **Next step:** The same tenant test as the adapter itself.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)

### Value Mapping

`cloud_integration.value_mapping` · checked 2026-09-26

- **Finding:** Create, read, deploy and delete are verified on a tenant; an in-place content update is not documented, so every change replaces the artifact.
- **Next step:** A tenant check of PUT on ValueMappingDesigntimeArtifacts, or SAP documentation of it, would allow in-place updates and save_as_version.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)
  - tenant probes and acceptance runs on a development tenant (September 2026)

### Key Pair

`security.key_pair` · checked 2026-09-26

- **Finding:** Generation, read, SSH export and single-alias delete verified on a tenant; no update is documented.
- **Next step:** Documentation of which generation parameters KeystoreEntries returns.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)
  - tenant probes and acceptance runs on a development tenant (September 2026)

### OAuth2 Client Credential

`security.oauth2_client_credential` · checked 2026-09-26

- **Finding:** Lifecycle verified on a tenant; CustomParameters is a navigation without a documented write path.
- **Next step:** A UI-created custom parameter read back through the API (pending tenant check), then a deep-insert test.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)
  - tenant probes and acceptance runs on a development tenant (September 2026)

### User Credential

`security.user_credential` · checked 2026-09-26

- **Finding:** Create, read, update and delete verified on a tenant; the password is write-only, and kinds other than default are corroborated only by example payloads.
- **Next step:** tenant-probe -GapTests lists the kinds that exist on the tenant; the deployment status property is still unknown.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)
  - tenant probes and acceptance runs on a development tenant (September 2026)

## 👁️ Read-only

Available as a data source only; the provider never creates, changes or deletes it.

### Service Endpoints

`cloud_integration.service_endpoints` · checked 2026-09-26

- **Finding:** Endpoints are generated from deployed content; the read contract is verified against $metadata and a tenant.
- **Next step:** None; discovery only by design.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)
  - tenant probes and acceptance runs on a development tenant (September 2026)

### Partner

`partner_directory.partner` · checked 2026-09-26

- **Finding:** No create exists; a partner appears with its first entry, and delete removes every entry of it.
- **Next step:** None; read-only by design.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - tenant probes and acceptance runs on a development tenant (September 2026)

### Keystore Entry

`security.keystore_entry` · checked 2026-09-26

- **Finding:** Read contract verified against $metadata and a tenant; certificates and key pairs have their own resources.
- **Next step:** None; read-only by design.
- **Sources:**
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)
  - tenant probes and acceptance runs on a development tenant (September 2026)

## 🧭 Unofficial

Implementation has been validated, but relies on an API contract that SAP does not fully publish or officially document.

### Data Type

`cloud_integration.data_type` · checked 2026-10-03

- **Finding:** TestAccDataType_basic passed on a tenant: create, in-place change of schema and description, import and SaveAsVersion. Create, update (PUT), SaveAsVersion and delete work on a tenant with the bundle SAP stores for a data type (XSD, additionalAttributes.json, metainfo.prop); elements added on create and update are kept. SAP Help documents no request, so the contract comes from the $metadata and the probes.
- **Next step:** None within the provider; it becomes supported if SAP documents the Data Types API (the Hub and SAP Help list none today).
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)
  - a package export of the development tenant (2026-09-26): API artifacts and data types travel as package content
  - tenant-probe -GapTests on a development tenant (2026-09-27), synthetic tfAccProbe* objects only

### Edge Integration Cell Access Policy Replication

`edge_integration_cell.access_policy_replication` · checked 2026-09-26

- **Finding:** AccessPolicyRuntimeAssignments is readable through the access policy; no write is documented.
- **Next step:** Documented creation of assignments.
- **Sources:**
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)
  - tenant probes and acceptance runs on a development tenant (September 2026)

### Integration Assessment Landscape Configuration

`integration_assessment.landscape_configuration` · checked 2026-09-28

- **Finding:** A second landscape probe (2026-09-27 19:33) verified create (201 with a UUID Id), read, PATCH and PUT (204) and delete (204, then 404) for vendors, applications, application instances, technologies and technology instances, with links written as {"Id": ...}; setting, removing and resetting an application's vendor worked. The first probe had failed only on its own request format (charset parameter, __metadata links). TestAccIntegrationAssessment_landscape passed on 2026-09-27 21:30. A third probe (2026-09-28) changed an instance's application and deployment model, a technology's vendor and a technology instance's name and deployment model with PATCH, each confirmed by a read; TechnologyDomain, TechnologyStyle and TechnologyKeyCharacteristic were created, read and deleted, and a key characteristic's PATCH was refused (400 V101).
- **Next step:** TestAccIntegrationAssessment_landscape with the in-place moves passed on 2026-09-29; TestAccIntegrationAssessment_technologyProfile confirms the technology profile resources and lookups. The Hub specification (EntitiesAPI) would make the contract official and allow the status supported.
- **Sources:**
  - Hub package SAPIntegrationAssessment (EntitiesAPI, ManagementAPI, OData; modified 2025-07-25)
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Integration Assessment Entities and Management $metadata, fetched live on 2026-09-27 (snapshots testdata/api-metadata/integration-assessment-*.json)
  - ia-probe -LandscapeTests on the development tenant's Integration Assessment (2026-09-27 17:29), synthetic tfacc-probe objects only
  - ia-probe -LandscapeTests on the development tenant (2026-09-27 19:33), synthetic tfacc-probe objects only
  - ia-probe -LandscapeTests, extended, on the development tenant (2026-09-28 07:54): link changes with PATCH and the technology association sets, synthetic tfacc-probe objects only
  - acceptance run of 2026-09-27 21:30 on the development tenant, including TestAccIntegrationAssessment_landscape
  - acceptance run of 2026-09-29 14:35 on the development tenant, including TestAccIntegrationAssessment_landscape with the in-place moves

### Integration Assessment Master Data

`integration_assessment.master_data` · checked 2026-09-29

- **Finding:** The live Entities $metadata (27 entity sets) confirms the ISA-M taxonomy entities field by field, and every set was read on a tenant (7 domains, 6 styles, 24 use case patterns, 12 integration patterns, 14 key characteristics in 5 groups with 29 values, 4 recommendation degrees, 9 domain determinations). SAP Help's entity list (last changed 2026-07-02) describes each entity; its text for Domain Determination repeats the recommendation degree's, so that entity's meaning comes from the $metadata (a domain between a source and a target deployment model). The Hub's EntitiesAPI specification still ends at a login page (2026-09-29).
- **Next step:** Every taxonomy entity has a read-only data source; TestAccIntegrationAssessment_taxonomy and _technologyProfile passed on 2026-09-29. SAP publishing the Entities API specification without a login, or its download from the Hub into .specs/specs, would make the contract official.
- **Sources:**
  - Hub package SAPIntegrationAssessment (EntitiesAPI, ManagementAPI, OData; modified 2025-07-25)
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Integration Assessment Entities and Management $metadata, fetched live on 2026-09-27 (snapshots testdata/api-metadata/integration-assessment-*.json)
  - ia-probe -LandscapeTests on the development tenant's Integration Assessment (2026-09-27 17:29), synthetic tfacc-probe objects only
  - SAP Help, Integration Assessment APIs (docs/ISuite_Integration_Assessment/integration-assessment-apis-47847b5.md in SAP-docs/btp-integration-suite, last changed 2026-07-02), read 2026-09-29, and the Hub catalog of package SAPIntegrationAssessment (EntitiesAPI 1.0.0, specification behind a login)

### Secure Parameter

`security.secure_parameter` · checked 2026-09-27

- **Finding:** SAP Help documents Secure Parameters only in the Monitor UI, and neither the Security Content resource table nor SAP Help's list of API resources names SecureParameters. The entity set comes from the $metadata; create, read, PUT update and delete were verified on a tenant, and the acceptance test passes.
- **Next step:** Supported as soon as SAP documents SecureParameters (Security Content API reference or its example requests).
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)
  - tenant probes and acceptance runs on a development tenant (September 2026)

## 🔬 Research required

The capability has been identified, but its public API coverage, lifecycle semantics, or suitability for Terraform still requires further investigation.

### API Proxy (classic API Management)

`api_management.classic.api_proxy` · checked 2026-09-27

- **Finding:** Upload through Transport.svc as SAP's Client SDK does it, read and delete through Management.svc; the update semantics of an import over an existing proxy are undocumented. The acceptance run of 2026-09-27 17:29 answered the import of SAP's sample bundle, renamed to a tfacc proxy, with 400 APIPROXY_ZIP_ERROR ("Verify the directory structure inside the zip"); the request itself matches the SDK.
- **Next step:** apim-probe -ProxyTests (2026-09-27 19:33): the tenant's own export of a proxy, renamed, was rejected the same way, so the request is at fault, not the bundle. The implementation is withheld from the provider; the official Transport API specification (APIPortal_Transport_CF) decides the request, and TestAccAPIProxy_sample passing registers the resource as experimental.
- **Sources:**
  - SAP API Management Client SDK 3.0.6 (Maven Central, published 2026-09-24; classes StandardAPIProxyClient, StandardAPIProductClient, StandardAPIKeyValueMapClient)
  - Hub package APIMgmt (API Portal, Developer Hub, Metering, Billing, Graph Configuration APIs; modified 2026-09-24)
  - API portal Management.svc $metadata snapshot testdata/api-metadata/classic-api-management.json (2026-09-26)
  - SAP/apibusinesshub-api-recipes (commit 2668274, 2026-05-07)
  - tenant probes and acceptance runs on a development tenant (September 2026)
  - acceptance run of 2026-09-27 17:29 on the development tenant
  - Business Accelerator Hub: the API artifacts of 173 Integration Suite, API Management, Edge, Graph and BTP packages (308 APIs) searched for API management, API artifact, Integration Cell, runtime profile, virtual host, MCP, API deployment, API policy, reusable API, proxy, transport and gateway (2026-09-27); the package list was unchanged since 2026-09-26
  - Maven Central metadata of apim-client-sdk (2026-09-27): latest version still 3.0.6

### API Management Key Value Map across API Proxies (Classic)

`api_management.classic.environment_key_value_map` · checked 2026-09-26

- **Finding:** KeyMapEntries and KeyMapEntryValues confirmed by $metadata; how they relate to the generic key value maps is not documented.
- **Next step:** SAP documentation of the difference, or a tenant test.
- **Sources:**
  - Hub package APIMgmt (API Portal, Developer Hub, Metering, Billing, Graph Configuration APIs; modified 2026-09-24)
  - API portal Management.svc $metadata snapshot testdata/api-metadata/classic-api-management.json (2026-09-26)

### API Management Virtual Host (Classic)

`api_management.classic.virtual_host` · checked 2026-09-26

- **Finding:** Create, update and delete are documented through Configuration.svc/VirtualHostRequests; the read schema is confirmed. The write path needs the APIManagement.SelfService.Administrator role and answered 403 with an administrator key.
- **Next step:** A service key with APIManagement.SelfService.Administrator and a tenant test (a tenant-wide change, so a destructive gate).
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - API portal Management.svc $metadata snapshot testdata/api-metadata/classic-api-management.json (2026-09-26)
  - Hub package APIMgmt (API Portal, Developer Hub, Metering, Billing, Graph Configuration APIs; modified 2026-09-24)
  - tenant probes and acceptance runs on a development tenant (September 2026)

### Message Type

`cloud_integration.message_type` · checked 2026-09-26

- **Finding:** $metadata has MessageTypeDesigntimeArtifacts and FaultMessageTypeDesigntimeArtifacts with SaveAsVersion; SAP Help documents neither.
- **Next step:** The data type probe result decides the pattern; a message type probe follows with content that references a data type.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)

### Service Interface

`cloud_integration.service_interface` · checked 2026-09-26

- **Finding:** $metadata has ServiceInterfaceDesigntimeArtifacts with SaveAsVersion and a Resources navigation; SAP Help documents only the UI and ESR import.
- **Next step:** Same as message types, after data types are settled.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)

### Data Space Integration

`data_space_integration` · checked 2026-09-26

- **Finding:** DSIAPI 2.0.0 (REST) is on the Hub; SAP Help shows only consumer runtime requests. Assets, policies and contract definitions are documented only in the UI.
- **Next step:** The DSIAPI OpenAPI document, parsed with cmd/apidiscovery -from, to confirm the configuration objects; then tenant tests.
- **Sources:**
  - Hub package dataspaceintegration (DSIAPI 2.0.0, REST; modified 2026-07-16)
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)

### Edge Integration Cell Runtime Targeting

`edge_integration_cell.deployment_target` · checked 2026-09-26

- **Finding:** SAP Help (2026-09-18) documents https://<host>/location/<runtime location id>/api/v1 for Integration Content, Security Content, Partner Directory, MPL, message stores and number ranges, without per-operation examples. Not verified on a tenant with an Edge Integration Cell.
- **Next step:** TestAccEdgeIntegrationCell_securityAndPartnerDirectory on a tenant with an Edge Integration Cell; passing it makes runtime_location_id supported for those resources.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Hub package sap-int-eic-eic-operations (Jobs, Components, Partner Directory, Message Stores, MPL; created 2026-04-30, modified 2026-09-21)

### Certificate Chain

`security.certificate_chain` · checked 2026-09-27

- **Finding:** $metadata has the CertificateChainResources media entity and ChainCertificates; SAP Help documents chain import only in the UI. The tenant returns a key pair's chain through KeystoreEntries('<hexalias>')/ChainCertificates.
- **Next step:** A read-only chain attribute on key pairs or a data source; an upload needs a documented media type.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)
  - tenant-probe -GapTests on a development tenant (2026-09-27), synthetic tfAccProbe* objects only

### PGP Keyrings

`security.pgp_keyring` · checked 2026-09-27

- **Finding:** $metadata declares keyrings, keys, sub keys, user IDs and upload media entities; SAP Help documents no request. On the tenant PgpPublicKeyrings answers 200 but PgpKeyrings 404: not every declared set is addressable.
- **Next step:** A documented upload format; the secret keyring would be write-only.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)
  - tenant-probe -GapTests on a development tenant (2026-09-27), synthetic tfAccProbe* objects only

## ❌ Public API incomplete

A public API exists, but SAP does not document enough of it, for example no update or delete, to manage it with Terraform.

### API Proxy Deployment (classic API Management)

`api_management.classic.api_proxy_deployment` · checked 2026-09-27

- **Finding:** Imported proxies are deployed by default; no public call deploys or undeploys an existing proxy.
- **Next step:** A documented deploy or undeploy call.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - SAP API Management Client SDK 3.0.6 (Maven Central, published 2026-09-24; classes StandardAPIProxyClient, StandardAPIProductClient, StandardAPIKeyValueMapClient)
  - Business Accelerator Hub: the API artifacts of 173 Integration Suite, API Management, Edge, Graph and BTP packages (308 APIs) searched for API management, API artifact, Integration Cell, runtime profile, virtual host, MCP, API deployment, API policy, reusable API, proxy, transport and gateway (2026-09-27); the package list was unchanged since 2026-09-26
  - Maven Central metadata of apim-client-sdk (2026-09-27): latest version still 3.0.6

### API Management Application and Developer (Classic)

`api_management.classic.application` · checked 2026-09-26

- **Finding:** Schema confirmed; the Hub describes the Applications API as view only, creation only through the Developer API.
- **Next step:** Documented create through the API portal; the app secret would need write-only handling.
- **Sources:**
  - Hub package APIMgmt (API Portal, Developer Hub, Metering, Billing, Graph Configuration APIs; modified 2026-09-24)
  - API portal Management.svc $metadata snapshot testdata/api-metadata/classic-api-management.json (2026-09-26)

### API Management Cache Resource (Classic)

`api_management.classic.cache_resource` · checked 2026-09-26

- **Finding:** Schema confirmed; the Hub lists a cache resource API only for Neo.
- **Next step:** A Cloud Foundry cache resource API.
- **Sources:**
  - Hub package APIMgmt (API Portal, Developer Hub, Metering, Billing, Graph Configuration APIs; modified 2026-09-24)
  - API portal Management.svc $metadata snapshot testdata/api-metadata/classic-api-management.json (2026-09-26)

### API Management Certificate Store and Certificate (Classic)

`api_management.classic.certificate_store` · checked 2026-09-26

- **Finding:** Schema confirmed; the Hub describes the KeyStore and TrustStore APIs as create and view only.
- **Next step:** Documented update and delete.
- **Sources:**
  - Hub package APIMgmt (API Portal, Developer Hub, Metering, Billing, Graph Configuration APIs; modified 2026-09-24)
  - API portal Management.svc $metadata snapshot testdata/api-metadata/classic-api-management.json (2026-09-26)

### Policy (classic API Management)

`api_management.classic.policy` · checked 2026-09-26

- **Finding:** Policies are XML inside the proxy bundle, not separate entities; the proxy resource manages them as content.
- **Next step:** None; policies stay part of the proxy bundle.
- **Sources:**
  - SAP/apibusinesshub-api-recipes (commit 2668274, 2026-05-07)
  - API portal Management.svc $metadata snapshot testdata/api-metadata/classic-api-management.json (2026-09-26)

### API Management Policy Template (Classic)

`api_management.classic.policy_template` · checked 2026-09-26

- **Finding:** Schema confirmed; no policy template API on the Hub, SAP Help describes only the UI.
- **Next step:** A policy template API.
- **Sources:**
  - Hub package APIMgmt (API Portal, Developer Hub, Metering, Billing, Graph Configuration APIs; modified 2026-09-24)
  - API portal Management.svc $metadata snapshot testdata/api-metadata/classic-api-management.json (2026-09-26)
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)

### API Management Product Access Control (Classic)

`api_management.classic.product_access_control` · checked 2026-09-26

- **Finding:** Schema confirmed; the Hub describes the Access Control Service as view and create rules.
- **Next step:** Documented update and delete.
- **Sources:**
  - Hub package APIMgmt (API Portal, Developer Hub, Metering, Billing, Graph Configuration APIs; modified 2026-09-24)
  - API portal Management.svc $metadata snapshot testdata/api-metadata/classic-api-management.json (2026-09-26)

### API Management Rate Plan (Classic)

`api_management.classic.rate_plan` · checked 2026-09-26

- **Finding:** Schema confirmed; the Hub lists only billing and metering APIs for monetization.
- **Next step:** A rate plan API.
- **Sources:**
  - Hub package APIMgmt (API Portal, Developer Hub, Metering, Billing, Graph Configuration APIs; modified 2026-09-24)
  - API portal Management.svc $metadata snapshot testdata/api-metadata/classic-api-management.json (2026-09-26)

## ❌ Unsafe Terraform lifecycle

A public API exists, but a Terraform resource could not implement its lifecycle safely.

### Archiving Configuration

`cloud_integration.archiving` · checked 2026-09-26

- **Finding:** activateArchivingConfiguration and activateB2BArchivingConfiguration exist without a deactivation; archiving settings cannot be changed once enabled.
- **Next step:** A documented deactivation. Without it, a resource could never implement destroy.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)

### Value Mapping Entry

`cloud_integration.value_mapping_entry` · checked 2026-09-27

- **Finding:** On a synthetic value mapping, the first UpsertValMaps (IsConfigured=true) dropped the pair's design-time values, a second upsert of the same source value added a duplicate that SAP made the default, and DeleteValMaps answered 202 without removing anything. There is no safe update or destroy.
- **Next step:** None within Terraform unless SAP documents an entry update and a delete that works; revisit when the Integration Content API reference changes.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)
  - tenant-probe -GapTests on a development tenant (2026-09-27), synthetic tfAccProbe* objects only

## ❌ No public API

Only UI procedures or internal endpoints, which the provider does not use.

### API Artifact — Current API Management

`api_gateway.api_artifact` · checked 2026-09-27

- **Finding:** No public API: the Hub's APIMgmt and CloudIntegrationAPI packages, the Client SDK and SAP Help describe API artifacts only in the UI. API artifacts travel as package content (resourceType API) and reach another tenant only through package export/import (POST IntegrationPackages with PackageContent) or transport. The documented package navigation of the Integration Content API lists only a package's integration flows, not its API artifacts or MCP servers.
- **Next step:** An API artifact API from SAP. Until then, whole-package import is the only public path; it is opaque to individual API artifacts and not modelled.
- **Sources:**
  - Hub package APIMgmt (API Portal, Developer Hub, Metering, Billing, Graph Configuration APIs; modified 2026-09-24)
  - Hub package CloudIntegrationAPI (Integration Content, Security Content, Partner Directory, Message Stores, MPL, Log Files, B2B Scenarios; modified 2026-08-07)
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)
  - SAP API Management Client SDK 3.0.6 (Maven Central, published 2026-09-24; classes StandardAPIProxyClient, StandardAPIProductClient, StandardAPIKeyValueMapClient)
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - What's New for API Management, Cloud Foundry (entries up to 2026-09-20)
  - a package export of the development tenant (2026-09-26): API artifacts and data types travel as package content
  - artifact bundles downloaded from the development tenant (2026-09-27): an API artifact is a RESTAPI bundle and an MCP server an MCPSERVER bundle, both for runtime profile integrationcell; the MCP server requires the capability of its source API artifact
  - tenant-probe -GapTests on the development tenant (2026-09-29 07:51): IntegrationPackages('<id>')/IntegrationDesigntimeArtifacts for a package with an integration flow, an API artifact and an MCP server returned only the integration flow
  - SAP/cicd-actions-for-sap-integration-suite (2026-06-02)
  - Business Accelerator Hub: the API artifacts of 173 Integration Suite, API Management, Edge, Graph and BTP packages (308 APIs) searched for API management, API artifact, Integration Cell, runtime profile, virtual host, MCP, API deployment, API policy, reusable API, proxy, transport and gateway (2026-09-27); the package list was unchanged since 2026-09-26
  - Maven Central metadata of apim-client-sdk (2026-09-27): latest version still 3.0.6

### API Artifact Deployment — Integration Cell

`api_gateway.api_artifact_deployment` · checked 2026-09-27

- **Finding:** Deployment is a UI action; IntegrationRuntimeArtifacts is not documented for API artifacts.
- **Next step:** tenant-probe -GapTests records whether deployed API artifacts appear among the runtime artifact types; a documented deploy call is still needed.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Hub package CloudIntegrationAPI (Integration Content, Security Content, Partner Directory, Message Stores, MPL, Log Files, B2B Scenarios; modified 2026-08-07)
  - Business Accelerator Hub: the API artifacts of 173 Integration Suite, API Management, Edge, Graph and BTP packages (308 APIs) searched for API management, API artifact, Integration Cell, runtime profile, virtual host, MCP, API deployment, API policy, reusable API, proxy, transport and gateway (2026-09-27); the package list was unchanged since 2026-09-26
  - Maven Central metadata of apim-client-sdk (2026-09-27): latest version still 3.0.6

### API Artifact Policy

`api_gateway.api_policy` · checked 2026-09-27

- **Finding:** Policies are edited inside the API artifact; no separate lifecycle.
- **Next step:** Depends on an API artifact API.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub: the API artifacts of 173 Integration Suite, API Management, Edge, Graph and BTP packages (308 APIs) searched for API management, API artifact, Integration Cell, runtime profile, virtual host, MCP, API deployment, API policy, reusable API, proxy, transport and gateway (2026-09-27); the package list was unchanged since 2026-09-26
  - Maven Central metadata of apim-client-sdk (2026-09-27): latest version still 3.0.6

### MCP Server

`api_gateway.mcp_server` · checked 2026-09-27

- **Finding:** MCP Gateway (2026-07-05) and remote MCP servers (2026-09-20) are UI features; MCP servers travel as package content (MCPSERVER bundles that depend on their source API artifact).
- **Next step:** An MCP server API from SAP.
- **Sources:**
  - What's New for API Management, Cloud Foundry (entries up to 2026-09-20)
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Hub package APIMgmt (API Portal, Developer Hub, Metering, Billing, Graph Configuration APIs; modified 2026-09-24)
  - SAP API Management Client SDK 3.0.6 (Maven Central, published 2026-09-24; classes StandardAPIProxyClient, StandardAPIProductClient, StandardAPIKeyValueMapClient)
  - artifact bundles downloaded from the development tenant (2026-09-27): an API artifact is a RESTAPI bundle and an MCP server an MCPSERVER bundle, both for runtime profile integrationcell; the MCP server requires the capability of its source API artifact
  - Business Accelerator Hub: the API artifacts of 173 Integration Suite, API Management, Edge, Graph and BTP packages (308 APIs) searched for API management, API artifact, Integration Cell, runtime profile, virtual host, MCP, API deployment, API policy, reusable API, proxy, transport and gateway (2026-09-27); the package list was unchanged since 2026-09-26
  - Maven Central metadata of apim-client-sdk (2026-09-27): latest version still 3.0.6

### Reusable API Artifact

`api_gateway.reusable_api_artifact` · checked 2026-09-27

- **Finding:** Released 2026-07-05 as a UI feature; same blocker as API artifacts.
- **Next step:** Depends on an API artifact API.
- **Sources:**
  - What's New for API Management, Cloud Foundry (entries up to 2026-09-20)
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub: the API artifacts of 173 Integration Suite, API Management, Edge, Graph and BTP packages (308 APIs) searched for API management, API artifact, Integration Cell, runtime profile, virtual host, MCP, API deployment, API policy, reusable API, proxy, transport and gateway (2026-09-27); the package list was unchanged since 2026-09-26
  - Maven Central metadata of apim-client-sdk (2026-09-27): latest version still 3.0.6

### Runtime Profile

`api_gateway.runtime_profile` · checked 2026-09-27

- **Finding:** Enabled under Settings > Integrations only; the Runtime Profiles page lists no Integration Cell row.
- **Next step:** An API; even then a weak data source.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub: the API artifacts of 173 Integration Suite, API Management, Edge, Graph and BTP packages (308 APIs) searched for API management, API artifact, Integration Cell, runtime profile, virtual host, MCP, API deployment, API policy, reusable API, proxy, transport and gateway (2026-09-27); the package list was unchanged since 2026-09-26
  - Maven Central metadata of apim-client-sdk (2026-09-27): latest version still 3.0.6

### Current API Management Capability Activation

`capabilities.api_gateway` · checked 2026-09-27

- **Finding:** Activated with the API Management capability in the UI; no API.
- **Next step:** None expected.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)
  - Business Accelerator Hub: the API artifacts of 173 Integration Suite, API Management, Edge, Graph and BTP packages (308 APIs) searched for API management, API artifact, Integration Cell, runtime profile, virtual host, MCP, API deployment, API policy, reusable API, proxy, transport and gateway (2026-09-27); the package list was unchanged since 2026-09-26
  - Maven Central metadata of apim-client-sdk (2026-09-27): latest version still 3.0.6

### API Management Capability Activation

`capabilities.api_management` · checked 2026-09-26

- **Finding:** Activated under Manage Capabilities; no API.
- **Next step:** None expected.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)

### Cloud Integration Capability Activation

`capabilities.cloud_integration` · checked 2026-09-26

- **Finding:** Activated in the subscription wizard; no API.
- **Next step:** None expected.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)

### Edge Integration Cell Capability Activation

`capabilities.edge_integration_cell` · checked 2026-09-26

- **Finding:** Activated in the UI and through Edge Lifecycle Management; no API.
- **Next step:** None expected.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)

### Integration Cell Capability Activation

`capabilities.integration_cell` · checked 2026-09-27

- **Finding:** Activated under Settings > Runtime; no API.
- **Next step:** None expected.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)
  - Business Accelerator Hub: the API artifacts of 173 Integration Suite, API Management, Edge, Graph and BTP packages (308 APIs) searched for API management, API artifact, Integration Cell, runtime profile, virtual host, MCP, API deployment, API policy, reusable API, proxy, transport and gateway (2026-09-27); the package list was unchanged since 2026-09-26
  - Maven Central metadata of apim-client-sdk (2026-09-27): latest version still 3.0.6

### Edge Integration Cell Registration

`edge_integration_cell.registration` · checked 2026-09-26

- **Finding:** Edge nodes are added through Edge Lifecycle Management and a local bridge executable.
- **Next step:** None expected; a registration API from SAP would reopen it.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)

### Integration Advisor Design-Time Content

`integration_advisor.design_time_content` · checked 2026-09-26

- **Finding:** No API page in the documentation; the Hub has only EDI integration templates (modified 2026-04-02).
- **Next step:** An Integration Advisor API from SAP.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)

### Integration Cell Runtime

`integration_cell.runtime` · checked 2026-09-27

- **Finding:** Activation, status and configuration are UI-only.
- **Next step:** An Integration Cell API.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)
  - Business Accelerator Hub: the API artifacts of 173 Integration Suite, API Management, Edge, Graph and BTP packages (308 APIs) searched for API management, API artifact, Integration Cell, runtime profile, virtual host, MCP, API deployment, API policy, reusable API, proxy, transport and gateway (2026-09-27); the package list was unchanged since 2026-09-26
  - Maven Central metadata of apim-client-sdk (2026-09-27): latest version still 3.0.6

### Integration Cell Virtual Host

`integration_cell.virtual_host` · checked 2026-09-27

- **Finding:** Managed in Monitor > Integrations and APIs > Virtual Host only.
- **Next step:** A virtual host API for Integration Cell.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)
  - Business Accelerator Hub: the API artifacts of 173 Integration Suite, API Management, Edge, Graph and BTP packages (308 APIs) searched for API management, API artifact, Integration Cell, runtime profile, virtual host, MCP, API deployment, API policy, reusable API, proxy, transport and gateway (2026-09-27); the package list was unchanged since 2026-09-26
  - Maven Central metadata of apim-client-sdk (2026-09-27): latest version still 3.0.6

### Migration Assessment Source System

`migration_assessment.source_system` · checked 2026-09-26

- **Finding:** Migration Assessment consumes APIs; it exposes none.
- **Next step:** A Migration Assessment API from SAP.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)

### OData Provisioning

`odata_provisioning` · checked 2026-09-26

- **Finding:** UI-only configuration; no OData Provisioning package on the Hub.
- **Next step:** A management API from SAP.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)

### Certificate-User Mapping

`security.certificate_user_mapping` · checked 2026-09-26

- **Finding:** Documented only for the Neo environment.
- **Next step:** A Cloud Foundry API from SAP.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)

### Known Hosts (SSH)

`security.known_hosts` · checked 2026-09-26

- **Finding:** UI-only; no entity in $metadata and none in the Security Content resource table.
- **Next step:** A Known Hosts entity in $metadata (the discovery test fails on any new entity set).
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)

### OAuth2 Password Credentials

`security.oauth2_password_credential` · checked 2026-09-26

- **Finding:** UI-only; no entity set in $metadata. It may be stored as a UserCredentials kind.
- **Next step:** tenant-probe -GapTests lists the UserCredentials kinds of a tenant that has such an artifact; a documented kind would make it a user_credential variant.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)

### OAuth2 SAML Bearer Assertion

`security.oauth2_saml_bearer` · checked 2026-09-26

- **Finding:** UI-only; no entity set in $metadata.
- **Next step:** Same as OAuth2 password credentials.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)

### Security Material Where-Used

`security.where_used` · checked 2026-09-26

- **Finding:** UI-only; no entity or function import in $metadata.
- **Next step:** A where-used entity in $metadata; it would become a data source.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)

### Trading Partner Management Agreement

`trading_partner_management.agreement` · checked 2026-09-26

- **Finding:** UI-only, like company profiles.
- **Next step:** A TPM configuration API.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)

### Trading Partner Management Agreement Template

`trading_partner_management.agreement_template` · checked 2026-09-26

- **Finding:** UI-only, like company profiles.
- **Next step:** A TPM configuration API.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)

### Trading Partner Management Company Profile

`trading_partner_management.company_profile` · checked 2026-09-26

- **Finding:** UI-only with a JSON download; the B2B Scenarios API covers monitoring only.
- **Next step:** A TPM configuration API.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Hub package CloudIntegrationAPI (Integration Content, Security Content, Partner Directory, Message Stores, MPL, Log Files, B2B Scenarios; modified 2026-08-07)
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)

### Trading Partner Management Partner Profile

`trading_partner_management.partner_profile` · checked 2026-09-26

- **Finding:** UI-only, like company profiles.
- **Next step:** A TPM configuration API.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)

## ❌ Out of scope

Runtime data, workflows or imperative actions, not desired configuration.

### Data Store

`cloud_integration.data_store` · checked 2026-09-26

- **Finding:** Only a monitoring GET is documented; data stores come into existence through deployed flows.
- **Next step:** None; out of scope.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)

### Data Store Entry

`cloud_integration.data_store_entry` · checked 2026-09-26

- **Finding:** Only GETs are documented; entries are business messages.
- **Next step:** None; out of scope.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)

### Message Processing Logs

`cloud_integration.message_processing_logs` · checked 2026-09-26

- **Finding:** Public and documented, but runtime monitoring data.
- **Next step:** None; out of scope by the provider's scope rule.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Hub package CloudIntegrationAPI (Integration Content, Security Content, Partner Directory, Message Stores, MPL, Log Files, B2B Scenarios; modified 2026-08-07)

### Message Store Entries / JMS Resources

`cloud_integration.message_stores` · checked 2026-09-26

- **Finding:** Public and documented, but runtime messages and queues; the operations are support actions.
- **Next step:** None; out of scope.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Hub package CloudIntegrationAPI (Integration Content, Security Content, Partner Directory, Message Stores, MPL, Log Files, B2B Scenarios; modified 2026-08-07)

### Variable

`cloud_integration.variable` · checked 2026-09-26

- **Finding:** Only GET Variables(...)/$value is documented; variables are written by deployed flows.
- **Next step:** None; out of scope.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)

### Edge Integration Cell Local API Access

`edge_integration_cell.local_api` · checked 2026-09-26

- **Finding:** The local OData APIs are public (Hub package created 2026-04-30); they cover jobs, components, message stores, MPL and, new in the package, the Partner Directory. Monitoring stays out of scope, and the Partner Directory of an Edge Integration Cell is reachable through /location/<id>/api/v1 from the cloud.
- **Next step:** None for monitoring; the Partner Directory goes through edge_integration_cell.deployment_target.
- **Sources:**
  - Hub package sap-int-eic-eic-operations (Jobs, Components, Partner Directory, Message Stores, MPL; created 2026-04-30, modified 2026-09-21)
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)

### Edge Integration Cell Runtime Operations

`edge_integration_cell.runtime` · checked 2026-09-26

- **Finding:** Runtime parameters configure Kubernetes-level resources; components can be restarted, an imperative action.
- **Next step:** None; out of scope.
- **Sources:**
  - Hub package sap-int-eic-eic-operations (Jobs, Components, Partner Directory, Message Stores, MPL; created 2026-04-30, modified 2026-09-21)
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)

### Integration Advisor Runtime Artifact Injection

`integration_advisor.runtime_artifact_injection` · checked 2026-09-26

- **Finding:** A UI wizard and a one-shot action.
- **Next step:** None; out of scope.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)

### Integration Assessment Requests and Assessment Workflow

`integration_assessment.assessment_workflow` · checked 2026-09-26

- **Finding:** A request status machine; workflow state.
- **Next step:** None; out of scope.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)

### Migration Assessment Extraction and Scenario Evaluation

`migration_assessment.extraction_and_evaluation` · checked 2026-09-26

- **Finding:** Action-triggered workflow and reports.
- **Next step:** None; out of scope.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)

### Open Connectors

`open_connectors` · checked 2026-09-26

- **Finding:** A catalog of third-party connectors with connector-specific APIs.
- **Next step:** None; out of scope.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - earlier catalog research, recorded in the feature's limitations

### SSH Key

`security.ssh_key` · checked 2026-09-26

- **Finding:** SSHKeyGenerationRequests and SSHKeyResources exist in $metadata without documented requests; key pairs cover SSH through the OpenSSH export.
- **Next step:** None needed; revisit only if SAP documents SSH keys as a separate artifact.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
  - Cloud Integration $metadata snapshot testdata/api-metadata/cloud-integration.json (2026-09-26)

### Trading Partner Management Partner Directory Generation

`trading_partner_management.partner_directory_generation` · checked 2026-09-26

- **Finding:** A side effect of activating an agreement.
- **Next step:** None; out of scope.
- **Sources:**
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)

## ↗️ Separate provider

Belongs to a separate, independently versioned Terraform provider.

### Developer Hub

`developer_hub` · checked 2026-09-26

- **Finding:** Public APIs exist (APIMgmt package: Product, Application, Discovery, Registering Users, External Governance); a separate provider by design.
- **Next step:** None for this provider.
- **Sources:**
  - Hub package APIMgmt (API Portal, Developer Hub, Metering, Billing, Graph Configuration APIs; modified 2026-09-24)

### Event Mesh

`event_mesh` · checked 2026-09-26

- **Finding:** A general BTP messaging service with its own management APIs (Hub package modified 2026-09-18); a separate provider by design.
- **Next step:** None for this provider.
- **Sources:**
  - Business Accelerator Hub catalog.svc, full package list (1,971 packages, read 2026-09-26)
  - SAP Help mirror SAP-docs/btp-integration-suite, commit 33f3395 (2026-09-18)
