# Roadmap

This roadmap lists open work, ordered by what unblocks it. What the provider already does is in
[`docs/feature-support.md`](docs/feature-support.md) (generated from the feature catalog) and
in [`CHANGELOG.md`](CHANGELOG.md); the dated evidence behind every feature that is not fully
supported is in [`docs/research/capability-evidence-2026.md`](docs/research/capability-evidence-2026.md).

Nothing moves up because an entity merely exists in a `$metadata` document: that shows the
entity, not that SAP supports writing it. A feature whose only contract is `$metadata` can at
most become `unofficial`, and stays behind `enable_unofficial`.

Priorities:

- **P0**: blocked only on a tenant run that is already prepared.
- **P1**: a public contract exists; needs a tenant check or a service key, then implementation.
- **P2**: a public contract is partly confirmed; needs more evidence first.
- **P3**: valuable but low demand or high effort.
- **WATCH**: no public API today; revisit when SAP publishes one.

## Release plan

Each 0.x minor release has one theme. Released so far: 0.1.0 (the broad first provider), 0.2.0
(the tenant-verified rework), 0.3.0 (contract sources, opt-in switches, Registry documentation)
with the patches 0.3.1 and 0.3.2, and 0.4.0 (the Integration Assessment landscape). Each minor
line has one release branch, `release/<minor>.x`, and every release is a tag on it: 0.5.0 is
prepared on `release/0.5.x`, 0.6.0 on `release/0.6.x`.

| Version | Theme | State | Precondition |
|---|---|---|---|
| 0.5.0 | Integration Assessment technology profiles and the ISA-M taxonomy | prepared on `release/0.5.x` | — |
| 0.6.0 | API Composition: business data graph hardening | prepared on `release/0.6.x`; every change is tenant-verified | releasing 0.5.0 first |
| 0.7.0 | Cloud Integration data types, message types, fault message types, service interfaces | in progress on `feature/design-time-types`; data types, message types and fault message types done (unofficial); service interfaces implemented (experimental) | the acceptance test of service interfaces, including a synchronous operation |
| 0.8.0 | Security Content completion | proposed | the upload formats in P2 item 5 |
| 0.9.0 | Classic API Management infrastructure, above all virtual hosts; the API proxy only if the official Transport API request passes its acceptance test | proposed | P2 item 6 and the Transport API specification |
| 0.10.0 | Edge Integration Cell targeting | proposed | a tenant with an Edge Integration Cell (P1 item 3) |
| 0.11.0 | Data Space Integration desired-state configuration | proposed | the DSIAPI specification (P2 item 7) |

The order follows what is closest to a tenant-verified result, not importance, and changes when a
precondition is met earlier. Current API Management (API artifacts, MCP servers, runtime
profiles) stays on WATCH until SAP publishes a supported design-time management API. Developer
Hub belongs to a separate provider. `v1.0.0` is a separate, explicit decision and not the
automatic successor of any 0.x release.

## P0 — tenant runs that decide promotions

These need no further research, only a run of the prepared tests:

| Item | Test | Decides |
|---|---|---|
| Classic API proxy | the official Transport API specification, then `TestAccAPIProxy_sample` | Even the tenant's own export, renamed, answered `APIPROXY_ZIP_ERROR`, so the import request is the problem, not the bundle. The resource is implemented but not registered; the specification (`APIPortal_Transport_CF`, needs an SAP login) shows the documented request, after which the resource is registered as experimental |
| Live contract check | `TestAccMetadata` (`SAP_INTEGRATION_SUITE_ACC_METADATA`) | First comparison of the committed snapshots with a tenant, including the service documents |
| Deployments with longer timeouts | message mapping and value mapping acceptance tests | Whether slow deployments need more than the documented timeout advice |

## P1 — public contract, next implementation

1. **Integration Assessment.** The landscape (0.4), the technology profiles and the complete
   ISA-M taxonomy (0.5) exist, all unofficial. `TestAccIntegrationAssessment_technologyProfile`
   and `TestAccIntegrationAssessment_taxonomy` passed on 2026-09-29. The Hub specification of
   the Entities API would make them supported; it still needs an SAP login. Requests and
   assessment results stay out of scope as workflow.
2. **API Composition hardening.** A key user's token (password grant through the
   `configuration` client) reads the Configuration API; a client-credentials token got 403. The
   provider has that login since `feature/api-composition-hardening`, `TestAccMetadata` passes
   with it, and the client is checked against the `$metadata` snapshot.
   `TestAccBusinessDataGraph_basic` passed on 2026-10-03, so the business data graph is
   supported; its update body and delete request work but stay behind `enable_unofficial`,
   because SAP does not document them. The settings the `$metadata` adds are settable behind
   `enable_unofficial`, and all of them, cue-scoped key mappings included, passed tenant tests
   on 2026-10-03. Nothing is open for 0.6.0.
3. **Edge Integration Cell targeting.** `TestAccEdgeIntegrationCell_securityAndPartnerDirectory`
   needs a tenant with an Edge Integration Cell. Passing it makes `runtime_location_id`
   supported for the tested resources; the deployment resources follow with their own test.
4. **Data types, message types, fault message types, service interfaces.** The `$metadata`
   defines all four with `SaveAsVersion`. For data types (September 2026): reading, deleting and
   `SaveAsVersion` work, and so does a create with content once the bundle carries
   `additionalAttributes.json` and `metainfo.prop` as SAP's own `$value` does. A media `PUT`
   on `$value` answers 501. SAP rewrites the XSD's type name to the artifact's name, keeps the
   elements of a type on create and update (probe, 2026-10-03), and takes description and
   namespace from the request body. `sapintegrationsuite_data_type` is implemented and passed its
   acceptance test, so it is unofficial. Message types and fault message types need no content:
   SAP generates their schema from `DataTypeUsed`; both are implemented and unofficial. Next:
   service interfaces, whose bundle is a JSON operation model in nested archives.

## P2 — partly confirmed, more evidence first

5. **Security Content.** Certificate chains (upload media type), PGP keyrings (upload format;
   secret keyrings write-only) and OAuth2 client credential custom parameters (write path). The
   OAuth2 password and SAML bearer artifacts become user credential kinds if a probe finds them
   among `UserCredentials`. Access policy string limits are settled: a probe found that SAP
   shortens the role name and descriptions to 200, a reference name to 50 and a reference value
   to 150 characters, and 0.3.2 checks all of them at plan time.
6. **Classic virtual hosts.** The write API (`Configuration.svc/VirtualHostRequests`) is
   documented and the read schema confirmed; needs a key with
   `APIManagement.SelfService.Administrator` and a destructive-gated tenant test, since virtual
   hosts are tenant-wide.
7. **Data Space Integration.** Parse the DSIAPI OpenAPI document (`cmd/apidiscovery -spec-dir`)
   once it is available, then model only desired-state configuration (assets, policies,
   contract definitions); negotiations, agreements and transfers stay out of scope.
8. **Unofficial features.** `sapintegrationsuite_secure_parameter`, the access policy runtime
   assignments, the eight access policy reference types SAP accepts but does not document
   (credentials, secure parameters, adapters, service interfaces, fault message types) and the
   undocumented operations listed in Feature Support become supported only if SAP documents them.
   The weekly discovery run and the Hub checks watch for that.

## P3 — later

9. Value mapping in-place update and `save_as_version` (needs a verified PUT).
10. Integration adapters tested with real adapter content (create and read are not documented
    by an example).
11. Classic certificate stores, applications, cache resources, rate plans, policy templates and
    product access control, each once SAP documents update and delete.
12. Discovery automation: the weekly strict discovery run fails on any change; it should open
    an issue with the semantic diff instead.

## WATCH — no public API today

- Current API Management: API artifacts, reusable API artifacts, API artifact deployment,
  policies, MCP servers and the MCP Gateway, runtime profiles; Integration Cell runtime and
  virtual hosts. Channels: the Hub's API Management and Cloud Integration packages, the Client
  SDK on Maven Central, What's New for Integrations and APIs. The design for an API artifact
  resource is in `docs/guides/current-api-management.md`.
- Value mapping entries: SAP's entry functions dropped values and deleted nothing in tenant
  tests (2026-09-27). Revisit when SAP documents an update and a delete that work.
- Capability activation for every Integration Suite capability.
- Known hosts, certificate-to-user mapping (Neo only today), where-used lists.
- OData Provisioning, Integration Advisor, Trading Partner Management configuration, Migration
  Assessment.

## Separate providers

- **Developer Hub**: its own API boundary and credentials; planned as a separate provider. MCP
  server products and subscriptions belong there.
- **Event Mesh**: a general BTP messaging service with its own broker APIs.

## Out of scope

- Runtime and monitoring data: message processing logs, message stores, JMS queues, data
  stores, variables, trace, logs, B2B interchange monitoring.
- Workflows and one-shot actions: Integration Assessment requests and results, Migration
  Assessment extraction and evaluation, Integration Advisor injection, agreement activation,
  restarts and retries.
- Edge Integration Cell local and Operations Cockpit APIs (runtime data and Kubernetes-level
  settings), OAuth2 authorization codes (interactive consent), Open Connectors.
- BTP control-plane objects (subaccounts, entitlements, subscriptions, destinations, role
  collections): use the `SAP/btp` provider.
