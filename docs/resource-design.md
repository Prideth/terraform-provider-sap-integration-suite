# Resource Design

For every resource, this document records the Terraform-suitability check from
`provider-scope.md`/rule set §15 (desired state, identity, read-back, create, update, delete,
drift, import) before committing to a schema.

## `sapintegrationsuite_integration_package`

- **Purpose**: manage a Cloud Integration content package (the container for flows, mappings,
  script collections).
- **SAP object**: `IntegrationPackages` (Integration Content API, OData V2).
- **Desired state**: yes — name, description, short text, version are stable, user-set fields.
- **Identity**: the SAP-assigned `Id` (a business key chosen at creation time) — stable, used
  as the Terraform ID directly.
- **Read-back**: `GET /api/v1/IntegrationPackages('{Id}')` returns the full object.
- **Create**: `POST /api/v1/IntegrationPackages`.
- **Update**: SAP does not support in-place metadata update for packages created via the API
  in every field; the provider only updates the fields SAP's API accepts and uses
  `RequiresReplace()` on `id` (immutable after creation).
- **Delete**: `DELETE /api/v1/IntegrationPackages('{Id}')`. A `404` on delete is treated as
  success.
- **Drift detection**: `Read` always re-fetches; renames/description edits made in the UI are
  visible on the next `terraform plan`.
- **Import**: `terraform import sapintegrationsuite_integration_package.example <Id>`.
- **Known limitation**: SAP-shipped/content-catalog packages are read-only; attempting to
  manage one as a Terraform resource will surface the SAP API's own error.

## `sapintegrationsuite_integration_flow`

- **Purpose**: manage the design-time content of a single integration flow.
- **SAP object**: `IntegrationDesigntimeArtifacts`, scoped under a package.
- **Desired state**: yes, but the "value" is binary content (a zip of the iFlow project), not
  a set of scalar fields. The provider models the artifact through a file plus a content
  hash, following the file-based pattern in §36 of the project brief:

  ```hcl
  resource "sapintegrationsuite_integration_flow" "metering" {
    package_id = sapintegrationsuite_integration_package.utilities.id
    flow_id    = "metering"
    name       = "Metering"

    content      = "${path.module}/iflows/metering.zip"
    content_hash = filesha256("${path.module}/iflows/metering.zip")
  }
  ```

- **Identity**: composite `<package_id>/<flow_id>`. Both are user-chosen business keys, stable
  across versions.
- **Read-back**: `GET IntegrationPackages('{package_id}')/IntegrationDesigntimeArtifacts` for
  metadata (name, version); the zip payload itself is only re-downloaded when
  `content_hash` indicates drift, to avoid unnecessary large downloads.
- **Create**: `POST IntegrationDesigntimeArtifacts` with the zip content base64-encoded.
- **Update**: SAP's design-time API is version-based, not in-place: an update creates a new
  design-time version of the same `flow_id`. The provider treats this as a normal Terraform
  update (no replace) because identity (`package_id`/`flow_id`) does not change.
- **Delete**: `DELETE IntegrationDesigntimeArtifacts(Id='{flow_id}',Version='{version}')`.
- **Drift detection**: content hash comparison against the last-read artifact; metadata
  (name) compared directly.
- **Import**: `terraform import sapintegrationsuite_integration_flow.metering UTILITIES/metering`.
- **Security**: content is handled via streaming/size-bounded reads to avoid zip-slip/oversized
  decompression issues on any future "read back and inspect" feature (see `SECURITY.md`).
- **Note**: deploying the flow is a **separate** resource (see below) — a design-time content
  change never implicitly deploys.

## `sapintegrationsuite_integration_flow_deployment`

- **Purpose**: express the desired *runtime* deployment state of an integration flow,
  independent of its design-time content lifecycle.
- **SAP object**: the `DeployIntegrationDesigntimeArtifact` action plus `IntegrationRuntimeArtifacts`
  for status.
- **Desired state**: yes — "this version of this flow should be deployed". `flow_version` is
  a Required input (typically wired to `sapintegrationsuite_integration_flow.<name>.version`)
  precisely so that a new design-time version produces a plannable diff on this resource; the
  design-time and deployment resources otherwise share no attribute that would change when
  content changes, so without `flow_version` a redeploy would never be triggered.
- **Identity**: the flow's `<package_id>/<flow_id>` (a tenant only ever has one active runtime
  deployment per flow id).
- **Create/Update**: `POST DeployIntegrationDesigntimeArtifact?Id='{flow_id}'&Version='{flow_version}'`,
  then poll `IntegrationRuntimeArtifacts(Id='{flow_id}')` until `Status` is `STARTED` or
  `ERROR`, using context-based exponential backoff with jitter and a configurable timeout —
  never a fixed sleep.
- **Delete**: `DELETE IntegrationRuntimeArtifacts(Id='{flow_id}')` (undeploy). `404` is success.
- **Drift detection**: `Read` writes the actually-deployed `Version` it observes back into
  `flow_version` (a Required, non-Computed attribute). Since Terraform diffs that against the
  desired value in configuration, a flow redeployed to a different version or left stale
  outside Terraform shows up as a plannable change on the next refresh.
- **Import**: `terraform import sapintegrationsuite_integration_flow_deployment.metering UTILITIES/metering`.
- **Async model**:

  ```
  Create/Update -> POST Deploy -> poll GET status -> STARTED | ERROR
  ```

## `sapintegrationsuite_access_policy`

*Re-audited September 2026. The wire contract comes from SAP's own access-policy automation;
see `docs/sap-api-references.md` for the evidence.*

- **Purpose**: manage the policy object itself, meaning the role name that unlocks access and
  its description. The rules that decide *which* artifacts are protected are separate
  resources.
- **SAP object**: `AccessPolicies` in the Security Content API (OData V2), keyed by an
  `Edm.Int64` `Id`.
- **Desired state**: `role_name`, `description`. Both are persisted properties. There is
  nothing computed besides the ID.
- **Identity**: the numeric `Id` SAP returns on create. `role_name` is unique per tenant and
  is the portable identifier across landscapes. The data source can resolve it to an ID, but
  the resource keys on the ID because that is what every write addresses.
- **Create**: `POST AccessPolicies` with `RoleName` and `Description`.
- **Update**: `PATCH AccessPolicies(<id>L)` with `Description` only. The re-audit briefly
  switched to `PUT` with `RoleName` and `Description`, as SAP's CI/CD upload action contains;
  a tenant answered that `PUT` with `501`, while `PATCH` and `MERGE` answered `204` and the
  change was read back (September 2026). `role_name` keeps `RequiresReplace()`: it is the
  policy's identity, and renaming in place was not tested.
- **Delete**: `DELETE AccessPolicies(<id>L)`. SAP deletes the policy's artifact references
  with it, including references Terraform never managed. This is the one place where
  destroying a Terraform resource reaches objects outside its own state, and the guide calls
  it out.
- **Drift detection**: `GET AccessPolicies(<id>L)`. A 404 removes the resource from state.
- **Import**: by numeric ID. Non-numeric IDs are rejected at import time rather than sent to
  SAP.
- **Data source**: looks up by `id` or by `role_name` (exactly one). The role-name lookup
  uses `$filter=RoleName eq '<name>'`, the same query SAP's tooling uses.
- **Runtime targeting**: read-only. `data.sapintegrationsuite_access_policy_runtime_assignments`
  reads the `AccessPolicyRuntimeAssignments` navigation property (structure confirmed by tenant
  `$metadata`); writing assignments is undocumented and not modeled. The former
  `reconciliation_status` attribute was removed because the entity has no such property.

## `sapintegrationsuite_access_policy_reference`

- **Purpose**: manage one artifact reference, the rule matching artifacts by type and by name
  or ID.
- **Suitability (§17)**: a reference has its own server-assigned ID and its own create and
  delete calls, and ordering between references has no documented meaning. A separate
  resource is the better fit than a nested block:
  - independent API identity: yes (`ArtifactReferences` keyed by `Edm.Int64`)
  - independent create/delete: yes, through the top-level `ArtifactReferences` set
  - import of a single reference: yes
  - drift detection per reference: yes
  - a nested block would force Terraform to own the complete reference set of a policy,
    which breaks shared policies
- **Schema and wire mapping**: `name` → `Name`, `description` → `Description`,
  `artifact_type` → `Type`, `attribute` → `ConditionAttribute`, `operator` →
  `ConditionType`, `value` → `ConditionValue`. The Terraform names follow the labels of SAP's
  UI, and the values are SAP's wire constants.
- **Validation**: non-empty strings only, plus plan-time rejection of `IntegrationFlow` and
  `EQUALS`. Those two were accepted by earlier releases and are now known to be wrong
  (`INTEGRATION_FLOW`, `exactString`). The complete constant sets are not public, so a closed
  enum would either be a guess or lock users out of valid types such as Integration Package.
- **Create**: `POST ArtifactReferences` with the six properties and
  `"AccessPolicy": {"Id": "<policy id>"}` binding the reference to its policy.
- **Read**: `GET AccessPolicies(<policy id>L)/ArtifactReferences`, then pick the reference by
  ID. Going through the policy also proves membership, and a deleted parent surfaces as a 404.
- **Update**: none. Every attribute has `RequiresReplace()`. SAP's UI can edit a reference,
  but no public API contract for it was found, and SAP's own sync deletes and recreates
  references instead. Replacement briefly removes the protection the old reference gave; the
  guide describes `create_before_destroy` as the mitigation.
- **Delete**: `DELETE ArtifactReferences(<id>L)`. A 404 counts as success, because deleting
  the parent policy already removed the reference.
- **Import**: `<access_policy_id>/<reference_id>`, both numeric.
- **Ownership boundary**: Terraform creates, reads and deletes only the reference IDs in its
  state. It never lists a policy's references to remove unknown ones. The exception sits on
  SAP's side: destroying the parent policy removes every reference.
- **Regular expressions**: the *Matches* operator takes a `java.util.regex.Pattern`, not a
  glob. `UTIL_.*` is the unambiguous way to say "starts with `UTIL_`".

## Value Mapping API model

Before designing the Terraform resources, this section separates two distinct SAP concerns
that are easy to conflate: the **design-time artifact** (a versioned container, structurally a
sibling of `IntegrationDesigntimeArtifacts` in the same Integration Content API), and
**individual mapping entries** inside it (managed through a different, entry-oriented set of
OData actions with its own preconditions).

**Confirmed operations** (Integration Content API, OData V2, `CloudIntegrationAPI` package):

| Operation | Method | Purpose |
|---|---|---|
| `ValueMappingDesigntimeArtifacts` | GET | Read artifact metadata (`Id`, `Version`, `Name`, `PackageId`), navigable from `IntegrationPackages` |
| `ValueMappingDesigntimeArtifacts` | POST | Create a new value mapping artifact from uploaded content |
| `ValueMappingDesigntimeArtifactSaveAsVersion` | POST | Save the artifact's content under an explicit, caller-supplied new version identifier |
| `ValueMappingDesigntimeArtifacts(Id=…,Version=…)` | DELETE | Delete the artifact |
| `DeployValueMappingDesigntimeArtifact?Id='…'&Version='…'` | POST | Deploy a specific version to the runtime |
| `IntegrationRuntimeArtifacts(Id=…)` | GET / DELETE | Read deployment status / undeploy — the **same shared runtime-artifacts entity** already used by `sapintegrationsuite_integration_flow_deployment`, not a distinct "ValueMappingRuntimeArtifacts" entity |
| `UpsertValMaps` | POST | Insert/update individual mapping-entry rows inside an *existing* scheme. Reachable secondary sources describe it as query-parameter-only (no request body — sending one reportedly causes a 400), returning 202 with no response body on success. |
| `UpdateDefaultValMap` | POST | Set the default value for a scheme, identified by a `ValMapId` GUID. That GUID is not returned by `UpsertValMaps`'s response (which per the above has no body); the same secondary sources describe retrieving it via a separate `ValueMappingDesigntimeArtifacts` read after the upsert, not from the upsert call itself. |
| `DeleteValMaps` | — | Delete mapping-entry rows (exact granularity unconfirmed) |
| Required roles | — | `WorkspacePackagesConfigure`, `WorkspacePackagesEdit`, `WorkspaceArtifactsDeploy` |

The deploy action's exact name (singular `...Artifact`, not plural `...Artifacts`) was re-checked
this phase after a report that current SAP documentation might use the plural form. Every
reachable secondary source (independent search results summarizing SAP community/blog content)
consistently gives the singular form used above, matching the sibling actions
`DeployIntegrationDesigntimeArtifact` and `DeployMessageMappingDesigntimeArtifact`; no source
found gave the plural form. `help.sap.com`/`api.sap.com`, which would settle this conclusively,
were not reachable from this environment to fetch and read directly (see the Update entry below
for the full list of what was tried and blocked).

**A real, confirmed constraint shapes the whole design**: a value mapping artifact cannot be
saved with zero entries — SAP requires at least one mapping entry to exist before the object
can be created at all. This, together with the entry-level API's own precondition (attempting
to upsert into a source/target agency-identifier combination that doesn't already exist as a
defined scheme returns 404 — the scheme must already exist before values can be added to it),
indicates the entry-level API manages *rows within* a scheme that is itself part of the
artifact's design-time content, not a fully independent object graph creatable purely through
`UpsertValMaps` calls.

## `sapintegrationsuite_value_mapping`

- **Purpose**: manage a Cloud Integration value mapping design-time artifact as a file-based
  content package, the same way `sapintegrationsuite_integration_flow` manages an integration
  flow's ZIP content — the content itself (the scheme definitions and their entries) is
  authored as a unit and uploaded, rather than field-by-field through Terraform.
- **SAP object**: `ValueMappingDesigntimeArtifacts`.
- **Desired state**: yes — `name`, and content identified by a hash, following exactly the
  `content`/`content_hash` pattern already proven for `sapintegrationsuite_integration_flow`.
- **Identity**: composite `<package_id>/<mapping_id>`, both user-chosen business keys.
- **Create**: `POST ValueMappingDesigntimeArtifacts` with `Id`, `Name`, `PackageId`, and
  base64-encoded `ArtifactContent`.
- **Read**: `GET ValueMappingDesigntimeArtifacts(Id='{mapping_id}',Version='active')`.
- **Update — resolved conservatively, not implemented via PUT**: an earlier version of this
  resource called `PUT` against the keyed `(Id, Version)` entity, by analogy with
  `sapintegrationsuite_integration_flow`. That assumption was re-investigated this phase and
  deliberately not kept:
  - `help.sap.com`, `api.sap.com`, `community.sap.com`, `blogs.sap.com`, and every reachable
    mirror/proxy for them (web.archive.org, a Google Translate proxy, a text-extraction proxy)
    were all blocked by this environment's network egress policy, so the exact
    `ValueMappingDesigntimeArtifactSaveAsVersion` request/response shape and any authoritative
    statement about what plain `PUT` does for this entity set could not be fetched and read
    directly.
  - What could be gathered from reachable secondary sources (search-engine summaries of SAP
    community/blog posts, plus a public SAP Knowledge Base Article, 3502529, titled "HTTP/403
    Forbidden response while trying to change Version of the ValueMapping") consistently treats
    changing a value mapping's version as a distinct, separately named, separately gated
    operation (`ValueMappingDesigntimeArtifactSaveAsVersion`, taking a caller-supplied new
    version identifier) — not something that happens implicitly as a side effect of a generic
    `PUT`, the way it does for `IntegrationDesigntimeArtifacts`.
  - An independent third-party OData client built directly against this same API
    (`github.com/lemaiwo/ci-mcp-server`) explicitly disables its generic "update" operation for
    `ValueMappingDesigntimeArtifacts` specifically, while leaving the identical operation enabled
    for `IntegrationDesigntimeArtifacts`, `MessageMappingDesigntimeArtifacts`, and
    `ScriptCollectionDesigntimeArtifacts` — the same family of design-time artifact entity sets.
    That is a deliberate, specific difference, not a gap in that project's coverage.

  None of this is primary-source confirmation, but taken together it points the same direction:
  retaining the `PUT` call would have been shipping a guess this project's own standing rule
  says not to ship, especially given the explicit instruction not to keep an unverified `PUT`
  just because it resembles the Integration Flow API. **Resolution (Option A from the phase's
  conservative-fallback list): `sapintegrationsuite_value_mapping` no longer has an in-place
  Update.** `name`, `content`, and `content_hash` are now all `RequiresReplace`; changing any of
  them makes Terraform create a new value mapping artifact and delete the old one, which only
  relies on `Create` and `Delete` — both independently confirmed, unlike the update path.
  `internal/client/cloudintegration/value_mapping.go` no longer has an `UpdateValueMapping`
  function at all; see the comment there for the full reasoning. Implementing true in-place
  update via `ValueMappingDesigntimeArtifactSaveAsVersion` is deferred (ROADMAP.md, P3), once its exact
  contract can be confirmed against a live tenant or a reachable primary source — see
  `docs/sap-api-references.md`.
- **Delete**: `DELETE ValueMappingDesigntimeArtifacts(Id='{mapping_id}',Version='active')`.
- **Version**: Computed only, exactly like `sapintegrationsuite_integration_flow` — SAP
  assigns it, Terraform never asks the user to manage a version string, which is also what
  keeps a `terraform apply` with unchanged content from producing a new version.
- **Import**: `terraform import sapintegrationsuite_value_mapping.example UTILITIES/company-codes`.
- **`content_hash` stays explicit, not provider-computed**: re-evaluated this phase, since
  requiring the user to write both `content` and `content_hash` (typically
  `content_hash = filesha256(content)`) could look like redundant manual bookkeeping the
  provider could do itself. Keeping it explicit was kept as the better design:
  - `filesha256()` is a Terraform built-in function evaluated as part of the configuration
    graph, not something the user maintains by hand — in practice this is the same
    `source_hash`/`etag`-style pattern used by file-backed resources across the Terraform
    ecosystem (for example `aws_s3_object`'s `source_hash`), not something unusual to this
    provider.
  - Making `content_hash` fully provider-computed would mean reading the local file from
    inside a custom plan modifier during `terraform plan`, purely to detect drift before any
    API call happens. That is technically possible with the Terraform Plugin Framework, but it
    reads a local file as a side effect of planning outside the declarative config-graph value
    the plan is otherwise built from — a bigger change to this provider's planning model than
    this phase's scope (resolving the Value Mapping API contract) justifies.
  - `sapintegrationsuite_integration_flow` already ships the identical explicit
    `content`/`content_hash` pattern; changing it only for `sapintegrationsuite_value_mapping`
    would make the two file-based resources behave inconsistently for no API-contract reason.
    Any change here belongs in its own cross-cutting pass covering both resources, not folded
    into a Value Mapping-focused phase.
  - Conclusion: `content_hash` remains an explicit, user-supplied attribute for both resources.
- **Drift detection**: `Read` re-fetches metadata on every refresh, exactly like
  `sapintegrationsuite_integration_flow`; the same import limitation applies to `content` (see
  that resource's entry above) since SAP does not return a local file path for existing
  content. Right after import, with `content`/`content_hash` left unset in configuration,
  `terraform plan` shows no changes — `UseStateForUnknown` keeps the planned value equal to the
  (null) imported state, so `RequiresReplace` sees no diff and does not fire. The first apply
  that *does* supply `content`/`content_hash` to bring an imported artifact's content under
  management is expected to replace the resource (see the Update entry above), not update it in
  place — this is called out explicitly in `examples/brownfield/main.tf`.

## `sapintegrationsuite_value_mapping_deployment`

- **Purpose**: express the desired runtime deployment state of a value mapping, independent of
  its design-time content, mirroring `sapintegrationsuite_integration_flow_deployment` exactly.
- **SAP object**: the `DeployValueMappingDesigntimeArtifact` action plus the shared
  `IntegrationRuntimeArtifacts` entity for status/undeploy (see the table above — this is not
  a separate runtime entity per artifact type).
- **Schema and lifecycle**: identical shape to `sapintegrationsuite_integration_flow_deployment`
  — a required `mapping_version` input (normally wired to
  `sapintegrationsuite_value_mapping.<name>.version`) that both triggers redeployment when the
  design-time version changes and, once `Read` writes the actually-deployed version back into
  it, surfaces drift from an out-of-band redeploy or undeploy. Uses the same context-aware
  polling with exponential backoff and jitter, never a fixed sleep.
- **Shared runtime abstraction, re-reviewed this phase**: `internal/client/cloudintegration/runtime_artifact.go`
  and `internal/provider/runtime_deployment.go` are shared, unmodified, between
  `sapintegrationsuite_integration_flow_deployment` and this resource. Re-checked specifically
  for whether that sharing forces genuinely different artifact types through one abstraction —
  it does not: SAP's own documentation of the "Runtime Status API" describes
  `IntegrationRuntimeArtifacts` and the deploy/undeploy/status model as covering "currently
  deployed integration artifacts" generically, not per design-time artifact type, and the same
  third-party OData client referenced under Update above (`ci-mcp-server`) independently models
  `IntegrationRuntimeArtifacts` as a single generic entity set, not one per artifact type.
  Deliberately, the sharing stops at runtime status/undeploy: `DeployValueMappingDesigntimeArtifact`
  is its own action (distinct from `DeployIntegrationDesigntimeArtifact`, just following the
  same naming and query-parameter convention), and nothing about design-time Update/versioning
  is shared — value mapping and integration flow diverge there (see Update above), and the code
  does not pretend otherwise. "Shared polling is good only where semantics are genuinely
  shared" continues to hold for this abstraction.
- **Import**: `terraform import sapintegrationsuite_value_mapping_deployment.example UTILITIES/company-codes`.

## Deliberately not implemented this phase: entry-level management

`sapintegrationsuite_value_mapping_entry` (or any resource wrapping `UpsertValMaps`,
`UpdateDefaultValMap`, or `DeleteValMaps` directly) is **not implemented**. This was a genuine
evaluation, not a default:

- **Independent identity**: plausible — `UpdateDefaultValMap` addresses a scheme through a
  `ValMapId` GUID, suggesting schemes (and possibly entries) do have server-assigned
  identities of their own.
- **What blocks implementation**: the exact request/response payload shapes for all three
  operations, the precise granularity of `DeleteValMaps` (a single entry, a whole scheme, or a
  bulk selection), and how `ValMapId` is obtained from a `ValueMappingDesigntimeArtifacts` read
  could not be confirmed against any reachable primary source. Guessing these would violate
  this project's standing rule against inventing wire-format details, and would risk exactly
  the destructive-ambiguity problem the ownership-semantics question warns about: an
  under-specified "Terraform owns all entries" model could delete rows Terraform never created
  if the delete operation's actual scope turns out to be broader than assumed.
  - a confirmed precondition (entries can only be upserted into an *already-existing* scheme,
    confirmed by the documented 404-on-nonexistent-combination behavior) means entry management
    cannot stand alone as a resource anyway — it would depend on scheme structure that is
    itself part of the artifact's uploaded content, not a separate creatable object.
- **Path forward**: once the exact API contract is confirmed (ideally against a live tenant's
  `$metadata` and an actual request/response trace), the ownership model most consistent with
  the evidence gathered so far is likely closer to Option B (separate
  `sapintegrationsuite_value_mapping_entry` resources with a composite ID derived from the
  artifact plus the source/target agency-identifier tuple) than Option A (nesting, which would
  force replacing the entire entry set on any single change) — but this is not yet a design
  decision, only a direction for the next investigation.

## Message Mapping API model

A reusable Message Mapping is a **package-level design-time artifact**, structurally the same
kind of object as an integration flow or a value mapping — not the inline/local message mapping
step that can be configured directly inside an integration flow without ever becoming a
standalone artifact. The two are easy to conflate because SAP's own UI uses the same term
"Message Mapping" for both:

```
Integration Package
│
├── Message Mapping Artifact (MessageMappingDesigntimeArtifacts)
│      │
│      └── reusable by Integration Flows via a message mapping flow step
│
└── Integration Flow
       │
       └── may reference a Message Mapping Artifact, or define an inline
           mapping that never becomes a separate artifact
```

This feature is only about the first kind — the reusable, package-level artifact reachable
through `MessageMappingDesigntimeArtifacts`. It does not model, own, or manage inline mapping
configuration embedded directly in an integration flow's own content, which stays entirely
inside that integration flow's `content`/`content_hash` as far as this provider is concerned.

**Confirmed via SAP's own documentation** (SAP Help Portal, read through the `SAP-docs`
GitHub organization's markdown mirror of the Cloud Integration documentation — the primary,
official source, not a blog or forum): creating a message mapping artifact requires a package
context, a technical ID, a display name, and optional description; the artifact's content is a
mapping definition (`*.mmap`) file, uploaded as (or bundled inside) a ZIP archive; and the
artifact must be deployed before any integration flow that references it can use it — there is
no automatic deployment of a referenced message mapping when the referencing integration flow
is deployed. That last point matters for this provider's ownership model (see the deployment
resource below): a message mapping's deployment is independent of, and not implicitly triggered
by, anything the flows that reference it do.

**Confirmed operations** (Integration Content API, OData V2, `CloudIntegrationAPI` package):

| Operation | Method | Purpose |
|---|---|---|
| `MessageMappingDesigntimeArtifacts` | GET | Read artifact metadata (`Id`, `Version`, `Name`, `PackageId`), navigable from `IntegrationPackages` |
| `MessageMappingDesigntimeArtifacts` | POST | Create a new message mapping artifact from uploaded content |
| `MessageMappingDesigntimeArtifacts(Id=…,Version=…)` | PUT | Update an existing artifact's content; the version stays the same (tenant test, September 2026) — see Update below |
| `MessageMappingDesigntimeArtifacts(Id=…,Version=…)` | DELETE | Delete the artifact — see Delete below for what "delete" actually removes |
| `DeployMessageMappingDesigntimeArtifact?Id='…'&Version='…'` | POST | Deploy a specific version to the runtime (singular action name, matching the sibling actions for integration flows and value mappings) |
| `IntegrationRuntimeArtifacts(Id=…)` | GET / DELETE | Read deployment status / undeploy — the same shared runtime-artifacts entity already used by `sapintegrationsuite_integration_flow_deployment` and `sapintegrationsuite_value_mapping_deployment`, confirmed applicable here too (see the deployment resource entry below) |
| `MessageMappingDesigntimeArtifactSaveAsVersion?Id='…'&SaveAsVersion='…'` | POST | Save the current content under an explicit, caller-supplied version string; used by `save_as_version`, since a `PUT` does not change the version |
| Required roles | — | `WorkspacePackagesConfigure`, `WorkspacePackagesEdit`, `WorkspaceArtifactsDeploy` (same Integration Content API roles already required for integration flows and value mappings) |

**`SaveAsVersion` is a universal action across this API family, not a Value-Mapping-specific
concept.** While re-investigating the Value Mapping API contract in an earlier phase, this
project treated `ValueMappingDesigntimeArtifactSaveAsVersion` as if it might be Value Mapping's
*only* real update path, precisely because a plain `PUT` for that entity set could not be
confirmed to work and a third-party OData client explicitly disabled generic update for it. This
phase's research shows the actual shape of `SaveAsVersion` more clearly: it exists as
`IntegrationDesigntimeArtifactSaveAsVersion?Id='…'&SaveAsVersion='…'` for integration flows too
— an entity set where `PUT`-based update is independently confirmed and already implemented — so
`PUT` and `SaveAsVersion` are evidently not mutually exclusive alternatives in general; they
coexist. A tenant test in September 2026 settled their roles: a `PUT` replaces the content and
keeps the version, and `SaveAsVersion` is the only way to move the version. This does not change any
conclusion this project reached about Value Mapping specifically (that remains governed by its
own entity-specific evidence, documented in `docs/sap-api-references.md`), but it does mean the
mere existence of a `SaveAsVersion` action for Message Mapping is not, by itself, a reason to
avoid `PUT` here. What actually decides Message Mapping's Update model is entity-specific
evidence for *this* entity set — see Update below.

## `sapintegrationsuite_message_mapping`

- **Purpose**: manage the design-time content of a reusable Cloud Integration message mapping
  artifact, uploaded from a local content file, the same way
  `sapintegrationsuite_integration_flow` manages an integration flow's ZIP content.
- **SAP object**: `MessageMappingDesigntimeArtifacts`.
- **Identity**: composite `<package_id>/<mapping_id>`, both user-chosen business keys, exactly
  mirroring `sapintegrationsuite_integration_flow` and `sapintegrationsuite_value_mapping`.
- **Create**: `POST MessageMappingDesigntimeArtifacts` with `Id`, `Name`, `PackageId`, and
  base64-encoded `ArtifactContent`. No SAP-documented minimum-content precondition was found for
  message mapping (unlike value mapping's confirmed "at least one entry" requirement) — none is
  enforced client-side either, consistent with this provider's rule of not inventing
  constraints SAP does not document.
- **Read**: `GET MessageMappingDesigntimeArtifacts(Id='{mapping_id}',Version='active')`. The
  `Version='active'` alias is the same one already confirmed and tested for
  `sapintegrationsuite_integration_flow` and `sapintegrationsuite_value_mapping`, on the same
  entity family; not re-derived from scratch for this entity, but not assumed to be
  automatically valid for every future entity in this family either.
- **Update — implemented via `PUT`, on different grounds than Value Mapping**: unlike Value
  Mapping, this phase found positive, entity-specific evidence that `PUT` is the right mechanism
  here, not just an analogy with integration flows:
  - `MessageMappingDesigntimeArtifacts` shares the exact same `(Id, Version)` composite-key
    shape as `IntegrationDesigntimeArtifacts`, whose `PUT`-based update is
    independently confirmed and already implemented.
  - An independent third-party OData client built directly against this API
    (`github.com/lemaiwo/ci-mcp-server`, the same one whose configuration was used as
    corroborating evidence against Value Mapping's `PUT`) explicitly enables its generic
    "update" operation for `MessageMappingDesigntimeArtifacts`, the same as it does for
    `IntegrationDesigntimeArtifacts` and `ScriptCollectionDesigntimeArtifacts` — and unlike
    `ValueMappingDesigntimeArtifacts`, where that same tool explicitly disables it. That is a
    deliberate difference the tool's author drew between these entity sets, not a gap in
    coverage.
  - No SAP Knowledge Base Article or other evidence of a documented problem with changing a
    message mapping's version via `PUT` was found (unlike Value Mapping, where KBA 3502529
    documents exactly that problem for that entity set specifically).
  - `PUT` replaces the content of the same artifact ID; this provider treats that as a normal
    Terraform Update (no replace), because identity (`package_id`/`mapping_id`) does not
    change — the same design as `sapintegrationsuite_integration_flow`. An earlier version of
    this document said the `PUT` creates a new version; a tenant test in September 2026 showed
    that it keeps the version, answers 200 without a body, and must carry only `Name` and
    `ArtifactContent` (empty `Id` and `PackageId` were rejected with a 500).
  - `MessageMappingDesigntimeArtifactSaveAsVersion` backs the optional `save_as_version`
    attribute: it is the only way to move the version, which a deployment needs to notice new
    content unless `redeploy_triggers` is used.
- **Delete**: `DELETE MessageMappingDesigntimeArtifacts(Id='{mapping_id}',Version='active')`.
  Whether this removes only the active version or every version of the artifact was not
  confirmed against a primary source (the same open question already flagged for
  `sapintegrationsuite_value_mapping`'s Delete) — documented as an open item rather than
  asserted as "removes all versions".
- **Version**: Computed. On create it comes from `Bundle-Version` in the ZIP; a `PUT` keeps it;
  only `save_as_version` changes it. An apply with unchanged content does not touch the
  artifact at all (idempotent apply — see Terraform version semantics below).
- **Import**: `terraform import sapintegrationsuite_message_mapping.example UTILITIES/customer-mapping`.
- **Drift detection**: `Read` re-fetches metadata on every refresh. `content`/`content_hash`
  follow the exact same explicit, user-supplied, `RequiresReplace`-free pattern already
  documented and re-evaluated for `sapintegrationsuite_value_mapping` (see that resource's
  entry above for the content-hash design reasoning, which applies here unchanged); the same
  import limitation applies since SAP does not return a local file path for existing content.
- **Content format**: transported opaquely as a ZIP archive (base64-encoded `ArtifactContent`),
  exactly like `sapintegrationsuite_integration_flow` and `sapintegrationsuite_value_mapping`.
  This provider does not parse, validate, or interpret what is inside the archive — not the
  `.mmap` mapping definition, and not any XSD/WSDL/EDMX/Swagger-OpenAPI schema files SAP's
  mapping editor lets a message mapping reference for its source/target message structures. It
  only transports and manages the artifact as a unit, the same boundary already established for
  the other file-based design-time resources.

## Terraform version semantics for Message Mapping

Selected **Model A** from this phase's three candidate designs (content changes update the
design-time artifact in place; the provider does not expose or require a caller-supplied version
string; a separate deployment resource references whatever version the artifact resource last
produced) over:

- Model B (an explicit `desired_version` Terraform input) — rejected because nothing in the
  confirmed API contract requires or even exposes a caller-chosen version number for a normal
  content update; `SaveAsVersion`'s caller-supplied version string is an optional, separate
  step (`save_as_version`, added later), not a required part of the update path.
- Model C (draft vs. published version as separate resource concerns) — rejected because no
  SAP documentation surfaced a draft/published distinction for message mapping artifacts
  independent of the same `Version='active'` alias already used uniformly across this API
  family; inventing that distinction without evidence would violate this project's standing
  rule against guessing wire/state semantics.

Model A is also exactly what `sapintegrationsuite_integration_flow` already implements, so this
is a consistency choice as well as an evidence-based one: an unchanged `terraform apply` does not
create a new SAP version, because content-hash comparison against the last-read artifact is what
decides whether `Update` is even called, and the `version` attribute is Computed-only so
Terraform itself never treats a version-string mismatch as configuration drift.

## `sapintegrationsuite_message_mapping_deployment`

- **Purpose**: express the desired runtime deployment state of a message mapping, independent
  of its design-time content, mirroring `sapintegrationsuite_integration_flow_deployment` and
  `sapintegrationsuite_value_mapping_deployment` exactly.
- **SAP object**: the `DeployMessageMappingDesigntimeArtifact` action plus the shared
  `IntegrationRuntimeArtifacts` entity for status/undeploy.
- **Why `IntegrationRuntimeArtifacts` and not `BuildAndDeployStatus`**: SAP's Integration
  Content API also exposes a `BuildAndDeployStatus(TaskId='…')` entity, which this phase
  investigated specifically rather than assuming it applies here. The evidence found ties
  `BuildAndDeployStatus` to a different artifact family's build-then-deploy pipeline (OData API
  artifacts, which SAP documents as needing to be built before they can be deployed, keyed by a
  `TaskId` a build operation returns — not by the artifact's own `Id`), not to
  `MessageMappingDesigntimeArtifacts`. By contrast, SAP's own Runtime Status API documentation
  describes `IntegrationRuntimeArtifacts` as covering "currently deployed integration
  artifacts" generally, and secondary sources specifically describe message mappings as
  existing as runtime artifacts inside deployed runtime packages alongside integration flows,
  script collections, and adapters — monitored through that same shared entity. `Deploy` for a
  message mapping does not return a `TaskId` the way a `BuildAndDeployStatus`-fronted deploy
  would; it follows the same fire-and-poll-`IntegrationRuntimeArtifacts` shape already
  implemented for integration flows and value mappings. `runtime_artifact.go` and
  `runtime_deployment.go` are reused unmodified, since the semantics genuinely match — the same
  standard already applied when this abstraction was reviewed for value mapping. A tenant test
  (September 2026) confirmed it: the mapping deploy answered 202 without a body. Integration
  flows are different: their deploy returns a task ID, and `BuildAndDeployStatus` reports it
  (`SUCCESS`), so the flow deployment reads that task while the runtime artifact is missing.
- **Schema and lifecycle**: identical shape to `sapintegrationsuite_value_mapping_deployment` —
  a required `mapping_version` input (normally wired to
  `sapintegrationsuite_message_mapping.<name>.version`) that both triggers redeployment when the
  design-time version changes and, once `Read` writes the actually-deployed version back into
  it, surfaces drift from an out-of-band redeploy or undeploy. Uses the same context-aware
  polling with exponential backoff and jitter, never a fixed sleep.
- **No hidden coupling to referencing integration flows**: confirmed by SAP's own documentation
  that deploying an integration flow does not automatically deploy a message mapping it
  references — this provider does the same: `sapintegrationsuite_message_mapping_deployment` is
  a resource in its own right that a configuration must explicitly create, exactly mirroring
  SAP's own behavior rather than adding automatic-deployment behavior SAP itself does not
  provide. The message mapping resource does not scan, own, or modify integration flow content
  that references it; ownership of that reference stays entirely with whichever integration
  flow's content contains it.
- **Import**: `terraform import sapintegrationsuite_message_mapping_deployment.example UTILITIES/customer-mapping`.

## `data.sapintegrationsuite_message_mapping`

- **Purpose**: read-only lookup of an existing message mapping's metadata, mirroring
  `data.sapintegrationsuite_value_mapping` exactly — useful for brownfield adoption or for a
  configuration that wants to reference metadata/version of a mapping without owning it.
- **SAP object**: `GET MessageMappingDesigntimeArtifacts(Id='{mapping_id}',Version='active')`,
  the same read path the resource uses.
- Implemented because the read semantics are exactly as stable as
  `sapintegrationsuite_value_mapping`'s — not added merely for symmetry with every resource
  having a matching data source.

## Script Collection API model

A Script Collection is a bundle of reusable Groovy/JavaScript scripts, created within an
integration package so the same scripts can be shared across any number of integration flows,
mirroring the same reusable-artifact pattern already implemented for Message Mapping. Confirmed
via SAP's own documentation (again read through the `SAP-docs` GitHub organization's markdown
mirror, a legitimate primary source despite `help.sap.com` itself being blocked): a script
collection's technical ID must be unique for the entire tenant (not just the package), and its
description is capped at 120 characters — both are documented constraints, not this provider's
own invention, and neither is enforced client-side, consistent with this project's rule against
inventing constraints SAP does not document.

**Confirmed operations** (Integration Content API, OData V2, `CloudIntegrationAPI` package):

| Operation | Method | Purpose |
|---|---|---|
| `ScriptCollectionDesigntimeArtifacts` | GET | Read artifact metadata (`Id`, `Version`, `Name`, `PackageId`) |
| `ScriptCollectionDesigntimeArtifacts` | POST | Create a new script collection artifact from uploaded content |
| `ScriptCollectionDesigntimeArtifacts(Id=…,Version=…)` | PUT | Update an existing artifact's content; the version stays the same (tenant test, September 2026) |
| `ScriptCollectionDesigntimeArtifacts(Id=…,Version=…)` | DELETE | Delete the artifact — scope unconfirmed, same open item as every other design-time artifact in this family |
| `DeployScriptCollectionDesigntimeArtifact?Id='…'&Version='…'` | POST | Deploy a specific version to the runtime (singular action name, confirmed via SAP's own documentation, matching the sibling actions for every other design-time artifact type) |
| `IntegrationRuntimeArtifacts(Id=…)` | GET / DELETE | Read deployment status / undeploy — the same shared runtime-artifacts entity already used by every other `*_deployment` resource in this provider; script collections are documented as existing as runtime artifacts alongside integration flows, value mappings, and message mappings inside the same deployed runtime packages |
| Required roles | — | `WorkspacePackagesConfigure`, `WorkspacePackagesEdit`, `WorkspaceArtifactsDeploy` (same Integration Content API roles already required for every other design-time artifact type) |

**Update — implemented via `PUT`, same grounds as Message Mapping.** `ScriptCollectionDesigntimeArtifacts`
shares the exact `(Id, Version)` composite-key shape as `IntegrationDesigntimeArtifacts` and
`MessageMappingDesigntimeArtifacts`, both of which have confirmed `PUT`-based update
behavior. The same independent third-party OData client (`github.com/lemaiwo/ci-mcp-server`)
referenced when Message Mapping's Update model was decided explicitly enables generic update for
`ScriptCollectionDesigntimeArtifacts` too — the same as `IntegrationDesigntimeArtifacts` and
`MessageMappingDesigntimeArtifacts`, and unlike `ValueMappingDesigntimeArtifacts`, where it is
disabled. No SAP Knowledge Base Article or other evidence of a documented `PUT` problem for this
entity set was found. `PUT` here is therefore treated as a normal Terraform Update (no replace),
identical to `sapintegrationsuite_message_mapping` and `sapintegrationsuite_integration_flow`.

**Content format**: transported opaquely as a ZIP archive (base64-encoded `ArtifactContent`),
the same convention already confirmed for every other file-based design-time resource in this
provider (Integration Flow, Value Mapping, Message Mapping) — SAP's own Web IDE export/import
mechanism for design-time artifacts uses this ZIP+`ArtifactContent` shape uniformly across the
whole `CloudIntegrationAPI` package, not something inferred solely from Integration Flow. This
provider does not parse, validate, or execute the Groovy/JavaScript scripts inside the archive —
only transports and manages the artifact as a unit, the same boundary already established for
the other file-based resources.

## `sapintegrationsuite_script_collection`

- **Purpose**: manage the design-time content of a reusable Cloud Integration script collection,
  uploaded from a local content file, the same way `sapintegrationsuite_message_mapping` manages
  a message mapping artifact's ZIP content.
- **SAP object**: `ScriptCollectionDesigntimeArtifacts`.
- **Identity**: composite `<package_id>/<script_collection_id>`, mirroring every other
  file-based design-time resource in this provider. SAP additionally documents the technical ID
  as unique across the whole tenant, not just the package — this provider does not need to
  enforce that itself, since a duplicate ID is something SAP's own Create call would reject.
- **Create**: `POST ScriptCollectionDesigntimeArtifacts` with `Id`, `Name`, `PackageId`, and
  base64-encoded `ArtifactContent`. No SAP-documented minimum-content precondition was found
  (unlike value mapping's confirmed "at least one entry" requirement).
- **Read**: `GET ScriptCollectionDesigntimeArtifacts(Id='{script_collection_id}',Version='active')`,
  the same `Version='active'` alias already confirmed for every other entity in this family.
- **Update**: `PUT` against the keyed `(Id, Version)` entity — see the Update entry in the API
  model above for the full reasoning. Treated as a normal Terraform update (no replace), since
  identity does not change.
- **Delete**: `DELETE ScriptCollectionDesigntimeArtifacts(Id='{script_collection_id}',Version='active')`.
  Whether this removes only the active version or every version of the artifact is unconfirmed
  against a primary source, the same open item already flagged for every sibling resource.
- **Version**: Computed. It comes from `Bundle-Version` on create, a `PUT` keeps it, and
  `save_as_version` changes it; an unchanged `terraform apply` does not touch the artifact, for the same reason already established for the other file-based
  resources (content-hash comparison decides whether Update is even called).
- **Import**: `terraform import sapintegrationsuite_script_collection.example UTILITIES/shared-scripts`.
- **Drift detection**: `Read` re-fetches metadata on every refresh; `content`/`content_hash`
  follow the same explicit, user-supplied pattern already documented and re-evaluated for
  `sapintegrationsuite_value_mapping` and `sapintegrationsuite_message_mapping` — see those
  resources' entries for the content-hash design reasoning, which applies here unchanged.

## `sapintegrationsuite_script_collection_deployment`

- **Purpose**: express the desired runtime deployment state of a script collection, independent
  of its design-time content, mirroring `sapintegrationsuite_message_mapping_deployment` exactly.
- **SAP object**: the `DeployScriptCollectionDesigntimeArtifact` action plus the shared
  `IntegrationRuntimeArtifacts` entity for status/undeploy.
- **Schema and lifecycle**: identical shape to `sapintegrationsuite_message_mapping_deployment`
  — a required `script_collection_version` input that both triggers redeployment when the
  design-time version changes and, once `Read` writes the actually-deployed version back into
  it, surfaces drift from an out-of-band redeploy or undeploy. Uses the same context-aware
  polling with exponential backoff and jitter, never a fixed sleep. `runtime_artifact.go` and
  `runtime_deployment.go` are reused unmodified, since script collections are documented as
  sharing the same runtime-artifact model as every other design-time artifact type here.
- **No hidden coupling to referencing integration flows**: a script collection's deployment is
  a resource in its own right that a configuration must explicitly create; this provider does
  not scan or modify integration flow content to manage that reference, the same ownership
  boundary already established for message mapping.
- **Import**: `terraform import sapintegrationsuite_script_collection_deployment.example UTILITIES/shared-scripts`.

## `data.sapintegrationsuite_script_collection`

- **Purpose**: read-only lookup of an existing script collection's metadata, mirroring
  `data.sapintegrationsuite_message_mapping` exactly.
- **SAP object**: `GET ScriptCollectionDesigntimeArtifacts(Id='{script_collection_id}',Version='active')`.

## `sapintegrationsuite_integration_adapter`

- **Purpose**: manage the design-time content of a custom Integration Adapter (a `*.esa`
  archive built with the SAP Adapter SDK), uploaded from a local content file. Cloud Foundry
  environment only.
- **SAP object**: `IntegrationAdapterDesigntimeArtifacts` (Integration Content API, OData V2).
  See `docs/sap-api-references.md` for the full evidence-tier breakdown — this entity has
  materially thinner public documentation than the other design-time artifact types above.
- **Desired state**: yes, in the same shape as the other file-backed design-time resources —
  content is opaque binary content plus a hash, not scalar fields.
- **Identity**: `id`, SAP's confirmed sole key for this entity (unlike every sibling design-time
  artifact type, which all use a composite `(Id, Version)` key). SAP documents this ID as unique
  across the entire tenant, not merely the containing package.
- **Read-back**: `GET IntegrationAdapterDesigntimeArtifacts(Id='{id}')` (not confirmed by an
  adapter-specific example, but the standard convention every entity set in this API follows).
  `package_id` is deliberately populated from the caller's own state/plan value rather than
  trusted from the GET response, the same defensive choice already made for
  `sapintegrationsuite_script_collection`'s `PackageId` handling, since this project could not
  confirm GET reliably returns it.
- **Create**: `POST IntegrationAdapterDesigntimeArtifacts` with
  `Id`/`PackageId`/`Name`/`ArtifactContent` (base64). A tenant `$metadata` document confirms
  these property names (plus `Version` and `Description`, both filled by SAP from the *.esa
  file). There is still no SAP-published Create example for this entity.
- **Update — deliberately absent**: SAP documents that importing a duplicate `Id` is rejected as
  an error, which is positive evidence against a working reimport-to-update flow. No
  PUT/PATCH/reimport example was found for this entity anywhere. Every attribute —
  `id`, `package_id`, `name`, `content`, `content_hash` — is
  `RequiresReplace`. This is a stricter model than every sibling design-time artifact type
  (which all have at least a confirmed content-replacing `PUT`), chosen deliberately given the
  much thinner evidence base and the concrete negative signal from the duplicate-ID error.
- **`type`/`application` — removed**: the import dialog's type and application are UI
  classifications without an API property; the tenant `$metadata` has neither on
  `IntegrationAdapterDesigntimeArtifact`. Earlier releases sent them anyway. `description`
  (read-only, taken from the *.esa) replaces them in the schema.
- **Delete**: `DELETE IntegrationAdapterDesigntimeArtifacts(Id='{id}')` — confirmed directly
  from SAP's own example request, the strongest evidence tier this entity has for any operation
  besides Deploy.
- **Import**: composite `<package_id>/<id>`, even though `id` alone is SAP's confirmed key for
  Delete/Deploy. This is a deliberate Terraform-side choice: `package_id` is `Required` and
  `RequiresReplace`, and this project could not confirm it is recoverable from a plain GET by
  `id`. Importing by `id` alone would leave it unrecoverable, and since it is `RequiresReplace`,
  the first plan after import would want to destroy and recreate the adapter purely because
  Terraform never learned its package — a materially worse outcome than asking the practitioner
  to supply a value they already know. See `docs/guides/integration-adapters.md`.
- **File size limit**: no SAP-documented maximum was found; this provider applies only its own
  generic 32 MiB protective bound, the same one already used for script collections and value
  mappings, not an SAP-documented limit.
- **Content handling**: treated as opaque. Never unpacked, executed, or inspected beyond the
  size bound and content-hash check already applied to every file-backed resource in this
  provider — see the "Content handling" note in `docs/guides/integration-adapters.md`.

## `sapintegrationsuite_integration_adapter_deployment`

- **Purpose**: express the desired runtime deployment state of a custom Integration Adapter,
  independent of its design-time content lifecycle — the same design-time/runtime split as
  every other artifact type this provider manages.
- **SAP object**: the confirmed `DeployIntegrationAdapterDesigntimeArtifact` action (`POST
  ...?Id='{id}'`, no `Version` parameter — see `docs/sap-api-references.md`), plus the shared
  `IntegrationRuntimeArtifacts` entity for status/undeploy, reused by analogy (not
  independently confirmed for this specific artifact type).
- **Desired state**: "this adapter Id should be deployed." Unlike every sibling `*_deployment`
  resource, there is no `*_version` attribute: the confirmed deploy action has nothing to
  select besides `Id`, since the design-time entity has no confirmed `Version`-keyed identity.
- **Identity**: the adapter's `id` directly (no composite needed — deployment has no package
  concept, matching the confirmed Deploy/Delete request shapes).
- **Create/Update**: `POST DeployIntegrationAdapterDesigntimeArtifact?Id='{id}'`, then poll
  `IntegrationRuntimeArtifacts(Id='{id}')` until `Status` is `STARTED` or `ERROR`, reusing
  `waitForRuntimeArtifact` — the exact same context-aware exponential-backoff-with-jitter poller
  every other `*_deployment` resource in this provider already uses. Update is unreachable in
  practice (`adapter_id` is the only settable attribute besides `timeouts`, and it is
  `RequiresReplace`).
- **Delete**: `DELETE IntegrationRuntimeArtifacts(Id='{id}')` (undeploy), reused by analogy from
  the shared runtime-artifacts entity. `404` is treated as already-undeployed, the same
  convention as every sibling `*_deployment` resource.
- **Drift detection**: `Read` re-fetches `IntegrationRuntimeArtifacts(Id='{id}')`; a `404`
  removes the resource from state (undeployed externally).
- **Import**: `terraform import sapintegrationsuite_integration_adapter_deployment.example <id>`
  — a single value, matching the confirmed single-key deploy/runtime lifecycle (unlike the
  design-time resource's composite import, which exists only because `package_id` cannot be
  recovered any other way).
- **Ordering against a consuming integration flow**: SAP states an adapter must be deployed
  before a consuming integration flow is deployed. This provider does not parse integration flow
  content to discover adapter dependencies automatically — practitioners declare it via
  `depends_on`. See `docs/guides/integration-adapters.md`.
- **Restart deliberately not modeled**: SAP's UI exposes a Restart action for a deployed
  adapter. This is an imperative, operational action, not declarative desired state — the same
  reasoning behind not managing any other artifact type's restart, and this provider has no
  `sapintegrationsuite_integration_adapter_restart` resource.

## `data.sapintegrationsuite_integration_adapter`

- **Purpose**: read-only lookup of an existing custom Integration Adapter's metadata by its
  tenant-wide ID, for brownfield discovery before import or for referencing an adapter this
  provider does not itself manage from an
  `sapintegrationsuite_integration_adapter_deployment` resource.
- **SAP object**: `GET IntegrationAdapterDesigntimeArtifacts(Id='{id}')`.
- **Why no `package_id` attribute here (unlike the resource)**: the resource needs `package_id`
  to satisfy `Required`+`RequiresReplace` semantics on create/import even though it cannot
  confirm the GET response includes it; the data source has no such constraint — it simply does
  not expose a field this project could not confirm the API actually returns.
- **Why no `data.sapintegrationsuite_integration_adapters` collection data source**: no
  confirmed list/filter contract was found for this entity set, unlike `ServiceEndpoints`,
  which SAP explicitly documents `Name`/`Protocol` `$filter` support for. Implementing a
  collection lookup without confirmed filtering semantics risks either silently returning
  everything (expensive, and a different contract than what a practitioner would reasonably
  expect from named filter arguments) or guessing at filter parameter names SAP might reject —
  both worse than not implementing it yet.

## `data.sapintegrationsuite_service_endpoints`

- **Purpose**: read-only discovery of the runtime service endpoints (entry point URLs, API
  definition document links) SAP has generated for deployed Cloud Integration content.
- **SAP object**: `ServiceEndpoints` (Integration Content API, OData V2), with `EntryPoints` and
  `ApiDefinitions` expanded in a single combined request. See `docs/sap-api-references.md` for
  the full field-by-field contract and its confirmation sources.
- **Why a data source, not a resource (§17 suitability check)**: SAP documents no create,
  update, or delete operation for `ServiceEndpoints` at all — it is generated automatically from
  deployed content and removed automatically on undeploy. A Terraform resource needs a
  Create/Delete pair this provider actually controls; here, the closest thing to "desired state"
  already has its own resource (`sapintegrationsuite_integration_flow_deployment` and siblings),
  and `ServiceEndpoints` is purely the runtime *consequence* of that state, the same category of
  gap as Message Processing Logs and Message Stores (see `docs/provider-scope.md`) but with one
  difference: unlike those, a service endpoint's *existence* is a direct, deterministic function
  of desired-state content this provider does manage, which is exactly what makes it a useful
  discovery data source rather than out-of-scope operational data.
- **Why no singular `data.sapintegrationsuite_service_endpoint` (§12/§13 suitability check)**:
  SAP's own example requests filter `ServiceEndpoints` by `Name`, but no source confirms `Name`
  (or `Name`+`Protocol` together) is guaranteed unique across every possible deployment
  configuration — for example, one integration flow exposed through more than one adapter at
  once is plausible and not ruled out by anything this project found. A singular data source
  that silently picked "the first match" when more than one exists would be actively misleading.
  This provider only implements the collection form; filtering by `name` narrows results in
  practice without pretending to guarantee uniqueness SAP itself does not document.
- **Identity**: this provider invents no synthetic ID for either the data source instance or
  individual endpoint entries — `name`/`protocol` (the confirmed, filterable properties) identify
  each entry as returned; there is no `id` attribute anywhere in this schema, following the same
  precedent as `data.sapintegrationsuite_partner_string_parameters`.
- **Filters**: `name` and `protocol`, mapping directly to SAP's documented `$filter` support —
  the only two properties SAP documents as filterable. No raw/arbitrary OData `$filter` passthrough
  is exposed, consistent with this provider's general policy of only exposing confirmed,
  documented query capability.
- **Determinism**: SAP does not document a guaranteed collection order, so the client layer
  (`internal/client/cloudintegration/service_endpoint.go`) sorts the top-level list by `name`
  then `protocol`, and each entry's `entry_points`/`api_definitions` by their own fields, before
  the provider layer ever sees them — Terraform state is therefore stable across applies even if
  SAP's own response order is not.
- **No resource invocation**: Read only issues `GET` requests. It never calls a discovered entry
  point URL, downloads a referenced API definition document, or triggers any deployment,
  redeployment, or other runtime change.
- **Dependency pattern**: since this data source is not wired to any specific Terraform resource
  by reference (it filters by `name`, a plain string), a practitioner must add an explicit
  `depends_on` pointing at the relevant `sapintegrationsuite_integration_flow_deployment` (or
  equivalent) resource if they want to guarantee ordering within a single apply. This provider
  does not attempt to infer that dependency automatically — see `docs/guides/service-endpoints.md`.
- **Eventual consistency**: no SAP source found confirms or rules out a propagation delay
  between a deployment completing and its service endpoint becoming visible through this API.
  This provider does not implement a blind fixed-duration retry to paper over an unconfirmed
  gap — see `docs/guides/service-endpoints.md` for the reasoning and the practical mitigation
  (re-plan/re-apply) until a specific, bounded window is confirmed.

## `sapintegrationsuite_custom_tag_configuration`

- **Purpose**: manage the tenant-wide Custom Tag Configuration — the set of attributes
  (mandatory or optional, each with an optional closed value list) integration package owners
  classify their packages with.
- **SAP object**: `CustomTagConfigurations` (Integration Content API, OData V2), confirmed
  singleton key `CustomTags`. See `docs/sap-api-references.md` for the full confirmed request/
  response shapes and the complete singleton-semantics investigation.
- **Desired state**: yes — the complete tag list is the desired state, no different in spirit
  from any other resource's set of managed fields; the difference is that there is exactly one
  instance of this resource per tenant rather than many.
- **Identity**: `id`, always the literal string `"CustomTags"` — Computed, never something a
  practitioner chooses, since SAP documents no other identity for this entity.
- **Read-back**: `GET CustomTagConfigurations('CustomTags')/$value`, confirmed verbatim. Decoded
  directly as the documented JSON shape rather than through the standard OData entity envelope,
  since `$value` is OData's raw-media-stream convention.
- **Create/Update — same operation, confirmed verbatim**: `POST CustomTagConfigurations` with
  `Overwrite=true`, sending the complete tag list every time; there is no partial/delta update.
  Both Create and Update call the same client method (`SetCustomTagConfiguration`), always with
  `Overwrite=true`, since SAP's documentation never shows what a plain `POST` does once a
  configuration already exists.
- **Delete (§17/§26 suitability check) — no confirmed destroy mechanism, therefore no fake one**:
  this project reverified SAP's documentation specifically looking for a delete or clear
  operation and found none — not even an unconfirmed one, unlike most other gaps in this
  provider's evidence base. Given that, and given this provider's standing policy against both a
  silent Delete no-op and a guessed "empty overwrite deletes everything" implementation, Delete
  returns an explicit, actionable error diagnostic instead. This is a deliberate correctness
  choice: a `terraform destroy` that silently succeeded while leaving the tenant's governance
  configuration untouched would be a far worse outcome than a clear failure directing the
  practitioner to `terraform state rm` (to stop managing it without touching the tenant) or the
  SAP UI (to actually clear it). See `docs/guides/custom-tag-configurations.md`.
- **Ordering and duplicate handling**: `tags` and each tag's `permitted_values` are modeled as
  Terraform **sets**, not lists, since SAP's documentation never states that submission or
  response order carries meaning — this guarantees `terraform plan` reports no change when SAP
  returns the same tags in a different order. The request body sent to SAP is independently
  canonicalized (sorted by tag name, then by permitted value) before encoding, in
  `internal/client/cloudintegration/custom_tag_configuration.go`, so what is actually
  transmitted is deterministic regardless of the order Terraform's own set-to-slice conversion
  happens to produce. Tag name uniqueness is enforced by this resource's own `ValidateConfig`
  implementation (`resource.ResourceWithValidateConfig`), independent of the Terraform type
  system, since two same-named tags with different other fields would otherwise be two distinct,
  contradictory set elements rather than a type-level conflict Terraform would catch on its own.
- **Import**: `terraform import sapintegrationsuite_custom_tag_configuration.example CustomTags`
  — the literal fixed key, rejected outright if anything else is supplied, rather than accepted
  and left to fail later against an API that only recognizes that one key.
- **Feature catalog status**: `partial` — Create/Read/Update are fully implemented against a
  confirmed API contract; the permanently absent Delete is the reason for `partial` rather than
  `supported`.

## Number Ranges / Variables / Data Stores / Data Store Entries — suitability check

Before any implementation, every object in SAP's Message Stores API family was walked through
the same 14-question suitability check (who creates it; configuration vs. runtime state; is
identity stable; is Create/Read/Update/Delete public; can drift be detected reliably; would
reconciliation be safe; could `apply` reset runtime state; could `destroy` destroy productive
data; is import meaningful; does it belong in desired-state infrastructure; Resource/Data
Source/unsupported/out of scope). Full API evidence is in
`docs/sap-api-references.md`; this section records the suitability conclusions.

### Number Range — resource with a version-gated counter

> **Re-audit September 2026.** Points 5, 7, 8, 11 and 12 below originally said "no": SAP still
> documents only POST and PUT, but a tenant answered `GET NumberRanges('<name>')` with the entity
> and `DELETE NumberRanges('<name>')` with `202` (the object was gone afterwards). The resource
> now reads, deletes and imports; the updated answers are given inline.

1. **Who creates it**: a practitioner or integration developer, explicitly, via the Monitor
   application or this API — the only object in this family a human deliberately defines rather
   than one that comes into existence as a side effect of deployed content running.
2. **Configuration vs. runtime state — both, and they must be separated**: `Name`/`MinValue`/
   `MaxValue`/`Description`/`Rotate`/`FieldLength` are static desired configuration.
   `CurrentValue` (the UI's "Next Value") is live runtime state that advances every time deployed
   EDI/EDIFACT content consumes a number — it must never be treated as ordinary desired-state
   input Terraform reconciles on every apply.
3. **Is identity stable**: yes — `Name` is SAP's documented OData key, and SAP's documentation
   never describes rename semantics, so it is `RequiresReplace`.
4. **Is Create public**: yes, confirmed (`POST /NumberRanges`).
5. **Is Read public**: not documented by SAP, but verified on a tenant: `GET` by name returns
   every field as sent (the collection rejects `$top` with `501`).
6. **Is Update public**: yes, confirmed (`PUT /NumberRanges('{name}')`).
7. **Is Delete public**: not documented (the UI shows "Undeploy"), but verified on a tenant:
   `DELETE` by name answers `202` and the object is gone.
8. **Can Terraform detect drift reliably**: yes for the static fields, through the GET.
9. **Would Terraform reconciliation be safe**: for the static fields, yes, if Update never
   silently resends a value it cannot confirm is current. For `CurrentValue`, no — no design can
   make blind reconciliation of an unreadable, externally-advancing counter safe.
10. **Could `apply` accidentally reset runtime state**: yes, this is the central risk this design
    exists to prevent — see the Update design below.
11. **Could `destroy` destroy productive runtime data**: yes — deleting a number range that
    deployed content uses breaks that content; the guide says to remove references first.
12. **Is import meaningful**: yes, by name; the first apply after an import records the version
    marker without sending the counter.
13. **Does it belong in desired-state infrastructure management**: the static configuration
    does; the runtime counter categorically does not.
14. **Resource / Data Source / unsupported / out of scope**: **Resource, but a deliberately
    narrower one than every other resource in this provider** — see below.

**Design decision and why**: a conventional Terraform resource contract (Create, Read that
verifies state, Update that reconciles drift, Delete, Import) cannot be honestly implemented
against an API with no GET and no DELETE. The two remaining options were "implement nothing" or
"implement a write-only-lifecycle resource that is explicit about what it cannot do." This
provider chose the latter, on the basis that Create+Update alone still give a practitioner a
real, auditable, idempotent-on-the-Terraform-side way to push desired static configuration —
which "implement nothing" would deny them for no benefit, since the static-configuration part of
this object genuinely is something a human defines and wants version-controlled. This differs
in kind from Variables/DataStores/DataStoreEntries below, where nothing is ever practitioner-
authored in the first place.

- **Identity**: `id`/`name` — SAP's confirmed OData key, `RequiresReplace`.
- **Static configuration**: `min_value`, `max_value`, `description`, `rotate`, `field_length` —
  ordinary `Required`/`Optional` attributes, all resent on every Create and Update. `min_value`/
  `max_value`/`field_length` are Terraform **strings**, not numbers: SAP's wire format is a JSON
  string for every numeric field, and SAP documents values up to 14–15 digits, well beyond what
  this provider is willing to assume any numeric encoding preserves exactly without confirming
  it against `$metadata` (which could not be reached — see `sap-api-references.md`). A custom
  validator enforces unsigned-decimal-digit-string shape, `max_value` < 15 digits, `field_length`
  between 0 and 14, and (via `ValidateConfig`) `min_value <= max_value` using `math/big` integer
  comparison, never string comparison.
- **The runtime counter — `current_value_wo` / `current_value_wo_version`**: modeled as a
  `WriteOnly` attribute paired with a version marker, the same pattern this provider already uses
  for credential rotation (`password_wo`/`password_wo_version` on
  `sapintegrationsuite_user_credential`). `current_value_wo` is `Required` (SAP's Create example
  always includes `CurrentValue`) and is sent to SAP:
  - **On every Create.**
  - **On Update, only when `current_value_wo_version` differs from the prior state.**
  - Every other Update sends the counter SAP currently holds. Earlier releases omitted
    `CurrentValue` instead, because there was no GET; a tenant test (September 2026) showed that
    SAP rejects a PUT without `CurrentValue` with `500` and leaves the object unchanged, so every
    ordinary update would have failed. With GET verified, the original design ("GET the live
    value, then PUT it back unchanged") is used. The client refuses an update without a counter
    (`TestClient_UpdateNumberRange_RejectsMissingCurrentValue`), and the resource sends the live
    value (`TestNumberRangeResource_Update_SendsLiveCounterWhenVersionUnchanged`). A number
    consumed between the GET and the PUT would be handed out again; the window is one round trip.
- **Read** (since the September 2026 re-audit): `GET NumberRanges('<name>')` without query
  options. The static fields come from SAP, so UI changes show as drift; a `404` removes the
  resource from state. `current_value`, `deployed_by` and `deployed_on` are computed; the
  version marker is kept from state.
- **Create**: reads the name first and stops if it exists, because SAP does not document what a
  POST on an existing name does. Every write answers `202` without a body, so Create and Update
  read the object back.
- **Delete**: `DELETE NumberRanges('<name>')`; a `404` counts as already deleted.
- **Import**: by name. The version marker is null afterwards; Update pushes the configured
  counter only when the prior marker is non-null and changed, so the first apply after an import
  sends the live counter back and never resets it.
- **Names**: a tenant rejected a hyphenated name with `500` while the same request with a plain
  name succeeded; a validator rejects hyphens at plan time.
- **Feature catalog status**: `supported`.

### Variable — no resource, no data source

1. **Who creates it**: exclusively an integration flow's "Write Variables" design-time step —
   never a practitioner acting directly against this API.
2. **Configuration vs. runtime state**: entirely runtime state — an arbitrary value written
   during message processing, expiring after 400 days of inactivity.
3. **Is identity stable**: the composite key (`VariableName`, `IntegrationFlow`) is stable, but
   irrelevant to suitability given the points below.
4. **Is Create public**: no.
5. **Is Read public**: yes, but only a single-item download (`GET .../Variables(...)/$value`)
   returning the raw value with no metadata — no collection GET, so nothing to discover from.
6. **Is Update public**: no.
7. **Is Delete public**: not confirmed via a REST example, though a `DataStoresAndQueuesDelete`
   role template documented for this family implies some delete capability exists somewhere.
8. **Can Terraform detect drift reliably**: moot — there is nothing Terraform could have written
   in the first place.
9. **Would Terraform reconciliation be safe**: moot, same reason.
10. **Could `apply` accidentally reset runtime state**: moot — no Create/Update exists to expose.
11. **Could `destroy` destroy productive runtime data**: moot — no Delete exposed.
12. **Is import meaningful**: no — there is no Terraform-manageable state to import into.
13. **Does it belong in desired-state infrastructure management**: no — this is runtime business
    data flowing through deployed content, category-identical to Message Processing Logs.
14. **Resource / Data Source / unsupported / out of scope**: **unsupported, `out_of_scope`**.

A read-only `data.sapintegrationsuite_variable` was seriously considered — the GET is real, and
there is a plausible use case (referencing a shared global variable's value from other Terraform
configuration). It was rejected: the only confirmed read endpoint returns nothing but the raw
runtime value itself, with no safer metadata-only projection to fall back to, and this provider's
`docs/provider-scope.md` principle against placing arbitrary runtime business content into
Terraform state applies without exception here — every `terraform plan`/`refresh` would either
show constant spurious churn as the underlying value changes for reasons outside Terraform's
control, or silently capture a snapshot of business data into `.tfstate` on every apply. Neither
is acceptable.

### Data Store — no resource, no data source

1. **Who creates it**: implicitly, the first time an integration flow's Data Store Write step
   (or an XI adapter configured with `Temporary Storage = Data Store`) writes an entry — never a
   practitioner acting directly.
2. **Configuration vs. runtime state**: a Data Store has no configuration of its own at all; it
   is purely a runtime container, identified by (`DataStoreName`, `IntegrationFlow`, `Type`).
3–7. **Identity / Create / Read / Update / Delete**: no independent Create, Update, or Delete
   API exists for this entity at all. The one documented GET
   (`/DataStores?overdueonly=true`) is an aggregate monitoring endpoint (message counts), not a
   configuration read.
8–13. All moot for the same reason as Variables above — there is no Terraform-manageable desired
   state to reconcile, detect drift on, or import.
14. **Resource / Data Source / unsupported / out of scope**: **unsupported, `out_of_scope`** —
    the one GET is the same class of runtime monitoring data as
    `cloud_integration.message_processing_logs`, already excluded on identical grounds.

### Data Store Entry — no resource, no data source

1. **Who creates it**: the same Data Store Write step that creates its parent Data Store, once
   per message.
2. **Configuration vs. runtime state**: entirely runtime state — `Status`, `MessageId`, `DueAt`,
   `CreatedAt`, `RetainUntil` describe a specific processed business message, not configuration.
3–7. **Identity / Create / Read / Update / Delete**: identity (`Id`, `DataStoreName`,
   `IntegrationFlow`, `Type`) is stable, and two GETs are confirmed (single entry; all entries
   for a store), but no Create, Update, or REST Delete exists — the documented Delete is a
   design-time integration-flow step operating on messages as they are processed, explicitly
   scoped to "single entries only," never a REST call this provider could wrap in
   `terraform destroy`.
8–13. Moot — there is no Terraform-manageable desired state here, only runtime business-message
   records.
14. **Resource / Data Source / unsupported / out of scope**: **`out_of_scope`, not merely
    `not_implemented`** — this is business message content and processing status, the same
    category this provider already refuses to place into Terraform state for Message Processing
    Logs, and "an entry can be deleted" does not make deleting it a `terraform destroy` any more
    than deleting a message processing log record would be.

## `sapintegrationsuite_partner_string_parameter` / `..._binary_parameter`

- **Purpose**: manage a single named value (text or binary) scoped to a Partner Directory
  partner ID (Pid).
- **SAP object**: `StringParameters` / `BinaryParameters` (Partner Directory API, OData V2 —
  SAP documents full CRUD for both, `PUT` for update).
- **Identity**: composite `(Pid, Id)`; the string-parameter resource's Terraform ID is
  `<partner_id>/<parameter_id>`, the same composite-ID pattern already used throughout this
  provider (`splitCompositeID`).
- **Create**: `POST StringParameters` / `POST BinaryParameters`. The Pid does not need to
  already exist — SAP creates it implicitly on first reference (see the Partners entry below).
- **Update**: `PUT` (full replace) addressed by the `(Pid, Id)` key predicate; only the mutable
  field(s) (`Value`, or `ContentType`+`Value` for binary) are sent, since Pid/Id are the key and
  never mutable.
- **Delete**: `DELETE` by the same key predicate.
- **Binary parameter specifics**: file-based (`content`/`content_hash`), the same pattern as
  every other file-based resource in this provider. `content_type` is not restricted to a fixed
  validator list, since SAP's documented values (`xml`, `xsl`, `xsd`, `json`, `text`, `zip`,
  `gz`, `zlib`, `crt`) coexist with encoding-suffixed variants like `xml;encoding=UTF-8` that a
  short fixed list would incorrectly reject. The value limit checked client-side before any
  request is 1,572,864 bytes, the `MaxLength` of `BinaryParameter.Value` in the tenant
  `$metadata`; older SAP pages still mention 260 KB.
- **Security note**: Partner Directory data is stored unencrypted. Neither resource is meant
  for secrets — see the User Credential Parameter entry below and
  `docs/guides/partner-directory.md`.
- **Import**: `terraform import sapintegrationsuite_partner_string_parameter.example PartnerZ/ReceiverAddress`.

## `sapintegrationsuite_alternative_partner`

- **Purpose**: manage a mapping from an external identity tuple (`agency`, `scheme`,
  `external_id`) to an internal Partner ID.
- **SAP object**: `AlternativePartners` (Partner Directory API, OData V2).
- **Identity — the unusual part**: SAP's actual OData entity key is not the plain
  `Agency`/`Scheme`/`Id` strings but their hex-encoded form (`Hexagency`/`Hexscheme`/`Hexid`),
  confirmed against a documented SAP example (`"agency1"` → `"6167656e637931"`). This provider
  computes the hex key internally (`EncodeAlternativePartnerKey`,
  `encoding/hex.EncodeToString` on the UTF-8 bytes) and never exposes it as something a
  practitioner sets — only the plain `agency`/`scheme`/`external_id` attributes exist in the
  schema.
- **Create**: `POST AlternativePartners` with the plain `Agency`/`Scheme`/`Id`/`Pid` fields; SAP
  computes and stores the hex key itself.
- **Read/Update/Delete**: addressed by the hex key predicate this provider computes from the
  same three plain values it already has in state.
- **Update**: `PUT` repoints an existing mapping at a different `partner_id`; `agency`,
  `scheme`, and `external_id` together are the mapping's identity and force replacement.
- **Import ID**: the three hex-encoded segments joined by `/` —
  `<hex_agency>/<hex_scheme>/<hex_external_id>`. A "pretty" syntax using the plain strings
  directly was rejected: all three can themselves contain `/` or other characters that would
  make a naive split ambiguous. Hex text can never contain `/`, so this format is always
  unambiguous to parse back apart, with dedicated round-trip tests covering ASCII, spaces,
  Unicode, and punctuation.

## `sapintegrationsuite_partner_authorized_user`

- **Purpose**: manage a mapping from a communication user to the Partner ID it is authorized to
  act as for inbound communication.
- **SAP object**: `AuthorizedUsers` (Partner Directory API, OData V2). SAP documents this as
  many-to-one: one communication user maps to exactly one Pid, a Pid can have several.
- **Identity**: `User` is the entity's key; the Terraform ID is the same value.
- **Update**: `PUT` repoints an existing mapping at a different `partner_id`; `user` forces
  replacement (it is the mapping's identity).
- **Case**: SAP stores `User` lowercased (its example creates `MyUser` and returns `myuser`).
  A mixed-case value would come back different from the configuration, so the resource and the
  data source reject uppercase letters at plan time instead of normalizing silently.
- **Boundary**: this resource manages only the Partner Directory mapping, never the BTP user,
  OAuth client, or communication user credential itself.
- **Import**: `terraform import sapintegrationsuite_partner_authorized_user.example commuser1`.

## `sapintegrationsuite_partner_user_credential_parameter`

- **Purpose**: manage a communication username/password credential scoped to a Pid, treated as
  a distinct, security-sensitive case rather than "one more Partner Directory parameter type."
- **SAP object**: `UserCredentialParameters` (Partner Directory API, OData V2). SAP documents
  `POST` with `Pid`/`Id`/`User`/`Password` in the request body, and that this entity type (along
  with `CertificateUserMapping`) cannot be combined with other entity types in a single OData
  ChangeSet (batch) request.
- **Password handling**: `password_wo` is a `WriteOnly` schema attribute (Terraform Plugin
  Framework v1.17.0 in this module supports it; requires Terraform CLI 1.11+), so Terraform
  never persists it to plan or state. The client-layer `UserCredentialParameter` struct used
  for `Get` and for decoding `Create`'s response has no `Password` field at all — a password
  can never end up copied into a Go value this provider exposes. SAP returns `Password` as
  `null`, or as a SHA-256 hash with `returnHashedPassword=SHA256`, which the provider never
  requests.
- **Update — POST, in place**: SAP documents that a POST with the same `Pid` and `Id` updates the
  entry and that PUT is not supported (re-audit September 2026). `user` and the
  `password_wo_version` marker are therefore updatable; changing either sends the POST with the
  configured user and password, and the credential never disappears in between. Only
  `partner_id` and `parameter_id` force replacement.
- **Create — refuses to overwrite**: the same POST overwrites an existing entry, so Create
  first reads `(Pid, Id)` and stops with an "import it instead" error when it exists.
- **Import**: recovers `partner_id`, `parameter_id`, and `user` only — never the password, since
  there is nothing to recover it from. The first apply after import is an in-place update that
  sends the configured password.
- **Feature catalog status**: `supported`. The password is never read back, which is a property
  of the security model, not a missing operation.

## `sapintegrationsuite_user_credential`

- **Purpose**: manage a Security Content "User Credentials" artifact: a username/password
  credential integration flow adapters use for outbound basic or username-token authentication.
- **SAP object**: `UserCredentials` (Security Content API, OData V2, same `/api/v1` host as
  Cloud Integration content). Unlike `UserCredentialParameters` (Partner Directory), this entity
  is not scoped to a Pid — it is a flat, tenant-level artifact identified by its own name.
- **Desired state / Identity**: `id` is the artifact's `Name`, both its OData key and the alias
  integration flow adapters reference; SAP's UI states this explicitly ("the artifact name is
  used as an alias for the confidential data"). Immutable — SAP does not document renaming a
  security material artifact.
- **Read-back**: `User`, `Description`, `Kind`, `CompanyId` are read back; `Password` never is.
  The Go client's `UserCredential` struct has no `Password` field, so a password could not end
  up in this provider's state even if a future SAP API version returned one.
- **Password handling**: `password_wo`/`password_wo_version`, the same `WriteOnly` pattern as
  `sapintegrationsuite_partner_user_credential_parameter` — see that section above for the
  general rationale. The difference here is Update: see below.
- **Create**: `POST UserCredentials` with `Name`/`Kind`/`Description`/`User`/`Password`/
  `CompanyId`.
- **Update — implemented, unlike the Partner Directory analog**: SAP's Manage Security Material
  UI documents an *Edit* action for Credentials artifacts ("You can also edit and redeploy an
  existing artifact"), so this provider implements Update as `PUT UserCredentials('<name>')`
  with every mutable field resent, followed by a `GET` to read back the result (this project
  could not confirm whether a successful `PUT` returns a body or `204 No Content`, so it does
  not trust the `PUT` response shape). `password_wo` is `Required`, not `RequiresReplace`, and is
  resent on every Update — this provider assumes SAP requires re-entering the secret on every
  edit here too, since it documents that requirement explicitly for the sibling OAuth2 Client
  Credentials artifact and no documented exception for User Credentials was found.
- **Rotation trigger**: changing `password_wo_version` (or any other mutable attribute) plans an
  in-place Update, never a replacement — see `docs/guides/security-content.md`. Only `id` and
  `kind` are `RequiresReplace`.
- **Delete**: `DELETE UserCredentials('<name>')`.
- **Drift detection**: readable metadata (`user`, `description`, `kind`, `company_id`) is
  compared normally on Read; the password itself can never be detected as drifted, since SAP
  never returns it.
- **Import**: `terraform import sapintegrationsuite_user_credential.example BACKEND_BASIC`
  recovers `id` and readable metadata only. `password_wo_version` starts unset in state; a
  configuration that has not yet added `password_wo`/`password_wo_version` plans no changes at
  all (Terraform has no opinion on an attribute absent from both state and config), so import
  never implicitly touches the secret. The first time a practitioner adds both attributes, that
  plans as an ordinary in-place Update, not a replacement, and is the deliberate, visible
  "take ownership of rotation" step described in `docs/guides/security-content.md`.
- **Feature catalog status**: `partial` — full CRUD is implemented, but `kind`/`company_id`
  field casing is corroborated by a third-party example payload rather than `$metadata`, and a
  deployment-status field is deliberately not exposed (unconfirmed property name).

## `sapintegrationsuite_oauth2_client_credential`

- **Purpose**: manage a Security Content "OAuth2 Client Credentials" artifact: the client ID,
  client secret, and token service URL an integration flow adapter uses for the OAuth2 client
  credentials grant (RFC 6749) on outbound requests.
- **SAP object**: `OAuth2ClientCredentials` (Security Content API, OData V2).
- **Desired state / Identity**: `id` is the artifact's `Name`/alias, immutable, same reasoning as
  `sapintegrationsuite_user_credential`.
- **Read-back**: `Description`, `TokenServiceUrl`, `ClientId`, `Scope` are read back;
  `ClientSecret` never is (no field for it on the Go client's `OAuth2ClientCredential` struct).
- **Fields deliberately not exposed**: Grant Type placement, Client Authentication mode
  (body vs. header), Resource, Audience, and up to 20 custom parameters are documented in SAP's
  UI in prose, but their OData property names/JSON shapes were not confirmed against
  `$metadata` or a documented example payload, so this resource does not expose them — see
  `docs/sap-api-references.md`.
- **Create**: `POST OAuth2ClientCredentials` with `Name`/`Description`/`TokenServiceUrl`/
  `ClientId`/`ClientSecret`/`Scope`.
- **Update**: `PUT OAuth2ClientCredentials('<name>')`, same full-redeploy-plus-readback pattern
  as `sapintegrationsuite_user_credential`. `client_secret_wo` is `Required` and resent on every
  Update: SAP documents this explicitly ("Every time you edit an OAuth2 Client Credentials
  artifact, you must re-enter the Client Secret").
- **Rotation trigger**: changing `client_secret_wo_version` (or any other mutable attribute)
  plans an in-place Update; only `id` is `RequiresReplace`.
- **Delete**: `DELETE OAuth2ClientCredentials('<name>')`.
- **Drift detection / Import**: same shape as `sapintegrationsuite_user_credential` — see that
  section and `docs/guides/security-content.md`.
- **Feature catalog status**: `partial` — full CRUD is implemented for the confirmed field
  subset, but several UI-documented fields are not yet exposed.

## Remaining Security Content — suitability check

The same 14-question suitability check applied to Number Ranges was walked through for every
object in this phase before writing any code. Full API evidence is in
`docs/sap-api-references.md`; this section records the conclusions.

### Keystore Entry — data source only, no resource

1. **Who creates it**: varies by underlying object — a practitioner (Certificate, Key Pair) or
   SAP itself (SAP-owned entries).
2. **Configuration vs. runtime state**: configuration, but of fundamentally different object
   types sharing one entity set.
3. **Is identity stable**: yes — `Alias` (hex-encoded as the OData key).
4-7. **Create/Read/Update/Delete public**: Read is confirmed (`GET KeystoreEntries`, both
   collection and by-alias). No generic Create/Update/Delete exists for "a keystore entry" as
   such — only for the specific object types beneath it (see Certificate/Key Pair below).
8. **Can Terraform detect drift reliably**: yes, for the confirmed field subset.
9-11. **Reconciliation/apply/destroy safety**: not applicable — no resource is proposed here.
12. **Is import meaningful**: not applicable.
13. **Does it belong in desired-state infrastructure management**: no, as a single generic
   entity — it spans certificates, SAP-generated key pairs, and potentially other keyed entries,
   each with a different lifecycle. Task guidance for this phase was explicit on this point:
   a generic mutable `sapintegrationsuite_keystore_entry` resource is not preferred, precisely
   because "that entity represents multiple fundamentally different entry types."
14. **Resource / Data Source / unsupported / out of scope**: **Data Source only** —
   `data.sapintegrationsuite_keystore_entry` (single, by alias) and
   `data.sapintegrationsuite_keystore_entries` (collection, sorted by alias for deterministic
   output). Confirmed fields: `alias`, `hex_alias` (informational only — never required as
   input), `key_type`, `key_size`, `valid_not_before`, `valid_not_after`. `subject_dn`/
   `issuer_dn`/serial number/fingerprint are deliberately not exposed here as raw SAP fields
   (unconfirmed property names — see `sapintegrationsuite_certificate` below for how this
   provider derives them safely instead).

### `sapintegrationsuite_certificate`

1. **Who creates it**: a practitioner, explicitly — importing a trusted certificate (for example
   a partner's or CA's public certificate) into the tenant keystore.
2. **Configuration vs. runtime state**: pure desired-state configuration — the certificate
   content itself, not something that changes on its own.
3. **Is identity stable**: yes — `alias` (SAP's OData key is its hex encoding, computed
   internally).
4. **Is Create public**: yes, confirmed (`PUT CertificateResources('<hexalias>')/$value`, which
   SAP's own documentation explicitly confirms creates a new entity despite the PUT verb).
5. **Is Read public**: yes, confirmed (`GET KeystoreEntries('<hexalias>')/Certificate/$value` for
   the certificate bytes; `GET KeystoreEntries('<hexalias>')` for the confirmed metadata subset).
6. **Is Update public**: yes — the same PUT operation as Create, confirmed as also updating an
   existing entry at the same alias.
7. **Is Delete public**: yes, via the shared `KeystoreResources('system')?deleteEntries=true`
   mass-deletion operation, called with exactly the one alias this resource owns.
8. **Can Terraform detect drift reliably**: yes, and deliberately not via raw PEM text — see the
   design note below on fingerprint-based comparison.
9. **Would Terraform reconciliation be safe**: yes for content; SAP's own server-side protection
   for SAP-owned entries (see Keystore Entry above) is relied on and surfaced as an error rather
   than pre-empted, since no API field exists to detect ownership in advance.
10. **Could `apply` accidentally reset runtime state**: no — a certificate has no separate
   runtime-mutable state the way Number Ranges' counter does.
11. **Could `destroy` destroy productive data**: the confirmed Delete mechanism (mass-deletion
   with one alias) is scoped tightly enough that this resource can only ever destroy the exact
   alias it owns, never an unrelated one — see the destructive-safety regression test in
   `resource_certificate_test.go`.
12. **Is import meaningful**: yes — `terraform import sapintegrationsuite_certificate.x <alias>`
   populates `certificate` and every derived metadata field from a genuine `GET`.
13. **Does it belong in desired-state infrastructure management**: yes.
14. **Resource / Data Source / unsupported / out of scope**: **Resource**, fully implemented.

**Design notes**:

- **certificate is not Sensitive.** Public X.509 certificate content is not confidential — task
  guidance for this phase was explicit that conflating it with private key material would be a
  mistake this provider must not make.
- **Drift detection compares a canonical fingerprint, not raw PEM text.** Two PEM encodings of
  the identical certificate can differ in line endings, wrapping, or a trailing newline without
  representing any real change. `Read` parses both the remote certificate and whatever is
  currently in Terraform state with Go's own `crypto/x509`, and only overwrites the state's own
  PEM text with SAP's re-serialization when the SHA-256 fingerprint of the certificate's DER
  bytes actually differs — proven by
  `TestCertificateResource_Read_PreservesFormattingWhenUnchanged` and
  `TestCertificateResource_Read_DetectsGenuineDrift`. `certificate_sha256`, `subject_dn`,
  `issuer_dn`, and `serial_number` are all derived this way, locally, never from a guessed SAP
  field name (SAP's own `KeystoreEntries` example is truncated before these properties, and its
  surrounding prose never gives their exact JSON casing).
- **Delete is destructively scoped.** `DeleteKeystoreEntries` is called with a slice containing
  exactly `[]string{alias}` — never a caller-assembled list — so this resource's `terraform
  destroy` can never be constructed in a way that also submits an unrelated alias.

### `sapintegrationsuite_key_pair`

1. **Who creates it**: a practitioner, explicitly, via generation — SAP creates the private key
   material internally.
2. **Configuration vs. runtime state**: desired-state generation parameters (algorithm, size,
   subject DN, validity); the private key itself is neither read nor stored by this provider at
   all.
3. **Is identity stable**: yes — `alias`.
4. **Is Create public**: yes, confirmed field-for-field (`POST KeyPairGenerationRequests`).
5. **Is Read public**: **partially** — `GET KeystoreEntries('<hexalias>')` confirms `KeyType`/
   `KeySize`/`ValidNotBefore`/`ValidNotAfter` back; `SignatureAlgorithm`,
   `KeyAlgorithmParameter`, and every subject DN field are not confirmed returned by any
   documented GET.
6. **Is Update public**: no — no update operation is documented for a generated key pair's
   material.
7. **Is Delete public**: yes, via the same shared `KeystoreResources` mass-deletion mechanism as
   Certificate.
8. **Can Terraform detect drift reliably**: only for the confirmed-readable subset (point 5) —
   this is the specific, permanent reason this feature is cataloged `partial` rather than
   `supported`.
9. **Would Terraform reconciliation be safe**: for the confirmed-readable subset, yes; the rest
   is trusted from the last successful write rather than guessed at via an unconfirmed GET.
10. **Could `apply` accidentally reset runtime state**: no in-place update exists at all
   (everything generation-defining is `RequiresReplace`), so there is no path for an ordinary
   apply to silently mutate key material.
11. **Could `destroy` destroy productive data**: same tightly-scoped single-alias mass-delete
   call as Certificate — never a caller-assembled list.
12. **Is import meaningful**: partially — `alias` and the confirmed-readable subset populate
   correctly; `common_name`, `country`, and every other generation-only parameter cannot be
   reconstructed from an existing tenant key pair (SAP's GET does not return them), and a
   configuration applied right after import that includes them will plan a replacement rather
   than silently accept a value this provider could not actually verify, since every such field
   is `RequiresReplace`.
13. **Does it belong in desired-state infrastructure management**: yes — the practitioner
   genuinely defines this object's existence, algorithm, and subject identity, even though the
   private key itself is intentionally opaque to Terraform.
14. **Resource / Data Source / unsupported / out of scope**: **Resource**, `partial` support
   status (reason: `unsafe_terraform_lifecycle`, for the same "cannot fully verify" reason as
   point 5, not because anything about it is unsafe to use).

**Design notes**:

- **The private key never enters this provider.** No field for it exists on the Go client type,
   the Terraform schema, or anywhere else in this codebase — not because it is marked sensitive,
   but because no code path ever requests one from SAP in the first place. Verified by
   `TestKeyPairResource_SchemaRequiredComputed`'s explicit check that no `private_key*` attribute
   exists.
- **`key_size`/`key_algorithm_parameter` cross-validation matches SAP's documented rules
   exactly**: `key_size` is mandatory for `RSA`/`DSA`; for `EC`, either `key_size` (112-571) or
   `key_algorithm_parameter` (one of SAP's documented named curves) is required, enforced via
   `ValidateConfig` rather than a plain schema constraint, since the rule is conditional on
   `key_type`.
- **`signature_algorithm` is validated against the exact enum SAP documents per `key_type`** —
   RSA/DSA/EC each have their own fixed list — with the hyphenated form from SAP's field-table
   enum treated as authoritative over the un-hyphenated form in SAP's own inline example (see
   `docs/sap-api-references.md` for the discrepancy).
- **SSH Key is deliberately not a separate resource.** `public_key_openssh` (Computed,
   `RequiresReplace`-free since it is purely derived) is populated via the confirmed
   `GetSSHPublicKey` export whenever `key_type` is `RSA` or `DSA` (SAP documents EC as
   unsupported for this export); left `null` for an EC key pair rather than surfacing an error.

### SSH Key — folded into Key Pair, no separate resource

Reverified and corrected from the prior "research required" status: SAP's own Security Content
API overview lists no independent "SSH Key" resource at all, and the "Creating a Key Pair/SSH Key
Pair" UI documentation uses the identical attribute set for both actions. There is nothing left
to design separately — see `sapintegrationsuite_key_pair` above.

### Certificate Chain — not implemented, no independent contract found

SAP's own documentation describes certificate chain import/export as a *capability of* the Key
Pair resource ("create a certificate signing request, or import and export the related
certificate chain"), not an independently exampled entity. No `CertificateChainResources` entry
appears in SAP's curated example-requests index for this API family, and no CSR/signing-response
field contract was found documented anywhere. Per the task guidance for this phase: "Do not
implement until the exact relationship to a key pair is clear" — that relationship is now
clearer (subordinate to Key Pair), but the contract itself remains unconfirmed, so nothing is
implemented. If confirmed later, `sapintegrationsuite_key_pair_certificate_chain` — scoped by
key-pair alias, matching the confirmed ownership relationship — is the likely name, not a vague
global `sapintegrationsuite_certificate_chain`.

### Secure Parameter and Known Hosts — both remain unconfirmed or absent

Neither has a confirmed public REST/OData contract — see `docs/sap-api-references.md` for the
full re-verification trail (Secure Parameter is conceptually listed in SAP's API overview but has
zero worked examples anywhere; Known Hosts does not appear in that overview at all). Both stay
unimplemented; Known Hosts is corrected from `research_required` to `no_public_api` given the
stronger negative signal.

### Whole-keystore management (`KeystoreResources`) — confirmed contract, deliberately out of scope

The full import/backup and mass-deletion contract is now confirmed (see
`docs/sap-api-references.md`), and the confirmed mass-deletion operation is reused internally —
with exactly one alias — by both `sapintegrationsuite_certificate` and
`sapintegrationsuite_key_pair`'s Delete. A `sapintegrationsuite_keystore` resource managing the
*whole* keystore via `POST KeystoreResources` (base64 JKS/JCEKS import) is deliberately not
implemented regardless: a single import call can create, update, leave unchanged, or remove many
entries at once based on the uploaded file's contents, with no way for this provider to know
whether any given affected entry belongs to a different Terraform module, a different
administrator, or SAP itself. This is the same category of blast-radius risk this provider
already refuses for `KeystoreResources`' bulk delete outside the narrow one-alias-only pattern
Certificate/Key Pair use internally — see `docs/provider-scope.md`.

### History Keystore Entries — confirmed contract, not a mutable resource

`PUT HistoryKeystoreEntries('<hexalias>')?copy=true&destinationAlias=<...>` (restore/copy from
SAP's key-history keystore) is confirmed, but it is a restore/copy action on SAP-managed
historical key material, not a generic CRUD entity — and no GET was found documented for it
either, ruling out even a read-only data source for now. Consistent with the task's explicit
instruction to treat this as audit/history data, never a mutable Terraform resource.

## Current API Management / API Artifacts / Integration Cell — suitability check

Before any implementation, this phase's research question was narrower than usual: not "is this
object Terraform-suitable" but "does a public API for it exist at all." The full evidence trail
is in `docs/sap-api-references.md`; this section records why every object in this family ends
up `unsupported`/`no_public_api`, walked through the same 14-question format used elsewhere in
this document where a question is answerable at all.

*September 2026 re-audit:* the verdict holds for every object below and for MCP servers, which
did not exist during the first pass. The re-audit and the design intent for a future API
(runtime profile as `RequiresReplace`, separate design-time and deployment-time virtual hosts,
a read-only default virtual host, reusable APIs as a type on the API artifact resource) are
written up in `docs/guides/current-api-management.md`. The subsections below keep the original
reasoning, which the re-audit did not change.

### API Artifact — no resource, no data source

1. **Who creates it**: a practitioner, via *Design* > *Integrations and APIs* in the SAP
   Integration Suite UI.
2. **Configuration vs. runtime state**: both exist conceptually (design-time definition,
   runtime deployment) — moot, since neither is reachable through a public API.
3–7. **Identity / Create / Read / Update / Delete public**: **no** for all five. The
   authoritative, exhaustive Integration Content API resource table — the same page this
   provider already relies on for `IntegrationDesigntimeArtifacts` and every sibling entity —
   does not list API Artifacts at all. No other page anywhere in the `ISuite_Integrations_APIs`
   documentation tree mentions an OData/REST endpoint, entity set, HTTP method, or SAP Business
   Accelerator Hub link for this object, despite every stage of its lifecycle (create by four
   different methods, configure, version, access-manage, delete, deploy, monitor) being
   documented in detail through the UI.
8–13. Moot — there is no API to detect drift against, reconcile through, or import from.
14. **Resource / Data Source / unsupported / out of scope**: **unsupported, `no_public_api`** —
   this is Outcome C from this phase's own research framework ("UI supports API Artifacts but
   SAP exposes no public design-time API yet"), reached only after exhaustively checking for
   Outcomes A (reuse of `IntegrationDesigntimeArtifacts`) and B (a separate REST API) first, not
   assumed from the start.

### API Artifact Deployment — no resource

Same conclusion as API Artifact, for the same reason: no deployment or undeploy operation is
documented anywhere outside the UI's *Deploy*/*Undeploy* actions. The confirmed prose about
runtime-profile immutability and the design-time/deployment-time virtual host split (see
`docs/sap-api-references.md` for both quoted verbatim) describes real, useful-to-document
behavior — but describes a UI workflow, not an API contract this provider could build a
declarative deployment resource against.

### API Policy — no resource

Not reached as an independent question: without a confirmed API Artifact API in the first place,
there is nothing to attach a policy resource to, and no separate `Policies` entity or
document-level contract was found either. Per the task framework for this phase (§32): whether
policies are nested/opaque artifact content or independent entities could not even be
determined, since neither representation is exposed publicly.

### Reusable API Artifact — no resource, folded conceptually into API Artifact

SAP's own documentation ("not accessible via HTTP endpoint," "invoked... via the API Direct
adapter," "cannot recursively call other reusable APIs," "a unique base path") describes this as
a variant of the same API Artifact concept, not a materially different object — consistent with
this provider's default assumption. Moot regardless: no API exists for API Artifacts of any
kind.

### Runtime Profile — no data source

1. **Who creates it**: SAP; the profile list (Cloud Integration, Cloud Integration – Starter,
   SAP Process Orchestration per release, Edge Integration Cell — notably *not* including a
   distinct "Integration Cell" row, an inconsistency in SAP's own documentation this project
   does not resolve) is fixed platform metadata, not tenant-created content.
2. **Is Read public**: no — configured and displayed only under *Settings* > *Integrations*, no
   API mentioned anywhere.
14. **Resource / Data Source / unsupported / out of scope**: **unsupported, `no_public_api`**.
   Even setting the missing API aside, this project judges the profile list a weak data-source
   candidate on its own merits: it is small, stable, effectively enum-like data better served by
   documentation than a live API call, the same reasoning already applied to Number Ranges'
   signature-algorithm/key-type enums in the Security Content phase.

### Integration Cell Runtime — manual bootstrap, no resource, no data source

Reconfirms the existing `capabilities.integration_cell` catalog entry without change: activation
is a `Settings` > `Runtime` UI step (`activate-integration-cell-1a627da.md`, confirmed), and
runtime status/content is visible only through `Monitor` > `Integrations and APIs`. No API for
either activation or status was found. This provider does not, and given the evidence available
today cannot, manage Integration Cell activation or expose its runtime status — Terraform's role
here begins only once content can be deployed *to* an already-active Integration Cell through a
confirmed public API, which does not yet exist either.

### Integration Cell Virtual Host — no resource, no data source

1. **Who creates it**: a practitioner with the `PI_Administrator` role collection, through
   *Monitor* > *Manage Virtual Host*.
2–13. Moot — no Create/Read/Update/Delete API was found documented anywhere for this entity,
   despite dedicated "Configuring Additional Virtual Host," "View and Edit Virtual Host," and
   "Delete an Eligible Virtual Host" UI-procedure pages existing for it.
14. **Resource / Data Source / unsupported / out of scope**: **unsupported, `no_public_api`**.
   Per this phase's own guidance, even if a future API surfaces, the default virtual host would
   need read-only/data-source treatment rather than an ordinary mutable resource, since SAP
   documents it as having special, restricted editability compared to an administrator-created
   additional virtual host — a design note for whenever this becomes possible, not something
   this provider can act on today.

### What this means for the roadmap

Every object this phase set out to evaluate ends up in the same place: real, UI-documented SAP
functionality with no public API this provider found evidence of, after a genuinely thorough
search (over twenty pages read in full, covering every lifecycle stage of every object, plus an
explicit resource-table cross-check against the existing Integration Content API). This is not a
gap in this project's research effort; it is the accurate current state of SAP's public API
surface for this product area. See `docs/guides/current-api-management.md` for how this is
explained to practitioners, and `ROADMAP.md` for how the priority list responds to it.

## Classic API Management — suitability check

Unlike Current API Management above, this phase's research question was the usual one: a public
REST/OData API is confirmed to exist for every object below (`Management.svc`, under
`/apiportal/api/1.0`, confirmed field-for-field from SAP's own official "SAP API Management
Standalone Service" user guide — see `docs/sap-api-references.md`), so the suitability question
is genuinely "is this the right shape for a Terraform resource," not "does an API exist at all."

### API Provider — resource, narrow lifecycle

1. **Who creates it**: a practitioner, describing a backend connection an API Proxy will target.
2. **Configuration vs. runtime state**: configuration — a stable, named object a practitioner
   authors and wants reconciled, not runtime/business data.
3. **Identity**: `name`, confirmed as the entity's OData key (`APIProviders('<name>')`).
4–5. **Create / Read public**: yes, confirmed verbatim (POST; GET by key and as a collection).
6. **Update public**: **no**. SAP's own Piper `apiProviderUpload` tooling documents create-only
   support; this provider does not invent an unconfirmed PUT/PATCH, so every attribute is
   `RequiresReplace` instead.
7. **Delete public**: yes, confirmed (`DELETE APIProviders('<name>')`).
8. **Drift-detectable fields**: every field except `password_wo` (write-only, never returned by
   GET — the API does not appear to return credential material back at all, the same category of
   gap this provider already handles with the `_wo` pattern elsewhere).
9. **Import**: supported — `id` is `name`, and GET is confirmed.
10. **Eventual consistency**: confirmed verbatim (~20 seconds of caching after a write); handled
    with bounded, jittered polling in `internal/client/apimanagementclassic/retry.go`, not a
    fixed sleep.
11–13. Not applicable — no batch/deployment/versioning concept documented for this entity.
14. **Resource / Data Source / unsupported / out of scope**: **Resource + Data Source, `partial`
    (`unsafe_terraform_lifecycle`)** — scoped to the "Internet" connection type only, since the
    other three documented connection types (On Premise, Open Connectors, Cloud Integration)
    have no confirmed field-level JSON mapping this provider could find.

### API Product — resource, create/read/delete only

1. **Who creates it**: a practitioner, bundling API Proxies for subscription.
2. **Configuration vs. runtime state**: configuration.
3. **Identity**: `name`.
4–7. **Create / Read / Update / Delete public**: Create, Read and Delete confirmed on a tenant
   (September 2026). Update is not available: SAP answers `PUT`, `PATCH` and `MERGE` with 405,
   although its user guide still shows a `PUT` example.
8. **Drift-detectable fields**: every top-level field, plus the linked proxies and additional
   properties, which are read through the `apiProxies` and `additionalProperties` navigation
   properties because the product itself only returns `__deferred` links.
9. **Replace-only**: every attribute is `RequiresReplace`. Additional properties are sent inside
   the create request, each with the product's name as `entityId`; SAP rejects creating them on
   their own with 405. Replacing a product drops its subscriptions, which the resource
   documentation warns about.
10. **Import**: supported. Import reads the linked proxies and properties, so a matching
    configuration plans no change.
14. **Resource / Data Source / unsupported / out of scope**: **Resource + Data Source,
    `supported`**.

### Certificate Store Reference — resource, full CRUD, best-evidenced object this phase

1–3. **Who creates it / configuration / identity**: a practitioner, naming a pointer to an
   existing keystore/truststore; identity is `name`.
4–7. **Create / Read / Update / Delete public**: all four confirmed verbatim, including error
   response bodies for every failure mode SAP documents (duplicate name, missing linked store).
8–9. **Drift / Import**: every field drift-detectable; import supported.
14. **Resource / Data Source / unsupported / out of scope**: **Resource + Data Source,
    `supported`**. The one deliberate scope boundary: this provider does not manage the
    keystore/truststore or its certificate content, since SAP documents that as a UI-only upload
    with no REST API found — `certificate_store_name` must reference a store created outside
    Terraform.

### Key Value Map — resource, narrow lifecycle

1. **Who creates it**: a practitioner, defining runtime configuration lookups for one or more API
   Proxies.
2. **Configuration vs. runtime state**: configuration — though note the entries themselves are
   *read* at runtime by deployed proxy content via the Key Value Map Operations policy, similar
   in spirit to how Cloud Integration's Number Ranges are configuration that runtime content
   reads from.
3. **Identity**: composite (`name`, `scope`, `scope_id`), confirmed from the worked Create
   example's field set.
4. **Create public**: yes, confirmed verbatim, including nested entries in the same call.
5. **Read public**: inferred from this API family's consistent GET-by-key convention; not shown
   as a standalone worked example for this specific entity, but the UI's own "view the updated
   key value map" step confirms a read path exists.
6–7. **Update / Delete public**: SAP's UI documentation confirms both operations exist (add/
   delete/update-value-only for entries; delete for the whole map) but shows no REST payload for
   either — this provider does not guess at the shape, so entries are `RequiresReplace` and there
   is no in-place Update at all.
8. **A field this provider refuses to support at all**: `encrypted` / `isEncrypted`. Real,
   documented, but with unconfirmed GET-response behavior for the entry value once encrypted —
   this provider's `ValidateConfig` rejects `encrypted = true` outright rather than risk leaking
   a secret into state or producing a resource that can never stabilize.
9. **Import**: supported, via the composite key as `"<name>/<scope>/<scope_id>"`.
14. **Resource / Data Source / unsupported / out of scope**: **Resource + Data Source, `partial`
    (`unsafe_terraform_lifecycle`)** — unencrypted maps only, no Update.

### API Proxy — resource, experimental

1. **Who creates it**: a practitioner, as a ZIP-bundled design-time artifact exported from the
   API portal or built by hand.
2. **Configuration vs. runtime state**: configuration. SAP documents that an imported proxy is
   deployed by default, so the runtime state follows the import and is shown as `state`.
3. **Identity**: `name`, the OData key of `Management.svc/APIProxies`, confirmed by `$metadata`
   and SAP's worked `GET`/`DELETE` references. SAP takes it from the bundle's
   `APIProxy/<name>.xml` descriptor; the provider checks that it equals the configured name
   before uploading.
4. **Create public**: yes, through the Transport API. SAP's API Management Client SDK 3.0.6 (official
   tooling, published 2026-09-24) posts the raw ZIP as `application/octet-stream` to
   `/apiportal/api/1.0/Transport.svc/APIProxies`; the provider sends the same request. The Hub
   lists "API Portal - Transport (CF)" as the official API; its specification needs an SAP login.
5. **Read public**: yes, `Management.svc/APIProxies('<name>')` (GET returned 200 on a tenant).
6. **Update public**: not confirmed. Nothing public says whether importing a changed bundle over an
   existing proxy replaces it cleanly, so the provider never does it: a new `content_hash`
   replaces the resource (delete, then import).
7. **Delete public**: yes, `DELETE Management.svc/APIProxies('<name>')`, as SAP's documentation
   references it.
8. **Import**: by name. SAP does not return the bundle file, so the first apply after an import
   only records `content` and `content_hash`.
14. **Resource / Data Source / unsupported / out of scope**: **Resource, `experimental`
    (`public_api_incomplete`)** until `TestAccAPIProxy_sample` passes on a tenant; the open
    point is the update semantics, which the replace-only design avoids.

### API Proxy Deployment — no separate resource

SAP's own documentation states a transported or exported proxy "by default gets imported to the
target in the deployed state", and no public call deploys or undeploys an existing proxy on its
own. Deployment is therefore a side effect of importing a proxy, the
opposite of the design-time/runtime split this provider uses for Cloud Integration artifacts.

### Policy — no resource, folded into API Proxy's opaque content

SAP's own sample repository confirms policies are XML files referenced by a `<policies>` element
in the proxy's root XML, not an independently addressable OData entity. Per this provider's
established pattern for opaque, nested design-time content (matching how Cloud Integration
artifact ZIP content is already treated), policies are managed as part of
the proxy's bundle content, never as a separate
`sapintegrationsuite_api_proxy_policy` resource reproducing SAP's policy schema catalog.

## Migration Assessment — suitability check

The smallest documentation footprint audited so far (roughly fifteen pages), and the cleanest
conclusion: no public API for Migration Assessment's own objects, and — uniquely among the
capabilities audited this run — a conclusion that would not change even if one were confirmed.

### Source System — no resource, no data source

1. **Who creates it**: a practitioner, registering an on-premises SAP Process Orchestration
   system for Migration Assessment to extract data from.
2. **Configuration vs. runtime state**: configuration — the closest thing in this capability to a
   legitimate Terraform candidate, if a public API existed.
3–13. Moot — no API found; Migration Assessment's own documentation describes it *consuming*
   APIs from the registered source system, never exposing one for managing the registration
   itself.
14. **Resource / Data Source / unsupported / out of scope**: **unsupported, `no_public_api`**.

### Data Extraction Request, Scenario Evaluation Request, and results — out of scope regardless

1. **Who triggers it**: a practitioner, choosing *Create* on a request.
2. **Configuration vs. workflow/reporting**: confirmed both — Create is an imperative action with
   a resulting status (workflow), and the eventual output is an assessment-category/readiness/
   effort-estimate report (reporting data).
14. **Resource / Data Source / unsupported / out of scope**: **unsupported, `out_of_scope`** —
   deliberately not `research_required`: a confirmed API contract for triggering these actions
   would not change that they represent an action-and-its-result, not desired configuration this
   provider's plan/apply model could reconcile.

## Integration Advisor — suitability check

Same research question as Trading Partner Management (does a public API exist), same negative-
evidence method, same answer.

### Message Implementation Guideline, Mapping Guideline, Type System, Codelist, Shared Code,
### Global Parameters — no resource, no data source

1. **Who creates it**: a practitioner, within Integration Advisor's own workspace.
2. **Configuration vs. runtime state**: configuration — design-time content, a legitimate
   Terraform candidate in principle.
3–13. Moot — no Create/Read/Update/Delete API found documented anywhere, despite create, update,
   version, migrate, simulate, and delete all being thoroughly documented as UI procedures for
   most of these object types.
14. **Resource / Data Source / unsupported / out of scope**: **unsupported, `no_public_api`**.
   One research pitfall specifically avoided here: a page describing OAuth credential creation
   for this capability's documentation area turned out, on full reading, to describe
   authenticating against Cloud Integration for artifact injection, not a credential for these
   objects themselves — recorded in `docs/sap-api-references.md` as a caution for future
   research passes in this codebase, not just this one.

### Runtime artifact injection into Cloud Integration — out of scope regardless

1. **Who triggers it**: a practitioner, choosing *Inject* on a Mapping Guideline.
2. **Configuration vs. imperative action**: confirmed imperative — a UI wizard that pushes
   generated artifacts into a chosen integration flow's resources, right now, not a state
   Terraform's plan/apply model reconciles.
14. **Resource / Data Source / unsupported / out of scope**: **unsupported, `out_of_scope`** —
   this provider already manages Cloud Integration flow content directly
   (`sapintegrationsuite_integration_flow`); wrapping the injection *trigger* itself as a
   resource would model a one-shot action, not desired state, the same reasoning this provider
   already applies to Trading Partner Management's Partner Directory generation and Developer
   Hub's Subscription approval workflow.

## Trading Partner Management — suitability check

This phase's research question was, again, whether a public API exists at all — and unlike
Classic API Management and Integration Assessment, the answer here is no, reached through the
same negative-evidence method (absence of a dedicated API-access documentation page, consistently
present for every capability in this provider that does have a confirmed public API) applied
across roughly ninety documentation pages.

### Company Profile, Trading Partner Profile, Communication Partner Profile, Agreement Template,
### Agreement — no resource, no data source

1. **Who creates it**: a practitioner, under *Design* > *B2B Scenarios*.
2. **Configuration vs. runtime state**: configuration — design-time content a practitioner
   authors, which would be a legitimate Terraform candidate if a public API existed.
3–13. Moot — no Create/Read/Update/Delete API found documented anywhere for any of these
   objects, despite export/import (JSON download/upload) and full lifecycle management (create,
   activate, deactivate, copy, migrate, version) all being thoroughly documented as UI
   procedures.
14. **Resource / Data Source / unsupported / out of scope**: **unsupported, `no_public_api`** —
   the same Outcome C this provider already reached for Current API Management's object family.

### Partner Directory generation on agreement activation — out of scope regardless

1. **Who triggers it**: a practitioner, choosing *Activate* on a trading partner agreement.
2. **Configuration vs. imperative action**: confirmed imperative — SAP's own documentation
   describes activation as an action that "pushes" the complete agreement into Partner Directory,
   not a state Terraform's plan/apply model reconciles.
14. **Resource / Data Source / unsupported / out of scope**: **unsupported, `out_of_scope`** —
   this provider already manages the Partner Directory entries this process happens to generate,
   directly, through the confirmed API those resources already use (see
   `docs/guides/partner-directory.md`). Wrapping the *generation trigger itself* as a resource
   would model an action, not desired state, the same reasoning this provider already applies to
   Developer Hub's Subscription approval workflow and Integration Assessment's Request objects.

## Integration Assessment — suitability check

This phase's research question was again the usual one (does a public API exist), and the answer
is genuinely yes — a separate BTP service subscription, a confirmed dual-base-URL OAuth-secured
API, and an exhaustively confirmed entity inventory (nineteen named entities, each with a
one-paragraph SAP description) are all real evidence, not assumptions. What could not be
confirmed, despite checking the entire SAP-docs mirror tree for this capability, an official
2400-line PDF user guide, and SAP's own TechEd hands-on sample repository, is a field-level
request/response schema for even one entity. The suitability check below is therefore grouped by
category rather than walked entity-by-entity through the full fourteen-question format — every
individual entity in a group shares the same answer to questions 4 through 14 ("unconfirmed"),
so repeating that nineteen times would not add information.

### Master data (Domain, Style, Use Case Pattern, Integration Pattern, Key Characteristic family,
### Deployment Model, Domain Determination) — no resource, no data source

1. **Who creates it**: primarily SAP (shipped ISA-M reference taxonomy), with a documented
   tenant "Update Content Maintained by SAP" adjustment capability.
2. **Configuration vs. reference data**: reference/master data — the category this provider
   already treats as, at most, a data-source candidate rather than a resource (see Runtime
   Profile's suitability check in the Current API Management section above for the same
   reasoning pattern).
3–13. Moot — no field-level API contract confirmed for Create, Read, Update, or Delete on any
   entity in this group.
14. **Resource / Data Source / unsupported / out of scope**: **unsupported,
   `research_required`** — `PublicAPI: true` (the capability and entity both confirmed real),
   but no schema confirmed to build even a read-only data source against safely.

### Landscape configuration (Application, Application Instance, Technology, Technology Instance,
### Vendor, and their association entities) — no resource, no data source, but the strongest
### candidate in this capability

1. **Who creates it**: a practitioner, describing their organization's actual application and
   integration-technology landscape.
2. **Configuration vs. reference data**: configuration — practitioner-authored, not SAP-shipped,
   and SAP's documented per-tenant limits (20,000 Applications, 20,000 Application Instances, 50
   Technologies, 150 Technology Instances, 10,000 Vendors) confirm this is real, bounded, durable
   tenant storage, not runtime/business data.
3–13. Moot for the same reason as Master Data — no field-level contract confirmed.
14. **Resource / Data Source / unsupported / out of scope**: **unsupported,
   `research_required`** — the first place to look if SAP's wire contract for this capability
   ever becomes reachable, since every other suitability signal (who owns it, why it's created,
   documented bounded limits) already points toward a legitimate Terraform resource.

### Assessment workflow (Request, Request Line Item, Integration Flow, Message Flow, Integration
### Flow Message Flow, Request Line Item Technology Instance Decision) — no resource, no data
### source, and not merely a research gap

1. **Who creates it**: a practitioner, as a business solution/interface request moving through an
   assessment workflow.
2. **Configuration vs. workflow state**: workflow/project state, confirmed by SAP's own
   documented Request status machine (`draft` → `new` → `in progress` → `completed`, plus a
   `Reopen` action available at specific states) — the same category this provider already
   excludes for Message Processing Logs and Developer Hub's Subscription object.
14. **Resource / Data Source / unsupported / out of scope**: **unsupported, `out_of_scope`** —
   deliberately not `research_required`, since finding a confirmed API contract for this group
   would not change the underlying suitability judgment. This mirrors the distinction this
   provider already draws for Developer Hub's Application/Subscription (`PublicAPI: true`,
   `out_of_scope`), applied here even though this specific group's field contract also happens to
   be unconfirmed — the reason recorded is the one that would still apply if it were confirmed
   tomorrow.

## `data.sapintegrationsuite_partner` / `data.sapintegrationsuite_partners`

- **Purpose**: read-only discovery of Partner IDs (Pids). `data.sapintegrationsuite_partner`
  confirms a single Pid exists; `data.sapintegrationsuite_partners` lists every Pid in the
  tenant, following server-driven paging.
- **Why no `sapintegrationsuite_partner` resource (§17 suitability check)**: SAP documents no
  confirmed create operation for `Partners` — a Pid comes into existence implicitly the first
  time a child entity (`StringParameter`, `BinaryParameter`, `AlternativePartner`,
  `AuthorizedUser`, or `UserCredentialParameter`) references it, and SAP's own documentation
  says Pid uniqueness "is ensured by the tenant owner application" rather than being
  server-allocated. Separately, SAP documents that deleting a Pid can cascade to remove every
  entity belonging to it in one call. A Terraform resource needs a safe, predictable
  Create/Destroy pair; Partners has neither a confirmed Create nor a Destroy that could not
  erase content owned by a completely different Terraform module. Read-only data sources give
  practitioners exactly what the API actually supports (existence checks and discovery) without
  inventing a lifecycle SAP does not offer.

## Deferred (see ROADMAP.md)

`sapintegrationsuite_capability`, `sapintegrationsuite_api_artifact`,
`sapintegrationsuite_api_artifact_deployment`, message mapping entry-level or dependent-resource
management (schema files referenced by a mapping are managed as part of the opaque content
archive, not as separate Terraform resources), and value mapping entry-level management (see
above) are designed at the API level in `api-capability-matrix.md` but intentionally not
implemented yet, to keep each release small and high quality (see `ROADMAP.md`). Partner
Directory resources are no longer in this list — see the sections above. Security material is
now split: user credentials and OAuth2 client credentials are implemented (see the sections
above); keystore entries, certificates, key pairs, SSH keys, and certificate chains remain
deferred pending `$metadata` confirmation — see `docs/guides/security-content.md`.
