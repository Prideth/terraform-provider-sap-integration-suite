# Feature Support

Generated from `internal/features/catalog.go` by `go run ./cmd/gendocs`. Do not edit by hand — regenerate it instead, and see `CONTRIBUTING.md` for when this is required.

This is what this provider version implements, derived from the same catalog the provider binary itself exposes through `sapintegrationsuite_provider_features` (whose `provider_version` attribute reports the exact version at apply time). It is queryable directly from Terraform, with no SAP tenant connection required:

```hcl
data "sapintegrationsuite_provider_features" "all" {}

output "supported_features" {
  value = [
    for f in data.sapintegrationsuite_provider_features.all.features :
    f.key
    if f.support_status == "supported"
  ]
}

data "sapintegrationsuite_provider_feature" "one" {
  key = "cloud_integration.value_mapping"
}
```

See README.md's "Feature Support" section for a compact, high-level dashboard generated from this same catalog (`go run ./cmd/gendocs -readme`); this document is the detailed per-operation matrix.

## Status legend

`support_status` in `sapintegrationsuite_provider_features` and `sapintegrationsuite_provider_feature` carries the value in the second column.

| Status | Value | Meaning |
|---|---|---|
| ✅ Supported | `supported` | Every operation the provider claims is implemented and backed by SAP's documentation, an official API specification or SAP's tooling. |
| 🟡 Partial | `partial` | Usable, but the provider deliberately leaves out part of the lifecycle; the limitations say which part and why. |
| 👁️ Read-only | `read_only` | Available as a data source only; the provider never creates, changes or deletes it. |
| 🧪 Experimental | `experimental` | Implementation exists, but its lifecycle has not yet been sufficiently validated against a real SAP tenant. |
| 🧭 Unofficial | `unofficial` | Implementation has been validated, but relies on an API contract that SAP does not fully publish or officially document. |
| 🔬 Research required | `research_required` | The capability has been identified, but its public API coverage, lifecycle semantics, or suitability for Terraform still requires further investigation. |
| ❌ Unsupported | `unsupported` | Not implemented, and investigated far enough to decide: there is no usable public API or safe Terraform lifecycle, or the feature is deliberately out of scope. |
| ↗️ Separate provider | `separate_provider` | Belongs to a separate, independently versioned Terraform provider. |

Three of them are easy to confuse. 🔬 Research required: the capability is known, but it has not been investigated far enough to decide whether and how to model it in Terraform, and there is no implementation. 🧪 Experimental: an implementation exists, but its lifecycle has not yet passed a test on a real SAP tenant; it needs `enable_experimental = true`. 🧭 Unofficial: the implementation passed on a tenant, but the SAP API contract behind it is not fully public or officially documented, so SAP may change it without notice; it needs `enable_unofficial = true`.

## Relationship to the other capability documents

This document, `docs/api-capability-matrix.md`, `docs/provisioning-capability-matrix.md`, and a possible future tenant-capability data source answer three related but distinct questions:

| Document | Answers |
|---|---|
| `docs/api-capability-matrix.md` / `docs/provisioning-capability-matrix.md` | What SAP's public APIs expose, independent of this provider |
| `docs/feature-support.md` (this document) / `sapintegrationsuite_provider_features` | What *this provider version* implements — static provider metadata, no SAP tenant required |
| A possible future `sapintegrationsuite_tenant_capabilities` data source (not implemented) | Which Integration Suite capabilities are *active in a specific SAP tenant* — would require SAP credentials and a reliable public discovery API, neither of which this provider assumes here |

Do not conflate these: a feature can be fully supported by this provider and still be unusable in a given tenant because the underlying SAP capability was never activated there, and vice versa a capability can be active in every tenant while this provider still does not implement a resource for it.

## All features

| Feature | Domain | Status | Contract source | Public API | Create | Read | Update | Delete | Import | Deploy | Terraform |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `api_composition.business_data_graph` | api_composition | supported | sap_documentation (some operations unofficial) | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource + Data Source |
| `api_gateway.api_artifact` | api_gateway | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `api_gateway.api_artifact_deployment` | api_gateway | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `api_gateway.api_policy` | api_gateway | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `api_gateway.mcp_server` | api_gateway | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `api_gateway.reusable_api_artifact` | api_gateway | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `api_gateway.runtime_profile` | api_gateway | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `api_management.classic.api_product` | api_management_classic | supported | sap_documentation | Yes | Yes | Yes | — | Yes | Yes | — | Resource + Data Source |
| `api_management.classic.api_provider` | api_management_classic | partial (unsafe_terraform_lifecycle) | sap_documentation | Yes | Yes | Yes | — | Yes | Yes | — | Resource + Data Source |
| `api_management.classic.api_proxy` | api_management_classic | research_required (public_api_incomplete) | — | Yes | Yes | Yes | — | Yes | Yes | — | — |
| `api_management.classic.api_proxy_deployment` | api_management_classic | unsupported (public_api_incomplete) | — | Yes | — | — | — | — | — | — | — |
| `api_management.classic.application` | api_management_classic | unsupported (public_api_incomplete) | — | Yes | — | — | — | — | — | — | — |
| `api_management.classic.cache_resource` | api_management_classic | unsupported (public_api_incomplete) | — | Yes | — | — | — | — | — | — | — |
| `api_management.classic.certificate_store` | api_management_classic | unsupported (public_api_incomplete) | — | Yes | — | — | — | — | — | — | — |
| `api_management.classic.certificate_store_reference` | api_management_classic | supported | sap_documentation | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource + Data Source |
| `api_management.classic.environment_key_value_map` | api_management_classic | research_required (public_api_incomplete) | — | Yes | — | — | — | — | — | — | — |
| `api_management.classic.key_value_map` | api_management_classic | partial (unsafe_terraform_lifecycle) | sap_documentation | Yes | Yes | Yes | — | Yes | Yes | — | Resource + Data Source |
| `api_management.classic.policy` | api_management_classic | unsupported (public_api_incomplete) | — | Yes | — | — | — | — | — | — | — |
| `api_management.classic.policy_template` | api_management_classic | unsupported (public_api_incomplete) | — | Yes | — | — | — | — | — | — | — |
| `api_management.classic.product_access_control` | api_management_classic | unsupported (public_api_incomplete) | — | Yes | — | — | — | — | — | — | — |
| `api_management.classic.rate_plan` | api_management_classic | unsupported (public_api_incomplete) | — | Yes | — | — | — | — | — | — | — |
| `api_management.classic.virtual_host` | api_management_classic | partial (public_api_incomplete) | sap_documentation | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource |
| `capabilities.api_gateway` | capability_provisioning | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `capabilities.api_management` | capability_provisioning | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `capabilities.cloud_integration` | capability_provisioning | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `capabilities.edge_integration_cell` | capability_provisioning | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `capabilities.integration_cell` | capability_provisioning | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `cloud_integration.archiving` | cloud_integration | unsupported (unsafe_terraform_lifecycle) | — | Yes | — | — | — | — | — | — | — |
| `cloud_integration.custom_tag_configuration` | cloud_integration | partial (unsafe_terraform_lifecycle) | sap_documentation | Yes | Yes | Yes | Yes | — | Yes | — | Resource + Data Source |
| `cloud_integration.data_store` | cloud_integration | unsupported (out_of_scope) | — | Yes | — | — | — | — | — | — | — |
| `cloud_integration.data_store_entry` | cloud_integration | unsupported (out_of_scope) | — | Yes | — | — | — | — | — | — | — |
| `cloud_integration.data_type` | cloud_integration | unofficial (public_api_incomplete) | metadata_only | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource |
| `cloud_integration.design_time_versioning` | cloud_integration | partial (not_implemented) | sap_documentation (some operations unofficial) | Yes | Yes | — | Yes | — | — | — | Resource |
| `cloud_integration.integration_adapter` | cloud_integration | partial (public_api_incomplete) | sap_documentation | Yes | Yes | Yes | — | Yes | Yes | — | Resource + Data Source |
| `cloud_integration.integration_adapter_deployment` | cloud_integration | partial (public_api_incomplete) | sap_documentation | Yes | Yes | Yes | — | Yes | — | Yes | Resource |
| `cloud_integration.integration_flow` | cloud_integration | supported | sap_documentation | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource |
| `cloud_integration.integration_flow_configuration` | cloud_integration | supported | sap_documentation | Yes | Yes | Yes | Yes | — | Yes | — | Resource |
| `cloud_integration.integration_flow_deployment` | cloud_integration | supported | sap_documentation | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Resource |
| `cloud_integration.integration_package` | cloud_integration | supported | sap_documentation | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource + Data Source |
| `cloud_integration.message_mapping` | cloud_integration | supported | sap_documentation (some operations unofficial) | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource + Data Source |
| `cloud_integration.message_mapping_deployment` | cloud_integration | supported | sap_documentation | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Resource |
| `cloud_integration.message_processing_logs` | cloud_integration | unsupported (out_of_scope) | — | Yes | — | — | — | — | — | — | — |
| `cloud_integration.message_stores` | cloud_integration | unsupported (out_of_scope) | — | Yes | — | — | — | — | — | — | — |
| `cloud_integration.message_type` | cloud_integration | unofficial (public_api_incomplete) | metadata_only | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource |
| `cloud_integration.number_range` | cloud_integration | supported | sap_documentation (some operations unofficial) | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource |
| `cloud_integration.script_collection` | cloud_integration | supported | sap_documentation (some operations unofficial) | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource + Data Source |
| `cloud_integration.script_collection_deployment` | cloud_integration | supported | sap_documentation | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Resource |
| `cloud_integration.service_endpoints` | cloud_integration | read_only (unsafe_terraform_lifecycle) | sap_documentation | Yes | — | Yes | — | — | — | — | Data Source |
| `cloud_integration.service_interface` | cloud_integration | unofficial (public_api_incomplete) | metadata_only | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource |
| `cloud_integration.value_mapping` | cloud_integration | partial (unsafe_terraform_lifecycle) | sap_documentation (some operations unofficial) | Yes | Yes | Yes | — | Yes | Yes | — | Resource + Data Source |
| `cloud_integration.value_mapping_deployment` | cloud_integration | supported | sap_documentation | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Resource |
| `cloud_integration.value_mapping_entry` | cloud_integration | unsupported (unsafe_terraform_lifecycle) | — | Yes | — | — | — | — | — | — | — |
| `cloud_integration.variable` | cloud_integration | unsupported (out_of_scope) | — | Yes | — | — | — | — | — | — | — |
| `data_space_integration` | other_capability | research_required (research_required) | — | Yes | — | — | — | — | — | — | — |
| `developer_hub` | other_capability | separate_provider (out_of_scope) | — | Yes | — | — | — | — | — | — | — |
| `edge_integration_cell.access_policy_replication` | edge_integration_cell | unofficial (public_api_incomplete) | metadata_only | Yes | — | Yes | — | — | — | — | Data Source |
| `edge_integration_cell.deployment_target` | edge_integration_cell | research_required (public_api_incomplete) | — | Yes | — | — | — | — | — | — | — |
| `edge_integration_cell.local_api` | edge_integration_cell | unsupported (out_of_scope) | — | Yes | — | — | — | — | — | — | — |
| `edge_integration_cell.registration` | edge_integration_cell | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `edge_integration_cell.runtime` | edge_integration_cell | unsupported (out_of_scope) | — | Yes | — | — | — | — | — | — | — |
| `event_mesh` | other_capability | separate_provider (out_of_scope) | — | Yes | — | — | — | — | — | — | — |
| `integration_advisor.design_time_content` | other_capability | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `integration_advisor.runtime_artifact_injection` | other_capability | unsupported (out_of_scope) | — | No | — | — | — | — | — | — | — |
| `integration_assessment.assessment_workflow` | integration_assessment | unsupported (out_of_scope) | — | Yes | — | — | — | — | — | — | — |
| `integration_assessment.landscape_configuration` | integration_assessment | unofficial (public_api_incomplete) | metadata_only | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource + Data Source |
| `integration_assessment.master_data` | integration_assessment | unofficial (public_api_incomplete) | metadata_only | Yes | — | Yes | — | — | — | — | Data Source |
| `integration_cell.runtime` | integration_cell | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `integration_cell.virtual_host` | integration_cell | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `migration_assessment.extraction_and_evaluation` | other_capability | unsupported (out_of_scope) | — | No | — | — | — | — | — | — | — |
| `migration_assessment.source_system` | other_capability | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `odata_provisioning` | other_capability | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `open_connectors` | other_capability | unsupported (out_of_scope) | — | Yes | — | — | — | — | — | — | — |
| `partner_directory.alternative_partner` | partner_directory | supported | sap_documentation | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource + Data Source |
| `partner_directory.authorized_user` | partner_directory | supported | sap_documentation | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource + Data Source |
| `partner_directory.binary_parameter` | partner_directory | supported | sap_documentation | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource + Data Source |
| `partner_directory.partner` | partner_directory | read_only (unsafe_terraform_lifecycle) | sap_documentation | Yes | — | Yes | — | — | — | — | Data Source |
| `partner_directory.string_parameter` | partner_directory | supported | sap_documentation | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource + Data Source |
| `partner_directory.user_credential_parameter` | partner_directory | supported | sap_documentation | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource |
| `security.access_policy` | security | supported | sap_tooling (some operations unofficial) | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource + Data Source |
| `security.access_policy_reference` | security | supported | sap_tooling (some operations unofficial) | Yes | Yes | Yes | — | Yes | Yes | — | Resource + Data Source |
| `security.certificate` | security | supported | sap_documentation | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource |
| `security.certificate_chain` | security | unofficial (public_api_incomplete) | metadata_only | Yes | Yes | Yes | Yes | — | Yes | — | Resource |
| `security.certificate_user_mapping` | security | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `security.key_pair` | security | partial (unsafe_terraform_lifecycle) | sap_documentation (some operations unofficial) | Yes | Yes | Yes | — | Yes | Yes | — | Resource |
| `security.keystore_entry` | security | read_only (unsafe_terraform_lifecycle) | sap_documentation | Yes | — | — | — | — | — | — | Data Source |
| `security.known_hosts` | security | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `security.oauth2_client_credential` | security | partial (public_api_incomplete) | sap_documentation (some operations unofficial) | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource + Data Source |
| `security.oauth2_password_credential` | security | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `security.oauth2_saml_bearer` | security | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `security.pgp_keyring` | security | unofficial (public_api_incomplete) | metadata_only | Yes | Yes | Yes | — | Yes | Yes | — | Resource |
| `security.secure_parameter` | security | unofficial (public_api_incomplete) | metadata_only | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource |
| `security.ssh_key` | security | unsupported (out_of_scope) | — | Yes | — | — | — | — | — | — | — |
| `security.user_credential` | security | partial (public_api_incomplete) | sap_documentation | Yes | Yes | Yes | Yes | Yes | Yes | — | Resource + Data Source |
| `security.where_used` | security | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `trading_partner_management.agreement` | other_capability | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `trading_partner_management.agreement_template` | other_capability | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `trading_partner_management.company_profile` | other_capability | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |
| `trading_partner_management.partner_directory_generation` | other_capability | unsupported (out_of_scope) | — | No | — | — | — | — | — | — | — |
| `trading_partner_management.partner_profile` | other_capability | unsupported (no_public_api) | — | No | — | — | — | — | — | — | — |

## Contract sources

Every implemented feature says where its contract comes from, because that decides how far it can be trusted across SAP releases:

| Source | Meaning | Highest status |
|---|---|---|
| `sap_documentation` | SAP Help describes the operations (example requests, the API's list of resources) | supported |
| `api_specification` | The official API specification on the SAP Business Accelerator Hub | supported |
| `sap_tooling` | SAP's own SDK or tooling sends these requests (API Management Client SDK, CI/CD actions, Project Piper) | supported |
| `metadata_only` | Only the service's `$metadata` and tests on a tenant; nothing official | unofficial once verified, experimental before |

**Unofficial** means that it works and was verified on a tenant, but SAP has not documented it, so SAP may change it without notice.

Resources and data sources whose status is `experimental` or `unofficial` are switched off by default, so nobody uses them by accident. A configuration that uses one fails until the provider block sets the matching switch:

```hcl
provider "sapintegrationsuite" {
  enable_experimental = true # or SAP_INTEGRATION_SUITE_ENABLE_EXPERIMENTAL=true
  enable_unofficial   = true # or SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL=true
}
```

| Type | Status | Switch |
|---|---|---|
| `sapintegrationsuite_access_policy_runtime_assignments` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_data_type` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_fault_message_type` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_application_instance` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_application` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_deployment_model` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_domain_determination` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_domain` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_integration_pattern` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_key_characteristic_group` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_key_characteristic_value` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_recommendation_degree` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_style` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_technology_domain` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_technology_instance` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_technology_key_characteristic` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_technology_style` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_technology` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_use_case_pattern` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_integration_assessment_vendor` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_key_pair_certificate_chain` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_message_type` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_pgp_public_key` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_pgp_secret_key` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_secure_parameter` | unofficial | `enable_unofficial` |
| `sapintegrationsuite_service_interface` | unofficial | `enable_unofficial` |

Individual operations of an otherwise documented feature can be unofficial too. They are switched off by `enable_unofficial` as well, but only the operation: the resource itself stays usable with its documented operations. A plan that needs one of them fails with an error that names the operation, for example an in-place update of a message mapping's content. Number ranges without the switch work with what SAP documents: a refresh keeps the state instead of reading the number range, an update has to change `current_value_wo_version` so that it can send the counter, and delete and import are refused.

| Feature | Implemented operation that SAP does not document |
|---|---|
| `api_composition.business_data_graph` | delete (DELETE on the graph) |
| `api_composition.business_data_graph` | update body (PATCH with the writable properties) |
| `api_composition.business_data_graph` | settings named only in the $metadata (description, odata_containment, locating_policy.description, key_mapping cues) |
| `cloud_integration.design_time_versioning` | save_as_version on script collections (ScriptCollectionDesigntimeArtifactSaveAsVersion is only in $metadata; integration flows and message mappings are documented) |
| `cloud_integration.message_mapping` | update of the content (PUT; SAP documents read, create and delete) |
| `cloud_integration.number_range` | read by name |
| `cloud_integration.number_range` | delete |
| `cloud_integration.number_range` | import |
| `cloud_integration.script_collection` | update of the content (PUT; SAP documents create, upload of resources and deploy) |
| `cloud_integration.value_mapping` | save_as_version (ValueMappingDesigntimeArtifactSaveAsVersion is only in $metadata; SAP Help lists read, download, create, upload, deploy and delete) |
| `security.access_policy` | description update (PATCH, verified on a tenant) |
| `security.access_policy_reference` | create a reference to an artifact type SAP does not document for access policies (eight types SAP's API lists, verified on a tenant) |
| `security.key_pair` | read of the certificate signing request (KeystoreEntries('<hexalias>')/SigningRequest/$value is only in $metadata) |
| `security.oauth2_client_credential` | custom_parameters, created by a deep insert and read through the CustomParameters navigation (only in $metadata) |

## Unsupported and partially supported features

Grouped by why, not just that. A feature can be `partial` and reachable via one of these reasons too — see docs/feature-support.md's per-feature `Limitations` (the `limitations` attribute in Terraform) for exactly what is and is not covered.

### Public API exists but provider implementation is pending

- **`cloud_integration.design_time_versioning`** — Saving a design-time artifact under an explicit version number (for example 1.0.3) instead of working only on the active draft. (partial support already implemented — see Limitations below)
  - save_as_version calls <Artifact>SaveAsVersion?Id=''&SaveAsVersion='' after the content upload, as SAP Help documents for IntegrationDesigntimeArtifactSaveAsVersion; the tenant $metadata confirms the same function import for message mappings and script collections. A new version is saved only when save_as_version changes.
  - Not yet available for value mappings: that resource replaces the artifact on every change, and a version bump must not recreate it. Data types, message types, fault message types and service interfaces have the function import too but no resource.
  - SAP does not document what happens when the version already exists or is lower than the current one; SAP's error is passed through unchanged.

### Public API details are not fully confirmed

- **`api_management.classic.api_proxy`** — A classic API Management API proxy definition: the ZIP-bundled design-time content (proxy endpoint, target endpoint, policies, resources) deployed as a callable API. (🔬 Research required)
  - An implementation exists in the repository but is not part of the provider yet: the API portal answered its import, the request SAP's Client SDK 3.0.6 sends, with 400 APIPROXY_ZIP_ERROR ("Verify the directory structure inside the zip"), even for a bundle the same API portal had exported (tenant test, 2026-09-27). It is added once the request documented in the official Transport API specification (APIPortal_Transport_CF) works.
  - Experimental until the acceptance test (TestAccAPIProxy_sample) passes on a tenant. Upload follows SAP's API Management Client SDK 3.0.6 (published 2026-09-24) byte for byte: POST /apiportal/api/1.0/Transport.svc/APIProxies with the SDK's own query and the raw ZIP as application/octet-stream. The Business Accelerator Hub lists "API Portal - Transport (CF)" ("Export and Import API Proxy via zip bundle") as the official API; its specification needs an SAP login.
  - The APIProxy entity is confirmed by the Management.svc $metadata (key name; provider_name, state, status_code, version, isPublished and navigations to endpoints, policies, resources and the API provider), and its GET returned 200 on a tenant. The read and delete paths Management.svc/APIProxies('<name>') are the ones SAP's documentation and worked examples use.
  - Replace-only: nothing public says whether importing a changed bundle over an existing proxy replaces it cleanly, so a different content_hash deletes the proxy and imports it again. SAP's documentation says an imported proxy is deployed by default, so there is no separate deployment step (see api_management.classic.api_proxy_deployment).
  - API providers a bundle's target endpoint references must already exist on the tenant: SAP's sample repository documents that the import fails otherwise. Bundles with a target URL (provider_id NONE) have no such dependency.
  - The SDK's JSON create path (/api/1.0/apis/ with isFromCli) is an internal endpoint and not used.
- **`api_management.classic.api_proxy_deployment`** — The runtime deployment state of a classic API Proxy, potentially independent of its design-time content.
  - SAP's own documentation states that a proxy transported or exported, individually or as part of a product, "by default gets imported to the target in the deployed state", so an import deploys the proxy, and exposes the resulting state. No documented API deploys or undeploys an existing proxy on its own, so there is no separate deployment resource.
- **`api_management.classic.application`** — Consumer applications subscribed to API products, with their generated application key and secret, and the developers who own them.
  - Management.svc $metadata: Applications (key id; app_key, app_secret, callbackurl, status_code, validity, subscribedRatePlan, navigation to apiProducts and developer) and Developers (key id; emailId, firstName, lastName, country). The Hub describes the CF Applications API as "view all available applications"; creating applications through the API is described only for the Developer API. The generated app_secret would have to be kept out of state or treated as sensitive.
- **`api_management.classic.cache_resource`** — A named cache used by response cache and lookup cache policies.
  - Management.svc $metadata: CacheResources (key name; sizes, compression, overflow and expiry settings). The Hub documents create, view, update and delete only for the Neo version of this API; no Cloud Foundry version is listed.
- **`api_management.classic.certificate_store`** — Key stores and trust stores of the API Portal and the certificates in them, used for TLS towards backends and for virtual hosts.
  - Management.svc $metadata: CertificateStores (key name; storeType) and Certificates (key name and storeName; content as Edm.Binary, format, password, validity and issuer fields). The Business Accelerator Hub describes the KeyStore and TrustStore APIs as "create and view"; update and delete are not described, so a Terraform lifecycle cannot be confirmed yet. certificate_store_reference covers pointing at an existing store.
- **`api_management.classic.environment_key_value_map`** — Key value maps shared across API proxies (KeyMapEntries), as opposed to the generic, scoped key value maps sapintegrationsuite_api_key_value_map manages. (🔬 Research required)
  - Management.svc $metadata: KeyMapEntries (key name; encrypted, scope) with KeyMapEntryValues (key map_name and name; value). The Hub lists "Key Value Maps (CF)" ("create key value pairs across the API proxies") next to the generic key value maps this provider implements. How the two relate, and which one SAP recommends, is not documented.
- **`api_management.classic.policy`** — An individual mediation policy (for example VerifyAPIKey, Quota, AssignMessage) attached to a classic API Proxy's proxy or target endpoint flow.
  - Confirmed to be XML content embedded inside the API Proxy ZIP bundle (a <policies> element in the proxy's root XML, referencing named files under a Policy/ folder), not an independently addressable OData entity with its own Create/Read/Update/Delete — so individual policies are not a separate resource candidate; they are managed as part of a proxy's bundle content.
- **`api_management.classic.policy_template`** — Reusable policy templates that can be applied to API proxies.
  - Management.svc $metadata: PolicyTemplateContainers (key name; proxy and target endpoint XML, navigations to policies and file resources). The token scopes include import, export and apply for policy templates, but the Hub lists no policy template API and SAP Help describes only the UI.
- **`api_management.classic.product_access_control`** — Rules that grant user groups access to API products in the Developer Hub.
  - Management.svc $metadata: ACLProductLinkages (key ruleId; entityId, entityType, permissionSet, operation, isPublished). The Hub describes the Access Control Service (CF) as "view and create rules"; update and delete are not described.
- **`api_management.classic.rate_plan`** — Monetization rate plans attached to API products.
  - Management.svc $metadata: RatePlans (key id; rate, currency, frequency, type, validity, isActive, isPublished). The Hub lists only billing and metering APIs for monetization, no rate plan API.
- **`api_management.classic.virtual_host`** — A virtual host of the Classic API Portal: the default-domain alias or custom domain (with one-way or mutual TLS) under which API proxies are exposed. (partial support already implemented — see Limitations below)
  - Create, update and delete are documented in SAP Help (Configuring a Default Domain / Custom Domain / Mutual TLS for a Virtual Host): POST /apiportal/operations/1.0/Configuration.svc/VirtualHostRequests with operation CREATE, UPDATE or DELETE. The resource sends the default domain bodies: accountId (the subaccount subdomain), virtualHostUrl (the alias), isDefaultVirtualHostRequest and, for update and delete, virtualHostId; for mutual TLS also isClientAuthEnabled and trustStore (a truststore name or ref://<certificate store reference>), as SAP Help documents them in Configuring Mutual TLS for Default Domain Virtual Host.
  - Tenant probe (2026-10-06) with a key of the role APIManagement.SelfService.Administrator: create, update and delete answered 201 and took effect at once (allocationStatus COMPLETE). virtualHostId is the id of Management.svc/VirtualHosts; SAP returns the full host name <alias>.<tenant domain>, not <alias>.sapdefaultdomain as in SAP Help. A duplicate alias, more than 63 characters and characters other than letters, digits and hyphens are refused with 400; an unknown or deleted ID with 400 VHR_NO_COMPLETED_RECORD_FOUND.
  - That key reads the list Management.svc/VirtualHosts but not a single entity (403), so the resource looks a host up in the list. Configuration.svc does not serve its $metadata to either key (403); the request fields come from SAP Help and the tenant's answers.
  - TestAccAPIManagementVirtualHost_basic passed on a tenant (2026-10-06): create, rename in place, import by alias and delete of a default domain host.
  - Mutual TLS on the default domain (client_auth_enabled, trust_store): a probe (2026-10-07) found the host list showing isClientAuthEnabled and the truststore as sent, ref:// included; an update without the TLS fields kept mutual TLS; isClientAuthEnabled false switched it off and cleared the truststore (SAP Help shows no body for that). A missing truststore is refused with 400 VHR_VIRTUALHOST_TRUST_STORE_MISSING, an unknown one with 400 VIRTUAL_HOST_CREATE_ERROR. TestAccAPIManagementVirtualHost_mutualTLS passed: create, move to a certificate store reference, import, switch off. The truststore itself is created in the UI; no documented API creates it.
  - Custom domains (keyStoreName, keyStoreAlias, isForCustomDomain) are documented but not tested, and the resource refuses to manage hosts that use them. It never makes a host the default one. Deletion is refused by SAP while proxies (deployed, draft or in a revision) reference the host or while it is the default.
- **`cloud_integration.data_type`** — A reusable XSD data type artifact (simple or complex) used by message types and mappings. (🧭 Unofficial)
  - The tenant $metadata defines DataTypeDesigntimeArtifacts (Id and Version as key, PackageId, Name, Namespace, Description, IsSimpleType, ArtifactContent) and a DataTypeDesigntimeArtifactSaveAsVersion function import. SAP Help documents only the UI and lists no API resource or example request for data types.
  - Gap probes on a tenant (September and October 2026): a create needs the bundle as SAP stores it, with additionalAttributes.json and metainfo.prop next to the XSD; a package export's bundle lacks both and fails with 500 "map is null". With them, create, update (PUT with Name and ArtifactContent), SaveAsVersion and delete work, and an element added on create or update is kept. The provider builds that bundle from xsd, namespace and description. SAP takes the description and the namespace from the entity, not from the bundle, so the provider sends them in the create and update bodies. TestAccDataType_basic passed on a tenant (2026-10-03): create, in-place change of schema and description, import, SaveAsVersion.
  - Complex data types only: the bundle of a simple type (IsSimpleType) was not examined.
  - SAP stores the complex type under the data type's name and normalizes the schema, so the schema is not read back for drift detection; xsd is taken from the configuration.
- **`cloud_integration.integration_adapter`** — A custom Integration Adapter design-time artifact (a *.esa archive built with the SAP Adapter SDK), imported into a Cloud Integration package. Cloud Foundry environment only. (partial support already implemented — see Limitations below)
  - This provider's evidence base for this entity is thinner than for the sibling design-time artifact types it manages: SAP's own "Integration Adapter Example Requests, Cloud Foundry Environment" documentation shows only Delete (confirming the entity is keyed by Id alone, not the composite (Id, Version) key every other design-time artifact type in this API uses) and the Deploy action — no Create or Read example was found. The property names and the single Id key are confirmed by a tenant $metadata document, but no SAP example shows a Create request for this entity. See docs/guides/integration-adapters.md.
  - No in-place update: SAP documents that importing an ID that already exists on the tenant is rejected as an error, which is positive evidence against a working reimport-to-update flow, so every attribute is RequiresReplace rather than an unverified PUT/PATCH.
  - The type and application shown in SAP's import dialog cannot be set: the IntegrationAdapterDesigntimeArtifact entity has only Id, Version, PackageId, Name, ArtifactContent and Description (tenant $metadata). Earlier releases sent Type and Application anyway; both attributes were removed. description is exposed read-only.
  - Distinct from SAP Business Accelerator Hub prebundled adapters (a different import/auto-deploy lifecycle reached from inside the integration flow editor) and from Integration Suite capability activation — see docs/guides/integration-adapters.md for why these are not the same feature.
  - There is no sapintegrationsuite_integration_adapters collection data source: no confirmed list/filter contract for this entity set was found, unlike ServiceEndpoints' documented Name/Protocol filters.
- **`cloud_integration.integration_adapter_deployment`** — The runtime deployment state of a custom Integration Adapter, independent of its design-time content lifecycle. (partial support already implemented — see Limitations below)
  - Deploy is confirmed directly from SAP's own example request: POST DeployIntegrationAdapterDesigntimeArtifact?Id='...', singular "Artifact" (matching every sibling deploy action in this API), with no Version query parameter (unlike every sibling deploy action, consistent with Id being this entity's only confirmed key).
  - Runtime status polling and undeploy reuse the same shared IntegrationRuntimeArtifacts entity every other *_deployment resource in this provider polls/undeploys through, by analogy — this project could not independently confirm that a deployed custom adapter surfaces through that same shared entity as opposed to an adapter-specific status/undeploy mechanism (for example BuildAndDeployStatus). See docs/guides/integration-adapters.md.
  - Whether a fresh deployment's runtime status becomes visible immediately or after a build/deploy delay specific to adapters (as opposed to ordinary content deployment) was not confirmed.
- **`cloud_integration.message_type`** — A message type artifact that wraps a data type as a message root element, and the fault message type, which adds SAP's standard fault data. (🧭 Unofficial)
  - The tenant $metadata defines MessageTypeDesigntimeArtifacts (Id and Version as key, PackageId, Name, Namespace, Description, DataTypeUsed, ArtifactContent) and a SaveAsVersion function import; FaultMessageTypeDesigntimeArtifacts has the same shape for fault messages. SAP Help documents only the UI for both.
  - ESR probes on a tenant (2026-10-03): a create without content works, and SAP generates the schema from DataTypeUsed (an element of that data type; a fault message type adds ExchangeFaultData). An update changes Description and DataTypeUsed and regenerates the content, but SAP refuses to update Name, so changing the name replaces the artifact. TestAccMessageType_basic passed on a tenant for both (2026-10-03): create, in-place switch of the data type and the description, import, SaveAsVersion.
  - A read returns DataTypeUsed empty even when it took effect; the provider reads the data type from the generated bundle: dtUniqueIdinMT in additionalAttributes.json (the artifact ID), dtUsedinMT (the name) only when the ID is missing. They differ for SAP's standard data types such as ExchangeFaultData, whose IDs carry a hash suffix (package export, 2026-10-03). Whether a create accepts a standard data type by ID or by name is not probed.
  - SAP does not check references: a data type that a message type uses can be deleted.
- **`cloud_integration.service_interface`** — A service interface artifact describing operations and their request, response and fault message types. (🧭 Unofficial)
  - The tenant $metadata defines ServiceInterfaceDesigntimeArtifacts (Id and Version as key, PackageId, Name, Namespace, Description, ArtifactContent, Resources navigation) and a SaveAsVersion function import. SAP Help documents creating, editing and importing service interfaces from the Enterprise Services Repository only in the UI.
  - The operation model (src/main/resources/json/<name>.json in a bundle nested in the $value) follows two service interfaces SAP's editor created (package exports, 2026-10-03 and 2026-10-04): an asynchronous operation with request and fault message, and a synchronous one with request, response and fault message, each message named by ID, name, namespace, version and package; the manifest names them in Require-Capability.
  - Tenant probe (2026-10-04): a create without content works (SAP generates one operation without messages). An update needs Name next to ArtifactContent (without it, 500 "name is null") and the nested bundle as $value returns it (the interface's own bundle alone answers 400 "The bundle is not of type ServiceInterface"); SAP stores the uploaded asynchronous operation and drops messageDetails. TestAccServiceInterface_basic passed on a tenant (2026-10-04): create with an asynchronous operation, an in-place switch to a synchronous operation with a response plus a second operation, import, SaveAsVersion.
- **`edge_integration_cell.access_policy_replication`** — The runtimes (Cloud Integration runtime, Integration Cell, specific Edge Integration Cells) an Access Policy is replicated to, and the replication state of each. (🧭 Unofficial)
  - Readable, not writable: the tenant $metadata defines AccessPolicyRuntimeAssignments (Id, RuntimeLocationId, TransferStatus, TransferErrors, StatusUpdatedAt) as a navigation property of AccessPolicies, which the data source reads. Whether assignments can be created or deleted through the API is not documented, and $metadata carries no creatable/updatable flags, so choosing runtimes stays a UI step.
  - transfer_status is passed through as SAP returns it: the UI shows Fail, Success and Pending, but the API values are plain strings with no documented enumeration.
- **`edge_integration_cell.deployment_target`** — Addressing an Edge Integration Cell instead of the cloud runtime: deploying content to it and managing its security material, selected with runtime_location_id. (🔬 Research required)
  - Not supported yet. SAP Help's Integration Content, Security Content, Partner Directory, message processing log, message store and number range pages (mirror of 2026-09-18) document https://<host>/location/<runtime location id>/api/v1/<path> for calling the same APIs against an Edge Integration Cell, once for all operations and without per-operation examples, and SAP's own CI/CD tooling still called the path unpublished in May 2026. It has not been verified against a tenant with an Edge Integration Cell.
  - The deployment, credential, certificate, key pair and Partner Directory resources and their data sources carry an optional runtime_location_id that sends requests to that path, and import IDs accept a location:<id>/ prefix. TestAccEdgeIntegrationCell_securityAndPartnerDirectory (gate SAP_INTEGRATION_SUITE_ACC_EDGE_INTEGRATION_CELL) checks both; until it passes on a tenant, leave runtime_location_id unset.
- **`integration_assessment.landscape_configuration`** — Tenant-owned integration landscape inventory: Application, Application Instance, Technology, Technology Instance, Vendor, and their association entities (Technology Domain, Technology Style, Technology Key Characteristic). (🧭 Unofficial)
  - SAP documents the Entities API, these entities and their per-tenant limits (20,000 applications, 20,000 application instances, 50 technologies, 150 technology instances, 10,000 vendors), but the field-level specification on the Business Accelerator Hub needs an SAP login. The requests follow the live $metadata and were verified on a tenant on 2026-09-27: create (201 with a UUID Id), read, PATCH and PUT (204), delete (204) for all five objects, and links written as {"Id": ...}; the service rejects links written as __metadata URIs (V124) and a Content-Type with a charset parameter (V122).
  - Names, descriptions and links change in place (PATCH, verified on 2026-09-28): an application's vendor, an application instance's application and deployment model, a technology's vendor and a technology instance's deployment model. Only moving a technology instance to another technology was not tested, so it replaces the instance.
  - A technology's profile (TechnologyDomain, TechnologyStyle, TechnologyKeyCharacteristic) is managed as three association resources. Create, read and delete work on a tenant (2026-09-28); the service has no update for them (a key characteristic answered PATCH with 400 V101), so every attribute forces a new association.
  - Needs provider.integration_assessment: the service key of an "Integration Assessment APIs" service instance; the other credential sets do not work there.
- **`integration_assessment.master_data`** — SAP Integration Solution Advisory Methodology (ISA-M) taxonomy: Domain, Style, Use Case Pattern, Integration Pattern, Key Characteristic (and its Group/Value/Recommendation), Deployment Model, Domain Determination — largely SAP-maintained reference content a tenant can review and adjust. (🧭 Unofficial)
  - SAP documents the Entities API and lists these entities with a description each; the field-level specification on the Business Accelerator Hub needs an SAP login. The contract comes from the live $metadata (snapshot testdata/api-metadata/integration-assessment-entities.json), and every entity set was read on a tenant in September 2026.
  - The whole taxonomy has read-only data sources, looked up by exact name: deployment models, domains, styles, use case patterns, integration patterns, key characteristic groups, key characteristic values (by their key characteristic's name and their own, since value names repeat) and recommendation degrees. A domain determination has no name and is found by its source and target deployment models.
  - SAP Help's description of Domain Determination repeats the recommendation degree's text; its meaning (the domain that applies between two deployment models) comes from the  only.
  - The taxonomy is SAP's reference content, adjusted in the UI if at all, so there are no resources for it.
  - Needs provider.integration_assessment: the service key of an "Integration Assessment APIs" service instance; the other credential sets do not work there.
- **`security.certificate_chain`** — A certificate chain associated with a key pair. (🧭 Unofficial)
  - SAP Help names importing and exporting a key pair's certificate chain and creating a certificate signing request as capabilities of the Key Pair API; the requests are documented only in the API specification on the Business Accelerator Hub, which needs a login. The provider uses what the tenant $metadata declares and a tenant probe verified on 2026-10-04: the CSR through KeystoreEntries('<hexalias>')/SigningRequest/$value (sapintegrationsuite_key_pair.certificate_signing_request), the upload as PUT CertificateChainResources('<hexalias>')/$value with fingerprintVerified=true, and the export through KeystoreEntries('<hexalias>')/ChainResource/$value, a PKCS#7 bundle.
  - SAP accepted PEM in any order, without the root and the leaf alone; it refuses a certificate issued for another key (400 "The public key of the CA Reply is different"). After the upload the key pair reports the signed certificate's issuer and validity; sapintegrationsuite_key_pair keeps the validity it was generated with.
  - SAP has no request that removes a chain: destroying the resource leaves the chain on the key pair, and only regenerating the key pair returns to a self-signed certificate.
  - TestAccKeyPairCertificateChain_signedByCA passed on a tenant (2026-10-04): a key pair's CSR signed by a hashicorp/tls CA, the chain uploaded, read back and imported, and after regenerating the key pair the new CSR signed and the chain uploaded in the same apply. SAP signs the CSR with another algorithm after a chain upload, so the key pair keeps the stored CSR while subject and public key stay the same.
  - Earlier research (kept for the record): key pair's Hexalias with a KeystoreEntry navigation, and a read-only ChainCertificates set (Hexalias, Index and certificate details); KeystoreEntries('<hexalias>')/ChainCertificates returned a key pair's chain on 2026-09-27.
- **`security.oauth2_client_credential`** — An "OAuth2 Client Credentials" security material artifact: the client ID, client secret, and token service URL an integration flow adapter uses for the OAuth2 client credentials grant (RFC 6749) on outbound requests. (partial support already implemented — see Limitations below)
  - The client secret is never returned by SAP's read API; client_secret_wo/client_secret_wo_version are write-only attributes (Terraform CLI 1.11+ required) and drift on the secret value itself cannot be detected.
  - client_authentication, scope_content_type, resource and audience map to the ClientAuthentication, ScopeContentType, Resource and Audience properties confirmed by a tenant $metadata. Their accepted constants are undocumented, so they are passed through; they are Optional+Computed so every PUT resends values set in the UI.
  - custom_parameters (unofficial, enable_unofficial): a tenant check of 2026-10-04 showed that SAP takes custom parameters only in the POST that creates the credential (deep insert), accepts SendAsPartOf body, header or url, deletes all parameters with every PUT that does not send them, and refuses a PUT that does (400), MERGE (405) and PATCH (501). With custom_parameters set, every change therefore replaces the credential. Without it, an update or replacement warns when SAP holds parameters set in the UI, because it deletes them. The UI's grant-type placement (URL or body) has no API property at all. TestAccOAuth2ClientCredential_customParameters passed on a tenant (2026-10-04): created with two parameters, replaced on a description change with the parameters intact, imported without a difference, and replaced without them.
  - Update is implemented as a full PUT redeploy and resends client_secret_wo on every apply that touches this resource, matching SAP's documented requirement to re-enter the client secret on every edit.
  - OAuth2 Authorization Code and OAuth2 SAML Bearer Assertion are separate SAP artifact types this provider does not implement: Authorization Code requires interactive human authorization (see security.oauth2_authorization_code note in docs/guides/security-content.md). OAuth2 SAML Bearer Assertion and the 2026 OAuth2 Password Credentials artifact have no entity set in the tenant $metadata of /api/v1, so there is no public API to manage them.
- **`security.pgp_keyring`** — The tenant's PGP public and secret keys used by the PGP encryptor, decryptor, signer and verifier steps. (🧭 Unofficial)
  - SAP Help documents PGP keys only in the Monitor UI (Manage Security > PGP Keys: add public or secret keys, download, delete). The resources use what the tenant $metadata declares and a tenant probe verified on 2026-10-04: PUT PgpKeyringPublicResources('pubring')/$value and PgpKeyringSecretResources('secring')/$value with an armored keyring add its key and answer with PgpKeyEntryImportResults, the secret keyring needs the key's passphrase in the request header Passphrase, and PgpKeyEntries('<KeyId>') reads and deletes one key.
  - One resource per key, not per keyring: an upload adds keys, a keyring holds the keys of other owners, and replacing a whole keyring would remove them. A keyring with more than one primary key is refused; a key ID that exists already fails the create (SAP answers "not imported" with HTTP 200), so import it instead.
  - SAP deletes a key's public and secret part together. sapintegrationsuite_pgp_public_key therefore leaves a key that also has a secret part in place on destroy, with a warning.
  - Cloud runtime only: on the Cloud runtime only the keyrings pubring and secring exist (other names answered 404 "PUBLIC_KEYRING ... does not exist"). The keyrings of an Edge Integration Cell (pubring-<name>/secring-<name>) were not tested.
  - Version 4 keys only: the plan derives key ID and fingerprint locally (RFC 4880), which is defined differently for version 5 and 6 keys.
  - TestAccPGPKeys_publicAndSecret passed on a tenant (2026-10-04): a public and a secret key generated by gpg added, read, imported and deleted; CheckDestroy confirmed both key IDs gone.
- **`security.secure_parameter`** — A "Secure Parameter" security material artifact: an opaque confidential value (for example for a custom adapter) deployed without an associated username. (🧭 Unofficial)
  - SAP Help documents the artifact only in the Monitor UI. The entity set comes from the tenant $metadata (key Name; Description, SecureParam, DeployedBy, DeployedOn, Status), and a tenant test in September 2026 verified create (POST), read by name, update (PUT) and delete, each write answering 202 without a body.
  - The value is write-only (secure_param_wo / secure_param_wo_version) and sent on create and every update; SAP returns SecureParam as null. Import recovers the name and description only, so the first apply after an import sends the configured value.
  - Create stops when the name already exists, because SAP does not document what a create on an existing name does. Edge Integration Cell targeting is not supported and not offered for this resource.
- **`security.user_credential`** — A "User Credentials" security material artifact: a username/password credential integration flow adapters use for outbound basic or username-token authentication. (partial support already implemented — see Limitations below)
  - The password is never returned by SAP's read API; password_wo/password_wo_version are write-only attributes (Terraform CLI 1.11+ required) and drift on the password value itself cannot be detected — only readable metadata (user, description, kind, company_id) is compared on Read.
  - Update is implemented as a full PUT redeploy, matching SAP's documented "Edit" action for Credentials artifacts, and resends password_wo on every apply that touches this resource (SAP documents re-entering the secret on every edit for the sibling OAuth2 Client Credentials artifact; this provider assumes the same requirement here since it could not find a documented exception for User Credentials).
  - Kind takes only default, successfactors and openconnectors, in lower case: a tenant (2026-10-04) refused every other value, including the SuccessFactors and OpenConnectors that releases before 0.5.1 documented. successfactors needs CompanyId; openconnectors needs a password in the undocumented Open Connectors credential format. kind is checked while planning.
  - Deployment status (SAP's UI shows Stored/Deployed/Error) is not exposed: this project could not confirm the OData property name for it, and would rather omit a computed attribute than expose one that is silently always empty.
  - Verified on a tenant (September 2026): SAP rejects a create without Kind ("must not be empty or null") or with Description null, and reports a generic credential's kind as "default". kind therefore defaults to "default", and Kind, Description and CompanyId are always sent. Create, read, update (PUT) and delete were exercised.

### Public lifecycle insufficient for safe Terraform management

- **`api_management.classic.api_provider`** — A classic API Management backend/API provider system definition — the connection an API Proxy targets. (partial support already implemented — see Limitations below)
  - Only the "Internet" connection type is supported (direct host/port, optionally over SSL). SAP documents three further connection types (On Premise via Cloud Connector, Open Connectors, Cloud Integration) with distinct field sets this provider could not confirm a field-level JSON mapping for from a reachable primary source.
  - No Update operation: SAP's own official Piper apiProviderUpload tooling documents that only Create is supported through this API; every attribute is RequiresReplace.
  - Eventual consistency: SAP documents up to approximately 20 seconds of caching before a just-created or just-deleted provider is reliably visible to GET; Create polls with bounded, jittered backoff to reduce (not eliminate) this window.
- **`api_management.classic.key_value_map`** — A classic API Management key-value map used for runtime configuration lookups, readable through the Key Value Map Operations policy. (partial support already implemented — see Limitations below)
  - No Update: SAP's documentation confirms a full Create/Update-entries/Delete UI lifecycle exists, but only Create's REST payload is shown verbatim anywhere reachable; every attribute, including entries, is RequiresReplace.
  - Encrypted maps are not supported: SAP's isEncrypted field is real and documented, but this provider could not confirm whether GET returns an encrypted entry's plaintext value back, a masked placeholder, or nothing at all. This resource always sends isEncrypted = false and rejects a configuration that sets encrypted = true.
- **`cloud_integration.archiving`** — Archiving of message processing logs (and, for Trading Partner Management, B2B interchange payloads) to an external CMIS repository.
  - Tenant-wide activation is a one-way switch: the tenant $metadata has activateArchivingConfiguration and activateB2BArchivingConfiguration (POST, no parameters) and read-only ArchivingConfigurations/B2BArchivingConfigurations (Id, Active), but no deactivation. SAP's "Enable Archiving" page adds that the CMS metadata properties cannot be changed once archiving is enabled. A resource could never implement destroy, so none is offered.
  - Per-integration-flow settings (sender and receiver channel messages, persisted messages, log attachments) are documented only in the Integration Content Monitor; the Archiving* flags in the $metadata sit on MessageProcessingLog and record what was active when a message ran.
  - The destination (CloudIntegration_Archive / CloudIntegration_B2BArchive) is a BTP destination, outside this provider. The KPI entity sets are monitoring data.
- **`cloud_integration.custom_tag_configuration`** — The tenant-wide set of custom tags integration package owners are asked, or required, to classify their packages with. (partial support already implemented — see Limitations below)
  - No confirmed delete or clear operation exists for this entity anywhere in SAP's public documentation. Destroying this resource in Terraform returns an explicit error rather than guessing that an empty overwrite means delete, or silently dropping Terraform state while leaving the tenant's configuration untouched — see docs/guides/custom-tag-configurations.md.
  - Create and Update both use the same confirmed POST .../CustomTagConfigurations?Overwrite=true operation (SAP documents no separate plain-POST-without-Overwrite path this provider relies on), sending the complete desired tag list every time. Whether Overwrite=true is a full replace (removing tags not present in the new list) is strongly implied by the word "Overwrite" and by the fact the documented payload is the complete configuration, not a delta, but SAP's documentation never uses the word "replace" explicitly.
  - Whether tag names must be unique, whether permitted values are case-sensitive, and whether SAP preserves submitted ordering are all unconfirmed by SAP's documentation. This provider enforces tag-name uniqueness itself and treats ordering (of both tags and permitted values) as not semantically meaningful, modeling both as unordered Terraform sets so a reordered response never produces a spurious diff.
  - One documented example response shows a single-element permittedValues array containing a comma-separated string ("Mr. Bean, Ms. Bean") rather than two separate array elements; this is treated as a documentation artifact, not a confirmed wire format, since every other array-typed field in SAP's own examples (and everywhere else in this provider) uses one array element per value.
- **`cloud_integration.service_endpoints`** — Read-only discovery of the runtime service endpoints (entry point URLs and API definition links) SAP generates for deployed Cloud Integration content. (👁️ Read-only)
  - Discovery only, by design: SAP generates service endpoints from deployed content and there is no create/update/delete API for them, so this provider intentionally has no matching resource type — see docs/guides/service-endpoints.md.
  - No single-endpoint (sapintegrationsuite_service_endpoint) data source exists: this project could not confirm that Name uniquely and stably identifies exactly one service endpoint, so only the collection data source (sapintegrationsuite_service_endpoints, with optional name/protocol filters) is implemented, to avoid a lookup data source that silently returns the wrong result if more than one endpoint ever matches.
  - All exposed fields are checked against a tenant $metadata document. API definition links carry a url and a name; the format "type" (oas-json, edmx, ...) that SAP's documentation mentions is not a property of the Definition entity, and earlier releases that exposed it always returned an empty value.
- **`cloud_integration.value_mapping`** — A value mapping design-time artifact's content, managed as file-based content. (partial support already implemented — see Limitations below)
  - No in-place update: a tenant probe (2026-10-06) got 501 Not Implemented for every PUT on ValueMappingDesigntimeArtifacts, with content, name or both, so changing name, content or content_hash replaces the resource (Terraform deletes the artifact and uploads it again).
  - save_as_version uses ValueMappingDesigntimeArtifactSaveAsVersion (only in $metadata), which answered 200 and relabels the one stored version, even to a lower number, without changing the content (tenant probe, 2026-10-06). A changed save_as_version relabels in place; it needs enable_unofficial. TestAccValueMapping_saveAsVersion passed on a tenant (2026-10-07).
  - SAP keeps one version of a value mapping: Delete removes the artifact completely (tenant probe, 2026-10-06).
- **`cloud_integration.value_mapping_entry`** — Individual source/target value pairs inside a value mapping's agency/identifier pair, managed through UpsertValMaps, UpdateDefaultValMap and DeleteValMaps.
  - Tenant check of 2026-09-27 on a synthetic value mapping: the first UpsertValMaps with IsConfigured=true switched the agency pair to State Configured and dropped the two values the design-time content had defined; a second UpsertValMaps with the same source value added a second entry instead of changing the first, and SAP made the newest one the default; DeleteValMaps for the pair answered 202 and removed nothing; every call left the artifact in version Draft. A resource could neither update an entry in place nor destroy what it created, so entries stay unmanaged.
  - Create and update are documented (POST /UpsertValMaps?Id=&Version=&SrcAgency=&SrcId=&TgtAgency=&TgtId=&SrcValue=&TgtValue=&IsConfigured=, returning a ValMap with Id and Value{SrcValue, TgtValue}), and reading goes through ValMapSchema(...)/ValMaps. Deletion is not documented per entry: DeleteValMaps takes only Id, Version and the agency/identifier pair, and the tenant $metadata has no ValMapId parameter for it.
  - A resource per entry could therefore not implement destroy. A resource owning a whole agency/identifier pair could, but SAP does not document whether UpsertValMaps creates a missing pair, what IsConfigured=false does (SAP only says it should always be true once configured), or whether DeleteValMaps removes the pair or only its entries. Those need a check against a tenant before business data is written.
  - Entries set through the API also compete with the value mapping's own content: uploading content through sapintegrationsuite_value_mapping replaces them.
- **`partner_directory.partner`** — A Partner ID (Pid) known to the tenant's Partner Directory. (👁️ Read-only)
  - No resource: SAP's API reads all partners and deletes a partner, but has no create operation. A Pid comes into existence implicitly the first time a StringParameter, BinaryParameter, AlternativePartner, AuthorizedUser, or UserCredentialParameter references it.
  - Deleting a partner removes the partner and all its entities (SAP's Delete Partner description). A Partner resource's destroy could therefore erase content owned by other Terraform resources or modules, which is the other reason this stays read-only.
  - SAP refuses to read a single partner by key ("Reading of single partner entities is not supported", tenant test September 2026); the data source filters the collection by Pid instead. A partner exists only while it has entries: after its last entry is deleted, DELETE Partners('<pid>') answers 404.
- **`security.key_pair`** — An SAP-generated key pair keystore entry (private key plus X.509 certificate), as opposed to one uploaded from outside the tenant. (partial support already implemented — see Limitations below)
  - Create confirmed field-for-field via SAP's own "Generate a Key Pair" documentation: POST KeyPairGenerationRequests. The private key never leaves SAP — this resource has no field for it and never requests one.
  - No update operation is documented for a generated key pair: every attribute that defines the generated key material is RequiresReplace.
  - Only the subset SAP confirms KeystoreEntries returns (key_type, key_size, valid_not_before, valid_not_after) is read back and refreshed on every plan; generation-only parameters SAP does not confirm returning (signature_algorithm, key_algorithm_parameter, the subject DN fields) are trusted from the last successful write, not re-verified — the same reason this is `partial`, not `supported`.
  - Delete uses the same documented keystore mass-deletion operation as sapintegrationsuite_certificate, with exactly the one alias this resource owns.
  - certificate_signing_request is read only with enable_unofficial (see Certificate Chain). After a certificate chain was uploaded, valid_not_before and valid_not_after keep the values the key pair was generated with; SAP then reports the signed certificate's validity, which sapintegrationsuite_key_pair_certificate_chain shows.
- **`security.keystore_entry`** — Any entry (certificate, SAP-generated key pair, or other RSA/DSA/EC-keyed entry) in the tenant's keystore, read-only. (👁️ Read-only)
  - Fields follow the tenant $metadata: besides alias, key type and size and validity, the data sources expose entry type, owner, status, subject and issuer DN, serial number, signature algorithm, elliptic curve, certificate version, SHA-1/256/512 fingerprints and creation/modification details. Dates are converted from OData V2 literals to RFC 3339.
  - owner shows who owns an entry, but its values are not documented. The certificate and key pair resources therefore still rely on SAP's own server-side protection when an Update or Delete targets an SAP-owned entry, rather than trying to detect it in advance — see docs/guides/security-content.md.
  - No resource: this entity represents fundamentally different object types (plain certificates, generated key pairs) with different lifecycles, so a single mutable sapintegrationsuite_keystore_entry resource was deliberately not created — see security.certificate and security.key_pair instead.

### No suitable public SAP API

- **`api_gateway.api_artifact`** — A design-time API (REST, SOAP or OData) in SAP's API-centric integration model: endpoints, policies and security, created inside an integration package under Design > Integrations and APIs and deployed to Integration Cell or Edge Integration Cell.
  - SAP has not published an API for API artifacts. The September 2026 re-audit checked the Integration Content API's resource table (Help version of 2026-07-10; no API artifact resource), both SAP API Documentation index pages (only the CloudIntegrationAPI and Classic APIMgmt packages), every current creation, versioning, copy, deletion and deployment page (UI procedures only), SAP's API Management Client SDK 3.0.6 (Classic API Portal endpoints only) and SAP's own 2026 CI/CD tooling (no API artifact automation).
  - The 2026 features around API artifacts, such as API-centric integration, simplified creation from a URL or specification, OpenAPI specifications drafted by SAP's assistant and product subscriptions, are all UI features.
  - Re-checked 2026-09-26 (Hub package APIMgmt of 2026-09-24, SAP Help mirror of 2026-09-18, Client SDK 3.0.6 of 2026-09-24): still no API. API artifacts and MCP servers are integration package content (resourceType API in a package export) and move between tenants with the package: export, POST IntegrationPackages with PackageContent (optionally ?Overwrite=true), SAP Cloud Transport Management or CTS+. That whole-package import is the only public path and is opaque to individual API artifacts, so it is not modelled as an API artifact resource.
  - A downloaded API artifact (2026-09-27) is a bundle of type RESTAPI for the Integration Cell runtime profile. It holds an OpenAPI 3.0 definition and an integration flow that SAP generates from it. The format is SAP-internal, so building such bundles is not a way around the missing API.
  - The documented Integration Content API does not see them: for a package holding an integration flow, an API artifact and an MCP server, IntegrationPackages('<id>')/IntegrationDesigntimeArtifacts listed only the integration flow (2026-09-29).
- **`api_gateway.api_artifact_deployment`** — The runtime deployment of an API Artifact or MCP Server on Integration Cell or Edge Integration Cell, including the virtual host chosen at deployment time.
  - SAP documents deployment only as a UI action; no deploy, undeploy or status API exists for API artifacts or MCP servers.
  - Semantics to preserve if an API appears: the runtime profile is fixed once the artifact exists (except for choosing the target Edge Integration Cell), and the deployment-time virtual host may differ from the design-time one, with the deployed endpoint URL following the deployment-time choice. Terraform would need to keep the two virtual hosts as separate attributes.
- **`api_gateway.api_policy`** — A policy or mediation step (authentication, quota, rate limiting, transformation, external callout and so on) inside an API Artifact.
  - Policies are edited inside the API artifact's policy editor and have no separate lifecycle in SAP Help. Without an API for the artifact itself there is nothing to show whether policies would be nested content or addressable entities; a separate resource would only make sense for the latter.
- **`api_gateway.mcp_server`** — A Model Context Protocol server artifact that exposes APIs as tools for AI agents, created from an API artifact, an HTTP endpoint with an OpenAPI specification, an RFC-enabled backend, a remote MCP server or a Classic API proxy, and deployed to Integration Cell.
  - New in 2026 (MCP Gateway, July 2026; remote MCP servers, September 2026). Creation, tool selection, authentication and deployment are documented only as UI procedures, and no MCP server resource appears in any public API or in SAP's Client SDK.
  - Like API artifacts, MCP servers are integration package content: a downloaded MCP server (2026-09-27) is a bundle of type MCPSERVER for the Integration Cell runtime profile that requires the bundle of its source API artifact. It can only be moved with the whole package.
  - Publishing an MCP server as a Developer Hub product belongs to the separate Developer Hub provider, not to this one.
- **`api_gateway.reusable_api_artifact`** — An internal-only API Artifact with no external endpoint, invoked by other API Artifacts through the API Direct adapter to share logic such as authentication or transformation.
  - A variant of the API artifact (unique base path, not reachable over HTTP, cannot call another reusable API), created through the same UI. It shares the API artifact's blocker. If an API appears, the intended model is a type discriminator on the API artifact resource rather than a second resource.
- **`api_gateway.runtime_profile`** — The target platform (Cloud Integration, Integration Cell, Edge Integration Cell, SAP Process Orchestration) an API Artifact or integration flow is designed and deployed for.
  - Runtime profiles are enabled and disabled under Settings > Integrations only. Even with an API, the list would be a weak data source: small, platform-defined and better served by documentation.
  - SAP's Runtime Profiles reference page still (September 2026) lists no Integration Cell row, although Integration Cell is offered as a runtime profile when creating API artifacts and MCP servers.
- **`capabilities.api_gateway`** — Activating SAP's current, API-centric API Management capability (API Artifacts, Integration Cell) itself within an Integration Suite tenant.
- **`capabilities.api_management`** — Activating the classic API Management capability itself within an Integration Suite tenant.
  - Documented as a UI step (Integration Suite → Manage Capabilities → activate API Management); no public API found.
- **`capabilities.cloud_integration`** — Activating the Cloud Integration capability itself within an Integration Suite tenant.
  - Activated as part of the Integration Suite subscription/onboarding wizard; stays a manual, one-time bootstrap step.
- **`capabilities.edge_integration_cell`** — Activating the Edge Integration Cell capability itself within an Integration Suite tenant.
- **`capabilities.integration_cell`** — Activating the Integration Cell capability itself within an Integration Suite tenant.
  - SAP's own documentation describes activation as choosing Activate in Integration Suite → Settings → Runtime; no public API was found.
- **`edge_integration_cell.registration`** — SAP-side registration and runtime association of an Edge Integration Cell, never the customer-managed Kubernetes workloads themselves.
  - Reconfirmed: an Edge Node is added and removed exclusively through the Edge Lifecycle Management (ELM) UI, and onboarding is completed by running the standalone Edge Lifecycle Management Bridge executable against the target Kubernetes cluster's kubeconfig — a local CLI/Kubernetes handshake, not an HTTP API this provider could call. No public registration, runtime-association, or deregistration API was found.
- **`integration_advisor.design_time_content`** — Message Implementation Guidelines (MIGs), Mapping Guidelines (MAGs, standard/overlay/XSLT), custom Type Systems, Codelists, Shared Code, and Global Parameters — SAP's collaborative B2B interface-content-design objects.
  - No API-access documentation page (the kind that, when present, confirmed a real public API for Classic API Management and Integration Assessment) exists anywhere in Integration Advisor's roughly eighty-five-page documentation tree. The one OAuth-credential page found in this area ("Creating OAuth Client Credentials for Cloud Foundry Environment") turned out, on full reading, to describe authenticating against the destination Cloud Integration tenant for artifact injection (Process Integration Runtime service, plan api), not a credential for Integration Advisor's own design-time content management.
  - Re-audit September 2026: all 80 pages of the current documentation checked again, and the full Business Accelerator Hub package list (1971 packages) contains only integration content for Integration Advisor (EDI Integration Templates), no API.
- **`integration_cell.runtime`** — Runtime status and configuration of an already-activated Integration Cell, distinct from activating the capability itself.
  - Split by concern, none has a public API: activation is a Settings > Runtime UI step; runtime discovery and status are shown only in Monitor > Integrations and APIs with the Integration Cell runtime selected; configuration such as trace log level (new for Integration Cell APIs in August 2026) is UI-only; and the only deployment targeting is the runtime profile chosen in the UI.
- **`integration_cell.virtual_host`** — A host name under the Integration Cell default domain (or a custom domain) through which API Artifacts and MCP Servers are exposed.
  - Virtual hosts are added, edited and deleted in Monitor > Integrations and APIs > Virtual Host (PI_Administrator); SAP documents no API for any of these steps.
  - Rules a future resource would have to respect: at most 11 virtual hosts per tenant, the name Default is reserved, the host alias is at most 22 characters, and deletion is blocked for the default virtual host and for hosts used by deployed APIs. APIs that referenced a deleted host fall back to the default one. The default host would have to be read-only in Terraform.
- **`migration_assessment.source_system`** — A registered SAP Process Orchestration system (7.31 SP28+, 7.40 SP23+, or 7.50 SP06+) Migration Assessment extracts integration scenario data from.
  - No API-access or service-key documentation page exists anywhere in Migration Assessment's documentation tree (a small, roughly fifteen-page tree, entirely checked). Its own documentation instead describes Migration Assessment as an API *consumer*: it reaches into a registered source system's own SAP Process Orchestration APIs (via Cloud Connector/Destination service) to extract data — the opposite direction from a public API this provider could manage Migration Assessment's own objects through.
  - Re-audit September 2026: the 11 current pages and the full Business Accelerator Hub package list show no Migration Assessment API.
- **`odata_provisioning`** — A capability that exposes SAP Business Suite backend OData services (SAP Gateway back-end-enablement) through SAP Integration Suite, without requiring an on-premise SAP Gateway hub.
  - Re-audit September 2026: registering OData services, adding destinations, switching a service on or off, error tolerance for multi-origin composition, and metadata validation and cache settings are documented only in the UI (Configure > OData Services). The complete Business Accelerator Hub package list has no OData Provisioning package.
  - ODPAPIAccess, earlier read as a sign of a management API, grants access to the service document of registered services; APIFullAccess grants runtime access and ODPManage the UI. The service key (Serverless Runtime, plan odpruntime) is for calling the registered services, not for configuring them.
  - Registered services would suit Terraform (named configuration with destinations and settings) if SAP publishes a management API.
- **`security.certificate_user_mapping`** — A mapping from a client certificate to an inbound user identity, used for inbound client certificate authentication.
  - Reverified for this feature family: SAP's own documentation ("Managing Certificate-to-User Mappings", "Client Certificate Authentication and Certificate-to-User Mapping (Inbound)", "Setting Up Inbound HTTP Connections with Certificate-to-User Mapping") exists only under the Neo environment, with no Cloud Foundry equivalent found in SAP's published documentation set. This provider targets the Cloud Foundry environment (its other Security Content resources use the Cloud Foundry "/api/v1" OData host), so this catalog entry is corrected from its previous "not_implemented"/PublicAPI:true state to "no_public_api": the feature cannot be implemented for this provider's target environment, not merely unimplemented yet. If SAP publishes a Cloud Foundry certificate-to-user-mapping API in the future, re-open this entry.
- **`security.known_hosts`** — The SSH "known_hosts" file artifact used to validate SFTP server host keys for outbound SFTP connections.
  - Reverified for this feature family and strengthened from research_required to no_public_api: unlike Secure Parameter (at least conceptually listed in SAP's Security Content API overview's resource table), Known Hosts does not appear in that table at all. SAP's "Deploying an SSH Known Hosts Artifact" documentation describes only the Manage Security Material UI (Create > Known Hosts (SSH), Browse/Add/Deploy), with no REST endpoint mentioned anywhere. The tenant $metadata of /api/v1 (September 2026) has no known-hosts entity either.
- **`security.oauth2_password_credential`** — An OAuth2 resource owner password credentials artifact (new in 2026): user name, password and optional client authentication for the OAuth2 password grant.
  - Documented only as a Security Material UI procedure (Create > OAuth2 Password Credentials). The tenant $metadata of /api/v1 has no entity set for it, and OAuth2ClientCredential has no user or password property it could be stored in.
- **`security.oauth2_saml_bearer`** — An OAuth2 SAML bearer assertion artifact for principal propagation to OAuth-protected receivers.
  - Documented only as a Security Material UI procedure. The tenant $metadata of /api/v1 has no SAML bearer entity set.
- **`security.where_used`** — The list of integration artifacts that reference a security material artifact.
  - Shown in the Security Material UI only; the tenant $metadata of /api/v1 has no where-used entity or function import. If one appears it would be a read-only data source, never mutable state.
- **`trading_partner_management.agreement`** — A trading partner agreement: business transaction activities, identifiers, and integration/message flow configuration between the tenant's company profile and a trading partner.
- **`trading_partner_management.agreement_template`** — A reusable template defining the shape of trading partner agreements created from it.
- **`trading_partner_management.company_profile`** — The tenant's own company profile and its subsidiaries, the initiator side of every trading partner agreement.
  - Reconfirmed: no "accessing APIs programmatically", API reference, or service-instance/service-key page exists anywhere in SAP's entire Trading Partner Management documentation tree (roughly 90 pages checked) — the same kind of page that, when present, confirmed a real public API for Classic API Management and Integration Assessment. Content is downloadable as JSON through a UI Download button only (company.json), never through a documented REST endpoint.
  - Re-audit September 2026: all 93 pages checked again. The Business Accelerator Hub's Cloud Integration package has a "B2B Scenarios" OData API, but the tenant $metadata shows what it covers: BusinessDocuments, interchanges, payloads, events and the reprocessing functions singleInterchangeProcess/massInterchangeProcess, all B2B monitoring data. Profiles, agreement templates and agreements still have no API. The only TPM configuration call SAP documents is B2B archiving activation, see cloud_integration.archiving.
- **`trading_partner_management.partner_profile`** — Trading partner profiles and communication partner profiles: the counterparty side of a trading partner agreement.
  - Same evidence as trading_partner_management.company_profile: UI-only (Design > B2B Scenarios), downloadable as JSON, no documented REST/OData endpoint found.

### Further research required

- **`data_space_integration`** — SAP's Dataspace-Protocol-based data space connectivity capability (Connectors, Assets, Policies, Contract Definitions, Contract Negotiations/Agreements) within Integration Suite, initially scoped to the Catena-X data space. (🔬 Research required)
  - A dedicated "Data Space Integration API Access" service instance (plan api, roles AuthGroup_DataspaceConsumer/AuthGroup_DataspaceProvider, client_credentials grant, one instance per connector) gives API access. The Business Accelerator Hub lists one REST API, DSIAPI 2.0.0; its specification needs an SAP login.
  - Re-audit September 2026: SAP's Help shows requests only for consumer runtime flows under /api/dsi/v1 (catalog, contract negotiation, transfer process). Assets, policies, contract definitions, company policies and contract references, the objects Terraform could manage, are documented only through the UI, so their API contract is not confirmed.
  - Being a REST API, it has no OData $metadata to read the contract from; the Hub specification or new SAP documentation is needed. See the Data Space Integration guide.

### Out of provider scope

- **`cloud_integration.data_store`** — A tenant-persisted runtime container of Data Store Entries, created implicitly by an integration flow's Data Store Write step (or an XI adapter's Temporary Storage option) the first time it writes an entry.
  - SAP documents exactly one public operation for this entity: GET .../DataStores?overdueonly=true, an aggregate monitoring endpoint (NumberOfMessages/NumberOfOverdueMessages per store) — the same class of runtime monitoring data as cloud_integration.message_processing_logs, not configuration. There is no independent declarative creation API: a Data Store comes into existence only as a side effect of deployed integration flow content.
  - The DataStores API does not support $filter, $inlinecount, $orderby, $skip, $top, $expand, or $select (confirmed directly from SAP's own documentation).
- **`cloud_integration.data_store_entry`** — A single runtime message (payload and headers) persisted inside a Data Store by an integration flow's Data Store Write step, read back by a Data Store Get or Select step, and deleted only by a Data Store Delete step.
  - SAP documents only GET operations for this entity (a single entry by composite key, all entries for a store, and all entries for a message ID) — every field (Status, MessageId, DueAt, CreatedAt, RetainUntil) is runtime business-message state, not infrastructure desired state. Delete exists only as a design-time integration flow step (entry-by-entry or bulk via an XPath-derived ID list at runtime), never as a REST call this provider could wrap in a Terraform destroy.
  - Deliberately out_of_scope rather than not_implemented: this provider does not manage business message payloads or place them into Terraform state, and 'terraform destroy' semantics are not an excuse to expose operational message deletion as desired infrastructure state — see docs/provider-scope.md.
- **`cloud_integration.message_processing_logs`** — Runtime message processing log records for deployed integration flows.
  - Monitoring/operational data, not infrastructure state this provider manages — see docs/provider-scope.md.
- **`cloud_integration.message_stores`** — Runtime persisted-message-store entries (created by the Persist step) and JMS queue resource metadata used by deployed integration flows. Data Stores, Data Store Entries, Variables, and Number Ranges — all part of the same broader Message Stores API family — each have their own dedicated catalog entry; see cloud_integration.data_store, cloud_integration.data_store_entry, cloud_integration.variable, and cloud_integration.number_range.
  - Payload/queue content is operational data, not desired state this provider manages.
- **`cloud_integration.variable`** — A tenant-persisted runtime value written by an integration flow's "Write Variables" step, shared across steps of the same flow (local) or across every flow deployed on the tenant (global).
  - SAP documents exactly one public operation for this entity: GET .../Variables(...)/$value, which downloads the raw value with no structured metadata (no Visibility/ UpdatedAt/RetainUntil fields are returned by this endpoint). There is no collection GET, no POST, no PUT, and no confirmed DELETE — Variables are created and updated exclusively by deployed integration flow content, an entirely different ownership domain than Terraform-managed infrastructure.
  - A read-only data source was deliberately not implemented: the only confirmed read operation returns nothing but the raw runtime value itself, with no safer metadata-only alternative available, and this provider does not place arbitrary runtime business values into Terraform state merely because an API can return them — see docs/provider-scope.md.
- **`developer_hub`** — SAP's API/Event/MCP Server catalog, publication, and subscription capability for Integration Suite, reachable through its own /api/1.0 REST API and its own devportal-apiaccess OAuth credentials, separate from every Cloud Integration and current API Management endpoint this provider otherwise talks to. (↗️ Separate provider)
  - Developer Hub has its own API boundary, its own OAuth client credentials, and a consumer/catalog object lifecycle (Products, Applications, Subscriptions) distinct in shape from this provider's Integration Suite content and capability model.
  - Planned as a separate, independently versioned Terraform provider (working name Prideth/terraform-provider-sap-developer-hub) rather than a domain inside this one, so its release cadence and credential surface never entangle with this provider's.
  - This entry intentionally represents the whole Developer Hub capability as a single scope statement; individual Developer Hub objects (Product, Application, Subscription, and so on) are not separately cataloged here.
- **`edge_integration_cell.local_api`** — SAP's public /local/api/v1 and /local/api/eic/v1 REST endpoints, reachable directly against a running Edge Integration Cell node without going through the cloud UI: Message Processing Logs, Message Stores/JMS, DataStores/Variables (OData V2), and the Operations Cockpit's Component/Job/RuntimeParameter resources (OData V4).
  - Confirmed reachable and documented (SAP Business Accelerator Hub package sap-int-eic-eic-operations, created 2026-04-30, with the APIs Jobs, Components, Partner Directory, Message Stores and Message Processing Logs), authenticated with the same certificate or clientId/clientsecret mechanisms as the rest of this provider, and CSRF-token gated for modifying calls — but every entity behind it is monitoring/operational runtime data (message processing records, message store/JMS contents, data store/variable values), the same category this provider already excludes for Cloud Integration's own MessageProcessingLogs/DataStores/Variables. See edge_integration_cell.runtime for the separate Operations Cockpit control-plane objects reachable through this same /local prefix.
  - The package's Partner Directory API is configuration, not monitoring, but it is the same Partner Directory the provider reaches from the cloud through /location/<runtime location id>/api/v1 (see edge_integration_cell.deployment_target), so a second, network-local path is not needed.
- **`edge_integration_cell.runtime`** — The Operations Cockpit API's Component, Pod, RuntimeParameter, and Job/JobSchedule entities: per-component status, pod resource limits/replica counts, log levels, and scheduled-job configuration for an Edge Integration Cell's own SAP-operator-managed pods.
  - Confirmed reachable at /local/api/eic/v1 (also documented on SAP Business Accelerator Hub as the Edge Integration Cell package's OData V4 API), and RuntimeParameter is explicitly described as changeable ("Save to run the changes in the back end") — a genuinely writable-looking configuration object, not read-only monitoring data.
  - Every RuntimeParameter documented (LOG_LEVEL, MIN_REPLICAS, MAX_REPLICAS, CPU_LIMIT, MEMORY_LIMIT, EPHEMERAL_STORAGE_LIMIT) configures Kubernetes-level pod resource requests, replica counts, or log verbosity for SAP-operator-managed components — the same category of concern docs/provider-scope.md already excludes as Kubernetes/Helm infrastructure, just fronted by an EIC-specific API instead of the raw Kubernetes API.
  - The API's own documentation states the Component resource lets a caller "restart components" — an imperative runtime operation this provider's standing policy refuses to model as a Terraform resource (see the retry/restart/cancel/purge rule this provider already applies elsewhere).
  - The exact entity keys and PATCH/POST payload shapes for RuntimeParameter and JobSchedule are not confirmed from a reachable primary source: the package's $metadata/EDMX and worked examples live behind SAP Business Accelerator Hub's authenticated catalog pages, which redirect unauthenticated requests to a login page, the same access limitation this project has documented repeatedly for other api.sap.com packages.
- **`event_mesh`** — SAP's Solace PubSub+-based event broker service for asynchronous, event-driven integration: queues, topic subscriptions, and webhook subscriptions. (↗️ Separate provider)
  - Reconfirmed, not just assumed: Event Mesh is activated through the same generic "Activating and Managing Capabilities" mechanism as every other Integration Suite capability (no dedicated public activation API, consistent with every other capability audited), but once active it exposes a genuine, well-documented broker management surface (service-key-based channel/queue/subscription creation, AMQP/MQTT/REST messaging APIs) confirmed across roughly forty-five documentation pages.
  - Event Mesh predates, and is usable entirely independently of, SAP Integration Suite — it is a general-purpose BTP messaging service consumed by many unrelated SAP products, with its own Solace PubSub+-derived API family fundamentally different in shape from the OData-centric model this provider is built around. Community Terraform support for Solace PubSub+ already exists independently. Rather than absorbing a broker-management surface into this Integration-Suite-scoped provider, this belongs in a separate, independently versioned provider, the same reasoning already applied to Developer Hub.
- **`integration_advisor.runtime_artifact_injection`** — Injecting generated runtime artifacts (from a Mapping Guideline) directly into an integration flow's resources on a target Cloud Integration tenant.
  - Confirmed as a UI wizard (Mapping Guideline > Inject > SAP Cloud Integration Flow Resources > choose tenant/package/integration flow > Inject), not a documented REST call, and — independent of that — an imperative one-shot action rather than desired-state configuration this provider's plan/apply model could represent even if an API for it were confirmed.
- **`integration_assessment.assessment_workflow`** — Business solution/interface Request and Request Line Item workflow objects, the Integration Flow/Message Flow content they reference, and Request Line Item Technology Instance Decision.
  - Confirmed to be workflow/project state, not desired-state configuration: SAP documents an explicit Request status machine (draft -> new -> in progress -> completed, with a Reopen action), matching this provider's established Message Processing Log/Developer Hub Subscription category of exclusion — out of scope regardless of whether a field-level API contract is ever confirmed for it.
- **`migration_assessment.extraction_and_evaluation`** — Data Extraction Requests, Scenario Evaluation Requests, and their resulting assessment-category/migration-readiness/effort-estimate reports.
  - Confirmed to be action-triggered workflow and reporting data, not desired-state configuration: SAP's own documentation describes choosing "Create" on a Data Extraction Request as starting an extraction process with a resulting Completed/Completed with warnings/Completed with errors status, and Scenario Evaluation results are assessment-category classifications, migration-readiness scores, and effort estimates — reporting output, the same category this provider already excludes for Message Processing Logs. Out of scope regardless of whether a public API is ever confirmed for triggering these actions.
- **`open_connectors`** — SAP's third-party SaaS connectivity capability within Integration Suite: a catalog of 170+ third-party connector types (each with its own normalized REST API and OpenAPI-documented instance configuration), originally the standalone Cloud Elements product.
  - A deliberate suitability judgment, not a research gap: Open Connectors is not one coherent API this provider could model with a handful of resources, the pattern every other capability in this catalog follows. It is a catalog of 170+ independent third-party connector types, each with its own authentication scheme, configuration schema, and normalized-but-still-connector-specific REST surface. Implementing even connector-instance management generically would mean either an unbounded, per-connector-type schema explosion, or an opaque untyped-JSON resource that gives up the type safety and validation this provider's schema-first design otherwise provides everywhere else.
- **`security.ssh_key`** — An SSH-capable key pair used for SFTP public-key authentication.
  - Reverified for this feature family and corrected: SAP's Security Content API overview lists no independent "SSH Key" resource, and the tenant keystore's own "Creating a Key Pair/SSH Key Pair" UI documentation uses the identical Key Pair attribute set (alias, key type, key size, signature algorithm, subject DN fields, validity) for both — "Create > Key Pair" and "Create > SSH Key" are the same underlying mechanism with a different label. The tenant $metadata does define SSHKeyGenerationRequests (with SSHFile and Password) and SSHKeyResources, but SAP documents no request for either, so no separate resource is built on them.
  - An RSA or DSA sapintegrationsuite_key_pair's public key can be exported in OpenSSH format via public_key_openssh, backed by SAP's confirmed KeystoreEntries('<hexalias>')/Sshkey/$value — this covers the SSH use case without a separate resource. EC key pairs are documented as unsupported for this export.
- **`trading_partner_management.partner_directory_generation`** — The generated Partner Directory entries (String/Binary Parameters, prefixed "SAP_TPM") an agreement's activation pushes into Partner Directory for runtime use by generic integration flows.
  - Confirmed, verbatim: "When a trading partner agreement gets activated, the complete agreement information gets pushed into the partner directory." This is a UI-triggered (Activate action), one-way generation side effect, not an independent create/update operation this provider's already-implemented Partner Directory resources (sapintegrationsuite_partner_string_parameter and siblings) could safely manage or even represent — out of scope for the same reason this provider does not model other imperative generation/replication actions as resources, regardless of whether a public API for the trigger itself is ever confirmed.

