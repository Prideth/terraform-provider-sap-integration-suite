# Changelog

This is the release history of the Terraform Provider for SAP Integration
Suite. Each entry is written for the people who run the provider: what
changed, why it changed, and what you have to do when you upgrade.

The provider is on a 0.x release line. A minor release (0.2.0, 0.3.0) can
contain breaking schema or lifecycle changes; each one is listed under
"Breaking changes" together with the steps it needs. Patch releases (0.2.1)
only fix defects in their minor release.

## 0.5.0 — 2026-10-04

0.5.0 extends Integration Assessment from the landscape inventory to the
technology profiles that ISA-M technology recommendations are based on.
0.4.0 described which vendors, applications and technologies exist and where
they run; 0.5.0 describes what a technology can do, in the terms of SAP's
Integration Solution Advisory Methodology taxonomy, and makes the whole
taxonomy readable.

### Highlights

- A technology's profile can be maintained with Terraform: the integration
  domains and styles it serves and how strongly it meets each key
  characteristic value. The assessment compares these profiles when it
  recommends a technology for an interface.
- Every entry of the ISA-M taxonomy can be looked up: deployment models,
  domains, styles, use case patterns, integration patterns, key
  characteristic groups, key characteristic values, recommendation degrees
  and domain determinations. All 27 entity sets of the Entities API are now
  either used by the provider or explicitly excluded as assessment workflow.

### New resources

- `sapintegrationsuite_integration_assessment_technology_domain`,
  `sapintegrationsuite_integration_assessment_technology_style` and
  `sapintegrationsuite_integration_assessment_technology_key_characteristic`
  (unofficial). A tenant created, read and deleted all three on 2026-09-28
  and answered a `PATCH` on a key characteristic rating with 400 V101, so
  the service has no update for them: every change replaces the association.
  All three can be imported by the Id the service assigned.

### New data sources

- `sapintegrationsuite_integration_assessment_domain`,
  `sapintegrationsuite_integration_assessment_style`,
  `sapintegrationsuite_integration_assessment_use_case_pattern`,
  `sapintegrationsuite_integration_assessment_integration_pattern`,
  `sapintegrationsuite_integration_assessment_key_characteristic_group` and
  `sapintegrationsuite_integration_assessment_recommendation_degree`
  (unofficial) find a taxonomy entry by its exact name. The pattern lookups
  also return the Ids of the domain and style the pattern refers to.
- `sapintegrationsuite_integration_assessment_key_characteristic_value`
  (unofficial) finds a value by its key characteristic's name and its own,
  because value names repeat across key characteristics.
- `sapintegrationsuite_integration_assessment_domain_determination`
  (unofficial) finds the domain that applies between a source and a target
  deployment model. A domain determination has no name; SAP Help's
  description of it repeats the recommendation degree's text, so what it is
  comes from the service's `$metadata`.

### Known limitations

- Still unofficial: SAP documents the Entities API and its entities, but the
  field-level specification on the Business Accelerator Hub needs an SAP
  login (checked again on 2026-10-04). The acceptance tests of the technology
  profile and of the taxonomy lookups passed on a tenant on 2026-10-04.
- The ISA-M taxonomy is read, never written. Assessment requests, their line
  items and decisions, the integration and message flows they describe and
  the interface request report stay out of scope: they are the state of an
  assessment, not configuration.

### Upgrade notes

Nothing to do. All additions are optional and need `enable_unofficial`; no
state migration is needed when upgrading from 0.4.

## 0.4.0 — 2026-10-03

0.4.0 adds the Integration Assessment landscape. It changes nothing for
existing configurations. The acceptance test of the landscape, including
moving objects between their links in place, passed on a tenant on
2026-09-29. It also contains the access policy checks of 0.3.1 and 0.3.2.

### Highlights

- The Integration Assessment landscape can be maintained with Terraform:
  vendors, applications, application instances, technologies and technology
  instances, with lookups for deployment models, vendors and technologies
  (unofficial, see below).

### New resources

- `sapintegrationsuite_integration_assessment_vendor`,
  `sapintegrationsuite_integration_assessment_application`,
  `sapintegrationsuite_integration_assessment_application_instance`,
  `sapintegrationsuite_integration_assessment_technology` and
  `sapintegrationsuite_integration_assessment_technology_instance`
  (unofficial): the landscape that Integration Assessment evaluates
  integration requests against. They use a new, optional
  `provider.integration_assessment` block with the service key of an
  "Integration Assessment APIs" instance. SAP documents the API and its
  entities, but the field-level specification needs an SAP login, so the
  requests follow the service's `$metadata`; create, read, update and delete
  of all five objects, and their links, were verified on a tenant. Names,
  descriptions and links change in place; only moving a technology instance
  to another technology replaces it, because that was not tested.

### New data sources

- `sapintegrationsuite_integration_assessment_deployment_model`,
  `sapintegrationsuite_integration_assessment_vendor` and
  `sapintegrationsuite_integration_assessment_technology` (unofficial) find
  an existing object by its exact name, for example one of SAP's deployment
  models or technologies, whose Ids differ between tenants.

### Upgrade notes

Nothing to do. The new types are optional; to use them, set
`enable_unofficial = true` and add the `integration_assessment` block.

### Improvements

- The Current API Management evidence records what a tenant showed on
  2026-09-27 and 2026-09-29: API artifacts and MCP servers are integration
  package content (bundles of type `RESTAPI` and `MCPSERVER` for the
  Integration Cell runtime profile), and the documented Integration Content
  API does not list them. There is still no public API for them.

### Fixes

- The GitHub release now shows the hand-written release notes. The releases
  of 0.2.0 and 0.3.0 were published without a description, because
  `changelog.disable` in the GoReleaser configuration also skipped reading
  the notes file.
- The provider page, the Classic API Management guide, the import guide and
  the README still described API proxies as manageable after the resource was
  withdrawn in 0.3.0. They no longer do.

### Known limitations

- Integration Assessment requests and assessment results stay out of scope
  (workflow state), as do the content import and export. The ISA-M taxonomy
  other than deployment models, and the domains, styles and key
  characteristics of a technology, are not exposed yet. A tenant test showed
  that key characteristics can only be created and deleted.
## 0.3.2 — 2026-10-03

0.3.2 is a patch release of 0.3.1. It extends the plan-time length check of
the access policy description to the four other access policy strings that
SAP shortens without an error. Nothing else changes.

### Fixed

- **Access policy strings that SAP would shorten are rejected at plan
  time.** A tenant probe wrote values of 200 to 5000 characters to every
  string of an access policy and its references. SAP answered each request
  with success and stored only the first characters:

  | Attribute | SAP keeps |
  |---|---|
  | `sapintegrationsuite_access_policy.role_name` | 200 characters |
  | `sapintegrationsuite_access_policy_reference.name` | 50 characters |
  | `sapintegrationsuite_access_policy_reference.description` | 200 characters |
  | `sapintegrationsuite_access_policy_reference.value` | 150 characters |

  The provider then read back a different value than it had planned, so the
  apply failed after the object existed in SAP, or every later plan showed a
  change. `sapintegrationsuite_access_policy` and
  `sapintegrationsuite_access_policy_reference` now check these lengths while
  Terraform validates the configuration, as 0.3.1 does for the policy
  description. The error names the limit and the length found; the provider
  never shortens a value itself.

  SAP counts characters, not bytes: 200 umlauts were kept. Characters outside
  the Basic Multilingual Plane, such as emoji, count as two, because how SAP
  counts them was not tested. No SAP source states these limits; they were
  observed on a tenant, and an acceptance test round-trips every string at
  its limit.

- **The GitHub release shows the hand-written release notes.** GoReleaser's
  `changelog.disable` setting skipped the release notes file, so the
  releases of 0.2.0, 0.3.0 and 0.3.1 first showed only the tag message. This
  fix was already part of the 0.4.0 line.

### Upgrade notes

Configurations within the limits need no change. A longer value now fails
the plan; shorten it. Such configurations never applied cleanly. A policy or
reference that an earlier apply left in SAP with a shortened value is
replaced if it is in the state; otherwise delete it in SAP or import it. The
Provider Upgrades guide describes the cases.

## 0.3.1 — 2026-10-01

0.3.1 is a patch release of 0.3.0. It moves two kinds of access policy
errors from apply time to plan time: descriptions SAP cannot keep, and
reference values SAP does not accept. Both used to leave policies in SAP
without their references. Nothing else changes.

### Fixed

- **Access policy descriptions longer than 200 characters are rejected at
  plan time.** SAP Integration Suite keeps at most 200 characters of an
  access policy description. Earlier releases accepted longer values: SAP
  accepted the create request, stored only the first 200 characters, and the
  provider's read-back no longer matched the plan. Terraform then reported
  "Provider produced inconsistent result after apply". The policy already
  existed in SAP at that point, but the
  `sapintegrationsuite_access_policy_reference` resources that depend on its
  `id` were never created, leaving policies with a role, a truncated
  description and no references.

  `sapintegrationsuite_access_policy` now checks the length while Terraform
  validates the configuration, so a description SAP cannot keep stops the
  plan before anything is sent to SAP. The error names the limit and the
  length found. The provider does not shorten descriptions itself: a silently
  cut text would hide the configuration error. An empty string is rejected as
  before; omit the attribute instead.

  The limit is not stated in SAP Help, the Business Accelerator Hub
  specification or the service's `$metadata`; it was observed on a tenant.

- **Access policy references accept only values SAP stores.**
  `artifact_type`, `attribute` and `operator` of
  `sapintegrationsuite_access_policy_reference` accepted any non-empty string,
  apart from the two wrong spellings `IntegrationFlow` and `EQUALS`. A UI
  label such as `MATCHES` or `IntegrationPackage` passed the plan, SAP
  rejected it during apply, and the policy created in the same apply was left
  without its references.

  The plan now accepts only SAP's wire values: the artifact type constants
  (for example `INTEGRATION_FLOW`, `INTEGRATION_PACKAGE`, `ODATA_SERVICE`,
  `MESSAGE_QUEUE`), the attributes `Name` and `ID`, and the operators
  `exactString` (*Equals* in the UI) and `regularExpression` (*Matches*). It
  also rejects the combinations SAP refuses: Integration Package references
  only take `exactString`, and message queues, global variables and global
  data stores can only be matched by `Name`. Every error lists the valid
  values and, for a UI label or an earlier spelling, names the wire value to
  use; the provider never converts a value itself.

  For `regularExpression`, the plan fails for patterns Java rejects as well
  (for example unbalanced parentheses or a leading `*`) and warns about
  glob-like patterns such as `SALES_*`, which matches `SALES` followed
  by underscores rather than every name starting with `SALES_`.

  The values come from SAP Help (types, attributes, operators and both
  restrictions), SAP's audit log documentation (`INTEGRATION_FLOW`, `Name`,
  `exactString`, `regularExpression`) and the tenant, which lists the allowed
  values in its error answers. A tenant test created every accepted
  combination and read it back unchanged. Eight artifact types SAP accepts
  but does not document for access policies (credentials, secure parameters,
  adapters, service interfaces, fault message types) need
  `enable_unofficial = true` to be created.

  Read and import are unchanged: a reference that already exists in SAP keeps
  whatever values SAP returns, even ones this version does not create.

### Upgrading from 0.3.0

- No resource is renamed and no state is migrated. Configurations whose
  access policy descriptions have 200 characters or fewer need no change.
- A configuration with a longer description fails at plan time. Shorten the
  description to a summary of the policy; the artifacts it protects belong in
  its references.
- Policies that 0.3.0 created with a truncated description are reconciled
  once the configuration is shortened. A policy left in the state as tainted
  is replaced by the next apply, or kept with `terraform untaint` (an
  in-place description change needs `enable_unofficial = true`). A policy
  that is not in the state can be imported by its numeric ID. The Provider
  Upgrades guide describes both cases.
- A reference configuration with a value outside the supported lists fails at
  plan time with the value to use. Such a configuration never worked against
  SAP, except for the eight undocumented artifact types: references to those
  keep working once `enable_unofficial = true` is set.

## 0.3.0 — 2026-09-29

0.3.0 changes how the provider treats anything SAP has not documented, fixes
what the September 2026 acceptance runs found in certificates, key pairs
and number ranges, and rewrites the Terraform Registry documentation. It adds
no resource types. Two changes are breaking; read them before you upgrade.

### Highlights

- Functionality that is not backed by an official SAP contract is now off
  by default. Experimental and unofficial resources, and the undocumented
  operations of otherwise documented resources, need an explicit switch in
  the provider block. Nobody ends up depending on them by accident.
- Every implemented feature now records where its contract comes from: SAP
  Help, an official API specification, SAP's own tooling, or only the
  service's `$metadata`. The support status follows from that.
- Replacing a certificate and importing a key pair work on a tenant. Both
  failed or planned a replacement in 0.2.0.
- Every Registry page was rewritten around lifecycle, import and limitations,
  with new guides for getting started, importing existing content,
  upgrading and troubleshooting.

### Breaking changes

#### Experimental and unofficial types need a provider switch

Old behavior: every registered resource and data source could be used
without further configuration, whatever its support status.

New behavior: a resource or data source whose status is `experimental` or
`unofficial` fails with an error that names it until the matching switch is
set:

| Type | Status | Switch |
|---|---|---|
| `sapintegrationsuite_business_data_graph` (resource and data source) | experimental | `enable_experimental` |
| `sapintegrationsuite_secure_parameter` | unofficial | `enable_unofficial` |
| `data.sapintegrationsuite_access_policy_runtime_assignments` | unofficial | `enable_unofficial` |

Why: `experimental` means the lifecycle has not passed an acceptance test
on a tenant yet; `unofficial` means it works on a tenant, but SAP documents
the API nowhere, so SAP can change it without notice. Both deserve a
conscious decision. `sapintegrationsuite_secure_parameter` and the runtime
assignments data source were listed as supported and read-only in 0.2.0;
they are reclassified because the `SecureParameters` entity set and the
runtime assignments are known only from the service's `$metadata`.

What to do: if you use one of these types, add the switch. Nothing else
changes; state and existing objects are untouched.

```terraform
provider "sapintegrationsuite" {
  host = var.integration_suite_host

  enable_experimental = true # business_data_graph
  enable_unofficial   = true # secure_parameter, access_policy_runtime_assignments

  oauth {
    # ...
  }
}
```

Both switches can also be set with `SAP_INTEGRATION_SUITE_ENABLE_EXPERIMENTAL`
and `SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL`; a value in the provider block
takes precedence.

#### Undocumented operations need `enable_unofficial`

Some resources are built on a documented API but use one operation that SAP
does not document. Those operations are now refused at plan time without
`enable_unofficial`, with an error that names the operation:

- `sapintegrationsuite_message_mapping`: an in-place update. SAP documents
  reading, creating and deleting message mappings; the content update is a
  `PUT` that works on a tenant but is not documented.
- `sapintegrationsuite_script_collection`: an in-place update, and
  `save_as_version` (the `SaveAsVersion` function import exists only in the
  service's `$metadata`).
- `sapintegrationsuite_access_policy`: changing `description` (`PATCH`).
- `sapintegrationsuite_business_data_graph`: an in-place update and
  deleting the graph, in addition to `enable_experimental`. SAP names the
  PATCH URL but shows no body, and documents no delete request.

`sapintegrationsuite_number_range` keeps working without the switch, but
only with the two operations SAP documents, `POST` and `PUT`:

- a refresh keeps the state instead of reading the number range, so drift
  is not detected and `deployed_by` / `deployed_on` stay empty;
- every update has to change `current_value_wo_version`, because SAP
  rejects an update without a counter and the live counter can only be read
  with the undocumented `GET`;
- `terraform destroy`, a replacement and `terraform import` are refused;
- create cannot check whether the name already exists.

What to do: to keep the behavior of 0.2.0, set `enable_unofficial = true`.
Otherwise, plans that need one of the operations above fail before anything
is sent to SAP. To stop managing a number range without deleting it, use
`terraform state rm`. Turning the switch on later is safe: the next refresh
reads the number range and fills in what the state is missing.

### Improvements

- The feature catalog records a contract source for every implemented
  feature (`sap_documentation`, `api_specification`, `sap_tooling`,
  `metadata_only`) and the operations that work but are not documented.
  Only an official source allows `supported`, `partial` or `read_only`;
  something known only from `$metadata` is `unofficial` once verified on a
  tenant and `experimental` before. `data.sapintegrationsuite_provider_feature`
  and `data.sapintegrationsuite_provider_features` return
  `contract_source` and `undocumented_operations`, and
  `docs/feature-support.md` explains the sources.
- The feature catalog now covers Integration Assessment's two services from
  their live `$metadata`: 27 entity sets in the entities service and the
  `ImportContent` / `ExportContent` operations of the management service.
  No resource yet; see Known limitations.
- The Current API Management re-audit was repeated on 2026-09-27 against 308
  APIs in 173 Business Accelerator Hub packages. There is still no public
  API for API artifacts, MCP servers, runtime profiles or Integration Cell
  virtual hosts.

### Fixes

- Replacing the certificate of a `sapintegrationsuite_certificate` failed
  with "Provider produced inconsistent result after apply", because the plan
  kept the old `certificate_sha256`, `subject_dn`, `issuer_dn` and
  `serial_number`. They are derived from the PEM, so the plan now computes
  them from the new certificate.
- After `terraform import`, a `sapintegrationsuite_key_pair` had empty
  subject fields, so the next apply planned a replacement, which would have
  generated a new key pair. The empty subject fields are now filled from the
  subject DN SAP returns.
- For the same reason, setting `signature_algorithm` or
  `key_algorithm_parameter` on an imported key pair planned a replacement:
  SAP does not return either value, so state held null. When the prior value
  is unknown, both now only record the configured value; changing a known
  value still generates a new key pair.
- `sapintegrationsuite_number_range` names are checked against the rule SAP
  states when it rejects one: letters, digits, spaces and underscores only.
  Earlier releases rejected only hyphens, so a name such as `INV.2026` passed
  the plan and failed on apply with a 500.

### Known limitations

- There is still no API proxy resource. An implementation that imports a
  proxy bundle the way SAP's API Management Client SDK does exists, but the
  API portal rejected that import with `APIPROXY_ZIP_ERROR`, even for a bundle
  it had exported itself. It is added once the request documented in the
  official Transport API specification works.
- Integration Assessment has no resources yet. Its two services are
  classified from their live `$metadata`, and a tenant test created, read and
  deleted vendors and applications.
- Tenant checks of value mapping entries (September 2026) showed that
  `UpsertValMaps` dropped design-time values, a duplicate source value became
  the new default, and `DeleteValMaps` answered 202 without deleting
  anything. Value mapping entries are therefore classified as unsafe for a
  Terraform lifecycle and stay unimplemented.
- Data types can be created on a tenant only without content; every request
  that carries content failed with a 500 ("map is null"). Data types and the
  other design-time types (message types, fault message types, service
  interfaces) stay unimplemented until content can be written.

### Documentation

- Every resource and data source page on the Registry was rewritten. Each
  page states the support status and the contract behind it, the
  prerequisites and roles, what create, read, update, replacement and delete
  do (and why an attribute forces replacement), what an import recovers and
  what it cannot, and SAP limitations separately from provider limitations.
  Pages are grouped by area in the sidebar.
- The provider page now covers scope and the boundaries to the SAP/btp
  provider, Developer Hub and Event Mesh, which service key field goes into
  which attribute, environment variables and their precedence, support
  statuses and the opt-in switches, runtimes, write-only secrets and
  troubleshooting.
- New guides: Getting Started (from a service key to a deployed integration
  flow and its endpoint URL), Importing Existing Content, Provider Upgrades
  and Troubleshooting.
- Two examples declared a flow deployment with a `version` argument that
  does not exist and without the required `flow_version`; they are fixed,
  and a test now checks every example and guide snippet against the schema.
- The Classic API Management data sources had no attribute descriptions;
  they have now.

### For contributors

- API discovery: `$metadata` (OData V2 and V4) and OpenAPI documents are
  parsed into one normalized model, and a snapshot of each service's
  contract is committed under `testdata/api-metadata/` without host names or
  tenant data. `go run ./cmd/apidiscovery` compares live documents with the
  snapshots and reports semantic changes, breaking ones separately; official
  REST specifications can be read from a local directory (`-spec-dir`).
  Every entity set is classified as used, candidate or excluded, and
  `docs/api-discovery-report.md` is generated from that.
- Acceptance tests are gated by capability: a test runs only with `TF_ACC=1`
  and either `SAP_INTEGRATION_SUITE_ACC_ALL=1` or its own gate (for example
  `SAP_INTEGRATION_SUITE_ACC_CLOUD_INTEGRATION=1`), and skips with the
  missing variable names otherwise. Destructive tests also need
  `SAP_INTEGRATION_SUITE_ACC_DESTRUCTIVE=1` and their gate set by name.
  `go run ./cmd/accplan` prints what would run. `TF_ACC=1` alone no longer
  runs anything.
- `go run ./cmd/repohygiene` checks commit authorship and message style; CI
  runs it on every push.
- `docs/research/capability-evidence-2026.md` is generated from
  `internal/features/evidence.go` and gives, for every feature that is not
  fully supported, the date of the last check, the sources and the step
  that would change the classification.

## 0.2.0 — 2026-09-27

Tag `v0.2.0` points at commit `9d65cfa` of 2026-09-26. The CHANGELOG inside
that tag still shows the changes as an unreviewed "Unreleased" list; this
entry is the reviewed one. From 0.3.0 on, the tagged commit contains its own
finished entry.

### Highlights

- This is the first release that works against a real tenant. 0.1.0 could
  not authenticate under Terraform at all (see its entry below). On
  2026-09-26 the provider ran against a tenant for the first time, through
  acceptance tests and API probes, and the defects those runs found are
  fixed here. Integration adapters, business data graphs and Edge
  Integration Cell targeting were not part of those runs.
- The wire contracts were checked against tenant `$metadata` documents and
  against SAP's own tooling. Several resources sent property names SAP does
  not have; they are corrected, some of them with breaking schema changes.
- Externalized integration flow parameters, explicit design-time versions,
  secure parameters and full number range management are new, as is an
  experimental resource for API Composition business data graphs.

### Upgrade from 0.1.0

0.1.0 failed every request to SAP (see below), so it cannot have created or
imported any SAP object. In practice the upgrade touches only your
configuration. Attributes that were removed from a schema are dropped from
state silently on the first refresh.

| Resource or data source | Change to make |
|---|---|
| `sapintegrationsuite_integration_package` | Add the new required `short_text`. SAP rejects a package without it. |
| `sapintegrationsuite_access_policy_reference` | Add the new required `name`. Use SAP's constants: `INTEGRATION_FLOW` instead of `IntegrationFlow`, `exactString` instead of `EQUALS`. |
| `sapintegrationsuite_integration_adapter` and data source | Remove `type` and `application`; the entity has neither. |
| `data.sapintegrationsuite_service_endpoints` | Read `api_definitions[*].name` instead of `api_definitions[*].type`. |
| `sapintegrationsuite_partner_authorized_user` and data source | Write `user` in lowercase. SAP stores it lowercased. |
| `sapintegrationsuite_api_product` | Set `api_proxy_names`, now required. Every attribute now forces a new product. |
| `sapintegrationsuite_access_policy` and data source | Remove references to `reconciliation_status`. Use `data.sapintegrationsuite_access_policy_runtime_assignments`. |
| Keystore entry data sources | `valid_not_before` / `valid_not_after` are RFC 3339 timestamps now, no longer `/Date(...)/` literals. |

Integration package, before and after:

```terraform
# 0.1.0
resource "sapintegrationsuite_integration_package" "order_processing" {
  id          = "ORDER_PROCESSING"
  name        = "Order Processing"
  description = "Order intake and confirmation flows"
}

# 0.2.0
resource "sapintegrationsuite_integration_package" "order_processing" {
  id          = "ORDER_PROCESSING"
  name        = "Order Processing"
  short_text  = "Order intake and confirmation"
  description = "Order intake and confirmation flows"
}
```

Access policy reference, before and after:

```terraform
# 0.1.0
resource "sapintegrationsuite_access_policy_reference" "order_flows" {
  access_policy_id = sapintegrationsuite_access_policy.order_team.id
  artifact_type    = "IntegrationFlow"
  attribute        = "Name"
  operator         = "EQUALS"
  value            = "ORDER_INTAKE"
}

# 0.2.0
resource "sapintegrationsuite_access_policy_reference" "order_flows" {
  access_policy_id = sapintegrationsuite_access_policy.order_team.id
  name             = "Order intake flow"
  artifact_type    = "INTEGRATION_FLOW"
  attribute        = "Name"
  operator         = "exactString"
  value            = "ORDER_INTAKE"
}
```

The provider rejects `IntegrationFlow` and `EQUALS` at plan time with a
pointer to the correct value. Every attribute of a reference forces
replacement, as before.

### Breaking changes

Each of these is part of the table above. Why they changed:

- **Integration package `short_text` is required.** Creating a package
  failed on a tenant with "Property 'ShortText' cannot be empty". Updates
  now use `PUT` (SAP answers `PATCH` with 501). Because that `PUT` replaces
  the whole package, the provider reads the package first and sends its
  version, vendor and tag fields back unchanged; without them SAP reset
  `Version` and `Vendor` to empty. SAP stores the description as HTML and
  wraps plain text in `<p>...</p>`; the provider strips that wrapper, so a
  plain description no longer shows as drift. Adding `short_text` to an
  existing package is an in-place update.
- **Access policy reference.** 0.1.0 sent property names the
  `ArtifactReferences` entity does not have (`ArtifactType`, `Attribute`,
  `Operator`, `Value`) and could not create a reference on a tenant. The
  real names (`Name`, `Description`, `Type`, `ConditionAttribute`,
  `ConditionType`, `ConditionValue`) and constants come from SAP's own
  access policy automation in `SAP/cicd-actions-for-sap-integration-suite`.
  `artifact_type`, `attribute` and `operator` now pass SAP's constants
  through instead of checking a closed list, which was incomplete (it
  lacked Integration Package, API, Data Type and Message Type).
- **Integration adapter `type` and `application` removed.** The tenant
  `$metadata` shows that `IntegrationAdapterDesigntimeArtifact` has neither
  property; sending them was a guess based on the UI's import dialog. Both
  the resource and the data source now expose the read-only `description`
  SAP takes from the `.esa` file, and the data source returns `package_id`.
- **Service endpoints `api_definitions[*].type` renamed to `name`.** The API
  definition entity has `Url` and `Name`, so `type` was always empty.
  Endpoints now also return `id`, `title`, `version`, `summary`,
  `description` and `last_updated`; entry points return
  `additional_information`.
- **Authorized users must be lowercase.** SAP lowercases the user (its own
  example creates `MyUser` and returns `myuser`), so a mixed-case value
  could never match what SAP reported back.
- **API product is replace-only.** 0.1.0 could neither create nor read a
  product: SAP returns the links to proxies and properties as `__deferred`
  objects. A tenant also answered every update (`PUT`, `PATCH`, `MERGE`)
  with 405 and refused a product without a linked proxy. Every attribute
  now forces a new product; replacing a product drops the subscriptions of
  applications that use it, so read plans carefully. `status_code`
  defaults to `PUBLISHED` (`DRAFT` also works). `additional_properties` are
  sent inside the create request, and refresh and import read the linked
  proxies and properties, so drift in both is detected.
- **Access policy `reconciliation_status` removed.** The `AccessPolicies`
  entity has no such property. Runtime replication is a separate navigation
  property, which the new runtime assignments data source reads.
- **Keystore dates.** `valid_not_before` and `valid_not_after` are RFC 3339
  timestamps. OData V2 date literals with a zone offset are now parsed
  correctly.

### New resources

- `sapintegrationsuite_integration_flow_configuration` sets externalized
  parameters of an integration flow through SAP's documented
  `$links/Configurations` update. Only the keys you list are managed, each
  parameter keeps its data type, and unknown keys are rejected before
  anything is written. `terraform destroy` leaves the values in place,
  because SAP offers no way to delete a parameter. Combine it with
  `redeploy_triggers` on the deployment to bring new values to the runtime.
- `sapintegrationsuite_secure_parameter` manages Security Content secure
  parameters, the confidential values that custom adapters and scripts read
  by alias. The value is write-only (`secure_param_wo` together with
  `secure_param_wo_version`) and is never stored in state. Requires
  Terraform 1.11 or later.
- `sapintegrationsuite_business_data_graph` (experimental) manages an API
  Composition business data graph through the Configuration API. It needs
  the new optional `provider.api_composition` block with credentials from an
  API Composition service instance of plan `configuration`; the Cloud
  Integration and API portal credentials do not work there. Create and
  update wait for SAP's asynchronous processing (`PROCESSING` until
  `DEPLOYMENT_INITIATED` or `FAILED`, 20 minutes by default, configurable
  with `timeouts`). A graph that ends in `FAILED` stays in state and is
  marked tainted. Not yet tested against a tenant.

### New data sources

- `data.sapintegrationsuite_access_policy_runtime_assignments` lists the
  runtimes an access policy is replicated to (Cloud Integration runtime,
  Integration Cell, Edge Integration Cells) with SAP's transfer status,
  errors and last status change. Choosing the runtimes remains a UI step.
- `data.sapintegrationsuite_business_data_graph` (experimental).

### Improvements to existing resources

- `save_as_version` on `sapintegrationsuite_integration_flow`,
  `sapintegrationsuite_message_mapping` and
  `sapintegrationsuite_script_collection` saves the uploaded content under a
  version you name, through the `...SaveAsVersion` function imports. A
  version is saved only when the value changes. Uploading new content does
  not change an artifact's version by itself (tenant test), and deployments
  redeploy only when the version they point at changes, so this is the
  clearest way to get new content to the runtime.
- The integration flow, message mapping, script collection and value
  mapping deployments have `redeploy_triggers`, a map that redeploys in
  place when it changes. Passing the artifact's `content_hash` redeploys on
  every content change.
- A deployment that SAP accepted but that did not reach `STARTED` before its
  timeout, or ended in `ERROR`, now stays in state with its last status and
  is marked tainted. Earlier it was dropped from state and kept running on
  the tenant untracked. The integration flow deployment also follows the
  deploy task (`BuildAndDeployStatus`) and stops with an explanation when
  SAP deployed the flow to another runtime, for example because
  `SAP_ProfileId` is `integrationcell`.
- `sapintegrationsuite_number_range` reads, deletes and imports by name.
  SAP documents only create and update, but a tenant confirmed
  `GET NumberRanges('<name>')` and `DELETE`. Refresh now detects drift in
  the static fields; the new `current_value`, `deployed_by` and
  `deployed_on` report what SAP holds. The first apply after an import
  records `current_value_wo_version` without touching the counter, and
  create stops if the name already exists. Updates that keep the counter
  read the live value right before the `PUT` and send it back, because SAP
  rejects a `PUT` without `CurrentValue`. Names with hyphens, which SAP
  rejects, are refused at plan time.
- `sapintegrationsuite_oauth2_client_credential` and its data source gain
  `client_authentication`, `scope_content_type`, `resource` and `audience`.
  They are optional and computed, so an update (a full `PUT`) resends what
  SAP holds instead of resetting settings made in the UI.
- `sapintegrationsuite_partner_user_credential_parameter` rotates the
  password and changes `user` in place. SAP documents a `POST` with the
  same keys as the update for this entity, so a rotation no longer deletes
  and re-creates the credential, and integration flows keep a valid
  credential throughout. Because that `POST` overwrites, create fails when
  the credential already exists and asks for an import.
- `sapintegrationsuite_user_credential`: `kind` defaults to `default`, the
  value SAP reports for a generic credential. SAP requires it.
- `data.sapintegrationsuite_access_policy` looks a policy up by `role_name`
  as well as by `id`; the role name is the same in every tenant, the ID is
  not. Import IDs of both access policy resources must be numeric and are
  checked before any request.
- The keystore entry data sources return `entry_type`, `owner`, `status`,
  `subject_dn`, `issuer_dn`, `serial_number`, `signature_algorithm`,
  `elliptic_curve`, `certificate_version`, `validity`, SHA-1/256/512
  fingerprints and the created/modified metadata.
- `sapintegrationsuite_partner_binary_parameter` accepts values up to
  1,572,864 bytes, the `MaxLength` the tenant `$metadata` declares, instead
  of 260 KB.
- Integration flows and message mappings whose ZIP was exported under a
  different ID can be updated. SAP writes the artifact ID into
  `Bundle-SymbolicName` on create and rejects every later update whose
  bundle ID differs. The provider uploads a copy with the artifact ID (and,
  for mappings, the same name in `Provide-Capability`) and shows a warning;
  your file is not changed.

### SAP API corrections found on a tenant

- Every request failed with `context canceled` on the token URL. The OAuth
  client was built with the context of the provider's configure call, which
  Terraform cancels as soon as that call returns. Tokens are now fetched
  with a context that outlives it, with a 60-second limit per request.
- SAP answers several successful writes with `200` or `202` and no body:
  content updates, `SaveAsVersion`, and the creates of OAuth2 and user
  credentials, Partner Directory entries and access policies. The provider
  reads the object back instead of failing on the empty body, which had left
  objects on the tenant that the next apply then failed to create.
- Content updates sent empty `Id` and `PackageId` fields, which SAP rejects
  for message mappings. Updates now send only `Name` and `ArtifactContent`.
- `data.sapintegrationsuite_partner` failed on every tenant, because SAP
  does not support reading a single partner by key. It now filters the
  partner list by `Pid`.
- `sapintegrationsuite_api_key_value_map` could not create or read a map,
  because SAP returns the entries as a `__deferred` link. They are read
  through `GenericKeyMapEntries(...)/genericKeyMapEntryValues`.
- Deployment status reads expected `ErrorInformation` as text; SAP declares
  it as a link to a media entity. A failed deployment now reports SAP's
  error text from `.../ErrorInformation/$value`.
- `sapintegrationsuite_certificate` could not import a self-signed or
  otherwise untrusted certificate (409 `notImported`) and could not replace
  a certificate (400 "already exists"). The import now sends
  `fingerprintVerified=true`, an update also `update=true`. Listing a
  certificate in your configuration is therefore the decision to trust it;
  compare `certificate_sha256` with the fingerprint you expect.
- Access policies and references are addressed with `Edm.Int64` keys
  (`AccessPolicies(1901L)`), and references are created and deleted through
  the top-level `ArtifactReferences` entity set.
- The Partner Directory resources no longer drop `runtime_location_id` from
  state on update or read.

### Known limitations

- `runtime_location_id` on the deployment, credential, certificate, key pair
  and Partner Directory resources sends requests to `/location/<id>/api/v1`,
  the service root SAP Help documents for Edge Integration Cells. It has not
  been tested against a tenant with an Edge Integration Cell and is **not
  supported**; leave it unset.
- The business data graph resource has not been run against a live system.
  SAP shows no PATCH body and no delete request; the provider sends the
  writable properties and `DELETE` on the graph's URL.
- Reading, deleting and importing number ranges, and the secure parameter
  entity set, rely on operations that SAP does not document but that worked
  on a tenant in September 2026.
- There is still no API proxy resource; value mapping entries and the
  remaining Security Content types (certificate chains, known hosts, PGP
  keyrings) are not implemented.

### For contributors

- Acceptance tests (`TF_ACC=1`) exist for the content, deployment,
  Security Content, Partner Directory, access policy and Classic API
  Management resources. Content tests use SAP's public integration content
  samples, or local exports named by an environment variable.
- Contract tests check every OData wire struct, key and function import
  against a tenant `$metadata` document when one is available locally.

### Documentation

- New guides: "Authorization and Roles" (the SAP role templates each
  resource family needs and how to diagnose a 403), "Integration Content and
  Deployments" (artifact ZIPs, bundle IDs, versions, redeploys, deploy times,
  `SAP_ProfileId`), "API Composition" and "Data Space Integration".
- Rewritten: the Access Policies guide (confirmed wire contract, upgrade
  steps) and the Current API Management guide (a status table per object and
  the 2026 evidence).
- `docs/research/sap-2026-public-api-gap-closure.md` summarizes the 2026
  re-audit: method, results per area and the earlier conclusions it
  corrected.

## 0.1.0 — 2026-09-24

The first release, built from documentation research without access to a
tenant.

### Known issue, found after the release

0.1.0 cannot talk to SAP under Terraform. It built its OAuth client with the
context of the provider's configure call, which Terraform cancels as soon as
that call returns, so every request fails with `context canceled` on the
token URL. Only the feature catalog data sources, which make no request,
work. Use 0.2.0 or later.

### Resources and data sources

Cloud Integration content:

- `sapintegrationsuite_integration_package` (resource and data source).
- `sapintegrationsuite_integration_flow` and
  `sapintegrationsuite_integration_flow_deployment`, with content uploaded
  from a local ZIP file and deployments polled until they start.
- `sapintegrationsuite_value_mapping`, `sapintegrationsuite_message_mapping`
  and `sapintegrationsuite_script_collection`, each with a data source and a
  `*_deployment` resource.
- `sapintegrationsuite_integration_adapter` (with data source) and
  `sapintegrationsuite_integration_adapter_deployment` for custom adapters
  (`.esa` files, Cloud Foundry only).
- `data.sapintegrationsuite_service_endpoints` for the entry points and API
  definitions of deployed content.
- `sapintegrationsuite_custom_tag_configuration` (with data source), the
  tenant-wide custom tag configuration.
- `sapintegrationsuite_number_range`.

Security:

- `sapintegrationsuite_access_policy` and
  `sapintegrationsuite_access_policy_reference`, each with a data source.
- `sapintegrationsuite_user_credential` and
  `sapintegrationsuite_oauth2_client_credential` (with data sources), whose
  secrets are write-only attributes.
- `sapintegrationsuite_certificate`, `sapintegrationsuite_key_pair` (SAP
  generates the private key; it never reaches Terraform), and the
  `keystore_entry` / `keystore_entries` data sources.

Partner Directory:

- `sapintegrationsuite_partner_string_parameter`,
  `sapintegrationsuite_partner_binary_parameter`,
  `sapintegrationsuite_alternative_partner`,
  `sapintegrationsuite_partner_authorized_user`, each with a data source,
  and `sapintegrationsuite_partner_user_credential_parameter`.
- `data.sapintegrationsuite_partner`, `data.sapintegrationsuite_partners`
  and `data.sapintegrationsuite_partner_string_parameters`.

Classic API Management, through a separate `provider.api_management` block:

- `sapintegrationsuite_api_provider`, `sapintegrationsuite_api_product`,
  `sapintegrationsuite_api_management_certificate_store_reference` and
  `sapintegrationsuite_api_key_value_map`, each with a data source, plus
  `data.sapintegrationsuite_api_providers`.

Provider:

- OAuth 2.0 client credentials with token caching and a one-time refresh on
  401, retries, CSRF token handling for OData writes and server-driven
  paging.
- `data.sapintegrationsuite_provider_feature` and
  `data.sapintegrationsuite_provider_features`: the feature catalog, usable
  without SAP credentials.

### Design decisions

- Secrets are write-only attributes paired with a `*_wo_version` marker, and
  are never read back or stored. Terraform 1.11 or later is required for
  them.
- `sapintegrationsuite_value_mapping` has no in-place update; `name`,
  `content` and `content_hash` force replacement, because no update path was
  documented for value mappings.
- `sapintegrationsuite_number_range` read nothing from SAP and refused import
  and destroy, because SAP documents only create and update. (0.2.0 added
  read, delete and import after a tenant test.)
- `sapintegrationsuite_custom_tag_configuration` cannot be destroyed: SAP
  documents no delete or clear operation for the tenant-wide configuration.
- `sapintegrationsuite_api_provider` and `sapintegrationsuite_api_key_value_map`
  have no in-place update; key value maps must be unencrypted, because it
  was not confirmed how SAP returns an encrypted value.
- There is no `sapintegrationsuite_partner` resource: SAP documents no
  create for partners, and deleting one cascades to everything that belongs
  to it.

### Known limitations

- No acceptance tests had been run against a tenant; coverage was unit tests
  against recorded responses.
- The access policy reference constants, the `reconciliation_status`
  attribute and the integration adapter's `type` and `application` were
  assumptions; 0.2.0 corrected all of them against a tenant.
- API proxies, certificate chains, secure parameters, known hosts and value
  mapping entries were not implemented.
