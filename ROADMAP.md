# Roadmap

This roadmap orders the open work by what unblocks it. Every item names the evidence it needs;
nothing moves up because an entity merely exists in a `$metadata` document, since that does not
show that SAP supports writing it. The authoritative per-feature status is
[`docs/feature-support.md`](docs/feature-support.md); the dated evidence behind every entry that
is not fully supported is [`docs/research/capability-evidence-2026.md`](docs/research/capability-evidence-2026.md);
the entity-level view of each service contract is [`docs/api-discovery-report.md`](docs/api-discovery-report.md).

Priorities:

- **P0**: in progress or blocked only on a tenant run that is already prepared.
- **P1**: a public contract exists; needs a tenant check or a service key, then implementation.
- **P2**: a public contract is partly confirmed; needs more evidence first.
- **P3**: valuable but low demand or high effort.
- **WATCH**: no public API today; revisit when SAP publishes one.
- **SEPARATE PROVIDERS** and **OUT OF SCOPE**: deliberately not in this provider.

## Status, September 2026

The provider covers Cloud Integration content and deployments, Security Content, Partner
Directory, access policies, number ranges, Classic API Management (API providers, products, key
value maps, certificate store references and, experimentally, API proxies) and, experimentally,
API Composition's business data graphs.

Since the September 2026 re-audit, every OData client declares its contract, and the contract is
checked against committed snapshots of the services' `$metadata` in every CI run. A discovery
command compares the snapshots with a tenant's live documents and reports new or changed
entities; every entity set of every snapshot is classified as used, a candidate or excluded, and
a test fails when SAP adds one that nobody has looked at. Acceptance tests run per capability
behind explicit gates (see `CONTRIBUTING.md`).

The re-audit of current API Management (API artifacts, reusable API artifacts, MCP servers,
Integration Cell, virtual hosts, runtime profiles) against SAP's 2026 releases found no public
API: the Business Accelerator Hub's API Management package (2026-09-24), the Client SDK 3.0.6
(2026-09-24) and SAP Help (2026-09-18) describe these objects only in the UI. They travel as
integration package content. See `docs/guides/current-api-management.md`.

## P0 — tenant runs that decide promotions

These need no further research, only a run of the prepared tests (`.specs/run-all-probes.ps1
-All` locally, or the acceptance workflow):

| Item | Test | Decides |
|---|---|---|
| Classic API proxy | `TestAccAPIProxy_sample` | `sapintegrationsuite_api_proxy` from experimental to supported (replace-only) |
| Live contract check | `TestAccMetadata` (`SAP_INTEGRATION_SUITE_ACC_METADATA`) | First comparison of the committed snapshots with a tenant; strict mode lists new SAP entities |
| Resources verified only by probes | the credential, certificate, key pair, number range, Partner Directory, access policy, key value map and API product acceptance tests | Terraform-level confirmation of state, drift and import |
| Deployments with longer timeouts | message mapping and value mapping acceptance tests | Whether slow deployments need more than the documented timeout advice |
| Design-time types, value mapping entries, security kinds | `tenant-probe -GapTests` | The P1 items below |

## P1 — public contract, next implementation

1. **Data types, message types, fault message types, service interfaces.** The `$metadata`
   defines all four with `SaveAsVersion`, and a package export carries data types in the same
   bundle format as message mappings. The first gap probe (2026-09-27) confirmed reading, but a
   create with the message mapping's body failed with 500; the next probe adds `Namespace`,
   `Description` and `IsSimpleType`. If create, update, version and delete then work, the four
   resources reuse the shared design-time client and the `save_as_version` attribute.
2. **API Composition hardening.** `TestAccBusinessDataGraph_basic` needs a service key of plan
   `configuration` and a destination; the service's `$metadata` goes through
   `cmd/apidiscovery` (service `api-composition-configuration`). Passing both promotes the
   business data graph from experimental and settles the PATCH body and delete request.
3. **Edge Integration Cell targeting.** `TestAccEdgeIntegrationCell_securityAndPartnerDirectory`
   needs a tenant with an Edge Integration Cell. Passing it makes `runtime_location_id` supported
   for the tested resources; the deployment resources follow with their own test.
4. **Integration Assessment.** Fetch the Entities and Management `$metadata` with a service key
   (services `integration-assessment-entities` and `-management` in `cmd/apidiscovery`),
   classify the entity sets, then implement the landscape configuration (applications,
   application instances, technologies, technology instances, vendors) where writes are
   verified. Requests stay out of scope as workflow.

## P2 — partly confirmed, more evidence first

5. **Value mapping entries — settled, not manageable.** The gap probe of 2026-09-27 showed that
   the first `UpsertValMaps` drops the pair's design-time values, a repeated source value adds a
   duplicate instead of an update, and `DeleteValMaps` removes nothing. Entries stay part of the
   value mapping's content; this item moves to WATCH until SAP documents an update and a delete
   that work.
6. **Security content.** Certificate chains (upload media type), PGP keyrings (upload format;
   secret keyrings write-only) and OAuth2 client credential custom parameters (write path). The
   OAuth2 password and SAML bearer artifacts become user credential kinds if the gap probe finds
   them among `UserCredentials`.
7. **Classic virtual hosts.** Write API documented (`Configuration.svc/VirtualHostRequests`) and
   read schema confirmed; needs a key with `APIManagement.SelfService.Administrator` and a
   destructive-gated tenant test, since virtual hosts are tenant-wide.
8. **Data Space Integration.** Parse the DSIAPI OpenAPI document (`cmd/apidiscovery -from`) once
   it is available, then model only desired-state configuration (assets, policies, contract
   definitions); negotiations and transfers stay out of scope.

## P3 — later

9. Value mapping in-place update and `save_as_version` (needs a verified PUT).
10. Integration adapters with real adapter content (create and read still undocumented).
11. Classic certificate stores, applications, cache resources, rate plans, policy templates and
    product access control, each once SAP documents update and delete.
12. Discovery automation: a scheduled strict discovery run that opens an issue with the semantic
    diff instead of failing silently.

## WATCH — no public API today

- Current API Management: API artifacts, reusable API artifacts, API artifact deployment,
  policies, MCP servers and the MCP Gateway, runtime profiles; Integration Cell runtime and
  virtual hosts. Channels: the Hub's API Management and Cloud Integration packages, the Client
  SDK on Maven Central, What's New for Integrations and APIs. The design for an API artifact
  resource is in `docs/guides/current-api-management.md`.
- Capability activation for every Integration Suite capability.
- Known hosts, certificate-to-user mapping (Neo only today), where-used lists.
- OData Provisioning, Integration Advisor, Trading Partner Management configuration, Migration
  Assessment.

## SEPARATE PROVIDERS

- **Developer Hub**: its own API boundary and credentials; planned as
  `Prideth/terraform-provider-sap-developer-hub`. MCP server products and subscriptions belong
  there.
- **Event Mesh**: a general BTP messaging service with its own broker APIs.

## OUT OF SCOPE

- Runtime and monitoring data: message processing logs, message stores, JMS queues, data
  stores, variables, trace, logs, B2B interchange monitoring.
- Workflows and one-shot actions: Integration Assessment requests, Migration Assessment
  extraction and evaluation, Integration Advisor injection, agreement activation, restarts and
  retries.
- Edge Integration Cell local and Operations Cockpit APIs (runtime data and Kubernetes-level
  settings), OAuth2 authorization codes (interactive consent), Open Connectors.
- BTP control-plane objects (subaccounts, entitlements, subscriptions, destinations, role
  collections): use the `SAP/btp` provider.

## Implemented

- Provider skeleton on the Terraform Plugin Framework, named `sapintegrationsuite`
- OAuth 2.0 client credentials authentication with a thread-safe, context-aware token cache
- Shared HTTP client: retry with backoff/jitter, `Retry-After` handling, bounded response
  sizes, custom User-Agent
- OData V2 request/query/pagination/error-parsing foundation
- Cloud Integration fundamentals:
  - `sapintegrationsuite_integration_package`
  - `data.sapintegrationsuite_integration_package`
  - `sapintegrationsuite_integration_flow` (uploads a copy whose bundle ID equals the flow ID,
    since SAP rejects updates otherwise)
  - `sapintegrationsuite_integration_flow_configuration` (externalized parameters, including
    `SAP_ProfileId`, which picks the runtime a flow is deployed to)
  - `sapintegrationsuite_integration_flow_deployment` (follows SAP's deploy task; a deployment
    that does not confirm stays in state as tainted)
- Access Policies (wire contract re-audited in 2026 against SAP's own tooling; lookup by role
  name; runtime targeting still open — see `docs/guides/access-policies.md`):
  - `sapintegrationsuite_access_policy`
  - `sapintegrationsuite_access_policy_reference`
  - `data.sapintegrationsuite_access_policy`
  - `data.sapintegrationsuite_access_policy_reference`
  - `data.sapintegrationsuite_access_policy_runtime_assignments`
- Value Mappings:
  - `sapintegrationsuite_value_mapping`
  - `sapintegrationsuite_value_mapping_deployment`
  - `data.sapintegrationsuite_value_mapping`
- Message Mappings (reusable, package-level artifacts — not the inline/local mapping step an
  integration flow can also define directly, see `docs/resource-design.md`):
  - `sapintegrationsuite_message_mapping`
  - `sapintegrationsuite_message_mapping_deployment`
  - `data.sapintegrationsuite_message_mapping`
- Script Collections (reusable Groovy/JavaScript bundles, same design-time/runtime split):
  - `sapintegrationsuite_script_collection`
  - `sapintegrationsuite_script_collection_deployment`
  - `data.sapintegrationsuite_script_collection`
- Partner Directory (string/binary parameters, alternative partners, authorized users, and a
  security-sensitive write-only user credential parameter resource — see
  `docs/guides/partner-directory.md`; no `sapintegrationsuite_partner` resource exists, since
  SAP documents no confirmed create operation for it, only discovery data sources):
  - `sapintegrationsuite_partner_string_parameter`
  - `sapintegrationsuite_partner_binary_parameter`
  - `sapintegrationsuite_alternative_partner`
  - `sapintegrationsuite_partner_authorized_user`
  - `sapintegrationsuite_partner_user_credential_parameter`
  - `data.sapintegrationsuite_partner`
  - `data.sapintegrationsuite_partners`
  - `data.sapintegrationsuite_partner_string_parameter`
  - `data.sapintegrationsuite_partner_string_parameters`
  - `data.sapintegrationsuite_partner_binary_parameter`
  - `data.sapintegrationsuite_alternative_partner`
  - `data.sapintegrationsuite_partner_authorized_user`
- Security Content credentials (write-only secrets, in-place redeploy via `PUT` — see
  `docs/guides/security-content.md` for the full boundary):
  - `sapintegrationsuite_user_credential`
  - `data.sapintegrationsuite_user_credential`
  - `sapintegrationsuite_oauth2_client_credential`
  - `data.sapintegrationsuite_oauth2_client_credential`
  - `sapintegrationsuite_secure_parameter` (Security Content "Secure Parameter", value write-only;
    the entity set comes from the tenant `$metadata`, SAP Help documents only the UI)
- Security Content keystore management (read-only entry discovery, X.509 certificates, and
  SAP-generated key pairs — private key material never enters this provider; certificate drift
  detection uses a locally-computed SHA-256 fingerprint, not raw PEM text; no separate "SSH Key"
  resource, since SAP's own documentation treats it as the same Key Pair mechanism — see
  `docs/guides/security-content.md`):
  - `data.sapintegrationsuite_keystore_entry`
  - `data.sapintegrationsuite_keystore_entries`
  - `sapintegrationsuite_certificate`
  - `sapintegrationsuite_key_pair`
- Cloud Integration Service Endpoints discovery (read-only by design — SAP generates these from
  deployed content, so there is no matching resource; combined `EntryPoints`/`ApiDefinitions`
  expansion, `name`/`protocol` filters, full server-driven pagination, deterministic ordering —
  see `docs/guides/service-endpoints.md`):
  - `data.sapintegrationsuite_service_endpoints`
- Custom Integration Adapter design-time and deployment support (Cloud Foundry only;
  conservative replace-on-any-change model since SAP documents a duplicate-ID import as an
  error and no in-place update was confirmed; no `*_version` attribute on the deployment
  resource, since the confirmed deploy action takes no `Version` parameter — see
  `docs/guides/integration-adapters.md`):
  - `sapintegrationsuite_integration_adapter`
  - `data.sapintegrationsuite_integration_adapter`
  - `sapintegrationsuite_integration_adapter_deployment`
- Custom Tag Configuration management (a tenant-wide singleton; Create/Read/Update confirmed
  verbatim against SAP's own documented example payloads; Delete deliberately returns an
  explicit error rather than a guessed clearing mechanism, since SAP documents no delete or
  clear operation for this entity at all — see `docs/guides/custom-tag-configurations.md`):
  - `sapintegrationsuite_custom_tag_configuration`
  - `data.sapintegrationsuite_custom_tag_configuration`
- Number Range configuration (the runtime counter is a version-gated write-only attribute,
  never an ordinary reconciled field; SAP documents only create and update, but a tenant check
  confirmed read and delete, so the resource detects drift, imports and deletes — see
  `docs/guides/runtime-stores-and-number-ranges.md`. Variables, Data Stores, and Data Store
  Entries were evaluated and are deliberately unsupported/out of scope — no independent
  creation API, and/or the object is runtime business data, not desired state):
  - `sapintegrationsuite_number_range`
- A shared, CSRF-aware HTTP client: every modifying (POST/PUT/PATCH/DELETE) request now
  transparently fetches and retries with an `X-CSRF-Token` if SAP's API asks for one,
  independently of OAuth authentication — see `docs/sap-api-references.md`
- A machine-readable provider feature support catalog, queryable with no SAP tenant
  credentials — see `docs/feature-support.md`:
  - `data.sapintegrationsuite_provider_features`
  - `data.sapintegrationsuite_provider_feature`
- Import support and drift detection for every resource above
- Unit test suite (`httptest`-based) for auth, HTTP retry, OData v2, and domain mapping
- Contract tests of every wire struct against the services' `$metadata` (property names, keys,
  function imports, field types, navigation properties)
- API discovery: `$metadata` and OpenAPI parsing, committed contract snapshots, semantic diffs,
  entity classification and a generated discovery report (`cmd/apidiscovery`)
- Acceptance tests (`TestAcc*`, gated per capability with `internal/testutil/accgate`;
  `cmd/accplan` shows what would run) for packages, integration flows and their
  configuration and deployment, message mappings, script collections and value mappings, using
  pinned content from SAP's public samples and, where those lack a type, local exports — see
  `CONTRIBUTING.md`
- Generated provider documentation, greenfield/brownfield examples
- CI (fmt/vet/test/build/lint) and GoReleaser-based release pipeline
- Edge Integration Cell classification (registration/activation confirmed UI/CLI-only; Access
  Policy replication has an undocumented public API surface; local monitoring and Operations
  Cockpit APIs confirmed real but out of scope) — see `docs/guides/edge-integration-cell.md`
- Classic API Management (a separate `provider.api_management` credential block and
  `internal/client/apimanagementclassic`; API Provider, API Product, Certificate Store Reference,
  and Key Value Map resources and data sources, each scoped to what SAP's `Management.svc` API
  actually confirms; API proxies experimentally, from their bundle ZIP, uploaded as SAP's
  Client SDK does it) — see `docs/guides/classic-api-management.md`:
  - `sapintegrationsuite_api_provider`
  - `data.sapintegrationsuite_api_provider`
  - `data.sapintegrationsuite_api_providers`
  - `sapintegrationsuite_api_product` (replace-only: SAP answers every update with 405)
  - `data.sapintegrationsuite_api_product`
  - `sapintegrationsuite_api_proxy` (experimental; a new bundle replaces the proxy)
  - `sapintegrationsuite_api_management_certificate_store_reference`
  - `data.sapintegrationsuite_api_management_certificate_store_reference`
  - `sapintegrationsuite_api_key_value_map`
  - `data.sapintegrationsuite_api_key_value_map`
- API Composition business data graph, experimental (separate `provider.api_composition`
  credentials from an API Composition instance with plan `configuration`; PATCH body and delete
  request inferred) — see `docs/guides/api-composition.md`:
  - `sapintegrationsuite_business_data_graph`
  - `data.sapintegrationsuite_business_data_graph`
