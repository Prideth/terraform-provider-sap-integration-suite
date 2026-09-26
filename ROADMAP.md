# Roadmap

This roadmap is adjusted after each round of API discovery documented under `docs/`. It
favors a small, high-quality resource set over broad but shallow coverage. Status labels used
below: **Implemented**, **Partial**, **Next**, **Planned**, **Blocked by API**, **Research
required**, **Out of scope**.

For what this provider *version* actually supports today, feature by feature, see
[`docs/feature-support.md`](docs/feature-support.md) and the
`sapintegrationsuite_provider_features` / `sapintegrationsuite_provider_feature` data sources —
this document describes direction and sequencing, not the authoritative current support
matrix.

## Current status

Every SAP Integration Suite capability this provider knows about (Cloud Integration, Security
Content, Partner Directory, Classic API Management, current API Management/API Artifacts,
Integration Cell, Edge Integration Cell, Developer Hub, API Composition, Integration Assessment,
Trading Partner Management, Integration Advisor, Migration Assessment, Data Space Integration,
Event Mesh, Open Connectors, OData Provisioning) has been researched and classified.

An earlier version of this section called public API coverage complete. The September 2026
re-audit showed that some contracts counted as confirmed had been reconstructed from UI labels
and did not match SAP's services (access policy references, integration adapters, service
endpoint API definitions). Those are fixed, and the Cloud Integration, Security Content,
Partner Directory and Classic API Management clients are now checked against the services'
`$metadata` by contract tests, including field types and navigation properties.

The same month brought the first checks against a real tenant, which found more than a dozen
defects that documentation research and unit tests could not: among them every request
failing under Terraform with `context canceled` on the token URL, SAP answering writes with 202
or 200 and no body, updates it rejects (PUT on access policies and API products, empty IDs on
message mapping updates), certificates that need a fingerprint confirmation, and content
updates that fail when a ZIP's bundle ID differs from the artifact ID. Each fix is listed in
`CHANGELOG.md`. Two kinds of checks now exist:

- **Probe scripts** send the provider's exact requests to a tenant and record SAP's answers.
  They verified the write paths of access policies, packages, Security Content (credentials,
  secure parameters, certificates, key pairs), Partner Directory, number ranges, API products,
  key value maps and the design-time content types.
- **Acceptance tests** (`TestAcc*`) run the provider itself through Terraform, with content
  from SAP's public samples and from local exports (see `CONTRIBUTING.md`). Packages,
  integration flows with content updates, import, configuration and `save_as_version`, flow
  deployments and script collections pass end to end; the message mapping and value mapping
  tests are waiting for a rerun with longer deployment timeouts.

What remains open is listed with the evidence each item needs in
[`docs/research/sap-2026-public-api-gap-closure.md`](docs/research/sap-2026-public-api-gap-closure.md):
chiefly the Hub specifications for the Transport API and Data Space Integration, service-key
access for API Composition and Integration Assessment, a virtual host key, and acceptance tests
for the resources that so far were only probed.

API Composition's business data graph is implemented as experimental: SAP documents its
contract, but the PATCH body and delete request are inferred and unverified on a live system
(see `docs/guides/api-composition.md`). Edge Integration Cell targeting is **not supported**:
an untested `runtime_location_id` remains on several resources, but verifying it is outside
this provider's scope (see `docs/guides/edge-integration-cell.md`).

Future work follows newly published SAP APIs and incoming feature requests, not a fixed backlog.
The clearest concrete opportunities already identified, in case SAP's documented API surface
moves first, are listed in the "Confirmed-but-not-yet-implemented capabilities" section of
`docs/provider-scope.md` — Classic API Management's API Proxy content upload, Data Space
Integration, and Integration Assessment's Landscape Configuration.

The remaining Integration Suite capability audit is done. Two capability areas had no prior
catalog entry at all and were added: **API Composition** (Business Data Graph, activated
alongside Developer Hub as a sub-capability of API Management) was, at the time of this audit,
the strongest confirmed-but-unimplemented finding of the whole provider; it has since been
implemented, see "Implemented" below — and **OData Provisioning**, which the September 2026
re-audit found to have no public management API (the `ODPAPIAccess` role grants runtime access
to registered services, not management). Three previously-placeholder entries
were reclassified with real evidence: **Event Mesh**
is now `separate_provider` (a genuine, well-documented Solace PubSub+ broker API, deliberately
excluded because it predates Integration Suite and belongs to a different provider's boundary,
the same reasoning already applied to Developer Hub); **Data Space Integration** is confirmed
`research_required` with a real, separately credentialed REST API and a genuinely complex
object model warranting its own future phase; **Open Connectors** is now `out_of_scope` as a
deliberate judgment (a catalog of 170+ independent third-party connector types that does not fit
this provider's schema-first design), not a research gap. Capability activation itself was
reconfirmed, once more, to have no public API for any capability this provider has ever audited
— see "Implemented" below, `docs/provider-scope.md`, and `docs/sap-api-references.md`.

Migration Assessment research is done — no public API was found for Source System registration,
Data Extraction Requests, or Scenario Evaluation Requests across Migration Assessment's small
(roughly fifteen-page) documentation tree. Uniquely among the capabilities audited this run, this
conclusion would hold even if an API were confirmed: Migration Assessment's documentation
describes it as an API *consumer* (reaching into a registered source system's own SAP Process
Orchestration APIs to extract data), and every one of its own objects is either action-triggered
workflow (Create starts an extraction with a resulting status) or reporting output (assessment
categories, readiness scores, effort estimates) — see "Implemented" below and
`docs/guides/migration-assessment.md`.

Integration Advisor research is done — no public API was found for any design-time object (MIG,
MAG, Type System, Codelist, Shared Code, Global Parameters) across roughly eighty-five
documentation pages checked. One page describing OAuth credential creation initially looked
promising but, on full reading, turned out to describe authenticating against Cloud Integration
for runtime-artifact injection, not a credential for Integration Advisor's own content — recorded
explicitly as a research caution for future phases. Injection itself is a confirmed UI wizard
with no REST equivalent, and is out of scope as an imperative action regardless — see
"Implemented" below and `docs/guides/integration-advisor.md`.

Trading Partner Management research is done — no public API was found for any design-time object
(Company/Trading Partner/Communication Partner Profile, Agreement Template, Agreement) across
roughly ninety documentation pages checked, using the same negative-evidence method (absence of a
dedicated API-access page) that has proven reliable for every capability audited so far.
Activating a trading partner agreement is confirmed to push generated entries into Partner
Directory, which this provider already manages directly and continues to treat as the correct,
existing surface for that runtime data — see "Implemented" below and
`docs/guides/trading-partner-management.md`.

Integration Assessment is done for its currently reachable public API surface — a separate BTP
service subscription, dual-base-URL OAuth authentication, and a fully confirmed 19-entity
inventory are all real findings, but no field-level wire contract was confirmed for any entity
despite checking the SAP-docs mirror, an official 2400-line PDF user guide, and SAP's own TechEd
hands-on sample repository, so nothing is implemented. Landscape Configuration
(Application/Technology/Vendor) is flagged as the strongest candidate if a schema is ever
confirmed; Requests/assessment workflow is out of scope regardless — see "Implemented" below and
`docs/guides/integration-assessment.md`.

Edge Integration Cell is done — SAP-side registration and activation remain UI/CLI-only (no public
API), Access Policy replication has a public but undocumented API surface
(`public_api_incomplete`), local monitoring APIs are confirmed real but out of scope as runtime data, and the Operations Cockpit API is confirmed real but excluded as
Kubernetes-adjacent operational configuration — see "Implemented" below and
`docs/guides/edge-integration-cell.md`.

Classic API Management is also done for its currently reachable public API surface — API
Provider, API Product, Certificate Store Reference, and Key Value Map are implemented (each with
a scope limitation specific to what SAP's API actually supports); API Proxy, API Proxy
Deployment, and Policy remain unimplemented because this phase could not confirm the API Proxy
content upload wire format from any reachable primary source, not because no API exists — see
"Implemented" below and `docs/guides/classic-api-management.md` for the full boundary and
exactly what would need to be confirmed to revisit API Proxy.

Current API Management / API Artifacts / Integration Cell (API Artifacts, MCP Servers, Runtime
Profiles, Integration Cell, Virtual Hosts, Policies, Reusable API Artifacts) was re-audited in
September 2026 after SAP's 2026 releases (API-centric integration, MCP Gateway, Client SDK
3.0.0). SAP still exposes no public API for any object in this family, so this provider
implements none of it — see `docs/guides/current-api-management.md` for the evidence and for the
design that applies once an API appears. The channels to watch are listed at the end of that
guide.

Developer Hub — SAP's API/Event/MCP Server catalog, publication, and subscription capability — is
not part of this provider's roadmap at all. It has its own API boundary, its own OAuth
credentials, and a consumer/catalog lifecycle unlike anything else this provider manages, and is
planned as a separate, independently versioned Terraform provider (working name
`Prideth/terraform-provider-sap-developer-hub`). See `docs/provider-scope.md` for the boundary
statement and `docs/feature-support.md`'s single `developer_hub` catalog entry.

Access Policies were re-audited in September 2026 against SAP's own access-policy automation and
then verified on a tenant. The re-audit corrected the artifact-reference wire format (property
names, Int64 keys, create and delete paths), removed the non-existent `reconciliation_status`,
added lookup by role name and a read-only runtime assignments data source; the tenant check
showed that description updates need PATCH (PUT answers 501) and that creates can return no
body — see "Implemented" below and `docs/guides/access-policies.md`. Managing runtime
assignments stays open until SAP documents how they are written.
Partner Directory is also done —
see "Implemented" below and `docs/guides/partner-directory.md`. Cloud Integration Service
Endpoints discovery is also done — see "Implemented" below and `docs/guides/service-endpoints.md`.
Security Content (User Credentials, OAuth2 Client Credentials, Secure Parameters, Keystore
Entry discovery, Certificates, and SAP-generated Key Pairs), custom Integration Adapter
design-time/deployment support, and Custom Tag Configuration management are also done, each at
a `partial` or better support level — see "Implemented" below, `docs/guides/security-content.md`,
`docs/guides/integration-adapters.md`, and `docs/guides/custom-tag-configurations.md`. Number
Ranges / Variables / Data Stores is also done — a Number Range resource whose counter is a
version-gated write-only attribute (SAP documents only create and update, but a tenant check
confirmed read and delete, so it also detects drift and imports), with Variables, Data Stores,
and Data Store Entries deliberately left unsupported/out of scope — see "Implemented" below and
`docs/guides/runtime-stores-and-number-ranges.md`.

This order describes what to work on **next**; it does not retroactively unimplement anything
already shipped (see "Implemented" below).

## Implemented (v0.1.x — Provider Foundation)

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
  `docs/guides/security-content.md`; most other Security Content artifact types remain
  unimplemented, see `docs/guides/security-content.md` for the full boundary):
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
- Acceptance tests (`TestAcc*`, gated on `TF_ACC=1`) for packages, integration flows and their
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
  actually confirms; API Proxy, API Proxy Deployment, and Policy deliberately unimplemented
  pending a confirmed content-upload wire format) — see
  `docs/guides/classic-api-management.md`:
  - `sapintegrationsuite_api_provider`
  - `data.sapintegrationsuite_api_provider`
  - `data.sapintegrationsuite_api_providers`
  - `sapintegrationsuite_api_product` (replace-only: SAP answers every update with 405)
  - `data.sapintegrationsuite_api_product`
  - `sapintegrationsuite_api_management_certificate_store_reference`
  - `data.sapintegrationsuite_api_management_certificate_store_reference`
  - `sapintegrationsuite_api_key_value_map`
  - `data.sapintegrationsuite_api_key_value_map`
- API Composition business data graph, experimental (separate `provider.api_composition`
  credentials from an API Composition instance with plan `configuration`; PATCH body and delete
  request inferred) — see `docs/guides/api-composition.md`:
  - `sapintegrationsuite_business_data_graph`
  - `data.sapintegrationsuite_business_data_graph`

## v0.2.x

- Value mapping entry-level management (`UpsertValMaps`, `UpdateDefaultValMap`,
  `DeleteValMaps`), once the exact payload/path shapes and delete granularity are confirmed
  against a reachable primary source or a live tenant — see `docs/resource-design.md`
  (**Research required**)
- Message mapping entry-level or dependent-resource management, if SAP ever exposes one
  independent of the opaque content archive this provider already transports (**Research
  required**)
- `sapintegrationsuite_api_proxy` and its deployment/policy siblings, once the API Proxy content
  upload wire format (multipart, base64 JSON field, or otherwise) is confirmed from a reachable
  primary source — see `docs/guides/classic-api-management.md` (**Research required**)
- Integration Assessment Landscape Configuration (Application/Application Instance/Technology/
  Technology Instance/Vendor), once a field-level wire contract is confirmed for it — see
  `docs/guides/integration-assessment.md` (**Research required**)
- Data Space Integration (Connector/Asset/Policy/Contract Definition/Contract Negotiation), once
  its field-level wire contract is confirmed — see `docs/sap-api-references.md` (**Research
  required**)

## v0.3.x

The provider hardening pass is done — see "Current status" above.

## Later

- Any further Integration-Suite capability for which SAP publishes a stable, documented
  public API, evaluated resource-by-resource against `docs/resource-design.md`'s suitability
  checklist before it is added.

## Explicitly not planned (Out of scope)

- Anything listed as "Out of Scope" in `docs/provider-scope.md`
- A generic `sapintegrationsuite_capability` resource, unless SAP publishes a public
  capability-activation API (none exists today — see
  `docs/provisioning-capability-matrix.md`)
- Terraform data sources for logs, metrics, or events (see `docs/architecture.md` and
  `docs/provider-scope.md` §66 — this provider is not a monitoring system)
