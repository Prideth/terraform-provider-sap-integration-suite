# SAP API References

Every implemented resource must trace back to an officially documented, SAP-supported public
API. This document is that trace.

> **Research note**: the first research phases ran in a sandbox that could not reach
> `help.sap.com`, `api.sap.com` or `community.sap.com` at all. The 2026 re-audit can reach
> them, but two limits remain, and they shape how evidence is gathered:
>
> - The SAP Help Portal renders its pages in JavaScript, so the re-audit reads the official
>   Markdown source of those pages in SAP's `SAP-docs/btp-integration-suite` GitHub
>   repository. It is the same text, not a secondary summary.
> - The SAP Business Accelerator Hub also renders in JavaScript, and its specification
>   downloads (EDMX, OpenAPI JSON) redirect to an SAP ID login. Where a specification could not
>   be downloaded, sections below say so. They then rely on SAP's own published tooling for
>   the same API, which shows the requests SAP itself sends.
>
> Sections re-audited in 2026 are marked as such. Everything else still reflects the earlier
> research and should be verified against a tenant's `$metadata` document before you rely on
> an exact property name.

## `sapintegrationsuite_integration_package`

- **SAP product area**: Integration Suite / Cloud Integration
- **Official API**: Integration Content API (`CloudIntegrationAPI` package, SAP Business
  Accelerator Hub)
- **Entity set**: `IntegrationPackages`
- **Protocol**: OData V2
- **Operations**: GET, POST, PATCH, DELETE (PATCH limited to SAP-permitted metadata fields;
  used rather than PUT so fields outside the Terraform schema are not reset to their defaults)
- **Required roles**: `IntegrationOperationServer` / `IntegrationDeveloper` OAuth scopes (role
  collection names vary per tenant; assign the least-privileged Integration Suite role
  collection covering "Integration Content" design-time operations)

## `sapintegrationsuite_integration_flow` / `..._deployment`

- **SAP product area**: Integration Suite / Cloud Integration
- **Official API**: Integration Content API
- **Entity sets / actions**: `IntegrationDesigntimeArtifacts`, `IntegrationRuntimeArtifacts`,
  `DeployIntegrationDesigntimeArtifact` (action)
- **Protocol**: OData V2
- **Operations**: GET, POST (create), PUT (create a new design-time version of an existing
  flow), DELETE, plus the `Deploy` action (POST)
- **Required roles**: as above, plus deploy-specific scopes for the runtime artifact
  operations

## `sapintegrationsuite_integration_flow_configuration`

*Added September 2026.*

- **SAP product area**: Cloud Integration, externalized parameters of integration flows
  (*Configure* in the Design UI)
- **Official API**: Integration Content API, resource *Configurations of Integration Flow*. SAP
  Help's Integration Content page: "You can read or update integration flow configurations. You
  can access integration flow configurations through the `IntegrationDesigntimeArtifacts`
  resource."
- **Documented request** (*Integration Flow Example Requests*, "Update Integration Flow
  Configuration Parameters"): `PUT /IntegrationDesigntimeArtifacts(Id='{Id}',Version='{Version}')/$links/Configurations('{ParameterKey}')`
  with body `{"ParameterValue": "...", "DataType": "xsd:string"}`, plus a `$batch` variant.
- **Corroboration**: SAP's Piper library, step `integrationArtifactUpdateConfiguration`
  (`cmd/integrationArtifactUpdateConfiguration.go`), sends the same PUT with only
  `ParameterValue`.
- **`$metadata`**: entity type `Configuration`, key `ParameterKey`, properties
  `ParameterValue`, `DataType`, `Description`; navigation `Configurations` on
  `IntegrationDesigntimeArtifact`; entity set `Configurations`. Checked by the contract test.
- **Read**: `GET IntegrationDesigntimeArtifacts(Id,Version)/Configurations`. Standard navigation
  read, covered by the documented "read" capability.
- **Not offered**: create or delete of a parameter. Parameters are defined in the flow model, so
  destroy only removes the resource from state.
- **Provider choices**: only listed keys are managed; the existing `DataType` is resent with
  each value; unknown keys are rejected before any write.

## Cloud Integration: tenant `$metadata` findings (September 2026)

The tenant `$metadata` of `/api/v1` settles several open points for Integration Content. As
elsewhere, it confirms names, keys and types but not which operations SAP accepts.

- **Value mapping entries.** `UpsertValMaps` (POST; `Id`, `Version`, `SrcAgency`, `SrcId`,
  `TgtAgency`, `TgtId`, `ValMapId`, `SrcValue`, `TgtValue`, `IsConfigured`) returns a `ValMap`
  (`Id`, complex `Value` with `SrcValue`/`TgtValue`). `UpdateDefaultValMap` takes the same
  identifiers plus `ValMapId` and `IsConfigured`. `DeleteValMaps` takes only `Id`, `Version` and
  the agency/identifier pair, so there is no documented per-entry delete. SAP Help's *Value
  Mapping Example Requests* shows `UpsertValMaps` and a filtered read of
  `ValMapSchema(...)/ValMaps`. Reclassified from `research_required` to
  `public_api_incomplete`; see `cloud_integration.value_mapping_entry`.
- **Explicit versions.** `…SaveAsVersion` function imports (`Id`, `SaveAsVersion`) exist for
  integration flows, message mappings, script collections, value mappings, data types, message
  types, fault message types and service interfaces. SAP Help's *Integration Flow Example
  Requests* documents `IntegrationDesigntimeArtifactSaveAsVersion` after a PUT. The provider uses
  it through `save_as_version` on integration flows, message mappings and script collections
  (`cloud_integration.design_time_versioning`, `partial`).
- **New design-time artifact types.** `DataTypeDesigntimeArtifacts`,
  `MessageTypeDesigntimeArtifacts`, `FaultMessageTypeDesigntimeArtifacts` and
  `ServiceInterfaceDesigntimeArtifacts`, all keyed by `Id` and `Version` with `Namespace` and
  `ArtifactContent`. SAP Help documents them only as UI procedures and does not list them in the
  Integration Content resource table.
- **Number ranges.** `NumberRanges` is keyed by `Name` and also carries `DeployedBy` and
  `DeployedOn`. GET by key and DELETE remain undocumented.
- **Integration packages.** `IntegrationPackage` also has `ResourceId`, `PartnerContent`,
  `UpdateAvailable`, `SupportedPlatform`, `Products`, `Keywords`, `Countries`, `Industries` and
  `LineOfBusiness`. The provider does not expose them yet; whether the classification fields can
  be written is undocumented.
- **Locks.** `IntegrationDesigntimeLocks` lists who has an artifact open in the editor. That is
  operational state, not configuration, and stays out of scope.

## `sapintegrationsuite_value_mapping` / `..._deployment`

- **SAP product area**: Integration Suite / Cloud Integration
- **Official API**: Integration Content API
- **Entity sets / actions**: `ValueMappingDesigntimeArtifacts`,
  `ValueMappingDesigntimeArtifactSaveAsVersion` (action, not currently used — see Update below),
  `IntegrationRuntimeArtifacts` (the same shared runtime-artifacts entity
  `sapintegrationsuite_integration_flow_deployment` uses — confirmed via SAP's own "Runtime
  Status API" description as covering "currently deployed integration artifacts" generally,
  not one entity per design-time artifact type), `DeployValueMappingDesigntimeArtifact` (action;
  singular form re-checked this phase — see `docs/resource-design.md`)
- **Protocol**: OData V2
- **Operations**: GET, POST (create), DELETE, plus the `Deploy` action (POST). No PUT/Update —
  see below.
- **Required roles**: `WorkspacePackagesConfigure`, `WorkspacePackagesEdit`,
  `WorkspaceArtifactsDeploy`
- **Confirmed constraint**: a value mapping cannot be created with zero entries — at least one
  mapping entry must be part of the artifact's content at creation time.
- **Update — resolved conservatively, no in-place update implemented**: SAP documents a distinct
  `ValueMappingDesigntimeArtifactSaveAsVersion` action (POST, taking the artifact's technical ID
  and a caller-supplied new version identifier). An earlier version of this provider called
  `PUT` against the keyed entity instead, by analogy with `IntegrationDesigntimeArtifacts`.
  Re-investigating this contract, `help.sap.com`, `api.sap.com`, `community.sap.com`,
  `blogs.sap.com`, and every reachable mirror/proxy for them were blocked by this environment's
  network egress policy, so the exact `PUT` vs. `SaveAsVersion` semantics could not be confirmed
  against primary documentation or an actual request/response trace. Reachable secondary
  evidence — a dedicated SAP Knowledge Base Article (3502529) treating "changing the version of
  a ValueMapping" as its own distinct, separately gated operation, and an independent
  third-party OData client that explicitly disables generic update for
  `ValueMappingDesigntimeArtifacts` while leaving it enabled for the sibling
  `IntegrationDesigntimeArtifacts`/`MessageMappingDesigntimeArtifacts`/`ScriptCollectionDesigntimeArtifacts`
  entity sets — points away from a working generic `PUT` for this entity set specifically.
  Given that, `sapintegrationsuite_value_mapping` does not retain the `PUT` call: `name`,
  `content`, and `content_hash` are all `RequiresReplace`, so any change to them replaces the
  resource (`Create` a new artifact, `Delete` the old one) instead of relying on an unverified
  update path. `UpdateValueMapping` no longer exists in
  `internal/client/cloudintegration/value_mapping.go`. Implementing true in-place update via
  `ValueMappingDesigntimeArtifactSaveAsVersion` is deferred to v0.2.x, once its request/response
  contract can be confirmed against a live tenant or a reachable primary source; see
  `docs/resource-design.md` for the full reasoning.
- **Delete semantics — unverified scope**: `DeleteValueMapping` deletes via the same
  `(Id, Version='active')` key used for reads. Whether this removes only the active version or
  every version of the artifact was not confirmed against a primary source (same network
  restrictions as above). This provider never creates more than one version of a value mapping
  concurrently, so it does not change this resource's observable behavior, but it means
  `terraform destroy` is not guaranteed to remove every version SAP stored — flagged here rather
  than asserted as "deletes all versions".
- **Deferred — entry-level operations**: `UpsertValMaps` (POST, insert/update individual
  mapping rows — confirmed to 404 if the target source/target agency-identifier scheme does not
  already exist), `UpdateDefaultValMap` (POST, sets a scheme's default value via a `ValMapId`
  GUID obtained from a separate lookup), and `DeleteValMaps` (exact deletion granularity
  unconfirmed) are real, existing APIs that this phase does not implement — see
  `docs/resource-design.md` for why guessing their wire format was rejected in favor of
  documenting the gap.

## `sapintegrationsuite_message_mapping` / `..._deployment`

- **SAP product area**: Integration Suite / Cloud Integration
- **Official API**: Integration Content API
- **Scope note**: this is the reusable, package-level Message Mapping *artifact*
  (`MessageMappingDesigntimeArtifacts`), not the inline/local message mapping step configurable
  directly inside an integration flow. See `docs/resource-design.md` for the distinction.
- **Entity sets / actions**: `MessageMappingDesigntimeArtifacts`,
  `MessageMappingDesigntimeArtifactSaveAsVersion` (action, confirmed to exist, not used by this
  provider — see Update below), `IntegrationRuntimeArtifacts` (the same shared runtime-artifacts
  entity `sapintegrationsuite_integration_flow_deployment` and
  `sapintegrationsuite_value_mapping_deployment` use), `DeployMessageMappingDesigntimeArtifact`
  (action; singular form, matching the sibling actions for the other design-time artifact types)
- **Protocol**: OData V2
- **Operations**: GET, POST (create), PUT (update — see below), DELETE, plus the `Deploy` action
  (POST)
- **Required roles**: `WorkspacePackagesConfigure`, `WorkspacePackagesEdit`,
  `WorkspaceArtifactsDeploy`
- **Sources used**: SAP Help Portal content read via the `SAP-docs` GitHub organization's
  markdown mirror of the official Cloud Integration documentation (a legitimate primary source —
  SAP's own published documentation, mirrored verbatim for community feedback — reached this
  phase despite `help.sap.com` itself being blocked by this environment's network egress
  policy), plus an independent third-party OData client
  (`github.com/lemaiwo/ci-mcp-server`) built directly against this same API, used as
  corroborating (not primary) evidence.
- **Content format — confirmed, not inferred from Integration Flow**: a message mapping
  artifact's content is a mapping definition (`*.mmap`) file; SAP's own artifact-upload UI
  accepts it as (or bundled inside) a ZIP archive. This provider transports that ZIP opaquely as
  base64-encoded `ArtifactContent`, the same as `IntegrationDesigntimeArtifacts` and
  `ValueMappingDesigntimeArtifacts` — it does not parse the `.mmap` file or any XSD/WSDL/EDMX/
  Swagger-OpenAPI schema files the mapping may reference for source/target message structures.
- **Update — resolved as `PUT`, on entity-specific grounds**: unlike
  `sapintegrationsuite_value_mapping` (which has no in-place update — see that resource's entry
  below), this phase found positive evidence supporting `PUT` specifically for
  `MessageMappingDesigntimeArtifacts`: it shares `IntegrationDesigntimeArtifacts`' exact
  `(Id, Version)` key shape and confirmed version-creating `PUT` behavior; the same third-party
  OData client that explicitly disables generic update for `ValueMappingDesigntimeArtifacts`
  explicitly *enables* it for `MessageMappingDesigntimeArtifacts` (matching
  `IntegrationDesigntimeArtifacts` and `ScriptCollectionDesigntimeArtifacts`); and no SAP KBA or
  other evidence of a documented `PUT` problem for this entity set was found (unlike Value
  Mapping's KBA 3502529). `MessageMappingDesigntimeArtifactSaveAsVersion` exists here too, but
  this phase's research clarified that `SaveAsVersion` is a universal action across this whole
  API family (confirmed to exist for `IntegrationDesigntimeArtifacts` as well, coexisting with
  its confirmed `PUT`), not evidence against `PUT` by itself — see `docs/resource-design.md` for
  the full reasoning. This provider does not use `SaveAsVersion` since it does not ask users to
  manage an explicit version string.
- **Delete — unverified scope, same open item as Value Mapping**: `DeleteMessageMapping` deletes
  via `(Id, Version='active')`, the same key used for reads. Whether this removes only the
  active version or every version of the artifact was not confirmed against a primary source.
- **Deployment — `IntegrationRuntimeArtifacts`, not `BuildAndDeployStatus`**: investigated both
  per this phase's instructions. `BuildAndDeployStatus(TaskId='…')` is documented in the context
  of a different artifact family's build-then-deploy pipeline (OData API artifacts, keyed by a
  `TaskId` a build operation returns), not `MessageMappingDesigntimeArtifacts`. SAP's Runtime
  Status API documentation and secondary sources both describe message mappings as deployed
  runtime artifacts monitored through the same shared `IntegrationRuntimeArtifacts` entity
  already used by the other `*_deployment` resources; `runtime_artifact.go` and
  `runtime_deployment.go` are reused unmodified. See `docs/resource-design.md` for the full
  reasoning.
- **No hidden coupling to referencing integration flows**: SAP's own documentation confirms an
  integration flow's deployment does not automatically deploy a message mapping it references;
  this provider does not add automatic-deployment behavior SAP itself does not provide, and does
  not scan or modify integration flow content to manage that reference.

## `sapintegrationsuite_script_collection` / `..._deployment`

- **SAP product area**: Integration Suite / Cloud Integration
- **Official API**: Integration Content API
- **Entity sets / actions**: `ScriptCollectionDesigntimeArtifacts`,
  `IntegrationRuntimeArtifacts` (the same shared runtime-artifacts entity every other
  `*_deployment` resource in this provider uses), `DeployScriptCollectionDesigntimeArtifact`
  (action; singular form, matching every other design-time artifact type's deploy action)
- **Protocol**: OData V2
- **Operations**: GET, POST (create), PUT (update), DELETE, plus the `Deploy` action (POST)
- **Required roles**: `WorkspacePackagesConfigure`, `WorkspacePackagesEdit`,
  `WorkspaceArtifactsDeploy`
- **Sources used**: SAP Help Portal content read via the `SAP-docs` GitHub organization's
  markdown mirror (the same primary-source path used for Message Mapping), plus the same
  independent third-party OData client (`github.com/lemaiwo/ci-mcp-server`) used as
  corroborating evidence for Message Mapping's Update model.
- **Documented constraints**: a script collection's technical ID must be unique across the
  entire tenant (not just the containing package), and its description is capped at 120
  characters — both confirmed via SAP's own documentation, neither enforced client-side by this
  provider.
- **Content format**: a ZIP archive of Groovy/JavaScript script files, transported opaquely as
  base64-encoded `ArtifactContent`, the same convention as every other file-based design-time
  resource in this provider. This provider does not parse, validate, or execute the scripts.
- **Update — resolved as `PUT`, same grounds as Message Mapping**: `ScriptCollectionDesigntimeArtifacts`
  shares `IntegrationDesigntimeArtifacts`' `(Id, Version)` key shape and confirmed `PUT`
  behavior; the same third-party OData client that disables generic update for
  `ValueMappingDesigntimeArtifacts` explicitly enables it here too, matching
  `IntegrationDesigntimeArtifacts` and `MessageMappingDesigntimeArtifacts`; no SAP KBA or other
  evidence of a documented `PUT` problem for this entity set was found. See
  `docs/resource-design.md` for the full reasoning.
- **Delete — unverified scope**: same open item as every other design-time artifact type in
  this provider — whether Delete removes only the active version or every version is
  unconfirmed against a primary source.
- **Deployment**: fire-and-poll through the shared `IntegrationRuntimeArtifacts` entity, the
  same model already confirmed for integration flows, value mappings, and message mappings —
  script collections are documented as existing as runtime artifacts inside the same deployed
  runtime packages.
- **No hidden coupling to referencing integration flows**: this provider does not scan or
  modify integration flow content to manage a reference to a script collection, the same
  ownership boundary already established for message mapping.

## `sapintegrationsuite_integration_adapter` / `..._deployment`

- **SAP product area**: Integration Suite / Cloud Integration — custom Integration Adapters
  built with the SAP Adapter SDK. **Cloud Foundry environment only**; every SAP source covering
  this feature repeats "This information is relevant only when you use SAP Cloud Integration in
  the Cloud Foundry environment."
- **Official API**: Integration Content API. The general "Integration Content" resource table
  (read via the `SAP-docs/btp-integration-suite` GitHub mirror) lists it as: "Integration
  Adapter: Represents an integration adapter (only available in the Cloud Foundry environment).
  You can use resource `IntegrationAdapterDesigntimeArtifacts` to import, deploy, or delete an
  integration adapter."
- **Entity set**: `IntegrationAdapterDesigntimeArtifacts`
- **Protocol**: OData V2
- **Evidence tier — weaker than the sibling design-time artifact types above**: SAP's own
  "Integration Adapter Example Requests, Cloud Foundry Environment" page — the direct
  counterpart to the complete example-request pages that back Integration Flow, Value Mapping,
  Message Mapping, and Script Collection above — shows only two operations. Everything else
  below is explicitly marked by its evidence tier.
- **Confirmed — Delete**: `DELETE /api/v1/IntegrationAdapterDesigntimeArtifacts(Id='SubsystemSymbolicName1')`.
  This is the only confirmed evidence for the entity's key shape: `Id` alone, not the composite
  `(Id, Version)` key every sibling design-time artifact entity set in this API uses.
- **Confirmed — Deploy**: `POST /api/v1/DeployIntegrationAdapterDesigntimeArtifact?Id='SubsystemSymbolicName1'`.
  Two details worth flagging explicitly since they're easy to get wrong by analogy:
  - Singular action name ("...Artifact"), matching every sibling deploy action in this API —
    confirmed directly, not assumed.
  - No `Version` query parameter, unlike every sibling deploy action (which all take both `Id`
    and `Version`). Consistent with the Id-only key finding above.
- **Create — properties confirmed by `$metadata`, request not shown by SAP**: implemented as
  `POST IntegrationAdapterDesigntimeArtifacts` with `Id`, `PackageId`, `Name` and base64
  `ArtifactContent`. A tenant `$metadata` document (September 2026) defines the entity type
  with exactly `Id` (sole key), `Version`, `PackageId`, `Name`, `ArtifactContent`
  (`Edm.Binary`) and `Description`. There is still no SAP-published Create example.
- **Confirmed (UI documentation) — identity and duplicate handling**: "The integration adapter
  ID needs to be unique across the tenant" (not merely the package) and "If there's already an
  integration adapter with the same ID, the system throws an error." The latter is treated as
  positive evidence against a working reimport-to-update flow — see Update below.
- **Not confirmed — Read**: `GET IntegrationAdapterDesigntimeArtifacts(Id='...')`, the ordinary
  OData GET-by-key convention every entity set in this API follows, not confirmed by an
  adapter-specific example.
- **Not confirmed, and deliberately not implemented — Update**: no PUT/PATCH/reimport example
  was found anywhere for this entity. Combined with the confirmed duplicate-ID-is-an-error
  behavior, this provider implements no update path at all: every attribute on
  `sapintegrationsuite_integration_adapter` is `RequiresReplace`.
- **`Type`/`Application` — UI concepts without API properties (corrected September 2026)**:
  SAP's UI documentation describes a line-of-business type and a target application for
  adapters, and earlier releases assumed matching OData properties. The tenant `$metadata` has
  neither, so the provider no longer sends them and no longer offers `type`/`application`.
- **Not confirmed — deployment runtime status / undeploy**: `sapintegrationsuite_integration_adapter_deployment`
  reuses the same shared `IntegrationRuntimeArtifacts` polling/undeploy (`GetRuntimeArtifact`/
  `UndeployRuntimeArtifact`, already used by every other `*_deployment` resource in this
  provider) by analogy. SAP's documentation for the shared `IntegrationRuntimeArtifacts` deploy
  mechanism explicitly states "You can only deploy BUNDLE type integration artifacts
  (integration flows, value mappings, or OData services)" — confirming adapters are *excluded*
  from that generic deploy path (which is exactly why they have their own dedicated deploy
  action) but not confirming or ruling out whether a custom adapter, once deployed through its
  own action, becomes readable/undeployable through that same shared entity. A dedicated
  `BuildAndDeployStatus`-based mechanism remains a plausible alternative this project could not
  rule out. See `docs/guides/integration-adapters.md`.
- **Distinct lifecycles — confirmed as separate concepts, addressed explicitly to prevent
  conflation**: "Import Integration Adapters" (a different SAP Help Portal page) describes
  importing a *prebundled SAP Business Accelerator Hub* adapter from inside the integration flow
  editor — an entirely different, UI-triggered, auto-deploying flow with no separate OData
  identity, not the custom-`.esa`-upload feature this provider manages. `docs/guides/integration-adapters.md`
  documents this distinction prominently, per this project's standing policy of never silently
  conflating two different SAP lifecycles that happen to share a name.
- **File size limit**: no SAP-documented maximum `.esa` upload size was found. This provider
  enforces only its own generic protective bound (32 MiB, the same bound already used for script
  collections and value mappings), not an SAP-documented limit.
- **Required roles**: SAP's "Importing Custom Integration Adapter, Cloud Foundry Environment"
  page names `WorkspacePackagesEdit` and `WorkspaceArtifactsDeploy` as the role templates
  required for the various adapter tasks.
- **CSRF audit performed for this feature, no changes needed**: this feature's Create (`POST`),
  Delete (`DELETE`), and Deploy (`POST`) client methods go through the exact same shared
  transport chain (`internal/client/cloudintegration.Client` → `internal/client/odata/v2.Client`
  → `internal/client/http.Client.Do`) as every other resource in this provider — the CSRF layer
  (`internal/client/http/csrf.go`) dispatches purely on HTTP method
  (`POST`/`PUT`/`PATCH`/`DELETE`), so it applies automatically with zero adapter-specific code.
  No new or adapter-specific CSRF handling was written or was needed; the full test suite
  (`internal/client/auth`, `internal/client/cloudintegration`, `internal/client/http`,
  `internal/client/odata/v2`, `internal/client/partnerdirectory`,
  `internal/client/securitycontent`, `internal/features`, `internal/provider`) was run after
  adding this feature specifically to confirm no regression to Integration Packages, Integration
  Flows, Value Mappings, Message Mappings, Script Collections, Access Policies, Partner
  Directory, or Security Content.

## `sapintegrationsuite_custom_tag_configuration`

- **SAP product area**: Integration Suite / Cloud Integration — tenant-wide governance
  configuration, not a per-package or per-artifact object
- **Official API**: Integration Content API. Confirmed directly from two SAP Help Portal pages
  (read via the `SAP-docs/btp-integration-suite` GitHub mirror): "Create New Custom Tags
  Configuration" and "Get Custom Tags Defined on the Tenant".
- **Entity set**: `CustomTagConfigurations`, with a single, fixed, confirmed key: `CustomTags`.
  There is exactly one configuration per tenant.
- **Protocol**: OData V2
- **Create/Update — confirmed verbatim, including the exact example payload**: `POST
  /api/v1/CustomTagConfigurations`, body `{"CustomTagsConfigurationContent":
  "<base64-encoded-content>"}`, where the decoded content for a mandatory "Owner" tag is
  `{"customTagsConfiguration":[{"tagName":"Owner","isMandatory":true}]}` — this exact
  base64-round-trip is asserted byte-for-byte in
  `internal/client/cloudintegration/custom_tag_configuration_test.go`. SAP's documentation
  states: "If a custom tags configuration is already available on the tenant, add the following
  query parameter to the request: `Overwrite=true`". This client always sends `Overwrite=true`
  on every write (see the doc comment on `SetCustomTagConfiguration` for why: SAP never
  documents what a plain `POST` does once a configuration exists, so always using the documented
  "already exists" path avoids depending on an unconfirmed distinction).
- **Read — confirmed verbatim, including the "$value" envelope difference**: `GET
  /api/v1/CustomTagConfigurations('CustomTags')/$value`. Unlike every other entity in this
  client, the response is the raw decoded JSON directly — OData's `$value` raw-media-stream
  convention — not wrapped in the standard `{"d": {...}}` envelope `v2.DecodeEntity` expects
  elsewhere, so this client decodes the response body directly instead.
- **Delete — reverified, confirmed absent**: no delete or clear operation is documented anywhere
  for this entity across either page or the general Integration Content resource table (which
  describes the entity only as maintainable "using the Cloud Integration Settings section" or
  "the OData API (Custom Tags interface)", with no mention of removal). This is a materially
  different situation than a design-time artifact type with an unconfirmed delete *scope* (for
  example whether delete removes one version or all) — here there is no delete operation
  documented at all. `sapintegrationsuite_custom_tag_configuration`'s Delete therefore returns
  an explicit error rather than a guessed "clear via empty overwrite" implementation or a no-op
  — see `docs/resource-design.md` and `docs/guides/custom-tag-configurations.md`.
- **`Overwrite=true` semantics — strongly implied, not stated verbatim**: the documented request
  body is always the complete configuration, never a delta, and the query parameter is literally
  named `Overwrite`, which this provider reads as "full replace" (tags absent from a new write
  are removed). SAP's documentation never uses the word "replace" itself, so this is flagged as
  this provider's own reasonable interpretation, not an independently confirmed fact.
- **Singleton semantics investigated (per the research checklist for this feature)**:
  - Is `CustomTags` always the fixed key? **Confirmed** — it appears literally in SAP's own GET
    example.
  - Can there be multiple configurations? No evidence of any mechanism for more than one; the
    entire API is structured around the one fixed key.
  - Does a plain `GET` (without `/$value`) return useful metadata? Not shown in either
    documented example; not implemented, since guessing at undocumented properties was rejected.
  - Is posting an empty configuration valid, and does it clear everything? **Not confirmed
    either way** — deliberately not relied upon by this provider's Delete (see above).
  - Tag name / permitted-value uniqueness, case sensitivity, and whether SAP preserves submitted
    order: **none of these are documented**. This provider enforces tag-name uniqueness itself,
    models `permitted_values` as a set (making exact duplicates unrepresentable), and treats
    both `tags` and `permitted_values` ordering as not semantically meaningful (see
    `docs/guides/custom-tag-configurations.md`).
- **A documentation artifact worth flagging**: SAP's "Get Custom Tags Defined on the Tenant"
  page's second example response renders two permitted values ("Mr. Bean" and "Ms. Bean") as a
  *single* array element containing a comma-separated string —
  `"permittedValues":["Mr. Bean, Ms. Bean"]` — rather than two separate array elements. This
  contradicts the one-element-per-value convention every other array-typed field in this API
  uses (including `EntryPoints`/`ApiDefinitions` on `ServiceEndpoints`), so it is treated as a
  documentation authoring artifact rather than a confirmed wire format; this client always
  serializes one array element per permitted value.
- **Required role**: SAP's general Integration Content API documentation states that maintaining
  custom tags requires the role template `WebToolingSettingsProductProfiles.savetenantconfiguration`
  ("part of role collection `PI_Administrator`" in the Cloud Foundry environment, "part of
  authorization group `AuthGroup.Administrator`" in Neo). The role **template**, not the entire
  `PI_Administrator` role collection, is what this provider's documentation states is actually
  required — a narrower custom role collection granting just that template should work equally
  well, and this provider does not claim the full `PI_Administrator` collection is mandatory.

## `data.sapintegrationsuite_service_endpoints`

- **SAP product area**: Integration Suite / Cloud Integration — runtime discovery of deployed
  content's exposed endpoints
- **Official API**: Integration Content API, documented as "Endpoints of Runtime Artifacts" —
  "You can use resource `ServiceEndpoints` to read all endpoints provided for integration flows
  and to get the number of endpoints" (confirming `$inlinecount`/count support). Base path
  `https://<host>/api/v1/ServiceEndpoints`, the same `/api/v1` OData V2 host as every other Cloud
  Integration resource this provider uses.
- **Entity set**: `ServiceEndpoints`, with two expandable navigation properties, `EntryPoints`
  and `ApiDefinitions`
- **Protocol**: OData V2. GET only — no create/update/delete operation is documented, since SAP
  generates these entities itself from deployed content.
- **Primary source**: SAP's own `ServiceEndpoints Example Requests` page (read via the
  `SAP-docs/btp-integration-suite` GitHub mirror of the official Help Portal content), which
  gives the complete confirmed contract:
  - `Name` and `Protocol` are the two documented, filterable top-level properties (`Name eq
    '...'`, `Protocol eq '...'`).
  - `EntryPoints` (expand via `$expand=EntryPoints`): an array of `EntryPoint`, with `Name`
    (required, String), `URL` (required — see casing note below), and `Type` (optional,
    enumerated String: `DEV`, `TEST`, `PROD`, `SANDBOX`).
  - `ApiDefinitions` (expand via `$expand=ApiDefinitions`): the documentation describes an
    array of `APIDefinition` with `URL` and `Type` (`oas-yaml`, `oas-json`, `raml`, `edmx`,
    `wsdl`). **Corrected September 2026:** the tenant `$metadata` defines the target type
    `Definition` with only `Url` (key) and `Name`; there is no `Type` property. The provider
    reads `Name` and exposes it as `api_definitions[].name`.
  - The same `$metadata` gives `ServiceEndpoint` the key `Id` and the additional properties
    `Title`, `Version`, `Summary`, `Description` and `LastUpdated`, and `EntryPoint` the
    additional property `AdditionalInformation`. All of them are exposed by the data source.
  - Protocol values, per the same page's adapter table: SOAP adapter → `SOAP`, IDoc adapter →
    `SOAP`, OData V2 adapter → `ODATAV2`, AS2 → `AS2`, AS4 → `AS4`, HTTPS → `REST`. This provider
    exposes exactly this value; it does not reverse-map it back to an adapter type, since the
    mapping is not one-to-one (SOAP and IDoc both report `SOAP`).
- **`Url` JSON casing — confirmed from SAP's own Piper (open-source CI/CD) library, not from
  prose alone**: SAP's documentation prose describes the property's *type* as "URL", which does
  not by itself establish the wire-format JSON key casing. `github.com/SAP/jenkins-library`'s
  `cmd/integrationArtifactGetServiceEndpoint.go` parses a real `ServiceEndpoints` response with
  `entryPoints.Path("results.0.Url")` — confirming the actual JSON property is `Url`, not
  `URL`. This provider's `EntryPoint.URL` Go field is tagged `json:"Url"` accordingly. The same
  Piper source also confirms the response envelope shape this provider already assumes
  everywhere (`{"d": {"results": [...]}}`) and that `EntryPoints`, once expanded, nests its own
  `{"results": [...]}` array (`internal/client/odata/v2.ExpandedCollection[T]`, a new small
  shared type distinct from the top-level paged-collection envelope, since SAP does not document
  paging for an expanded nested navigation property the way it does for a top-level collection
  request).
- **`ApiDefinitions[].Url` casing — confirmed by `$metadata`**: the `Definition` entity type
  names the property `Url`.
- **Combined `$expand=EntryPoints,ApiDefinitions`**: SAP's example requests demonstrate each
  expansion separately, never combined in one request. This provider combines them in a single
  request using OData V2's standard comma-separated `$expand` syntax (the same `Query.Expand
  []string` mechanism already used for `$select` elsewhere in this codebase) to avoid an N+1
  request pattern; no documented reason was found that `ServiceEndpoints` would reject a
  combined expand, and this is standard, unremarkable OData V2 behavior.
- **Pagination**: `$top`/`$skip`/`$inlinecount` support was added in the February 2020 release
  (v3.21.x) per SAP's own release notes (community source). This provider always fetches every
  page via the shared `GetAllPages` helper (the same server-driven `__next`-link paging every
  other collection-returning client method in this provider uses), rather than assuming a single
  page.
- **Technical ID — corrected September 2026**: the tenant `$metadata` keys `ServiceEndpoint`
  by `Id`, which the data source now exposes as `endpoints[].id`. The earlier statement that no
  technical ID exists was wrong. Whether that ID is stable across redeployments is still not
  documented, which is why the reasoning about a singular data source below still applies (see
  `docs/resource-design.md` for why this rules out a singular `data.sapintegrationsuite_service_endpoint`
  lookup).
- **Deterministic ordering**: SAP does not document a guaranteed response order for the
  collection, or for either expanded nested collection. This provider sorts all three
  deterministically before writing Terraform state — see `internal/client/cloudintegration/service_endpoint.go`.
- **Required roles**: not separately confirmed for this research pass; assumed to fall under the
  same Integration Content read scopes already required for `IntegrationDesigntimeArtifacts`/
  `IntegrationRuntimeArtifacts`, pending confirmation.

## `sapintegrationsuite_number_range` and the Message Stores API family (Variables, Data Stores, Data Store Entries)

- **SAP product area**: Integration Suite / Cloud Integration — the "Message Stores" OData V2 API
  (`https://api.sap.com/api/MessageStore`), covering `NumberRanges`, `Variables`, `DataStores`,
  and `DataStoreEntries` alongside `Entries`/`EntryAttachments`/`EntryProperties` (Message Store,
  already covered elsewhere) and JMS Resources.
- **Research method**: every page below was fetched from the `SAP-docs/btp-integration-suite`
  GitHub mirror (`docs/ci/Development/` and `docs/ci/Operations/`), the same official-mirror
  technique used throughout this project. `api.sap.com` itself redirects unauthenticated
  requests to a login page and could not be used directly to inspect `$metadata`.

### Number Ranges — confirmed operations and the missing GET

The curated **"Message Stores Example Requests"** index page
(`message-stores-example-requests-02c57df.md`) is the authoritative list of every documented
example for this API family. For every sibling entity it links a "Get ..." example page; for
Number Ranges it links exactly two:

- [Add a Number Ranges Object](https://raw.githubusercontent.com/SAP-docs/btp-integration-suite/main/docs/ci/Development/add-a-number-ranges-object-b1bd945.md):
  `POST /api/v1/NumberRanges`, body
  `{"CurrentValue":"0","Name":"My NRO Object","MinValue":"0","MaxValue":"9999","Description":" Number Range Object ","Rotate":"true","FieldLength":"4"}`
  — asserted byte-for-byte in `internal/client/cloudintegration/number_range_test.go`.
- [Update a Number Ranges Object](https://raw.githubusercontent.com/SAP-docs/btp-integration-suite/main/docs/ci/Development/update-a-number-ranges-object-139a6b2.md):
  `PUT /api/v1/NumberRanges('{objectName}')`, same body shape. This page also states explicitly:
  "The Current Value returned by the API corresponds to the Next Value shown in the Monitoring
  tab of the UI. The difference is only in terminology" — confirmed verbatim, cross-referenced
  by `managing-number-ranges-b6e17fa.md` (the UI documentation) independently.

**No GET operation — for either the collection or a single object — is documented anywhere.**
This was checked exhaustively, not assumed from absence on one page:

- The example-requests index above lists nothing beyond Add/Update.
- No `get-a-number-range*`/`get-number-range*` page exists under `docs/ci/Development/` (checked
  via a directory listing of the whole folder).
- `message-stores-1aab5e9.md` (the Message Stores overview/resource table) documents the
  `NumberRanges` resource in prose only, with no GET example, unlike its entries for
  `DataStores`, `DataStoreEntries`, and `Variables`, which each state their unsupported query
  options explicitly (implying a GET exists to apply those options to) — Number Ranges has no
  such statement.
- The overview page's resource table has a genuinely truncated caveat directly against
  "Number Ranges": `> ### Note: \n> Not supported in.` with no recoverable clause — re-fetched
  twice to rule out a fetch artifact; this is a truncation in SAP's own published source. Its
  meaning (an environment? a runtime type?) is **not confirmed**, and this provider does not
  guess at it.

**No DELETE operation is documented for Number Ranges either.** The overview page's general
CSRF-token paragraph mentions "POST, PUT, and DELETE" as the three modifying-action types across
the whole Message Stores API family, but this is generic boilerplate applying to the family as a
whole (DELETE is separately, specifically confirmed for Data Store Entries/Variables via the
`DataStoresAndQueuesDelete` role template — see below), not evidence of a Number-Ranges-specific
DELETE. `managing-number-ranges-b6e17fa.md` (the UI/Operations page) instead documents an
**"Undeploy"** action, explicitly distinct from "Delete" in its own Actions list, with no visible
REST equivalent anywhere in the API documentation.

**A UI-only multi-runtime deployment dimension was also found, with no API-level counterpart.**
The same Operations page documents a "Runtimes" field on the Add/Edit dialog: "One or more
runtime nodes to deploy the artifact to... including Cloud Integration and any active Edge
Integration Cell nodes," and describes name-uniqueness as evaluated against "any of the selected
runtimes." Neither of the two documented API examples (Add, Update) shows any runtime/location
parameter. This provider's client always targets the implicit default runtime and does not
attempt to reconstruct or guess at this parameter.

**Field semantics confirmed from `managing-number-ranges-b6e17fa.md`** (UI documentation,
consistent with the API examples):

- `MinValue`: "should be greater than or equal to 0."
- `MaxValue`: "should be less than 15 digit[s]" / "Must be fewer than 15 digits."
- `FieldLength`: zero-pads the displayed value; "maximum value allowed for this attribute is 14";
  a value of 0 applies no padding.
- `Rotate`: "If this attribute is set and the number range reaches specified maximum value, then
  the current value resets to specified minimum value" — confirmed verbatim, matching this
  provider's `rotate` semantics exactly.

**Role template**: no Cloud Foundry role template specific to Number Ranges creation/update was
found documented anywhere (unlike Data Stores/Variables, which have explicit
`DataStoresAndQueuesRead`/`DataStorePayloadsRead`/`DataStoresAndQueuesDelete` templates — see
below). The Neo-environment `tasks-and-permissions-556d557.md` page lists Monitor-app tasks
"View number ranges" and "Add, edit, or undeploy number ranges" gated behind the Integration
Developer / Tenant Administrator roles, but this documents UI authorization in the older Neo
environment, not a confirmed Cloud Foundry API role/scope for the public REST endpoints.

**Consequence for this resource's design** (see `docs/resource-design.md` for the full
suitability walkthrough): without a GET, `sapintegrationsuite_number_range` cannot implement a
Read that verifies anything against the tenant, cannot detect drift, and cannot support
`terraform import`. It is implemented as a **write-only-lifecycle resource**: Create and Update
call the two confirmed operations for real, Read is a documented no-op that trusts local state,
Delete and Import both return explicit errors rather than guessing at unconfirmed operations. The
runtime counter (`CurrentValue`) is handled as a version-gated write-only attribute
(`current_value_wo`/`current_value_wo_version`) that is sent to SAP only when the practitioner
deliberately bumps the version — every other Update omits `CurrentValue` from the request body
entirely, since this provider has no way to confirm any remembered value is still correct. Two
unconfirmed risks are inherent to this design and documented rather than hidden: (1) whether an
omitted `CurrentValue` on `PUT` is preserved unchanged or reset to a default by SAP is not
confirmed either way, since there is no GET to check the result; (2) whether `POST`ing a `Name`
that already exists on the tenant errors, conflicts, or silently overwrites is likewise
unconfirmed.

### Variables — read-only download, no creation API

[Download a Variable](https://raw.githubusercontent.com/SAP-docs/btp-integration-suite/main/docs/ci/Development/download-a-variable-94e6799.md)
documents the **only** public operation for this entity: `GET
/api/v1/Variables(VariableName='{VariableName}',IntegrationFlow='{IntegrationFlowName}')/$value`,
which downloads the raw value via OData's `$value` convention — no structured metadata
(`Visibility`/`UpdatedAt`/`RetainUntil`) is returned by this endpoint. No collection GET exists
(the composite key must already be known), and no POST/PUT/DELETE is documented anywhere.
[Define Write Variables](https://raw.githubusercontent.com/SAP-docs/btp-integration-suite/main/docs/ci/Development/define-write-variables-de04b75.md)
confirms Variables are created and updated exclusively by an integration flow's "Write
Variables" step (Constant/Header/XPath/Expression/Property source, a "Global Scope" checkbox
controlling Global vs. local `IntegrationFlow` visibility), and separately confirms: "A variable
gets expired after the retention period, which is 400 days," extended by every successful
processing run, and "Variables can't be downloaded using the data store viewer."

**No resource, no data source** — see `docs/resource-design.md` and `docs/provider-scope.md`.
There is no Create/Update API to back a resource at all, and the one confirmed read endpoint
returns nothing but the arbitrary runtime value itself, with no safer metadata-only projection
available.

### Data Stores and Data Store Entries — GET-only, runtime-created

[Get All Data Stores with Overdue Messages](https://raw.githubusercontent.com/SAP-docs/btp-integration-suite/main/docs/ci/Development/get-all-data-stores-with-overdue-messages-5173f5c.md):
`GET /api/v1/DataStores?overdueonly=true`, an aggregate monitoring endpoint returning
`NumberOfMessages`/`NumberOfOverdueMessages` per store — not configuration.
[Get Single Data Store Entry](https://raw.githubusercontent.com/SAP-docs/btp-integration-suite/main/docs/ci/Development/get-single-data-store-entry-8b86912.md)
and
[Get All Data Store Entries for a Data Store](https://raw.githubusercontent.com/SAP-docs/btp-integration-suite/main/docs/ci/Development/get-all-data-store-entries-for-a-data-store-acbef52.md)
confirm `DataStoreEntries` fields verbatim: `Id`, `DataStoreName`, `IntegrationFlow`, `Type`,
`Status`, `MessageId`, `DueAt`, `CreatedAt`, `RetainUntil` — all runtime message state.

[Define Data Store Write Operations](https://raw.githubusercontent.com/SAP-docs/btp-integration-suite/main/docs/ci/Development/define-data-store-write-operations-46260ee.md)
(referenced by the overview page as how both `DataStores` and `DataStoreEntries` come into
existence) and
[Define Data Store Delete Operations](https://raw.githubusercontent.com/SAP-docs/btp-integration-suite/main/docs/ci/Development/define-data-store-delete-operations-5efa3ac.md)
confirm both write and delete are **design-time integration-flow steps**, not REST operations on
these entities: "This step deletes an entry from a transient data store... the delete operation
can't be used to delete whole data stores, but single entries only." No POST/PUT/DELETE against
`DataStores` or `DataStoreEntries` is documented as a REST call anywhere.

Confirmed verbatim from `message-stores-1aab5e9.md`: both the `DataStore` and `Variables` APIs
"do not support the following query options: `$filter`, `$inlinecount`, `$orderby`, `$skip`,
`$top`, `$expand`, `$select`" — two separate, explicit statements, one per entity, not a single
blanket statement this provider generalized across the whole API family. `NumberRanges` carries
no such statement (and, per above, has no GET to apply query options to regardless).

**Required roles** (from `message-stores-1aab5e9.md`'s Permissions section, Cloud Foundry): to
view data store entries, `DataStoresAndQueuesRead`; to download payloads and view variables,
`DataStorePayloadsRead`; to delete data store entries/variables, `DataStoresAndQueuesDelete`. The
existence of `DataStoresAndQueuesDelete` confirms a delete capability is authorized for Data
Store Entries/Variables at the permission-template level, consistent with the confirmed
design-time Delete step above, even though no REST DELETE example was found for either entity.

**No resource, no data source for either entity** — see `docs/resource-design.md` and
`docs/provider-scope.md`. `DataStores` has no independent creation API (it is an implicit runtime
container); its one GET is monitoring data, the same class already excluded via
`cloud_integration.message_processing_logs`. `DataStoreEntries` is unambiguously runtime business
message data with no creation/deletion REST API at all.

### Edge Integration Cell

*Updated September 2026: the path variant this section once looked for is now documented.*

SAP Help's Integration Content (`integration-content-d1679a8`), Security Content
(`security-content-e01d3f0`) and Partner Directory (`partner-directory-0fe80dc`) pages, in the
version synced on 2026-07-10, all state: "For Edge Integration Cell, use this,
`https://<host address>/location/<runtime location id>/api/v1/<relative resource path>`". The
statement is general; there are no per-operation examples. Corroborating and contrary signals:

- Int4's Edge Integration Cell preparation guide uses the same prefix, "/location/<runtimeLocationId>"
  (example "/location/myedge"), and says the ID appears in the Integration Suite monitoring URL
  after selecting the Edge Integration Cell as runtime.
- SAP's `cicd-actions-for-sap-integration-suite`, action `sync-pid-to-eic` version 1.2.0
  (2026-05-05), takes the Edge URL as a template parameter and describes it as "Not published.
  Pre-work to support EIC — do not use productively. Subject to change." SAP Help published the
  pattern two months later.

The provider implements it as `runtime_location_id` on the deployment, credential, certificate,
key pair and Partner Directory resources and their data sources, marked experimental until verified on a tenant
with an Edge Integration Cell. For Number Ranges, Variables and Data Stores the same prefix would
apply; those resources are out of scope or write-only for other reasons.

## `sapintegrationsuite_access_policy` / `..._reference`

*Re-audited September 2026. The earlier version of this section guessed the reference wire
format; those guesses turned out to be wrong and have been replaced by the evidence below.*

- **SAP product area**: Integration Suite / Security (Monitor > Manage Security > Access
  Policies)
- **Official API**: Security Content API on the SAP Business Accelerator Hub
  (<https://api.sap.com/api/SecurityContent/resource>), resource group *Access Policies*. SAP
  Help's *Managing Access Policies* page points there and states that access policies can be
  read and written "by an OData V2 application programming interface".
- **Base path**: `/api/v1` on the tenant's Cloud Integration API host
- **Protocol**: OData V2, JSON (`$format=json` or `Accept: application/json`)

### Where the wire contract comes from

The Business Accelerator Hub page is a JavaScript application, and its specification
downloads (EDMX and OpenAPI JSON) redirect to an SAP ID login. The specification could not be
downloaded without an SAP account for this audit. The contract below therefore comes from the
next source in this project's evidence order: SAP's own automation for exactly this API,
published as <https://github.com/SAP/cicd-actions-for-sap-integration-suite> (Apache-2.0,
access-policy actions first released 2026-04-23). The `download-access-policy`,
`upload-access-policy`, `update-access-policy` and `delete-access-policy` actions call the API
directly with `curl`, so their requests and the example files in their READMEs show the real
property names, key format and payloads.

| Operation | Request (from SAP's tooling) | Provider usage |
|---|---|---|
| List / find by role | `GET /AccessPolicies?$filter=RoleName eq '<name>'` | data source lookup by `role_name` |
| Create policy | `POST /AccessPolicies` with `{"RoleName", "Description"}`, answers `201` with `d.Id` | Create |
| Update policy | SAP's tooling: `PUT /AccessPolicies(<Id>L)` with `{"RoleName", "Description"}`. **A tenant answered this `PUT` with `501`** (September 2026); `PATCH` with `{"Description"}` answered `204` | Update (`PATCH`) |
| Delete policy | `DELETE /AccessPolicies(<Id>L)`, deletes the policy "incl. all Artifact References" | Delete |
| List references | `GET /AccessPolicies(<Id>L)/ArtifactReferences` | Read, data source |
| Create reference | `POST /ArtifactReferences` with `{"Name", "Description", "Type", "ConditionAttribute", "ConditionValue", "ConditionType", "AccessPolicy": {"Id": "<Id>"}}`, answers `201` | Create |
| Delete reference | `DELETE /ArtifactReferences(<Id>L)` | Delete |

What this establishes:

- Both entity sets are keyed by `Edm.Int64` (the `L` literal suffix). OData V2 JSON serializes
  the value as a string, for example `"Id": "1901"`. The earlier client quoted keys as strings
  (`AccessPolicies('1901')`), which is wrong for an Int64 key.
- The policy's persisted properties are `RoleName` and `Description`. The README's example
  `AccessPolicy.json` holds exactly those two after `Id` and navigation properties are
  stripped. There is **no** `ReconciliationStatus` property, so the provider's former
  `reconciliation_status` attribute was removed.
- A policy has two navigation properties: `ArtifactReferences` and
  `AccessPolicyRuntimeAssignments`.
- Reference properties are `Name`, `Description`, `Type`, `ConditionAttribute`,
  `ConditionType` and `ConditionValue`. The earlier client sent `ArtifactType`, `Attribute`,
  `Operator` and `Value`, none of which exist.
- Confirmed enum values from the README example: `Type = INTEGRATION_FLOW`,
  `ConditionAttribute = Name`, `ConditionType = exactString`.
- SAP's tooling never updates a reference in place. Its sync deletes all references and
  recreates them.

The single-entity `GET /AccessPolicies(<Id>L)` used by the resource's Read is not shown in
SAP's tooling. It is standard OData V2 addressing for an entity set that supports `PUT` and
`DELETE` by the same key, and it is the only call in this resource that rests on protocol
convention rather than a published example.

### SAP Help findings (current documentation)

Read from the official Markdown source of the Help Portal pages in
`SAP-docs/btp-integration-suite` (`docs/ISuite_Integrations_APIs/access-policies-e0009f3.md`,
`defining-access-policies-b0d7950.md`, `managing-access-policies-318d107.md`,
`access-policies-examples-f1dc1a7.md`, `creating-custom-roles-for-access-policies-7db3c87.md`,
and `docs/ISuite_Edge_Integration_Cell/manage-access-policies-for-edge-integration-cell-d6503a5.md`):

- **Artifact types offered by the UI**: Integration Package, Integration Flow, API, OData API,
  REST API, SOAP API, Script Collection, Value Mapping, Message Mapping, Message Queue, Global
  Data Store, Global Variable, Data Type, Message Type. **Correction**: an earlier version of
  this document relied on KBA 3447540 to conclude that Integration Package is not a valid
  type. Current Help lists it as a selectable type and has a dedicated *Package-Level Access
  Policy* section; SAP's community announcement of the feature dates it to increment 2401.
  Whatever the KBA's original context, it no longer describes the product. The provider's
  former closed list of ten types was both incomplete and spelled wrong.
- **Attributes**: *Name* and *ID*. **Operators**: *Equals* (exact name or ID) and *Matches*
  (a Java regular expression; not available for Integration Package). Message queues, global
  variables and global data stores can only be matched by name.
- **Reference name**: every reference has a mandatory *Name* and an optional *Description*,
  which the provider did not model before.
- **Role association**: the BTP role must be created from role template `CustomRoleTemplate`
  of application `it`, with `custom_role` set to *Static* and the policy's role name in
  *Values*. It is granted through a role collection, and takes effect after the user logs in
  again. `PI_Administrator` is required to create and edit policies.
- **Scope of protection**: design-time operations, operations on deployed artifacts, and
  runtime data (MPL attachments, traces, data stores, variables, queues). Both UI and API
  access are covered. Unauthorized users can still see that an artifact exists. Custom header
  properties are not protected.
- **Runtimes**: when creating a policy, the user picks the runtimes it is created in, and can
  edit the selection later ("replicating access policies"). A per-runtime reconciliation
  status reads *Fail*, *Success* or *Pending*. Offline runtimes receive the policy once they
  come back.

### Tenant `$metadata` confirmation (September 2026)

A `$metadata` document from a Cloud Foundry tenant (`/api/v1/$metadata`, OData V2,
namespace `com.sap.hci.api`) confirms the contract above property for property:

| Entity type | Key | Properties | Navigation |
|---|---|---|---|
| `AccessPolicy` | `Id` (`Edm.Int64`) | `RoleName`, `Description` | `ArtifactReferences`, `AccessPolicyRuntimeAssignments` |
| `ArtifactReference` | `Id` (`Edm.Int64`) | `Name`, `Description`, `Type`, `ConditionAttribute`, `ConditionValue`, `ConditionType` | `AccessPolicy` |
| `AccessPolicyRuntimeAssignment` | `Id` (`Edm.Int64`) | `RuntimeLocationId`, `TransferStatus`, `TransferErrors`, `StatusUpdatedAt` (`Edm.DateTime`) | `AccessPolicy` |

All three entity sets exist (`AccessPolicies`, `ArtifactReferences`,
`AccessPolicyRuntimeAssignments`). The document carries no `sap:creatable`/`sap:updatable`
annotations, so it says nothing about which operations each set accepts. The provider's
contract test (`internal/client/cloudintegration/metadata_contract_test.go`) checks the
structs and keys against this document whenever it is available locally.

### Still not publicly documented

- The wire constants for every artifact type other than Integration Flow, for the *ID*
  attribute, and for the *Matches* operator. The provider therefore passes these values
  through and only rejects the two former provider values proven wrong (`IntegrationFlow`,
  `EQUALS`).
- Whether `AccessPolicyRuntimeAssignments` can be written (their structure is now known, see
  above), the values `TransferStatus` takes, and which runtimes a policy created through the
  API is assigned to by default.
- Whether `ArtifactReferences` supports `PATCH`/`MERGE` (the provider replaces references
  instead), and whether `PATCH` with a different `RoleName` renames a policy. A tenant test
  settled the rest: `PUT` on `AccessPolicies` answers `501`, `PATCH` and `MERGE` of
  `Description` answer `204`, and create, `$filter`, reads and deletes behave as in the table.

`$metadata` cannot settle any of these. They need the Security Content API specification on
the Business Accelerator Hub (login required) or read requests against a tenant with existing
policies.

## `sapintegrationsuite_user_credential` / `sapintegrationsuite_oauth2_client_credential`

- **SAP product area**: Integration Suite / Security — Security Content ("Security Material" in
  the tenant UI, under *Monitor* > *Manage Security* > *Security Material*)
- **Official API**: Security Content API, published on SAP Business Accelerator Hub, same
  `/api/v1` OData V2 host as the Cloud Integration content APIs (confirmed via
  `internal/client/cloudintegration/client.go`'s existing doc comment, which already noted this
  client "covers the 'Integration Content' and 'Security Content' OData V2 services under
  `/api/v1`" before this feature family was implemented). A dedicated
  `internal/client/securitycontent` client package is used instead, kept separate from
  `cloudintegration` because these entities have fundamentally different secret semantics.
- **Entity sets**: `UserCredentials`, `OAuth2ClientCredentials`
- **Protocol**: OData V2
- **Operations**: GET, POST, PUT, DELETE. `PUT` is used for Update (full redeploy), matching
  SAP's Manage Security Material UI, which documents an explicit *Edit* action for Credentials
  artifacts ("You can also edit and redeploy an existing artifact") and states that editing an
  OAuth2 Client Credentials artifact specifically requires re-entering the client secret every
  time. This project could not confirm whether a successful `PUT` returns the updated entity
  body or `204 No Content`, so both resources re-read the entity with `GET` after every `PUT`
  rather than trusting the `PUT` response shape — the same defensive pattern already used by
  `UpdateAccessPolicy`.
- **`UserCredentials` fields — confirmed**: `Name` (the artifact's alias, used as both display
  name and OData key — SAP's own UI documentation states "the artifact name is used as an alias
  for the confidential data"), `User`, `Password` (write-only, never read back), `Description`.
  Corroborated by a documented third-party example payload (`POST .../UserCredentials` with a
  JSON body of `Name`, `Kind`, `Description`, `User`, `Password`, `CompanyId`) rather than this
  project's own inspection of a live tenant's `$metadata`.
- **`UserCredentials` fields — confirmed existence, unconfirmed casing**: `Kind` (SAP's UI calls
  this "Type": empty/unset for a generic Basic/username-token credential, `SuccessFactors`, or
  `OpenConnectors`) and `CompanyId` (only meaningful when `Kind` is `SuccessFactors`; SAP's UI
  hides this field for every other kind). Both are corroborated by the same third-party example
  payload as above, not `$metadata`.
- **`UserCredentials` fields — not exposed**: a deployment status (SAP's UI shows
  Stored/Deployed/Error for security material generally). This project could not confirm the
  OData property name for it and would rather omit a `Computed` attribute than expose one that
  is silently always empty.
- **`OAuth2ClientCredentials` fields — confirmed**: `Name`, `Description`, `TokenServiceUrl`,
  `ClientId`, `ClientSecret` (write-only, never read back), `Scope`. Confirmed directly from
  SAP's Help Portal documentation for "Deploying an OAuth2 Client Credentials Artifact", which
  describes each field in prose with an unambiguous meaning — a stronger source tier than the
  `UserCredentials` third-party example payload.
- **`OAuth2ClientCredentials` fields — documented in the UI, not implemented**: Grant Type
  (whether the grant type is sent as part of the URL or the request body), Client Authentication
  (whether the client ID/secret are sent as a body parameter or an `Authorization` header),
  Resource, Audience, and up to 20 custom Key/Value/"Send as Part of" parameters. SAP's Help
  Portal documents all of these in prose, but this project could not confirm their OData
  property names or JSON shapes against `$metadata` or a documented example payload, so they are
  deliberately left unimplemented rather than guessed at.
- **Write-only secret design**: see `docs/guides/security-content.md` for the full rationale.
  In short: `password_wo`/`client_secret_wo` are Terraform Plugin Framework `WriteOnly`
  attributes (requires Terraform CLI 1.11+), paired with a plain `password_wo_version`/
  `client_secret_wo_version` string that is the sole signal Terraform uses to decide whether to
  redeploy the credential. Both Go client types (`UserCredential`, `OAuth2ClientCredential`)
  structurally have no field for the secret, so it cannot end up in state, a diagnostic, or a
  log line regardless of what SAP's response body contains.
- **Required roles**: not separately confirmed for this research pass; assumed to fall under the
  same Integration Suite "Manage Security" scopes as Access Policies and Keystore
  administration, pending confirmation.

## `sapintegrationsuite_certificate`, `sapintegrationsuite_key_pair`, and the rest of Security Content

- **SAP product area**: Integration Suite / Cloud Integration — the "Security Content" OData V2
  API (`https://api.sap.com/api/SecurityContent`), covering `KeystoreEntries`, `Keystores`,
  `KeystoreResources`, `HistoryKeystoreEntries`, `CertificateResources`,
  `KeyPairGenerationRequests`, `UserCredentials`, `OAuth2ClientCredentials`, and (conceptually,
  per SAP's own overview) `SecureParameters` and `CertificateUserMapping`.
- **Research method**: the same official-mirror technique used throughout this project —
  `docs/ci/Development/security-content-e01d3f0.md` (the overview/resource table) and
  `docs/ci/Development/security-content-example-requests-acb89ef.md` (the curated,
  authoritative index of every worked example SAP documents for this API family — the same
  pattern that resolved the Number Ranges research: an entity absent from this index has no
  documented example anywhere, even if it is mentioned conceptually elsewhere).

### Tenant `$metadata` findings (September 2026)

A tenant `$metadata` document (`/api/v1`, namespace `com.sap.hci.api`) settles several
questions that the documentation left open. It confirms names, keys and types. It does not say
which operations an entity set accepts, because the document carries no `sap:creatable` or
`sap:updatable` annotations.

| Entity type (set) | Key | Relevant properties | Effect on the provider |
|---|---|---|---|
| `OAuth2ClientCredential` (`OAuth2ClientCredentials`) | `Name` | `TokenServiceUrl`, `ClientId`, `ClientSecret`, `ClientAuthentication`, `Scope`, `ScopeContentType`, `Resource`, `Audience`, `SecurityArtifactDescriptor`; navigation `CustomParameters` | Four new pass-through attributes; custom parameters documented as a gap |
| `CustomParameter` (`CustomParameters`) | `Key`, `Value`, `SendAsPartOf` | same | Write path undocumented |
| `UserCredential` (`UserCredentials`) | `Name` | `Kind`, `Description`, `User`, `Password`, `CompanyId`, `SecurityArtifactDescriptor` | Matches the implementation |
| `KeystoreEntry` (`KeystoreEntries`) | `Hexalias` (inherited) | via `BaseType` chain: `Alias`, `KeyType`, `KeySize`, `ValidNotBefore`/`ValidNotAfter` (`Edm.DateTimeOffset`), `SerialNumber`, `SignatureAlgorithm`, `EllipticCurve`, `Validity`, `SubjectDN`, `IssuerDN`, `Version`, `FingerprintSha1/256/512`; own: `Type`, `Owner`, `Status`, `CreatedBy`/`CreatedTime`, `LastModifiedBy`/`LastModifiedTime` | Keystore data sources expose all of them |
| `SecureParameter` (`SecureParameters`) | `Name` | `Description`, `SecureParam`, `DeployedBy`, `DeployedOn`, `Status` | Exists; operations unverified |
| `CertificateChainResource` (`CertificateChainResources`), `ChainCertificate` (`ChainCertificates`) | `Hexalias` / `Hexalias`, `Index` | media entity / certificate details | Upload format undocumented |
| `OAuth2AuthorizationCode` (`OAuth2AuthorizationCodes`) | `Name` | provider, auth/token URL, client, refresh token fields, `RuntimeId`, `IsEdge`; function imports `OAuth2AuthorizationCodeFullAuthUrl`, `OAuthTokenFromCode`, `OAuth2AuthorizationCodeRefreshTokenUpdate` | Out of scope (interactive authorization) |
| PGP: `PgpKeyrings`, `PgpPublicKeyrings`, `PgpSecretKeyrings`, `PgpKeyEntries`, `PgpSubKeys`, `PgpUserIds`, keyring resources | various | keyring metadata, key details | No documented requests |

Absent from the document entirely: OAuth2 SAML Bearer Assertion, OAuth2 Password Credentials,
Known Hosts and where-used information. For those, the conclusion "no public API" now rests on
the service's own model, not only on the absence of documentation.

Two consequences for existing code:

- `Edm.DateTimeOffset` values arrive in JSON as `/Date(<millis>+0000)/`.
  `internal/client/odata/v2.ParseDateLiteral` now accepts the offset, and the keystore data
  sources return RFC 3339 instead of the raw literal.
- The OAuth2 update is a `PUT`, which replaces the entity. Earlier releases sent only the six
  fields they knew about, so an update could reset settings made in the UI. The four new
  attributes are optional and computed and are always resent.

### Hex alias encoding — confirmed explicitly, not just by example

SAP's Security Content overview page states the rule directly, not merely by example: **"Hex
representation is used for the `Alias` field... Hex representation of the alias does mean that a
UTF-8-encoded byte array is built from the alias string. A hex string is then calculated from
this byte array."** It also explains why: **"the server doesn't allow slashes or backslashes in a
URI, even if they're percent encoded."** This is the same rule this project had already inferred
for Partner Directory's `AlternativePartners` (`Hexagency`/`Hexscheme`/`Hexid`) from one example
request URL alone; Security Content states it as an explicit rule. The shared implementation
(`internal/client/odata/v2/hexkey.go`, `EncodeUTF8Hex`/`DecodeUTF8Hex`) is asserted against every
alias SAP's own documentation uses as a worked example across both API families (`smtp.mail.
yahoo.com`, `mycertificate`, `mykeypair`, `baltimore cybertrust root`, `agency1`), plus Unicode,
punctuation, semicolon, slash, and backslash cases — see
`internal/client/odata/v2/hexkey_test.go`.

### Keystore Entries — confirmed read operations, confirmed field gap

*Superseded in part (September 2026): the "field gap" below is closed. The tenant `$metadata` names every property; see "Tenant `$metadata` findings" above.*

- `GET /api/v1/KeystoreEntries` (all entries) and `GET /api/v1/KeystoreEntries('{Hexalias}')`
  (single entry by alias) are both confirmed with an identical documented example response:
  `Hexalias`, `Alias`, `KeyType`, `KeySize`, `ValidNotBefore`, `ValidNotAfter`, then a literal
  `....` truncation in SAP's own source.
- The overview page's prose separately states a keystore entry also carries "Subject DN and
  Issuer DN, and administrative information such as the time when the entry was last modified" —
  confirming those properties *exist*, but never giving their exact JSON property name or
  casing. This project does not guess at them: `sapintegrationsuite_certificate` derives subject
  DN, issuer DN, serial number, and a SHA-256 fingerprint locally instead, by parsing the
  certificate bytes it can already confirmedly retrieve with Go's own `crypto/x509` — see
  `internal/client/securitycontent/x509meta.go`.
- `GET /api/v1/KeystoreEntries('{Hexalias}')/Certificate/$value` (Export Certificate): confirmed,
  response content type `application/pkix-cert`, PEM body.
- `GET /api/v1/KeystoreEntries('{Hexalias}')/Sshkey/$value` (Export Public Key in OpenSSH
  Format): confirmed, response is the raw `ssh-rsa AAAA... <comment>` line. SAP documents this as
  supported only for RSA- and DSA-keyed entries ("Other algorithms, for example elliptic curve
  (EC), aren't supported").
- `PUT /api/v1/KeystoreEntries('{Hexalias}')?renameAlias=<new>` (Rename the Alias of a Keystore
  Entry): confirmed to exist, but deliberately not used by this provider — `alias` is
  `RequiresReplace` on both `sapintegrationsuite_certificate` and `sapintegrationsuite_key_pair`
  instead, keeping lifecycle behavior predictable, matching this provider's established
  precedent (`sapintegrationsuite_number_range`'s `name`). SAP's documentation for this operation
  states "The request body must contain a property of the `KeystoreEntry` entity type with a
  null value. Otherwise, you get an exception" — an unusual enough contract on its own to avoid
  relying on until there is a concrete reason to.
- **No field distinguishes SAP-owned from tenant-administrator-owned entries.** The overview page
  states in prose that "a keystore typically contains entries that belong to the tenant
  administrator and entries that are owned by SAP," but no property name for this distinction
  appears anywhere. This provider does not invent a heuristic (for example matching alias name
  patterns) to detect ownership in advance; instead, `sapintegrationsuite_certificate` and
  `sapintegrationsuite_key_pair` rely on and surface whatever error SAP's own server-side
  protection returns when an Update or Delete is actually attempted against a protected alias —
  the only mechanism this provider can trust for a fact SAP does not expose as data.

### Certificate — confirmed Create/Update, confirmed Delete via the shared keystore mass-delete

- `PUT /api/v1/CertificateResources('{Hexalias}')/$value` (Import and Update Certificate):
  confirmed for both create and update — SAP's own documentation flags the quirk explicitly:
  **"Although an HTTP PUT method is used, a new OData entity is created with the sample
  request."** SAP's documented example request body is enclosed in literal square brackets,
  `[-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----]`; no other example anywhere in
  either API family this project reviewed uses that convention for an inline value example, so
  this is treated as documentation formatting, not literal bytes to send — this client's
  `PutCertificate` sends the caller's PEM content exactly as given, with no bracket wrapping.
  **Tenant test, September 2026:** `Content-Type: application/pkix-cert` with a plain PEM body
  (no brackets) is accepted. The plain request answered a self-signed certificate with `409`,
  the parsed certificate as body and `Status: notImported`;
  `?fingerprintVerified=true&returnKeystoreEntries=false` imported it (`204`). Replacing the
  certificate of an existing alias answered `400 Entry with alias "…" already exists in
  keystore "system"` with and without `fingerprintVerified=true`, and `204` once `update=true`
  was added; the new serial number read back. The option names match a community blog; the SAP
  Help page itself could not be read without a browser.
- No per-entity `DELETE` is documented for `CertificateResources` or `KeystoreEntries`. Delete
  uses the same confirmed `KeystoreResources('system')?deleteEntries=true` mass-deletion
  operation described below, with exactly the one alias the resource owns.

### Key Pair — confirmed generation, confirmed field table, confirmed absence of Update/GET-by-request

- `POST /api/v1/KeyPairGenerationRequests` (Generate a Key Pair): every field's
  mandatory/optional status, type, and default is confirmed verbatim from SAP's own "Input
  Properties" table — `Hexalias` (mandatory but "any dummy hex value," recalculated by SAP from
  `Alias`), `Alias`, `KeyType` (enum `RSA`/`DSA`/`EC`, default `RSA`), `SignatureAlgorithm`
  (enum depends on `KeyType`, default `SHA-512/RSA`), `KeySize` (mandatory for `RSA`/`DSA`,
  default 4096; for `EC`, 112-571 or use `KeyAlgorithmParameter` instead),
  `KeyAlgorithmParameter` (named EC curve, `EC` only), `CommonName` (mandatory), `Country`
  (mandatory, "two characters required"), `OrganizationUnit`/`Organization`/`Locality`/`State`/
  `Email` (all optional), `ValidNotBefore`/`ValidNotAfter` (optional, OData V2 JSON date literal
  wire format `/Date(<millis>)/`, confirmed directly from SAP's own worked example
  `"ValidNotBefore":"/Date(1469685029780)/"` — see `internal/client/odata/v2/date.go`).
  This provider's client always sends a fixed dummy `Hexalias` ("00"), matching SAP's own
  documented statement that the value is ignored and recalculated.
- **A documentation inconsistency worth flagging**: SAP's own worked request example writes
  `"SignatureAlgorithm":"SHA256/RSA"` (no hyphen), while the same page's field-table enum for the
  identical value is `"SHA-256/RSA"` (with a hyphen, matching every other enum value on the same
  table, including the default `"SHA-512/RSA"`). This client always sends the hyphenated form
  from the enum table and validates against it, treating the un-hyphenated inline example as a
  documentation artifact — the same category of quirk as Custom Tag Configuration's
  "Mr. Bean, Ms. Bean" comma-separated array example.
- **No GET is documented for `KeyPairGenerationRequests` or a distinct `KeyPairResources`
  entity** — the generation request is a one-shot command, not a persistent object with its own
  read-back. The persistent representation is the resulting `KeystoreEntries` entry, read via the
  same confirmed `GET KeystoreEntries('{Hexalias}')` used for Certificate — but that GET only
  confirms `KeyType`/`KeySize`/`ValidNotBefore`/`ValidNotAfter` back; `SignatureAlgorithm`,
  `KeyAlgorithmParameter`, and every subject DN field (`CommonName`, `Country`, etc.) are **not**
  confirmed returned by any documented GET. `sapintegrationsuite_key_pair` reads back and
  refreshes only the confirmed subset on every plan; the rest is trusted from the last successful
  write, never re-verified, which is why this feature is cataloged `partial` rather than
  `supported`.
- **No update operation is documented** for a generated key pair's material — every attribute
  that defines it is `RequiresReplace`.
- Delete uses the same confirmed `KeystoreResources('system')?deleteEntries=true` mass-deletion
  operation, exactly one alias.

### SSH Key — reverified and corrected: no separate resource exists

*Updated September 2026: the tenant `$metadata` does contain `SSHKeyGenerationRequests` (with `SSHFile` and `Password`) and `SSHKeyResources`, so SSH key generation has its own entities after all. Neither is documented with a request. The approach below (an RSA or DSA key pair plus the OpenSSH export) remains the supported path.*

SAP's Security Content API overview's own Resources table lists **no independent "SSH Key"
entry** — only Certificate, Key Pair, Keystore Entry, Keystore, Keystore History, User
Credentials, Secure Parameter, OAuth2 Client Credentials, Certificate-to-User-Mapping (Neo), and
Access Policies/Artifact References. The Operations-guide page
`docs/ci/Operations/creating-a-key-pair-ssh-key-pair-b8a8601.md` ("Creating a Key Pair/SSH Key
Pair") confirms why: its UI dialog is literally the same field set for both — "In the *Current*
tab, choose *Create* > *Key Pair* or *Create* > *SSH Key* depending upon your requirement," with
one shared attribute table (Alias, Key Type, Key Size, Signature Algorithm, the subject DN
fields, validity dates). No separate `SSHKeyGenerationRequests` field contract (mandatory/
optional status, types, example body) was found documented anywhere in either the Development or
Operations documentation tree. Given this, `sapintegrationsuite_key_pair` covers the SSH use case
directly: an RSA or DSA key pair's public key is exposed as `public_key_openssh` via the
confirmed `GetSSHPublicKey` export, with no separate resource. One inconsistency worth noting:
the "Creating a Key Pair/SSH Key Pair" UI dialog lists Key Type options as RSA/EC ("DSA (only for
key pair)"), while the SSH-export documentation says RSA and DSA are supported and EC is not —
these two SAP-authored pages do not fully agree with each other; this provider follows the
export endpoint's own documented restriction (RSA/DSA) for `public_key_openssh` population.

### Certificate Chain — reverified: folded into Key Pair, no independent contract found

*Updated September 2026: `CertificateChainResources` and `ChainCertificates` exist in the tenant `$metadata`; the upload format is still undocumented. See "Tenant `$metadata` findings" above.*

The overview page's Key Pair resource description states the API can be used to **"create a
certificate signing request, or import and export the related certificate chain"** — Certificate
Chain is documented as a *capability of* Key Pair, not an independently exampled resource. No
`CertificateChainResources` entry appears in the curated example-requests index, and no CSR/
signing-response field contract was found documented in either the Development or Operations
tree. Not implemented this phase; see `docs/resource-design.md` for the suitability conclusion
and the likely eventual naming (`sapintegrationsuite_key_pair_certificate_chain`) if a concrete
contract is later confirmed.

### Secure Parameter and Known Hosts — reverified, both remain without a confirmed contract

*Updated September 2026: `SecureParameters` exists in the tenant `$metadata` (operations unverified); Known Hosts does not. See "Tenant `$metadata` findings" above.*

- **Secure Parameter**: the overview page's Resources table does list "Secure Parameter"
  conceptually, alongside User Credentials/OAuth2 Client Credentials (which do have confirmed
  full contracts). But the curated example-requests index — the same authoritative page that
  correctly enumerated every other confirmed Security Content operation — lists **zero** example
  requests for it, and its own "Deploying a Secure Parameter Artifact" documentation page
  describes only the Eclipse/Node-Explorer design-time deployment wizard ("In the Node Explorer
  you have selected a tenant, in the context menu you have selected *Deploy Artifacts*..."), not
  a REST contract. This, combined with prior third-party evidence of an OData error resolving a
  `SecureParameters` entity set, keeps this feature unconfirmed rather than implemented.
- **Known Hosts**: does not appear in the overview page's Resources table at all — a stronger
  negative signal than Secure Parameter's conceptual-but-unexampled listing. Its own "Deploying
  an SSH Known Hosts Artifact" documentation describes only the Manage Security Material UI
  (*Create* > *Known Hosts (SSH)*, *Browse*/*Add*/*Deploy*), with no REST endpoint mentioned
  anywhere. Corrected from `research_required` to `no_public_api`.

### Certificate-User Mapping — unchanged from prior research: Neo-only, no Cloud Foundry equivalent

Reconfirmed during this pass: SAP's certificate-to-user mapping documentation ("Managing
Certificate-to-User Mappings", "Client Certificate Authentication and Certificate-to-User
Mapping (Inbound)", "Setting Up Inbound HTTP Connections with Certificate-to-User Mapping")
exists only under the **Neo** environment, with no Cloud Foundry equivalent found anywhere in
SAP's published documentation. Since this provider targets Cloud Foundry, this stays
`PublicAPI: false` / `no_public_api`.

### Required role

SAP's Security Content overview page documents the Cloud Foundry role template
`MonitoringDataRead` as required to access this OData API — the same role already documented for
the Message Stores API family. This provider does not manage that role or role collection.

### Whole-keystore operations — reverified, confirmed high-risk, deliberately not implemented

- `POST /api/v1/KeystoreResources` (Import/Back up a Keystore): confirmed contract — `Name`
  (fixed value `"system"`, SAP's documentation states "Currently, only `system` is allowed (SAP
  supports only one keystore)"), `Resource` (base64-encoded JKS/JCEKS keystore file),
  `password` (the keystore/private-key password — SAP documents "All private keys must have the
  same password"). Confirms the exact blast-radius risk this provider's `provider-scope.md` and
  `ROADMAP.md` already flag: a single call can create, update, leave unchanged, or remove many
  keystore entries at once, based on the imported file's contents — entries this provider has no
  way to know are or are not owned by a different Terraform module or administrator. Deliberately
  not implemented, independent of how well the contract is now understood.
- `PUT /api/v1/KeystoreResources('system')?deleteEntries=true` (Trigger Mass Deletion of
  Keystore Entries): confirmed contract, and the operation `sapintegrationsuite_certificate` and
  `sapintegrationsuite_key_pair` both use for Delete — see
  `internal/client/securitycontent/keystore.go`. Body: `{"Aliases":"alias1;alias2;alias3"}`,
  **plain alias text, not hex-encoded** (unlike `KeystoreEntries`' own key predicate). SAP
  documents exact escaping rules with two worked examples: a literal `;` in an alias becomes
  `\;`, and a literal `\` becomes `\\`. This project's own inference (confirmed by reproducing
  both worked examples byte-for-byte in
  `internal/client/securitycontent/keystore_test.go`) is that backslashes must be escaped
  *before* semicolons — escaping in the other order would re-escape the backslash this operation
  itself inserts for a semicolon, corrupting the result; SAP's documentation states both
  individual rules but does not spell out their combined order. Both
  `sapintegrationsuite_certificate` and `sapintegrationsuite_key_pair` call this with exactly the
  one alias they own — never a caller-supplied list — so a Terraform destroy of one resource can
  never be constructed in a way that also submits an unrelated alias.
- `PUT /api/v1/HistoryKeystoreEntries('{Hexalias}')?copy=true&destinationAlias=<...>` (Restore an
  Entry from the SAP Key History Keystore): confirmed contract, a restore/copy operation on
  SAP-managed historical key material, not a generic CRUD entity — no GET was found documented
  for `HistoryKeystoreEntries` either, so even a read-only data source is not implemented.
  Consistent with this provider's stance that history/audit data should never become a mutable
  Terraform resource.
- `GET /api/v1/Keystores('system')?$expand=Entries&$select=LastModifiedTime,Entries` (Get
  Keystore Properties and Entries): confirmed, but not separately implemented as a data source —
  it returns the same entry list `data.sapintegrationsuite_keystore_entries` already exposes,
  plus a keystore-level `LastModifiedTime` this project judged not worth a second, overlapping
  data source for.

## Current API Management: API Artifacts, MCP Servers, Integration Cell, Virtual Hosts, Runtime Profiles

*Re-audited September 2026. Result unchanged: no public API for any object in this family.*

The first research pass concluded that SAP publishes no API for this family. SAP shipped a lot
of new functionality here in 2026 (API-centric integration, the MCP Gateway, reusable APIs,
remote MCP servers, Client SDK 3.0.0), so the re-audit checked every channel again instead of
relying on that result. Each item below names what was checked and what it showed.

| # | Source | Identity / version | Finding |
|---|---|---|---|
| 1 | Integration Content API resource table | Help page `integration-content-d1679a8`, version synced 2026-07-10 | Resources: integration packages (Discover, Design), custom tags, integration flows with configurations and resources, message mappings, script collections, build and deploy status, MDI delta token, runtime artifacts with errors and endpoints, value mappings, integration adapters. No API artifact, MCP server, runtime profile or virtual host. |
| 2 | *API Documentation* index pages | `api-documentation-3fd9fc9` (Cloud Integration), `api-documentation-e26b332` (API Management) | Point to `https://api.sap.com/package/CloudIntegrationAPI/odata` and `https://api.sap.com/package/APIMgmt/all` only. |
| 3 | Lifecycle pages in `SAP-docs/btp-integration-suite`, `docs/ISuite_Integrations_APIs/` | ~60 pages matching api-artifact, mcp, runtime-profile, virtual-host, integration-cell | All UI procedures. None names an endpoint, entity set, HTTP method or Business Accelerator Hub page. |
| 4 | SAP API Management Client SDK | Maven Central `com.sap.apimgmt.client.sdk:apim-client-sdk`, versions up to 3.0.6 (published 2026-09-18); What's New "Update Client SDK to Version 3.0.0", 2026-09-20 | Classes: `APIProxyClient`, `APIProductClient`, `APIKeyValueMapClient` and models for proxies, products, providers, key maps, virtual hosts (Classic). Endpoints: `/apiportal/api/1.0/Management.svc`, `/apiportal/api/1.0/ContentArchive.svc`, `/api/1.0/apis/`. Classic API Portal only. |
| 5 | SAP's CI/CD tooling | `github.com/SAP/cicd-actions-for-sap-integration-suite`, 2026 | Actions for packages, integration flows, deployments, Partner Directory and access policies. None for API artifacts or MCP servers. |
| 6 | API Management What's New (Cloud Foundry) | `what-s-new-for-sap-api-management-cloud-foundry-d9d60be`, entries to 2026-09-20 | 2026 Integration Cell entries (API-centric integration, MCP Gateway, reusable API artifact, simplified creation, product subscriptions, trace data, tool limit 30, remote MCP servers) are UI features. No API announced. |
| 7 | Business Accelerator Hub | `api.sap.com` search and index | No page for API artifacts, MCP servers, Integration Cell or virtual hosts is indexed. The hub renders in the browser and gates specification downloads behind an SAP login, so this channel was only checked through search engines. |
| 8 | Tenant `$metadata` of `/api/v1` | Cloud Foundry tenant, September 2026, 136 entity sets, 38 function imports | No entity type for API artifacts, MCP servers, virtual hosts or runtime profiles. `APIDefinitions` is the set of definition links behind `ServiceEndpoints` (association `ServiceEndpoint_2_r_ApiDefinitions`), not an API artifact. |

*Accessing API Management APIs Programmatically* (`accessing-api-management-apis-programmatically-24a2c37`)
is still about the Classic API Portal's `apiportal-apiaccess` plan (roles
`APIPortal.Administrator`, `APIPortal.Guest`, `APIManagement.SelfService.Administrator`).

### Semantics recorded for a future implementation

These come from the current Help pages and would constrain any future resource design:

- **Runtime profile** is fixed after an API artifact is created, except for choosing the
  target Edge Integration Cell (*Creating an API Artifact*). SAP's *Runtime Profiles* page
  (`runtime-profiles-8007daa`) still lists Cloud Integration, Cloud Integration – Starter, SAP
  Process Orchestration and Edge Integration Cell, but no Integration Cell row.
- **Design-time and deployment-time virtual host** can differ. The deployed endpoint follows
  the deployment-time choice (*Deploy an MCP Server*, `deploy-an-mcp-server-017ba42`, and the
  corresponding API artifact page).
- **Virtual host rules** (`configuring-additional-virtual-host-26c7416`,
  `delete-an-eligible-virtual-host-102257d`): at most 11 virtual hosts per tenant; name
  `Default` reserved; alias up to 22 characters of letters, digits and hyphens, not starting
  with a hyphen; deletion blocked for the default virtual host and for hosts referenced by
  deployed APIs; APIs referencing a deleted host fall back to the default host.
- **API artifact versioning** (`api-artifact-versioning-3f2e06b`): versions coexist at design
  time and runtime, one version is active, and revert keeps all versions.
- **MCP server sources**: API artifact, HTTP endpoint with OpenAPI specification, RFC-enabled
  backend, remote MCP server, Classic API proxy. Deployment targets Integration Cell only.
- **Destinations for Integration Cell** (`create-an-api-artifact-using-a-destination-a0bdfd4`)
  need the label `IntegrationCell.Include = true`, and only proxy types `Internet` and
  `OnPremise` are supported.

### Side findings for other areas

- **Edge Integration Cell runtime targeting**: the Integration Content page (item 1) now
  documents `https://<host>/location/<runtime location id>/api/v1/<relative resource path>`
  for calls against an Edge Integration Cell. SAP's CI/CD tooling still called the EIC path
  "not yet published" in its version of 2026-05-05, two months before that Help version. This
  affects `edge_integration_cell.deployment_target` and is evaluated in the Edge Integration
  Cell section.
- **Classic API Management virtual hosts**: the custom-domain and mutual-TLS virtual host
  pages (`configuring-a-custom-domain-for-a-virtual-host-6b9e5a3`,
  `configuring-mutual-tls-for-default-domain-virtual-host-9faf7ce`) describe an API for the
  **Classic** API Portal, used with a service key for `APIManagement.SelfService.Administrator`.
  They say nothing about Integration Cell and are evaluated in the Classic API Management
  section.

## Classic API Management: API Providers, API Products, Certificate Store References, Key Value Maps

- **SAP product area**: Integration Suite / Classic API Management (the API Portal)
- **Official API**: `Management.svc`, a REST/OData V2 API under `/apiportal/api/1.0`, confirmed
  by direct inspection of the official "SAP API Management Standalone Service" PUBLIC user guide
  PDF (`help.sap.com/doc/fc5a7d8c89db4a448903df7d526b36f4/Cloud/en-US/...`), which contains
  complete verbatim worked request/response examples for several of this API's entities —
  the strongest primary-source evidence this project has had for any Classic API Management
  object, stronger even than most of the Cloud Integration API family, which this project has
  mostly had to confirm from individual UI-procedure pages rather than one consolidated
  reference document.
- **Authentication**: the `apiportal-apiaccess` service plan (`APIPortal.Administrator` for full
  access, `APIPortal.Guest` for read-only, `APIManagement.SelfService.Administrator` for virtual
  host self-service), generating a service key with `url`/`tokenUrl`/`clientId`/`clientSecret` —
  the same OAuth 2.0 client-credentials shape this provider already uses everywhere else, wired
  through the new `provider.api_management` block. Confirmed verbatim from
  `accessing-api-management-apis-programmatically-24a2c37.md`.
- **Response envelope**: confirmed, by direct comparison of worked examples, to be the identical
  OData V2 `{"d": {...}}` / `{"d": {"results": [...]}}` envelope, the identical
  `{"error": {"code", "message": {"lang", "value"}}}` error format, and the identical
  single-quoted key-predicate convention (`EntitySet('key')`,
  `EntitySet(field1='a',field2='b')` for composite keys) this project's
  `internal/client/odata/v2` package already implements for Cloud Integration — not assumed, but
  independently confirmed against this API's own worked examples before reusing that package
  here (`internal/client/apimanagementclassic`).

### API Provider (`APIProviders`)

Confirmed: `POST`, `GET`, `DELETE` (endpoint and worked examples confirmed across multiple
sources: the official user guide's UI-procedure section, a Business Accelerator Hub-derived
worked JSON payload, and web search results explicitly showing `DELETE
.../APIProviders('ES5_1')`). Fields confirmed for the "Internet" connection type specifically:
`name`, `title`, `description`, `destType` ("INTERNET"), `host`, `port`, `useSSL`, `trustAll`,
`pathPrefix`, `url` (Catalog Service Settings), `authType` ("BASIC" confirmed), `userName`,
`password`. Three further connection types (On Premise, Open Connectors, Cloud Integration) are
described in full UI-procedure detail in the official user guide but without a confirmed
field-level JSON mapping — this project did not guess at one.

**No Update exists.** SAP's official Piper `apiProviderUpload` pipeline step (`project-piper.io`,
SAP's own CI/CD tooling project) documents: "ApiProviderUpload only supports create
operation" — existing providers must be deleted before re-creation. A dedicated SAP Knowledge
Base Article (3459828, "Default payload of POST APIM OData API /APIProviders in Business
Accelerator Hub is incorrect") independently confirms the `POST /APIProviders` endpoint's
existence while also warning that SAP's own example payload has previously been wrong (HTTP 400
on the documented default) — evidence this project treats as a reason for extra caution around
this entity's exact payload shape, not a reason to avoid it (the shape used here is the one
corroborated by multiple independent sources, not the one flagged as broken).

**Eventual consistency, confirmed verbatim** from the official user guide's "Create an API
Provider" procedure: "When you create, update, or delete an API provider using the API, changes
may not be immediately reflected in subsequent GET API provider requests. API provider data is
cached to reduce calls to the destination service, which can result in stale data being returned
for a short period after a modification. The updated data is typically available within
approximately 20 seconds." Implemented as bounded, jittered exponential-backoff polling in
`internal/client/apimanagementclassic/retry.go`, not a fixed sleep.

### API Product (`APIProducts`, `APIProductAdditionalProperties`)

Confirmed verbatim from the official user guide's custom-attribute section, which — despite its
heading — shows the complete API Product entity shape in its Create (`POST`) example: `name`,
`version`, `isPublished`, `status_code` ("PUBLISHED" confirmed), `title`, `description`,
`isRestricted`, `scope`, `quotaCount`/`quotaInterval`/`quotaTimeUnit` (confirmed nullable —
`-99`/`-99`/`null` in one example, `null`/`null`/`null` in another), `additionalProperties`
(nested, `entityId`/`name`/`value`), `apiProxies` (nested `__metadata.uri` deep-insert
references, `"APIProxies(name='SampleAPI')"`), `apiResources`. Update (`PUT
APIProducts(name='...')`) is also shown verbatim, and — this is the key finding for this
provider's resource design — its payload is narrower than Create's: `name`, `title`, `scope`,
`description`, `version`, `status_code`, `isRestricted`, `isPublished`, `quotaCount`,
`quotaInterval`, `quotaTimeUnit`, with no `apiProxies`, `apiResources`, or `additionalProperties`
field at all. `APIProductAdditionalProperties` has its own confirmed `PUT
APIProductAdditionalProperties(entityId='...',name='...')` example for updating a single custom
attribute's value. DELETE for `APIProducts` itself is not shown verbatim anywhere reachable; this
provider infers it from the identical key-predicate DELETE convention confirmed directly for
`APIProviders` and `CertificateStoreReferences` within this same API.

**Tenant test, September 2026 (overrides parts of the guide above).** Run against an API Portal
with a service key holding `APIPortal.Administrator`:

- `POST APIProducts` with `status_code: "PUBLISHED"` and one `apiProxies` reference answered 201.
  `status_code: "DRAFT"` also answered 201 and returned `isPublished: false`. Without
  `status_code` SAP failed with a `NullPointerException` on `getStatus_code()`; without a proxy
  it refused with "At least one API Proxy should be linked to an API Product". SAP set
  `version` to `"1"` when none was sent.
- `PUT`, `PATCH` and `MERGE APIProducts('<name>')` on an existing product all answered `405
  UPDATE operation not supported on APIProduct entity`. The guide's `PUT` example no longer
  works.
- `POST APIProductAdditionalProperties` answered `405 CREATE operation not supported on
  APIProductAdditionalProperty entity`. Nested in the product's create without `entityId`,
  SAP answered `400 ADDITIONAL_PROPERTY_ENTITY_ID_MATCH_ERROR` ("Entity Id in Additional
  Property does not match"), so each nested property carries the product's name as
  `entityId`, as the guide's example does.
- `GET APIProducts('<name>')` returns every navigation property (`apiProxies`,
  `additionalProperties`, `apiResources`, `ratePlans` and others) as a `__deferred` link.
  `GET APIProducts('<name>')/apiProxies` lists the linked proxies with their `name`;
  `?$expand=apiProxies` works as well. `GET APIProducts('<name>')/additionalProperties`
  answers `{"d":{"results":[...]}}`.
- `DELETE APIProducts('<name>')` answered 204.

### Certificate Store Reference (`CertificateStoreReferences`)

The most completely confirmed entity in this whole research pass: the official user guide
dedicates four full subsections ("Creating the References", "Updating the References", "Reading
the References", "Deleting the References") to this exact entity, each with a complete, verbatim
worked request and response — `POST` with `{"name", "certificateStoreName"}`, response including
`storeType` ("TRUSTSTORE" confirmed) and a `life_cycle` block; `PUT
CertificateStoreReferences('<name>')` with `{"certificateStoreName"}` only; `GET` returning the
standard OData V2 collection envelope; `DELETE`. Documented error codes:
`CERTIFICATE_STORE_REFERENCE_NAME_DUPLICATION_ERROR`,
`CERTIFICATE_STORE_REFERENCE_CREATE_FAILED_LINKED_CERTIFICATE_STORE_VALIDATION_ERROR`,
`NO_SUCH_CERTIFICATE_STORE_REFERENCE_EXISTS`,
`CERTIFICATE_STORE_REFERENCE_UPDATE_FAILED_LINKED_CERTIFICATE_STORE_VALIDATION_ERROR`. The same
section explicitly states: "References can only be used for the keystore and truststore, not for
the certificates" — the actual certificate/keystore content upload remains a UI-only "Upload the
PKCS12/PFX file" step with no accompanying REST API found anywhere.

### Key Value Map (`GenericKeyMapEntries`)

Create confirmed verbatim from a worked example embedded in the API Proxy target-endpoint UI
documentation (an Open Connectors instance-token setup walkthrough): `POST
.../Management.svc/GenericKeyMapEntries` with `{"name", "scopeId", "scope" ("APIPROXY"
confirmed), "isEncrypted", "genericKeyMapEntryValues": [{"name", "mapName", "value", "scopeId",
"scope"}]}`. The official user guide's dedicated "Key Value Map" chapter separately confirms a
full UI lifecycle (Create/Update/Delete) exists, and one specific, important semantic: "You can
only update the Value field and not the Key field" — but shows no REST payload for Update or
Delete at the entry level, only the Create payload above. `isEncrypted` is confirmed as a real,
per-map field (the UI's "Encrypted" checkbox), but neither GET's response shape for an encrypted
map's entry values, nor an Update/Delete REST example of any kind, was found in any reachable
source.

### API Proxy (`APIProxies`) — confirmed real, Create mechanism not confirmed

The entity set's existence, `GET`, and `DELETE` are confirmed directly (the very first
programmatic-access documentation page uses `GET .../Management.svc/APIProxies` as its worked
example, and `DELETE .../APIProxies('<name>')` appears in search-indexed community content). The
proxy content bundle's structure is confirmed field-for-field from SAP's own public sample
repository, `SAP/apibusinesshub-api-recipes` (`apimanagement-security-mini-series/APIProxies/`):
a root `<Name>.xml` (`APIProxy` element: `name`, `title`, `description`, `service_code`,
`life_cycle`, `proxyEndPoints`, `targetEndPoints`, `policies`, `fileResources`),
`APIProxyEndPoint/default.xml`, `APITargetEndPoint/default.xml`, `Policy/*.xml`. The same
repository's own `readme.md` confirms a dependency this provider's client design accounts for:
"Direct import of the API Proxies will fail in most cases citing the non-availability of API
Providers that the proxies depend on."

What is not confirmed, despite a specific search for it (SAP Community threads discussing
`multipart/form-data`, a `Transport.svc` alternative service mentioned once in passing, and a
"Uploading Proxy to API Management via REST API" community thread that could not be fetched due
to that site's bot-blocking): the exact Create/Update wire format for the ZIP content itself. The
official user guide's own "Import an API Definition" section describes only the UI wizard
procedure, never a REST call. This provider does not implement `sapintegrationsuite_api_proxy`
on the strength of a confirmed bundle *shape* alone — Create needs an independently confirmed
request format, the same bar this provider applies everywhere else.

### Re-audit September 2026: Client SDK 3.0.6 and virtual hosts

**SAP API Management Client SDK.** `com.sap.apimgmt.client.sdk:apim-client-sdk` 3.0.6 (Maven
Central, published 2026-09-18; What's New "Update Client SDK to Version 3.0.0", 2026-09-20) was
disassembled with `javap` to read the endpoints and headers it sends. `StandardAPIProxyClient`:

| SDK method | Request |
|---|---|
| `importAPIProxy(byte[])` | `POST /apiportal/api/1.0/Transport.svc/APIProxies?name=?virtualhost=default`, `Content-Type: application/octet-stream`, body = proxy ZIP |
| `exportAPIProxy(name)` | `GET /apiportal/api/1.0/Transport.svc/APIProxies?name=<name>` |
| `exportAPIProxies(names)` | `ContentArchive.svc`, JSON listing proxy names with `"includedependencies": true` |
| `getAPIProxies()` | `GET Management.svc/APIProxies` (fields `name`, `title`, `description`, `version`, `state`, `status_code`, `service_code`, `isPublished`, `life_cycle`) |
| `updateAPIProxy(payload)` | `PUT Management.svc/APIProxies(name='<name>')` with `name` and `policyTemplateNames` |
| `createAPIProxy(payload)` | `POST /api/1.0/apis/` with `isFromCli: true`: an internal endpoint, not used |
| default virtual host | `GET Management.svc/VirtualHosts?$filter=isDefault eq true&$select=id` |

The odd import URL (`?name=?virtualhost=default`) is reproduced as the SDK sends it. A community
write-up of the same endpoint uses `?virtualhost=<GUID>` and a base64 string body instead. SAP
Help documents `Transport.svc` nowhere, so the upload format stays disputed and API Proxy stays
`public_api_incomplete`.

**Classic virtual hosts.** SAP Help (*Configuring a Default Domain for a Virtual Host*,
*Configuring a Custom Domain for a Virtual Host*, *Configuring Mutual TLS …*) documents a
request-style API: `POST /apiportal/operations/1.0/Configuration.svc/VirtualHostRequests` with
`operation` `CREATE`, `UPDATE` or `DELETE`, requiring a service key with
`APIManagement.SelfService.Administrator`. Request fields: `accountId` (subaccount subdomain),
`virtualHostUrl`, `isDefaultVirtualHostRequest`, `isForCustomDomain`, `keyStoreName`,
`keyStoreAlias`, `trustStore`, `isClientAuthEnabled`, `virtualHostId` (update/delete). The 201
response returns an `apimgmtconfiguration.VirtualHostRequest` with `virtualHostId`,
`allocationStatus`, `allocatedPort` and the TLS settings. Reading goes through
`Management.svc/VirtualHosts`, whose properties are not documented; the SDK confirms only `id`
and `isDefault`. Recorded as `api_management.classic.virtual_host`, `public_api_incomplete`,
until `Management.svc/$metadata` is available.

### Not evaluated this phase

Monetization, Rate Plans, and API Analytics were not researched: these are operational/reporting
concerns by their nature (the same category this provider already excludes for Message
Processing Logs and Edge Integration Cell's local monitoring API), and this phase's research
budget was directed at the desired-state-configuration candidates above instead.

## Partner Directory API

- **SAP product area**: Integration Suite / Cloud Integration — Partner Directory
- **Official API**: Partner Directory OData V2 API, under the same `/api/v1` service root as
  Integration Content and Security Content — confirmed via SAP's own "Partner Directory",
  "Partner Directory Concepts", "Partner Directory Entity Types", and "Requests for String
  Parameter, Binary Parameter, and Authorized User" documentation pages.
- **Entity sets and confirmed operations**:
  - `Partners` — "You can read all partners or delete a partner from the Partner
    Directory" (`partner-directory-0fe80dc`); no create. The tenant `$metadata` has only the key
    `Pid` (`MaxLength=60`). SAP documents Pid uniqueness as "ensured by the tenant owner
    application", and the UI's *Delete Partner* deletes "a partner and all its entities". No
    `sapintegrationsuite_partner` resource exists because of this — see `docs/resource-design.md`.
  - `StringParameters` — GET/POST/PUT/DELETE, key `(Pid, Id)`. A documented example request
    body: `{"Pid":"partner1","Id":"sp1","Value":"sp1v"}`.
  - `BinaryParameters` — GET/POST/PUT/DELETE, key `(Pid, Id)`, `Value` base64-encoded.
    Documented `ContentType` values: `xml`, `xsl`, `xsd`, `json`, `text`, `zip`, `gz`, `zlib`,
    `crt` (with encoding-suffixed variants such as `xml;encoding=UTF-8` also valid — this
    provider does not restrict `content_type` to the short documented list for that reason).
    The encoding suffix is confirmed by `partner-directory-0fe80dc` ("you can also specify the
    encoding for xml, xsl, xsd, json, and text (separated by semicolon)"). Size: the request
    examples say 260 KB, the entity types page says 262,144 bytes and 1.5 MB, and the tenant
    `$metadata` declares `Value` as `Edm.Binary` with `MaxLength=1572864`. Re-audit September 2026: the
    provider enforces the `$metadata` value, pinned by the partnerdirectory contract test.
  - `AlternativePartners` — GET/POST/PUT/DELETE. The documented entity carries both plain
    (`Agency`, `Scheme`, `Id`, `Pid`) and hex-encoded (`Hexagency`, `Hexscheme`, `Hexid`) fields;
    a documented example request URL, `AlternativePartners(Hexagency='6167656e637931',...)`,
    confirms the hex form is the actual OData key and that it is the lowercase hex encoding of
    the plain value's UTF-8 bytes (`"agency1"` → `"6167656e637931"`). Creation uses the plain
    fields; SAP computes the hex key itself.
  - `AuthorizedUsers` — GET/POST/PUT/DELETE, key `User`. SAP documents this as many-to-one
    (one communication user per Pid, a Pid can have several). Documented example body:
    `{"User": "...", "Pid": "PartnerZ"}`. Re-audit September 2026: SAP lowercases `User`. The
    example request creates `MyUser` and the response shows `myuser`; filters on `User` must be
    lowercase (Locale.English); the scripting API returns users lowercased. The provider rejects
    uppercase `user` values.
  - `UserCredentialParameters` — POST, GET (a `Pid` filter is mandatory for the collection)
    and DELETE. Re-audit September 2026 (`requests-for-usercredentialparameter-79c06dd`): "You
    can also use the POST request to update a User Credentials parameter with the same values
    for PID and Id", and "PUT requests are not supported". The provider updates user and
    password in place with that POST and checks for an existing entry before create. Documented example body:
    `{"Pid":"Receiver_1","Id":"USER","User":"...", "Password":"..."}`. SAP documents the
    generated security-artifact alias format as `pd:<Pid>:<Id>:UserCredential`, matching the
    property set via the exchange property `RECEIVER_CREDENTIAL` in scripts. SAP additionally
    documents that `UserCredentialParameter` and `CertificateUserMapping` cannot be included
    together with other entity types in a single OData ChangeSet (batch) request.
- **Password read-back**: SAP documents that "the returned value for the `Password` property
  is always null", unless the query option `returnHashedPassword=SHA256` asks for a SHA-256
  hash. The provider never sends that option, and its `UserCredentialParameter` Go type has no
  `Password` field at all.
- **Runtime cache**: string parameters, binary parameters and authorized users are cached on
  runtime nodes; invalidation after a change "can take a few minutes"
  (`partner-directory-cache-1577f77`).
- **CSRF protection**: SAP's OData V2 services on Cloud Foundry/BTP require a valid
  `X-CSRF-Token` for POST/PUT/PATCH/DELETE, obtained via a GET with `X-CSRF-Token: fetch`
  against the same resource, independently of OAuth authentication (OAuth proves identity;
  CSRF proves the write was not forged). This is documented generically for SAP's Cloud
  Integration `api/v1` OData services, not specifically restated on the Partner Directory
  pages, but there is no indication Partner Directory's shared service root is exempt from it.
  Implemented once in the shared HTTP client (`internal/client/http/csrf.go`) rather than
  per-resource, so it applies to every write this provider makes, not just Partner Directory's.
- **Pagination**: SAP's OData V2 services return a `__next` link (a complete absolute URL) in
  `d.__next` for server-driven paging; `GetAllPages` in `internal/client/odata/v2` follows it
  until exhausted. String Parameters in particular is documented as capable of holding large
  numbers of entries per tenant.
- **Required roles**: the Cloud Foundry role template `AuthGroup_TenantPartnerDirectoryConfigurator`
  is documented as required to work with a tenant's Partner Directory entities via the OData
  API (`AuthGroup_Administrator` also works). This provider does not manage that role or role
  collection — see `docs/provider-scope.md`.
- **Security note**: SAP explicitly documents that ordinary Partner Directory data (string and
  binary parameters) is stored unencrypted. This provider's documentation and resource
  descriptions explicitly warn against storing secrets there; only
  `UserCredentialParameters` is treated as a credential store, and even that is modeled
  conservatively — see the write-only `password_wo` design in `docs/resource-design.md`.
- **SAP's own client code**: SAP's
  [Partner Directory accelerator](https://github.com/SAP-samples/integration-suite-partner-directory-accelerator-for-pipeline-concept)
  (`HttpRequestHandler.java`) sends the same requests as this provider: `POST AlternativePartners`
  with `Agency`, `Scheme`, `Id`, `Pid` and `PUT AlternativePartners(Hexagency=…,Hexscheme=…,Hexid=…)`
  with only `Pid`; `PUT StringParameters(Pid=…,Id=…)` with only `Value`; `PUT
  BinaryParameters(Pid=…,Id=…)` with `ContentType` and the base64 `Value`; `DELETE` on the same
  keys; lists with `$filter=Pid eq '…'`. It sends no CSRF token, which matches this provider
  fetching one only when SAP answers 403 with `X-CSRF-Token: Required`. It also uses `$select`
  and `startswith(…)` in `$filter` on these entity sets, which this provider does not need.

## Remaining Integration Suite capability audit

A final sweep of every SAP Integration Suite capability area this provider had not yet formally
classified, using the current SAP-docs mirror as the source of truth (not the pre-existing
catalog) — auditing the actual documentation tree turned up two capability areas
(`ISuite_API_Composition/`, `ISuite_OData_Provisioning/`) with no prior catalog entry at all,
alongside the three (Event Mesh, Data Space Integration, Open Connectors) already tracked as
`research_required` placeholders.

### API Composition — business data graph (re-audited September 2026)

API Composition (formerly Graph) is activated as a capability of API Management, together with
Developer Hub (`initial-setup-12ad448.md`). Its business data graphs are managed through a
Configuration API. Sources, all from the `ISuite_API_Composition` folder of the SAP-docs mirror
of the Integration Suite help:

- `configuration-api-specification-and-usage-b5b27c9.md`: service root
  `{region-specific host}/configuration/v1/sap.graph`, OData metadata at `.../$metadata`, the
  `GraphConfiguration` resource, the property table, a full create example
  (`POST .../GraphConfiguration`, 201), `GET .../GraphConfiguration/{BDG-Id}` (200) and
  `PATCH .../GraphConfiguration/{BDG-Id}` (200, no body shown), and the status model
  (`PROCESSING`, then `DEPLOYMENT_INITIATED` or `FAILED`, with `logMessages`). The table marks
  `businessDataGraphIdentifier`, `dataSources` and `locatingPolicy` as required, and
  `effectiveGraphModelVersion`, `statusDetails`, `logMessages` and `status` as read-only.
- `business-data-graph-configuration-file-e93d38c.md`: the field-level model. The identifier is
  up to 20 lowercase alphanumeric characters with hyphens. Services have `destinationName` and an
  optional `path`. `locatingPolicy` is an object with `cues` (objects with `name` and
  `description`), `keyMapping` (`foreignKey` and `references`, each with `dataSource`,
  `entityName`, `attributes` limited to one entry, and an optional `strategy` with `name` =
  `format`, `match`, `replace`) and `rules` (`name`, `leading`, optional `local`, `cues`,
  `sourceEntity`). `exclude` takes entity names with an optional trailing wildcard. The page also
  describes OData containment and cue-scoped key mappings without naming a property for either.
- `data-locating-policy-28d2c2c.md`: rule selection prefers a specific name over a wildcard and
  a rule with a matching cue over the default rule; order carries no meaning.
- `manage-business-data-graphs-using-api-composition-configuration-api-655bf12.md`: the API
  creates, updates and deletes graphs, and "managing extensions is not supported".
- `initial-setup-12ad448.md`: the API Composition service plan `configuration` is the one for the
  Configuration API; Process Integration Runtime plan `integration-flow` is for consuming graphs.
  Roles `Graph_Key_User` (role collection `Graph.KeyUser`) and the read-only `Graph_Guest`.
- `connect-to-your-business-systems-1a0dd22.md`: destinations need the additional property
  `IntegrationCell.Include = true`.

Not documented, and therefore inferred or left out by the provider: the PATCH body (the provider
sends the writable properties), the delete request (`DELETE` on the graph's URL), the field names
of the `configuration` plan's service key, the structure of a `logMessages` entry, and properties
for OData containment and cue-scoped key mappings. The property table's "Array of locating
policies" contradicts both examples, which show a single object; the provider follows the
examples. The `$metadata` document needs credentials and has not been checked.

**Correction:** the original audit named Process Integration Runtime with plan `integration-flow`
as the credentials for the Configuration API. That plan is for client applications that consume
a graph. The Configuration API uses an API Composition instance with plan `configuration`.

Implemented as the experimental `sapintegrationsuite_business_data_graph` resource and data
source; see `docs/guides/api-composition.md` and the catalog entry
`api_composition.business_data_graph`.

### OData Provisioning — no public management API (re-audited September 2026)

Exposes SAP Business Suite backend OData services (SAP Gateway back-end enablement) through
Integration Suite without an on-premise SAP Gateway hub. The first audit read the role
`ODPAPIAccess` as a sign of a management API. The re-audit of all pages in
`docs/ISuite_OData_Provisioning/` corrects that: `runtime-access-and-role-assignment-for-odata-provisioning-b46816c.md`
defines `ODPManage` ("View and register OData services. Monitor errors, manage metadata validation
and cache settings"), `ODPAPIAccess` ("Access the service document from a link against each of
the registered OData services") and `APIFullAccess` ("Access the registered OData services via the
runtime"). The service instance (Serverless Runtime `xfs-runtime`, plan `odpruntime`, no
parameters) and its key serve runtime calls to registered services. Registration, destinations,
on/off status, multi-origin error tolerance and metadata/cache settings are documented only as UI
procedures, and the complete package list of the Business Accelerator Hub catalog (1,971
packages, read page by page; the catalog ignores `$filter`, so filtered searches are not
evidence) contains no OData Provisioning package. Reclassified `no_public_api`.

### Event Mesh — reclassified from a guess to a confirmed separate-provider candidate

Previously catalogued with an unverified "likely a separate BTP service" note. Reconfirmed
directly: Event Mesh activates through the same generic "Activating and Managing Capabilities"
mechanism as every other capability (no dedicated activation API, consistent with every other
capability this provider has audited), but its broker management surface itself — channels,
queues, topic subscriptions, webhook subscriptions — is genuinely public and well-documented
across roughly forty-five pages, built on Solace PubSub+'s own AMQP/MQTT/REST API family.
Event Mesh predates SAP Integration Suite and is consumed independently by many unrelated SAP
products; community Terraform support for Solace PubSub+ already exists separately. Reclassified
`StatusSeparateProvider` (the same status this provider already uses for Developer Hub), on the
same reasoning: a real, confirmed API, deliberately excluded because it belongs to a different
provider's boundary, not because it is unreachable.

### Data Space Integration — confirmed real API, flagged for a future dedicated phase

`using-apis-to-work-with-data-space-integration-411fd1e.md` confirms a dedicated "Data Space
Integration API Access" service instance (plan `api`, roles
`AuthGroup_DataspaceConsumer`/`AuthGroup_DataspaceProvider`, `client_credentials` grant) and
OData REST APIs listed on SAP Business Accelerator Hub at
`api.sap.com/package/dataspaceintegration/rest` — unreachable to this project without an SAP
support login, the same limitation hit repeatedly elsewhere. The object model (per-connector
service instances, Assets, Policies, Contract Definitions, Contract Negotiations/Agreements,
built on the external Dataspace Protocol / International Data Spaces standard) is genuinely
complex enough that this audit intentionally did not attempt a shallow implementation.
`research_required`, flagged as a second strong candidate (after Landscape Configuration in
Integration Assessment) for a future dedicated phase.

**Re-audit September 2026.** All 48 pages of `docs/ISuite_Data_Space_Integration/` were read.
The Hub's public catalog lists a single API in the package, `DSIAPI` ("Data Space
Integration", "Manage Data Space Integration resources via API"), `SubType: REST`, version
2.0.0, modified July 2026; the earlier "OData" label came from boilerplate on SAP's API page.
Concrete requests appear only for consumer runtime flows: `POST /api/dsi/v1/catalog`
(`counterPartyAddress`, `counterPartyId`, `filterExpression`), `POST
/api/dsi/v1/contract-negotiation` with `…/{id}/state` and `…/{id}/agreement`, `POST
/api/dsi/v1/transfer-process` with `…/{id}/edr`, and the Convenient Data Request API.
Assets, policies, contract definitions, company policies and contract references are
documented as UI procedures only; the role table grants `DataspaceProvider` "create assets,
policies, and contract definitions" without saying whether that covers the API. UI
lifecycle facts relevant to a future design: asset IDs are immutable, contract definitions
are editable only while unused, SAP-delivered company policies are read-only, and asset data
addresses carry credentials. See `docs/guides/data-space-integration.md`.

### Open Connectors — a deliberate scope judgment, not a research gap

Confirmed to be a catalog of 170+ independent third-party connector types (the former standalone
Cloud Elements product), each with its own normalized-but-still-connector-specific REST API and
configuration schema. Reclassified from `research_required` to `out_of_scope`: this is not "not
yet researched," it is a considered judgment that a 170-plus-connector-type catalog does not fit
this provider's schema-first, one-API-family-per-resource design without either an unbounded
per-connector-type schema explosion or an opaque untyped-JSON escape hatch this provider does not
otherwise offer anywhere.

### Capability activation — reconfirmed once more, not reopened

No public activation API was found for any capability audited across this entire multi-phase
run (Cloud Integration, API Management — classic and current —, Integration Cell, Edge
Integration Cell, Developer Hub, API Composition, Integration Assessment, Trading Partner
Management, Integration Advisor, Migration Assessment, Data Space Integration, Event Mesh, Open
Connectors, OData Provisioning). Every one of them activates through the same generic *Settings*
> *Add Capabilities* / *Activate Capabilities* UI flow. This is treated as a settled finding
across the whole provider, not something to keep re-verifying capability-by-capability.

## Migration Assessment

- **SAP product area**: Integration Suite / Migration Assessment (SAP Process Orchestration to
  Integration Suite migration evaluation).
- **Research method and result**: Migration Assessment's documentation tree is small — around
  fifteen pages, entirely checked. No dedicated API-access or service-key page exists anywhere in
  it, the same negative-evidence signal used for Trading Partner Management and Integration
  Advisor.
- **A finding worth stating precisely**: Migration Assessment's documentation does mention APIs,
  but in the opposite direction from what this research question needs. To extract data, it
  reaches into a registered source system's own SAP Process Orchestration APIs (via the SAP
  Destination service and typically Cloud Connector) — confirmed from
  `add-an-sap-process-orchestration-system-5f76723.md`: "As API endpoints and subpaths are used
  to extract data from your SAP Process Orchestration system, make sure that the SAP Destination
  service can access the endpoints listed..." This is Migration Assessment *consuming* an API
  from the source system, never Migration Assessment *exposing* one for its own objects.
- **Object model, confirmed from `create-a-data-extraction-request-ce0ad0e.md` and
  `concepts-324507c.md`**: choosing *Create* on a Data Extraction Request immediately starts an
  extraction with a resulting `Completed`/`Completed with warnings`/`Completed with errors`
  status — an imperative action, not a declarative object. Scenario Evaluation results are
  assessment-category classifications (*Ready to Migrate* / *Adjustment Required* / *Evaluation
  Required*), migration-readiness scores, and effort estimates — reporting output.
- **Consequence for this phase's Terraform decisions**: no resource or data source implemented.
  Unlike every other capability audited so far, this conclusion would hold even if a public API
  were confirmed tomorrow: both remaining object types (extraction requests, evaluation results)
  are workflow/reporting data by nature, not desired-state configuration. See
  `docs/guides/migration-assessment.md`.

## Integration Advisor

- **SAP product area**: Integration Suite / Integration Advisor (B2B interface content design —
  MIGs, MAGs, Type Systems, Codelists, Shared Code, Global Parameters).
- **Research method and result**: the same dedicated-API-access-page search applied to Classic
  API Management, Integration Assessment, and Trading Partner Management returned nothing across
  roughly eighty-five pages checked. `grep`-searching the runtime-artifact and export/import
  pages for "API", "REST", "OData", "curl", and "endpoint" found no matches beyond a single
  generic sentence about SAP Integration Suite supporting "any kind of interface/API format" for
  generated *content*, not a statement about Integration Advisor's own management API.
- **A finding that needed careful reading, not a quick dismissal**:
  `creating-oauth-client-credentials-for-cloud-foundry-environment-50b63c6.md` initially looked
  like exactly the kind of page that confirmed a public API elsewhere. On full reading, it
  describes creating a *Process Integration Runtime* service instance (plan `api`, role
  `WorkspacePackagesEdit`) — this provider's own existing Cloud Integration OAuth mechanism, not
  a separate Integration Advisor credential. It exists in this documentation area because those
  credentials authenticate the *injection* step below, not Integration Advisor's own content.
- **Runtime artifact injection** (`inject-mapping-artifacts-to-sap-cloud-integration-47ad97e.md`):
  confirmed as a UI wizard — *Mapping Guideline* > *Inject* > *SAP Cloud Integration Flow
  Resources* > choose tenant/package/integration flow > *Inject* — that pushes generated runtime
  artifacts directly into a Cloud Integration integration flow's resources, targeting either the
  built-in tenant or an externally configured BTP Destination. No REST equivalent documented.
- **Consequence for this phase's Terraform decisions**: no resource or data source implemented.
  Outcome C (real, UI-documented SAP functionality, no public API found), the same category as
  Current API Management and Trading Partner Management. See
  `docs/guides/integration-advisor.md`.

## Trading Partner Management

- **SAP product area**: Integration Suite / Trading Partner Management (B2B/EDI partner
  onboarding), under *Design* > *B2B Scenarios*.
- **Research method and result**: the method that worked for Classic API Management
  (`accessing-api-management-apis-programmatically-...md`) and Integration Assessment
  (`creating-service-instance-and-service-key-to-enable-api-calling-...md`) — search the
  capability's entire documentation tree for a dedicated API-access or service-key page — was
  applied here and found nothing. Roughly ninety pages under
  `docs/ISuite_Trading_Partner_Management/` in the SAP-docs mirror were enumerated by path;
  a targeted `grep` for "API", "REST", "OData", "service key", and "service instance" across the
  capability overview, task/permissions, export, and configuration-manager pages (the ones most
  likely to mention programmatic access if it existed) returned zero matches.
- **What is documented, all UI-only**: a *Download* button on Company Profile, Trading Partner,
  Agreement Template, and Agreement produces a browser-downloaded JSON file
  (`company.json`, `TradingPartner_<name>.json`, `Template_<name>.json`, `Agreement_<name>.json`)
  — confirming these objects are internally JSON-shaped, but not evidence of a REST endpoint.
- **Confirmed relationship to Partner Directory** (`partner-directory-data-1d92d5c.md`, quoted
  verbatim): "When a trading partner agreement gets activated, the complete agreement information
  gets pushed into the partner directory. An entry is created in the partner directory for each
  business transaction activity in the agreement and for each Interchange Envelope extraction."
  Generated entries are visible read-only under *Partner Directory Data*, prefixed `SAP_TPM`.
  This confirms Trading Partner Management is a design-time workflow that bulk-generates entries
  in the same Partner Directory store this provider already manages directly — not a separate
  storage layer, and not something this provider represents as its own set of resources even if
  an API for the trigger itself were confirmed, since activation-triggered generation is an
  imperative side effect, not desired-state configuration.
- **Consequence for this phase's Terraform decisions**: no resource or data source implemented.
  This is an Outcome-C finding (real, UI-documented SAP functionality, no public API found),
  the same category as Current API Management's family, reached through the same negative-
  evidence method that has proven reliable across every capability audited so far. See
  `docs/guides/trading-partner-management.md`.

## Integration Assessment

- **SAP product area**: SAP Integration Solution Advisory Methodology (ISA-M) / Integration
  Assessment, provisioned as its own separate BTP service subscription (entitlement
  `integration-assessment`, service `Integration Assessment APIs`, plan `default`) — not a
  sub-feature of Cloud Integration or either API Management model.
- **Authentication**: confirmed from `creating-service-instance-and-service-key-to-enable-api-calling-749897f.md`.
  A service key for this instance exposes `entities` (base URL for the "Entities API"),
  `management` (base URL for the "Management API" — two distinct base URLs, a genuinely
  confirmed finding, though what the split means at the wire level was not investigated further
  since no implementation was reached), `clientid`, `clientsecret`, and `url` (the OAuth 2.0
  token server, append `/oauth/token`) — the same client-credentials shape this provider uses
  everywhere else.
- **Entity inventory**: confirmed exhaustively from `integration-assessment-apis-47847b5.md`,
  which lists every entity in this capability with a one-paragraph description: Domain, Style,
  Use Case Pattern, Integration Pattern, Key Characteristic (+ Group/Value/Recommendation),
  Deployment Model, Domain Determination (ISA-M reference taxonomy); Application, Application
  Instance, Technology, Technology Instance, Vendor, Technology Domain, Technology Style,
  Technology Key Characteristic (landscape configuration, with documented per-tenant limits: a
  maximum of 20,000 Applications, 20,000 Application Instances, 50 Technologies, 150 Technology
  Instances, 10,000 Vendors); Request, Request Line Item, Integration Flow, Message Flow,
  Integration Flow Message Flow, Request Line Item Technology Instance Decision (assessment
  workflow, with a documented Request status machine: `draft` → `new` → `in progress` →
  `completed`, plus a `Reopen` action).
- **What is not confirmed**: a field-level JSON request/response schema for any single entity.
  Checked and found to contain UI procedures only: the entire `docs/ISuite_Integration_Assessment/`
  SAP-docs mirror tree (every page read), the official "SAP Integration Solution Advisory
  Methodology" PUBLIC PDF user guide (`help.sap.com/doc/ac5a3b73452548cb887f3963877eb9ab/...`,
  2400+ lines, entirely conceptual/methodology content, zero REST/curl examples), and SAP's own
  TechEd IN262 hands-on sample repository (`SAP-archive/teched2022-IN262` — a UI-screenshot
  walkthrough, no API calls). The SAP Business Accelerator Hub package
  (`hub.sap.com/package/SAPIntegrationAssessment/overview`) is unreachable without an SAP support
  login, the same limitation this project has hit repeatedly for other packages.
- **Re-audit September 2026**: the SAP-docs pages are unchanged. The Hub's catalog service
  (`api.sap.com/odata/1.0/catalog.svc`) is readable anonymously at package level:
  `ContentPackages('SAPIntegrationAssessment')/Artifacts` lists `EntitiesAPI` ("Entities",
  "Access entities of Integration Assessment") and `ManagementAPI` ("Management", "Manage
  content of Integration Assessment"), both `SubType: ODATA`, version 1.0.0, state `ACTIVE`,
  last modified July 2025. The artifacts' `$value` and `APIContent.APIs(...)` both redirect to
  the Hub's OAuth flow (public client `sb-hubXsuaa-public`), which ends at a login page, so the
  specifications stay out of reach. A GitHub repository search found no SAP sample calling
  these APIs beyond the UI-only `teched2022-IN262`. Since both are OData services, their
  `$metadata` documents, fetched with a service key, are the way to confirm the contract (see
  `CONTRIBUTING.md`, "Checking wire contracts against `$metadata`").
- **Consequence for this phase's Terraform decisions**: no resource or data source is
  implemented. Unlike Current API Management's family (no API exists at all) or Developer Hub's
  Product (a specific unconfirmed detail blocking an otherwise well-understood entity), this is a
  capability with a confirmed API surface and a completely confirmed entity inventory, but with
  every single entity's wire contract unconfirmed — this provider does not build a schema from
  entity descriptions and documented limits alone. See `docs/guides/integration-assessment.md`
  for the full three-group classification (master data / landscape configuration / assessment
  workflow) this research produced.

## Tenant probe results (September 2026)

A read-only probe against a development tenant, with the Cloud Integration client holding the
full set of role templates, answered questions the `$metadata` cannot:

| Entity set | Bare `GET` | `$format=json` only | `$top` only | `$select` (with or without `$expand`) |
|---|---|---|---|---|
| `OAuth2ClientCredentials` | 200 | 200 | 501 | 501 |
| `SecureParameters` | 200 | 200 | 501 | 501 |
| `UserCredentials` | 200 | 200 | 501 | 501 |
| `NumberRanges` | 200 | 200 | 501 | — |
| `KeystoreEntries` | 200 | **400** | 400 | 400 |

The provider's clients send no query options (the OData client negotiates JSON through the
`Accept` header), so their reads correspond to the bare column. A regression test in
`internal/client/securitycontent` fails if a read ever adds a query string. `KeystoreEntries`
returned all 14 SAP-delivered entries in one response without `__next`, with `Type` values
`Certificate` and `Key Pair`, `Owner` `SAP`, `Status` `unchanged`, an empty `Validity`, and
dates as `/Date(ms)/`. `GET NumberRanges` answering 200 is the first evidence that number
ranges can be read; reading one by key and deleting it still need a write test.

A write test with throw-away objects (all cleaned up) settled the lifecycles:

| Entity set | `POST` | `GET` by key | `PUT` | `DELETE` |
|---|---|---|---|---|
| `SecureParameters` | 202, empty body | 200, `SecureParam` null, `Status` `DEPLOYED` | 202, empty body | 202, then 404 |
| `OAuth2ClientCredentials` | 202, empty body | 200, `ClientSecret` null | — | 202, then 404 |
| `NumberRanges` | 202, empty body | 200 | — | 202, then 404 |

Every write answered `202 Accepted` without a body, so a client must read the entry back rather
than decode the create response. Defaults SAP fills in for an OAuth2 client credential created
without optional settings: `ScopeContentType` `urlencoded`, `Scope`, `Resource` and `Audience`
empty, `ClientAuthentication` null. A number range created with SAP's documented example values
(`MinValue` 0, `MaxValue` 9999, `FieldLength` 4, `CurrentValue` 0) and the name `tfAccProbeNr`
succeeded; the same values with the name `tf-acc-probe-nr` failed with a bodiless `500`, so
hyphens in the name are rejected. A `PUT` that set `CurrentValue` to 5 succeeded (202, read back
as 5); a following `PUT` without `CurrentValue` failed with a bodiless `500` and changed nothing
(counter and description as before), so an update must always carry the counter.

A further write test (September 2026), with SAP's error messages captured, found three more
contract gaps and confirmed the Partner Directory writes:

| Entity set | Finding |
|---|---|
| `IntegrationPackages` | `POST` without `ShortText`: `400` "Property 'ShortText' cannot be empty"; with it `201`. `PATCH`: `501` "Not implemented"; `PUT` of `{Id, Name, Description, ShortText}`: `202`. `Description` is returned as HTML (`<p>text</p>`, empty as `<p></p>`). |
| `UserCredentials` | `POST` without `Kind`: `500` "Property 'Kind' must not be empty or null"; with `Kind` but without `Description`: `500` "Property 'Description' must not be null"; with `Kind` `default`, `Description` and `CompanyId` `""`: `202`. Read returns `Kind` `default`. `PUT` update `202`. |
| `OAuth2ClientCredentials` | `PUT` update with the secret resent: `202`. |
| `Partners` | Read by key: `400` "Reading of single partner entitities is not supported"; collection and `$filter=Pid eq '<pid>'`: `200`. `DELETE Partners('<pid>')` after the partner's last entry was deleted: `404` "Partner not found". |
| `StringParameters`, `BinaryParameters`, `AlternativePartners` (hex key), `AuthorizedUsers`, `UserCredentialParameters` | Create `201`, read `200`, `PUT` update (for credential parameters a second `POST`) `204`/`201`, delete `204`; repointing an alternative partner or authorized user to another Pid with `PUT {Pid}` works. |

On the API portal, a key with `APIPortal.Administrator` read `APIProviders`, `APIProxies`,
`APIProducts`, `CertificateStoreReferences`, `GenericKeyMapEntries` and `VirtualHosts` (200),
while `Configuration.svc` and its `VirtualHostRequests` answered 403: virtual host changes need
`APIManagement.SelfService.Administrator`.

## Business Accelerator Hub catalog (re-audit September 2026)

The Hub's own catalog service, `https://api.sap.com/odata/1.0/catalog.svc`, answers anonymously
at package level. `ContentEntities.ContentPackages` lists every package (1,971 in September 2026,
read by following `__next`), and `ContentPackages('<name>')/Artifacts` lists a package's APIs
with name, type (`ODATA`, `ODATAV4`, `REST`), version and a one-line description. Two
limitations: the service ignores `$filter`, so a filtered query returns an unrelated page and
proves nothing; and everything below package level (`APIContent.APIs(...)`, an artifact's
`$value`, the specification files) redirects to the Hub's OAuth flow and ends at a login page.

What the package and artifact lists add:

| Package | API | Finding |
|---|---|---|
| `CloudIntegrationAPI` | Integration Content, Security Content, Partner Directory, Message Stores, Log Files, Message Processing Logs, **B2B Scenarios** | B2B Scenarios is B2B monitoring (business documents, interchanges, reprocessing), confirmed by the tenant `$metadata` |
| `APIMgmt` | 29 APIs, among them API Portal API Proxy, API Provider, Product, Key Value Maps, Generic Key Value Maps, Certificate Store Reference, KeyStore, TrustStore, Virtual Host Request, Rules, Access Control Service, Applications, Developer, Endpoint (all OData, CF), **Transport (CF)** and **Content Archive Transport (CF)** (REST, zip import and export), Analytics (OData V4) | The transport APIs are the official route for proxy content; the other CF APIs are to be checked against `Management.svc/$metadata` |
| `APIMgmt` | `Graph_ConfigurationAPI` | API Composition's Configuration API is OData V4 |
| `SAPIntegrationAssessment` | `EntitiesAPI`, `ManagementAPI` | Both OData |
| `dataspaceintegration` | `DSIAPI` 2.0.0 | REST |
| `com.sap.integration.dsi` | — | Integration content (flows, scripts) for Data Space Integration, no API |
| `ICAPrepackagedContent` | — | Integration Advisor EDI templates, integration content, no API |

No package in the full list covers OData Provisioning, Migration Assessment, Trading Partner
Management configuration (profiles, agreements) or Integration Advisor's design-time content.

**Archiving** was found through Trading Partner Management's documentation and was missing from
the catalog: the tenant `$metadata` has `activateArchivingConfiguration` and
`activateB2BArchivingConfiguration` (POST, no parameters), read-only `ArchivingConfigurations`
and `B2BArchivingConfigurations` (`Id`, `Active`), and KPI entity sets. There is no
deactivation, and per-flow settings are UI-only; see the catalog entry
`cloud_integration.archiving`.

## Deferred APIs (tracked, not yet implemented)

| Area | API | Status |
|---|---|---|
| API Proxy content upload | `Management.svc/APIProxies` (Classic API Management) | Entity, GET, and DELETE confirmed; the ZIP content Create/Update wire format is not — see `docs/guides/classic-api-management.md` |
| Certificate Chain, Secure Parameter, Known Hosts | Security Content API | Reverified during the Security Content phase; remain without a confirmed public contract (Certificate Chain is documented only as a Key Pair capability) or without any public API at all (Known Hosts) — see `docs/guides/security-content.md` |
| Value mapping entry-level management | `UpsertValMaps`, `UpdateDefaultValMap`, `DeleteValMaps` | Confirmed public, deferred — exact payload/path shapes and delete granularity not confirmed against a reachable primary source; see `docs/resource-design.md` |
| Integration Assessment | `EntitiesAPI`/`ManagementAPI`, both OData (see above) | Confirmed public with a fully confirmed entity inventory; no field-level wire contract confirmed for any entity. Unblocked by the services' `$metadata`, fetched with a service key — see `docs/guides/integration-assessment.md` |

## Explicitly ruled out

- **Integration Suite capability activation** (Cloud Integration, API Management, Integration
  Cell, Edge Integration Cell): no public activation API found. SAP's own documentation
  describes activation as a UI action ("Settings" → "Runtime" → *Activate*). Not implemented;
  see `docs/provisioning-capability-matrix.md`.
- **Current API Management / API Artifacts / Integration Cell content** (API Artifacts,
  deployment, Policies, Reusable API Artifacts, Runtime Profiles, Integration Cell Virtual
  Hosts): reverified thoroughly, not merely re-asserted — see the dedicated section above and
  `docs/guides/current-api-management.md`. No public API found for any of them despite an
  exhaustive documentation sweep and an explicit cross-check against the Integration Content
  API's own resource table.
- Any endpoint only reachable by reverse-engineering the Integration Suite UI's network
  traffic. None were used, and none will be, regardless of how convenient they would be.
