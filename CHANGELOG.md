# Changelog

All notable changes to this project are documented in this file.

## Unreleased

### Fixed

- Every request failed under Terraform with `context canceled` on the token
  URL. The provider built its OAuth client with the context of the
  ConfigureProvider call, which Terraform cancels as soon as that call
  returns, so the first token fetch in a resource was already cancelled.
  Tokens are now fetched with a context that is not tied to that call, and
  each token request has its own 60-second limit. The first acceptance test
  run against a real tenant found this; unit tests could not, because they
  never cancel the context.
- A deployment that SAP accepted but that did not reach `STARTED` before
  its timeout (or ended in `ERROR`) was dropped from state and kept running
  on the tenant untracked; `terraform destroy` then left it behind. All
  deployment resources now keep such a deployment in state with its last
  runtime status, so Terraform marks it tainted and the next apply or
  destroy undeploys it. On a tenant, message mapping and value mapping
  deployments sometimes stayed `STARTING` for several minutes before
  starting; raise `timeouts.create` if that is common on yours.
- Changing the content of an integration flow, message mapping or script
  collection failed on a real tenant. The update sent empty `Id` and
  `PackageId` fields, which SAP rejects for message mappings ("Update of
  PackageId and Id are not allowed"), and SAP answers a successful update
  with 200 and no body, which the provider tried to decode. Updates now send
  only `Name` and `ArtifactContent` and read the new version back.
- An integration flow or message mapping whose ZIP was exported under a
  different ID could be created but never updated. SAP writes the artifact
  ID into `Bundle-SymbolicName` on create and rejects every later content
  update whose bundle ID differs (400 for flows, 500
  `BUNDLE_SYMBOLIC_NAME_CANNOT_BE_UPDATED` for mappings). The provider now
  uploads a copy that carries the artifact ID (and, for mappings, the same
  name in `Provide-Capability`) and shows a warning; the local file is not
  changed, and `content_hash` still refers to it.
- `sapintegrationsuite_integration_flow_deployment` waited until its timeout
  when SAP deployed a flow to another runtime. A flow with the externalized
  parameter `SAP_ProfileId = "integrationcell"` is deployed successfully,
  but to Integration Cell, and never appears in the Cloud Integration
  runtime. The resource now follows the task the deploy returns
  (`BuildAndDeployStatus`) and stops with an explanation once the task has
  succeeded and the flow is still missing, or at once when the task fails.
  The integration flow configuration guide shows how to set
  `SAP_ProfileId` to `iflmap`.

- Access policy artifact references now use the property names SAP's
  `ArtifactReferences` entity actually has (`Name`, `Description`, `Type`,
  `ConditionAttribute`, `ConditionType`, `ConditionValue`). Earlier releases
  sent invented names (`ArtifactType`, `Attribute`, `Operator`, `Value`) and
  could not create a reference on a real tenant. The contract was confirmed
  from SAP's own access-policy automation in
  `SAP/cicd-actions-for-sap-integration-suite`.
- Access policies and references are now addressed with `Edm.Int64` keys
  (`AccessPolicies(1901L)`) instead of quoted string keys. References are
  created and deleted through the top-level `ArtifactReferences` entity set.
- **Breaking:** `sapintegrationsuite_integration_adapter` and its data source
  no longer have `type` and `application`. A tenant `$metadata` document
  shows that `IntegrationAdapterDesigntimeArtifact` has no such properties;
  sending them on create was a guess based on the UI's import dialog. Both
  now expose the read-only `description` SAP takes from the *.esa file, and
  the data source also returns `package_id`.
- **Breaking:** `data.sapintegrationsuite_service_endpoints` returns
  `api_definitions[].name` instead of `api_definitions[].type`. The API
  definition entity has `Url` and `Name`, not `Type`, so `type` was always
  empty. Endpoints now also carry `id`, `title`, `version`, `summary`,
  `description` and `last_updated`, and entry points carry
  `additional_information`.
- The Partner Directory resources (string and binary parameters, alternative
  partners, authorized users) no longer drop `runtime_location_id` from state
  on update, and the user credential parameter no longer drops it on create
  and read. With an Edge Integration Cell location set, Terraform previously
  reported an inconsistent result or planned a replacement on every run.
- **Breaking:** `sapintegrationsuite_partner_authorized_user` and its data
  source reject `user` values with uppercase letters. SAP stores authorized
  users lowercased (its own example creates `MyUser` and returns `myuser`),
  so a mixed-case value could never match what SAP reported back and failed
  after apply.
- Creating OAuth2 client credentials failed on a real tenant after SAP had
  already created them: SAP answers the POST with `202 Accepted` and an
  empty body, which the client tried to decode. The next apply then failed
  because the credential existed. The Security Content creates (OAuth2 and
  user credentials) and the Partner Directory creates now read the entry
  back when the response has no body, and other creates report an empty
  response clearly instead of a JSON parse error. Access policy creates do
  the same: a policy is found by its role name, a reference by its content
  in the policy's reference list, since SAP assigns both IDs.
- Updating a `sapintegrationsuite_number_range` without changing
  `current_value_wo_version` failed on a real tenant. The update left out
  `CurrentValue` to avoid resetting the counter, but SAP rejects a PUT
  without it (500, nothing changed). The update now reads the live counter
  right before the PUT and sends it back unchanged. Names with hyphens,
  which SAP also rejects with a 500, are refused at plan time.
- **Breaking:** `sapintegrationsuite_integration_package` has a new required
  `short_text`. Creating a package failed on a real tenant because SAP
  requires it ("Property 'ShortText' cannot be empty"), and updating one
  failed because SAP answers PATCH with 501; updates now use PUT. Because
  that PUT replaces the package, the provider reads it first and sends its
  version, vendor and tag fields back unchanged; a PUT without them reset
  Version and Vendor to empty on a tenant. SAP stores
  the description as HTML and wraps plain text in `<p>...</p>`; the provider
  removes that wrapper when reading, so plain descriptions no longer show
  as drift. The package data source returns `short_text` as well.
- Creating a `sapintegrationsuite_user_credential` without `kind` failed on
  a real tenant: SAP requires `Kind` and a non-null `Description`. `kind`
  now defaults to `default`, the value SAP reports for a generic
  credential, and `Kind`, `Description` and `CompanyId` are always sent.
- `data.sapintegrationsuite_partner` failed on every real tenant: SAP does
  not support reading a single partner by key. It now filters the partner
  list by Pid.
- The size check for `sapintegrationsuite_partner_binary_parameter` allows
  values up to 1,572,864 bytes, the `MaxLength` the tenant `$metadata`
  declares for `BinaryParameter.Value`, instead of 260 KB. Older SAP pages
  still give the lower figure; the contract tests pin the new value to the
  `$metadata`.
- **Breaking:** `sapintegrationsuite_api_product` could neither create nor
  read a product on a real API Portal. SAP returns the product's links to
  proxies and properties as `__deferred` objects, which the client tried to
  decode as lists. A tenant test also showed that SAP answers every update
  of a product (PUT, PATCH and MERGE) with 405 and rejects a separate
  create of an additional property with 405. The resource now changes as
  follows:
  - Every attribute forces a new product. Replacing a product drops the
    subscriptions of applications that use it, so check plans carefully.
  - `api_proxy_names` is required. SAP refuses a product without a linked
    proxy.
  - `status_code` defaults to `PUBLISHED`. `DRAFT` also works and creates an
    unpublished product. Without a status, SAP's create fails.
  - `version`, `title`, `is_published` and `is_restricted` take the values
    SAP sets when they are left out, for example version `1`.
  - `additional_properties` are sent inside the create request.
  - Refresh and import read the linked proxies and the properties from SAP,
    so drift in both is detected. The data source's `api_proxy_names` was
    always empty and is now filled.
- `sapintegrationsuite_api_key_value_map` and its data source failed to
  create or read a map on a real API Portal for the same reason: SAP returns
  `genericKeyMapEntryValues` as a `__deferred` link. The entries are now read
  through `GenericKeyMapEntries(...)/genericKeyMapEntryValues`.
- Every `*_deployment` resource read the deployment status into a structure
  that expected `ErrorInformation` as text. The tenant `$metadata` declares
  it only as a link to a separate media entity, so a status read would fail
  as soon as SAP includes that link. The status read now leaves it out, and
  a failed deployment reports SAP's error text from
  `IntegrationRuntimeArtifacts('<id>')/ErrorInformation/$value`.
- `sapintegrationsuite_certificate` could not import a self-signed or
  otherwise untrusted certificate, and could not change any certificate
  after creating it. On a tenant, SAP answered the import with `409` and
  status `notImported` until the fingerprint was confirmed, and answered a
  replacement with `400 Entry with alias ... already exists` unless
  `update=true` was sent. The import now sends `fingerprintVerified=true`,
  and an update additionally sends `update=true`. Listing a certificate in
  the configuration is therefore the decision to trust it; compare
  `certificate_sha256` with the fingerprint you expect.
- `save_as_version` on integration flows, message mappings and script
  collections failed on a real tenant after SAP had saved the version: SAP
  answers `...SaveAsVersion` with `200` and an empty body, although the
  `$metadata` declares the artifact as its result. The provider now reads
  the active version back when the body is empty.

### Added

- `sapintegrationsuite_secure_parameter` manages Security Content "Secure
  Parameter" artifacts, the confidential values custom adapters and scripts
  read by alias. SAP Help documents the artifact only in the UI; the entity
  set comes from the tenant `$metadata`, and create, read, update and delete
  were verified on a tenant. The value is write-only
  (`secure_param_wo` / `secure_param_wo_version`) and redeployed in place.
- `sapintegrationsuite_number_range` reads, deletes and imports. SAP
  documents only create and update, but a tenant test confirmed
  `GET NumberRanges('<name>')` and `DELETE`. Read now detects drift in the
  static fields and removes number ranges deleted outside Terraform, the new
  `current_value`, `deployed_by` and `deployed_on` attributes report what SAP
  holds, `terraform destroy` deletes, and `terraform import` works by name.
  The first apply after an import records `current_value_wo_version`
  without touching the counter. Create stops if the name already exists.
- Experimental `sapintegrationsuite_business_data_graph` resource and data
  source for API Composition's Configuration API, with a new optional
  `provider.api_composition` block. Its credentials come from an API
  Composition service instance with plan `configuration`; the Cloud
  Integration and API Portal credentials do not work there. The schema
  follows SAP's configuration file format, including locating cues, key
  mappings with format strategies, `source_entity` and `exclude`. Create and
  Update wait for SAP's asynchronous processing (`PROCESSING` until
  `DEPLOYMENT_INITIATED` or `FAILED`, 20 minutes by default, configurable
  with `timeouts`). A graph that ends in `FAILED` stays in state and is
  tainted. SAP shows no PATCH body and no delete request; the provider sends
  the writable properties and `DELETE` on the graph's URL. See the new API
  Composition guide. This also corrects an earlier research note that named
  the `integration-flow` plan of Process Integration Runtime for
  configuration; that plan is for client applications consuming a graph.
- `save_as_version` on `sapintegrationsuite_integration_flow`,
  `sapintegrationsuite_message_mapping` and
  `sapintegrationsuite_script_collection` saves uploaded content under an
  explicit version through SAP's documented `…SaveAsVersion` function
  imports. A new version is saved only when the value changes.
- An optional `runtime_location_id` on the deployment resources, the user
  and OAuth2 credentials, certificates, key pairs and the Partner Directory
  resources, plus their data sources, sends requests to
  `/location/<id>/api/v1`, the service root SAP Help documents for Edge
  Integration Cells; import IDs accept a `location:<id>/` prefix. **Edge
  Integration Cell targeting is not supported:** it has never been tested
  against a tenant with an Edge Integration Cell and is outside the
  provider's supported scope. Leave the attribute unset.
- `sapintegrationsuite_integration_flow_configuration` sets externalized
  parameters of an integration flow through SAP's documented
  `$links/Configurations` update. Only the listed keys are managed, each
  parameter's data type is kept, and unknown keys are rejected before
  anything is written. Destroy leaves the values in place, since SAP offers
  no way to delete a parameter.
- `sapintegrationsuite_integration_flow_deployment` has a `redeploy_triggers`
  map that redeploys the flow in place when it changes, so new parameter
  values reach the runtime.
- `sapintegrationsuite_oauth2_client_credential` and its data source gain
  `client_authentication`, `scope_content_type`, `resource` and `audience`,
  the property names confirmed by a tenant `$metadata`. They are optional
  and computed, so every update resends the value SAP holds. Previously an
  update, which is a full `PUT`, could reset settings made in the UI.
- The keystore entry data sources return `entry_type`, `owner`, `status`,
  `subject_dn`, `issuer_dn`, `serial_number`, `signature_algorithm`,
  `elliptic_curve`, `certificate_version`, `validity`,
  `fingerprint_sha1/256/512`, `created_by`, `created_time`,
  `last_modified_by` and `last_modified_time`.
- The feature catalog lists OAuth2 Password Credentials, OAuth2 SAML Bearer,
  security material where-used and PGP keyrings with their current status,
  and the Cloud Integration artifact types data type, message type, fault
  message type and service interface, plus explicit design-time versioning.
  Value mapping entries move from "research required" to "public API
  incomplete": create and read are documented, per-entry delete is not.
- Classic API Management: a catalog entry for virtual hosts (documented
  request API, undocumented read schema), and the API Proxy entry now
  records the upload and export calls SAP's Client SDK 3.0.6 makes.
- `data.sapintegrationsuite_access_policy_runtime_assignments` lists the
  runtimes an access policy is replicated to (Cloud Integration runtime,
  Integration Cell, Edge Integration Cells) with SAP's transfer status,
  errors and last status change. Choosing the runtimes remains a UI step
  because writing assignments is not documented.
- Contract tests that check every OData wire struct, key and function
  import against a tenant `$metadata` document when one is available
  locally. See "Checking wire contracts against `$metadata`" in
  CONTRIBUTING.md.

### Changed

- **Breaking:** `sapintegrationsuite_access_policy_reference` has a new
  required `name` and an optional `description`. `artifact_type`, `attribute`
  and `operator` now take SAP's wire constants (for example
  `INTEGRATION_FLOW`, `Name`, `exactString`) and are passed through
  unchanged instead of being checked against a closed list. The old list was
  incomplete (it lacked Integration Package, API, Data Type and Message Type)
  and spelled wrong. `IntegrationFlow` and `EQUALS` are rejected at plan time
  with a pointer to the correct value.
- `data.sapintegrationsuite_access_policy` can look a policy up by
  `role_name` as well as by `id`. The role name is the same in every tenant;
  the ID is not.
- Import IDs for both access policy resources must now be numeric and are
  checked before any request is sent.
- The feature catalog lists MCP servers (`api_gateway.mcp_server`) as a
  current API Management object without a public API. The other Current API
  Management and Integration Cell entries now cite the September 2026
  re-audit, which confirmed that none of them has a public API yet.
- **Behavior change:** the keystore entry data sources return
  `valid_not_before` and `valid_not_after` as RFC 3339 timestamps instead of
  SAP's raw `/Date(...)/` literals. OData V2 date literals with a zone
  offset are now parsed correctly.
- `sapintegrationsuite_partner_user_credential_parameter` rotates passwords
  and changes `user` in place. SAP documents a POST with the same Pid and Id
  as the update for this entity (PUT is not supported), so changing
  `password_wo_version` or `user` no longer deletes and re-creates the
  credential, and integration flows keep a valid credential throughout.
  Because that POST also overwrites, create now fails when the credential
  already exists and asks for an import. The catalog lists the resource as
  supported.
- The feature catalog classifies OData Provisioning as having no public
  management API instead of needing research. The `ODPAPIAccess` role that
  earlier suggested one grants access to registered services at run time;
  registering and configuring services is documented only in the UI.
- The feature catalog has a new entry, `cloud_integration.archiving`, for
  message processing log and B2B payload archiving. Activation exists in the
  API only as a one-way switch without deactivation, so there is no
  resource; the entry explains why. API Composition's protocol is recorded
  as OData V4, and the Classic API Proxy entry names the official Transport
  APIs listed on the Business Accelerator Hub.
- The Classic API Management client is checked against an API portal's
  `Management.svc/$metadata` by a contract test; every property and key it
  uses exists. The same document confirms the virtual host read schema,
  which the catalog listed as the gap, and the API proxy entity. Seven
  entity sets the catalog did not cover yet (certificate stores and
  certificates, applications and developers, key value maps across proxies,
  cache resources, rate plans, policy templates, product access control)
  have catalog entries with what the `$metadata` and the Business
  Accelerator Hub say about them.
- A tenant probe showed that several Security Content and Message Store
  entity sets reject query options (`$top` and `$select` with 501,
  `KeystoreEntries` even `$format` with 400). The provider's reads send
  none; a regression test now keeps it that way, and CONTRIBUTING warns
  about it. The keystore entry data sources describe the values SAP
  actually returns for `entry_type`, `owner`, `status` and `validity`.
- Trading Partner Management, Integration Advisor and Migration Assessment
  were re-audited against the current documentation and the complete
  Business Accelerator Hub package list. Their classification is unchanged;
  the guides say what was checked.

### Documentation

- New research summary `docs/research/sap-2026-public-api-gap-closure.md`:
  method, results per area, corrected earlier conclusions, breaking changes
  and the open items with the evidence each one needs. The ROADMAP no longer
  claims complete public API coverage.
- New guide "Authorization and Roles": the SAP role templates each resource
  family needs, where they are assigned, and how to diagnose a 403.
- The Current API Management guide was rewritten around a per-object status
  table, the 2026 evidence (Integration Content API, Client SDK 3.0.6, SAP's
  CI/CD tooling, What's New), MCP servers, virtual host rules, and what can
  be automated around these objects today.
- The Access Policies guide was rewritten around the confirmed wire
  contract, with lifecycle side effects and upgrade steps.
- The Integration Assessment guide records the September 2026 re-audit:
  both APIs are OData services according to the Business Accelerator Hub,
  their specifications still need an SAP login, and a `$metadata` document
  fetched with a service key is the step that would unblock an
  implementation. CONTRIBUTING describes how to fetch it.
- New guide "Data Space Integration": which objects would suit Terraform
  (assets, policies, contract definitions, company policies, contract
  references), what SAP's API documentation covers (only consumer runtime
  flows under `/api/dsi/v1`), the one-credential-set-per-connector rule, and
  what would unblock an implementation.

### Removed

- **Breaking:** `reconciliation_status` on the access policy resource and
  data source. The `AccessPolicies` entity has no such property; runtime
  replication lives in the `AccessPolicyRuntimeAssignments` navigation
  property, which the new runtime assignments data source reads. Existing
  state is unaffected.

## 0.1.0 - 2026-09-24

Initial release. See `ROADMAP.md` for what is planned next and `docs/` for
the API discovery this release is based on.

### Added

- Provider foundation: OAuth 2.0 client credentials authentication with
  token caching, a retrying HTTP client, and an OData V2 request/pagination/
  error-handling layer.
- `sapintegrationsuite_integration_package` resource and data source.
- `sapintegrationsuite_integration_flow` resource (file-based content).
- `sapintegrationsuite_integration_flow_deployment` resource, with
  context-aware polling instead of fixed sleeps.
- `sapintegrationsuite_access_policy` and
  `sapintegrationsuite_access_policy_reference` resources, plus matching
  `data.sapintegrationsuite_access_policy` and
  `data.sapintegrationsuite_access_policy_reference` data sources. See
  `docs/guides/access-policies.md` for role/BTP semantics, supported
  artifact types/attributes/operators, and runtime reconciliation findings.
- `sapintegrationsuite_value_mapping` resource and data source (file-based
  content), and `sapintegrationsuite_value_mapping_deployment`, sharing the
  runtime-artifact polling and status model already proven for integration
  flow deployments.
- `sapintegrationsuite_message_mapping` resource and data source (file-based
  content), and `sapintegrationsuite_message_mapping_deployment`, for the
  reusable, package-level message mapping artifact — not the inline/local
  message mapping step an integration flow can also define directly inside
  its own content. Unlike `sapintegrationsuite_value_mapping`, this
  resource has a confirmed in-place Update via `PUT`, on entity-specific
  evidence documented in `docs/sap-api-references.md`. Reuses the same
  shared runtime-artifact polling and status model, confirmed applicable to
  this entity type rather than assumed.
- `sapintegrationsuite_script_collection` resource and data source
  (file-based content), and `sapintegrationsuite_script_collection_deployment`,
  for reusable Groovy/JavaScript script bundles. Shares its Update model
  with `sapintegrationsuite_message_mapping` (a confirmed in-place `PUT`,
  on the same entity-specific evidence) and reuses the same shared
  runtime-artifact polling and status model as every other `*_deployment`
  resource.
- Partner Directory support: `sapintegrationsuite_partner_string_parameter`,
  `sapintegrationsuite_partner_binary_parameter` (file-based, with SAP's
  documented 260 KB size limit checked before upload),
  `sapintegrationsuite_alternative_partner` (hiding SAP's hex-encoded
  `Hexagency`/`Hexscheme`/`Hexid` entity key behind plain
  agency/scheme/external_id attributes), and
  `sapintegrationsuite_partner_authorized_user` resources, each with a
  matching data source, plus `data.sapintegrationsuite_partner` and
  `data.sapintegrationsuite_partners` for discovery (there is no
  `sapintegrationsuite_partner` resource: SAP documents no confirmed create
  operation for Partners, and deleting one is documented as cascading to
  every entity belonging to it). See `docs/guides/partner-directory.md`.
- `sapintegrationsuite_partner_user_credential_parameter`, a
  security-sensitive Partner Directory resource using a write-only
  `password_wo` attribute (Terraform CLI 1.11+) paired with a
  `password_wo_version` marker, since Terraform never stores the password
  and this provider never reads one back from SAP.
- CSRF token handling in the shared HTTP client for every modifying
  (POST/PUT/PATCH/DELETE) request: SAP's OData V2 services protect writes
  with an `X-CSRF-Token` independently of OAuth, and this had no handling
  anywhere in this provider before now. Fetches and retries transparently
  when SAP asks for a token; a no-op when it does not.
- Generic server-driven paging (`__next` link following) in the OData v2
  client, so a large collection (Partner Directory's String Parameters in
  particular) is read completely rather than silently truncated to its
  first page.
- A machine-readable provider feature support catalog
  (`internal/features`), queryable via `data.sapintegrationsuite_provider_features`
  and `data.sapintegrationsuite_provider_feature` with no SAP host or
  OAuth credentials required — this provider's `Configure` no longer fails
  just because SAP connectivity is unconfigured; every SAP-backed resource
  and data source instead reports a clear, specific error when actually
  used without one. See `docs/feature-support.md`.
- Provider scope, boundary, architecture, and API discovery documentation.
- `sapintegrationsuite_user_credential` and `sapintegrationsuite_oauth2_client_credential`
  resources (plus matching data sources) for SAP's Security Content API, the first
  Security Content credential artifacts this provider manages. Both use write-only
  `password_wo`/`client_secret_wo` attributes paired with a `_wo_version` marker (the
  same pattern as `sapintegrationsuite_partner_user_credential_parameter`), but unlike
  that resource, both have a confirmed in-place Update via `PUT` (SAP documents an
  "Edit and redeploy" action for Credentials artifacts), so rotating a secret redeploys
  the credential rather than replacing the resource. A dedicated
  `internal/client/securitycontent` client package backs both. See
  `docs/guides/security-content.md` for the full security model, which fields could and
  could not be confirmed, and why most other Security Content artifact types (keystore
  entries, certificates, key pairs, SSH keys, certificate chains, secure parameters,
  known hosts) are not implemented yet.
- `data.sapintegrationsuite_service_endpoints`, a read-only discovery data source for
  SAP's `ServiceEndpoints` API: the runtime entry point URLs and API definition links
  SAP generates for deployed Cloud Integration content. Supports the documented `name`/
  `protocol` filters, combines `EntryPoints`/`ApiDefinitions` expansion into a single
  request, fetches every page via server-driven paging, and sorts its result
  deterministically since SAP does not document a guaranteed response order. There is
  deliberately no matching resource (SAP offers no create/update/delete API for these)
  and no singular per-endpoint lookup (uniqueness of `name` is not confirmed) — see
  `docs/guides/service-endpoints.md`.
- A generated "Feature Support" dashboard in `README.md`, between
  `<!-- BEGIN GENERATED FEATURE SUPPORT -->`/`<!-- END GENERATED FEATURE SUPPORT -->`
  markers, produced by `go run ./cmd/gendocs -readme` (also run by `make docs`) from the
  same `internal/features/catalog.go` that already generates `docs/feature-support.md`,
  so the two can never drift apart into independently maintained copies. Grouped by
  domain, using the ✅/⚠️/👁️/🧪/❌ icon legend, and — unlike the README table it
  replaces — shows unsupported and out-of-scope features alongside supported ones, not
  just a curated list of what works.
- `sapintegrationsuite_integration_adapter`, `data.sapintegrationsuite_integration_adapter`,
  and `sapintegrationsuite_integration_adapter_deployment` for custom Integration Adapter
  design-time and runtime management (Cloud Foundry environment only — SAP does not expose this
  artifact type in Neo). This is the SAP Adapter SDK `*.esa` upload lifecycle, distinct from
  importing a prebundled SAP Business Accelerator Hub adapter from inside the integration flow
  editor and from tenant capability activation. SAP's own "Example Requests" documentation for
  this entity confirms only Delete and Deploy (unlike the complete example set backing every
  sibling design-time artifact type), which confirmed two structural findings that differ from
  every other design-time resource in this provider: the entity is keyed by `Id` alone, not a
  composite `(Id, Version)` key, and the deploy action takes no `Version` query parameter.
  Because SAP documents that importing a duplicate `Id` is rejected as an error, and no
  reimport/update example was found, this resource implements no in-place update at all — every
  attribute is `RequiresReplace`. See `docs/guides/integration-adapters.md` for the full
  breakdown of what is confirmed versus inferred by analogy.
- `sapintegrationsuite_custom_tag_configuration` and its matching data source, for the
  tenant-wide Custom Tag Configuration — the set of attributes integration package owners
  classify their packages with. Unlike every other resource this provider manages, this is a
  tenant-level singleton, addressed by a single confirmed fixed key ("CustomTags"), and Create/
  Update both call the same confirmed `POST .../CustomTagConfigurations?Overwrite=true`
  operation, sending the complete desired tag list every time. `tags` and each tag's
  `permitted_values` are modeled as Terraform sets rather than lists, since SAP's documentation
  never states that submission or response order carries meaning, so a reordered response from
  SAP never produces a spurious plan diff. Delete deliberately returns an explicit error instead
  of a real destroy operation: this project specifically reverified SAP's documentation for a
  delete or clear mechanism and found none anywhere, and guessing that an empty overwrite means
  "delete everything" was rejected as unsafe for tenant-wide governance configuration. See
  `docs/guides/custom-tag-configurations.md`.
- Go 1.27.1 (up from 1.25.0), terraform-plugin-framework v1.19.0 (up from v1.17.0), and
  terraform-plugin-go v0.31.0 (up from v0.29.0), upgraded together since v0.31.0 requires the
  `GenerateResourceConfig` RPC that only landed in framework v1.19.0. Every other dependency,
  direct and transitive, is now on a current stable release rather than whatever minimum version
  selection happened to resolve — including replacing a `google.golang.org/grpc` prerelease
  pseudo-version that had crept into `go.sum` with a real tagged release. See SECURITY.md for
  why grpc is deliberately pinned one release behind the very latest.
- Every GitHub Actions workflow now runs against current action releases
  (`actions/checkout@v7`, `actions/setup-go@v7`, `golangci-lint-action@v9` pinned to
  `golangci-lint` v2.13.2, `hashicorp/setup-terraform@v4`) instead of versions that had drifted
  behind what those actions currently require, which is why lint CI had started failing before
  ever reaching the actual linters.
- `sapintegrationsuite_number_range`, a write-only-lifecycle resource for Cloud Integration
  Number Ranges: SAP documents confirmed `Create`/`Update` for this entity but no `GET` or
  `DELETE` anywhere, so Read is a documented no-op that trusts state rather than contacting SAP,
  and Import/Delete both refuse explicitly with an actionable error instead of guessing at an
  unconfirmed operation. The runtime counter (`CurrentValue`) is a version-gated write-only
  attribute (`current_value_wo`/`current_value_wo_version`), never an ordinary reconciled field,
  so an apply that only changes static configuration can never reset a counter that has since
  advanced through live EDI/EDIFACT processing. Variables, Data Stores, and Data Store Entries
  were evaluated in the same research pass and are deliberately unsupported: no independent
  creation API, and/or the object is runtime business data, not desired-state configuration. See
  `docs/guides/runtime-stores-and-number-ranges.md`.
- Security Content keystore management: `data.sapintegrationsuite_keystore_entry` and
  `data.sapintegrationsuite_keystore_entries` for read-only entry discovery,
  `sapintegrationsuite_certificate` for X.509 certificate lifecycle management (drift detection
  uses a locally-computed SHA-256 fingerprint of the DER bytes, not raw PEM text, so re-wrapped or
  CRLF-converted PEM content never causes a spurious diff), and `sapintegrationsuite_key_pair` for
  SAP-generated key pairs — private key material never enters this provider or its state. There is
  no separate "SSH Key" resource: SAP's own documentation treats it as the same Key Pair mechanism.
  Certificate Chain, Secure Parameter, and Known Hosts were evaluated and remain unimplemented,
  either without a confirmed public contract or without any public API at all. See
  `docs/guides/security-content.md`.
- Research findings for SAP's current, API-artifact-centric API Management model (API Artifacts,
  Runtime Profiles, Integration Cell, Virtual Hosts, Policies, Reusable API Artifacts) and for Edge
  Integration Cell: after a thorough documentation pass covering both areas, this provider found no
  public API for API Artifacts or their deployment, Integration Cell activation/runtime status, or
  Integration Cell/Edge Integration Cell Virtual Hosts — real, UI-documented SAP functionality with
  no REST/OData contract behind it. Edge Integration Cell's own local monitoring API
  (`/local/api/v1`, Message Processing Logs/Message Stores) and Operations Cockpit API
  (`/local/api/eic/v1`) are confirmed real and reachable, but excluded as runtime/monitoring data
  and Kubernetes-adjacent operational configuration respectively, the same category this provider
  already excludes for Cloud Integration's own Message Processing Logs. See
  `docs/guides/current-api-management.md` and `docs/guides/edge-integration-cell.md`.
- Classic API Management: an optional, independent `provider.api_management` configuration block
  (and matching `SAP_INTEGRATION_SUITE_API_MANAGEMENT_*` environment variables) authenticating
  against the API Portal's own `apiportal-apiaccess` service plan, entirely separate from the
  `oauth` block used for Cloud Integration. A new `internal/client/apimanagementclassic` package
  backs four resources, each scoped to exactly what SAP's `Management.svc` OData API confirms:
  `sapintegrationsuite_api_provider` (Create/Read/Delete only — SAP's own Piper tooling documents
  create-only support for this entity — Internet connection type only, with bounded jittered-backoff
  polling after Create to wait out SAP's documented ~20-second eventual-consistency window),
  `sapintegrationsuite_api_product` (full CRUD, confirmed verbatim from SAP's own worked Create/
  Update examples, including reconciled custom `additional_properties`),
  `sapintegrationsuite_api_management_certificate_store_reference` (full CRUD — the best-confirmed
  object in this whole provider, with complete request/response bodies documented for every
  operation), and `sapintegrationsuite_api_key_value_map` (Create/Read/Delete, unencrypted maps
  only — this provider could not confirm how an encrypted entry's value is returned by `GET`, so it
  rejects `encrypted = true` outright rather than risk leaking a secret into state). Matching data
  sources exist for all four. Classic API Proxy, its deployment, and its policy model are
  deliberately not implemented: the entity, its `GET`/`DELETE`, and its ZIP bundle structure are
  all confirmed, but no reachable primary source shows the Create/Update wire format for the bundle
  content itself. See `docs/guides/classic-api-management.md`.
- Research findings, without any resulting resources, for Integration Assessment (a separate BTP
  service with a fully confirmed 19-entity inventory but no confirmed field-level schema for any
  entity), Trading Partner Management (no public API found for any design-time object across
  roughly ninety documentation pages; agreement activation is confirmed to push generated entries
  into the Partner Directory this provider already manages directly), Integration Advisor (no
  public API found across roughly eighty-five pages; artifact injection into Cloud Integration is a
  confirmed UI wizard with no REST equivalent), and Migration Assessment (no public API for its own
  objects — it is documented as an API *consumer* of a registered source system's own SAP Process
  Orchestration APIs — and every object is action-triggered workflow or reporting output by nature,
  so this conclusion would hold even if an API were later confirmed). See
  `docs/guides/integration-assessment.md`, `docs/guides/trading-partner-management.md`,
  `docs/guides/integration-advisor.md`, and `docs/guides/migration-assessment.md`.
- A full sweep of the remaining Integration Suite capability surface against current SAP
  documentation, adding two previously-uncatalogued areas (API Composition's Business Data Graph —
  the strongest confirmed-but-unimplemented finding in this whole catalog, with complete verbatim
  Create/Read/Update worked examples; OData Provisioning) and reclassifying three placeholders with
  real evidence in place of guesses (Event Mesh and Developer Hub are both tracked as
  `separate_provider`, deliberately excluded because each belongs to a different provider's
  boundary by design — Developer Hub is planned as its own, independently versioned Terraform
  provider, working name `Prideth/terraform-provider-sap-developer-hub`; Data Space Integration is
  confirmed `research_required` with a real, separately credentialed API; Open Connectors is a
  deliberate `out_of_scope` judgment, a catalog of 170+ independent third-party connector types
  that does not fit this provider's schema-first design). See `docs/provider-scope.md` and
  `docs/sap-api-references.md`.

### Changed

- `sapintegrationsuite_value_mapping` no longer implements Update via
  `PUT`. Re-verifying the Value Mapping API contract found no confirmed
  in-place update path for this entity set (unlike
  `sapintegrationsuite_integration_flow`'s equivalent, which is confirmed);
  `name`, `content`, and `content_hash` are now `RequiresReplace`, so
  changing any of them replaces the resource instead of relying on an
  unverified `PUT`. See `docs/sap-api-references.md` for the full
  reasoning and `docs/resource-design.md` for what was checked.
- `sapintegrationsuite_access_policy`'s Update now sends a PATCH payload
  containing only `Description`, instead of resending the immutable
  `RoleName` unchanged on every description update.
- The `security.certificate_user_mapping` feature catalog entry is corrected from
  `public_api: true` / `not_implemented` to `public_api: false` / `no_public_api`:
  reverifying it found SAP's certificate-to-user mapping documentation exists only for
  the Neo environment, with no Cloud Foundry equivalent, and this provider targets
  Cloud Foundry.
- `docs/feature-support.md` no longer carries a UTF-8 byte-order mark. `cmd/gendocs` now writes
  it (and `README.md`'s generated feature table) directly with `os.WriteFile` instead of relying
  on a shell to redirect stdout into the file, which was the actual source of the BOM — a
  Windows shell's redirection can prepend one where a POSIX shell's never does, so the file's
  bytes previously depended on which platform last regenerated it.
- Documentation CI now also fails if `README.md`'s generated feature table is out of date, not
  only `docs/`, and its failure message now correctly says to run `make docs` instead of a
  `go generate ./...` command this project has never used.
- The `terraform-fmt` lint job now runs against a small matrix (Terraform 1.11.0, the documented
  floor for write-only attribute support, and 1.16.3, the current stable release) instead of
  whatever `hashicorp/setup-terraform` happened to install by default.
- `sapintegrationsuite_api_provider`'s `password_wo` attribute is now also marked `Sensitive`,
  matching every other credential-shaped write-only attribute in this provider (it was previously
  `WriteOnly` without `Sensitive`, an inconsistency a repository-wide hardening audit found and
  corrected before this release).

### Known limitations

- `sapintegrationsuite_value_mapping` has no in-place update (see Changed
  above); SAP separately documents a `ValueMappingDesigntimeArtifactSaveAsVersion`
  action this provider does not yet use, deferred to v0.2.x pending
  confirmation of its exact contract.
- Whether `sapintegrationsuite_value_mapping`'s,
  `sapintegrationsuite_message_mapping`'s, or
  `sapintegrationsuite_script_collection`'s Delete removes only the active
  version or every version of the artifact has not been confirmed against
  a primary source.
- Individual value mapping entries are not yet manageable through this
  provider — see `docs/resource-design.md`.
- `sapintegrationsuite_access_policy`'s `reconciliation_status` is
  best-effort and not polled to a terminal state: SAP's documentation
  confirms access policies can be replicated to the Cloud Integration
  runtime, Integration Cell, and Edge Integration Cell with a per-runtime
  `Fail`/`Success`/`Pending` reconciliation status, but this project could
  not confirm that mechanism is exposed through the public `AccessPolicies`
  OData API as opposed to being UI-only. See
  `docs/guides/access-policies.md`.
- The exact wire-format casing SAP's `AccessPolicies` OData API expects for
  `Attribute` (`Name`/`Id`) and `Operator` (`EQUALS`/`MATCHES`) enum values
  has not been confirmed against a live tenant or `$metadata`.
- `sapintegrationsuite_partner_user_credential_parameter` has no in-place
  update and no password read-back — a permanent property of its security
  model, not a gap expected to close later. See
  `docs/guides/partner-directory.md`.
- Whether `sapintegrationsuite_partner_authorized_user`'s `user` value is
  case-normalized by SAP internally has not been confirmed against a
  primary source; this provider does not normalize it.
- There is no `sapintegrationsuite_partner` resource. SAP documents no
  confirmed create operation for `Partners`, and deleting one is
  documented as capable of cascading to every entity that belongs to it —
  see `docs/guides/partner-directory.md`.
- `sapintegrationsuite_user_credential` and `sapintegrationsuite_oauth2_client_credential`
  never read a password/client secret back from SAP — a permanent property of their
  security model. `sapintegrationsuite_oauth2_client_credential` also only exposes name,
  description, token service URL, client ID, client secret, and scope; grant type
  placement, client authentication mode, resource, audience, and custom parameters are
  documented by SAP but not yet implemented. See `docs/guides/security-content.md`.
- `data.sapintegrationsuite_service_endpoints`'s `ApiDefinitions[].url` JSON property
  casing is inferred by consistency with the independently confirmed `EntryPoints[].url`
  casing (confirmed from SAP's own open-source Piper library), not independently
  confirmed itself. Whether a fresh deployment's service endpoint appears immediately or
  after a propagation delay is also unconfirmed — see `docs/guides/service-endpoints.md`.
- `sapintegrationsuite_integration_adapter` has no in-place update: every attribute is
  `RequiresReplace`. Its Create request shape is corroborated by analogy to sibling
  design-time artifact types rather than confirmed by an SAP-published example for this
  specific entity, `type`/`application` are not validated against a fixed value set (not
  confirmed as a closed enum), and there is no `sapintegrationsuite_integration_adapters`
  collection data source (no confirmed list/filter contract for this entity set). Its
  deployment resource reuses the shared runtime-artifact status/undeploy mechanism by
  analogy, not independent confirmation for this artifact type. See
  `docs/guides/integration-adapters.md`.
- `sapintegrationsuite_custom_tag_configuration` does not support `terraform destroy`: SAP
  documents no delete or clear operation for the CustomTagConfigurations API at all, and
  Delete returns an explicit error rather than a guessed implementation. Whether
  `Overwrite=true` performs a full replace (removing tags not present in the new list) is
  strongly implied but not stated explicitly by SAP's documentation, and whether tag names
  must be unique, whether permitted values are case-sensitive, and whether SAP preserves
  submitted ordering are all unconfirmed. See `docs/guides/custom-tag-configurations.md`.
- `sapintegrationsuite_number_range` has no Read, no Import, and no Delete — see the Added entry
  above and `docs/guides/runtime-stores-and-number-ranges.md` for the full reasoning.
- `sapintegrationsuite_api_provider` has no in-place Update (every attribute is
  `RequiresReplace`) and only supports the "Internet" connection type; SAP documents three
  further connection types (On Premise, Open Connectors, Cloud Integration) with no confirmed
  field-level JSON mapping this provider could find. `sapintegrationsuite_api_product`'s
  `api_proxy_names` is set only at Create time (also `RequiresReplace`), since SAP's confirmed
  Update payload never includes that association. `sapintegrationsuite_api_key_value_map` has no
  in-place Update and does not support encrypted maps at all. There is no
  `sapintegrationsuite_api_proxy` resource — see `docs/guides/classic-api-management.md` for all
  four limitations in full.
- No acceptance tests exist in this repository yet. Every phase of this provider's development so
  far has worked from documentation research without live SAP tenant credentials, so test coverage
  is unit-test (`httptest`-based) only; the `TF_ACC=1`-gated acceptance test workflow and
  convention are in place for future contributors with tenant access. See `CONTRIBUTING.md`.
