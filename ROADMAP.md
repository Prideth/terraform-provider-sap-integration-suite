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

## Next release: 0.3.0

0.3.0 is planned around Integration Assessment (P1 item 1) together with what is already on
`dev` and the current feature branch: API discovery, the experimental Classic API proxy, the
contract source of every feature and the opt-in switches. It is cut from `dev` once the
Integration Assessment landscape resources are verified or explicitly deferred. `v1.0.0` is a
separate, explicit decision and not the automatic successor of any 0.x release.

## P0 — tenant runs that decide promotions

These need no further research, only a run of the prepared tests:

| Item | Test | Decides |
|---|---|---|
| Classic API proxy | `apim-probe -ProxyTests`, then `TestAccAPIProxy_sample` | Why the import answers `APIPROXY_ZIP_ERROR` (the tenant's own export shows the expected layout); then `sapintegrationsuite_api_proxy` from experimental to supported (replace-only) |
| Live contract check | `TestAccMetadata` (`SAP_INTEGRATION_SUITE_ACC_METADATA`) | First comparison of the committed snapshots with a tenant, including the service documents |
| Deployments with longer timeouts | message mapping and value mapping acceptance tests | Whether slow deployments need more than the documented timeout advice |

## P1 — public contract, next implementation

1. **Integration Assessment landscape.** The Entities and Management `$metadata` are committed as
   snapshots and classified (27 entity sets). A tenant test created, read and deleted vendors and
   applications (2026-09-27). Next: the same test with updates sent as plain `application/json`
   and links as `{"Id": ...}`, plus technologies and technology instances. If updates and links
   work, the landscape objects become resources and SAP's taxonomy (technology domains, styles,
   key characteristics, deployment models) becomes data sources. Requests and assessment
   results stay out of scope as workflow.
2. **API Composition hardening.** `TestAccBusinessDataGraph_basic` needs a service key of plan
   `configuration` and a destination; the Configuration API's `$metadata` goes through
   `cmd/apidiscovery`. Passing both promotes the business data graph from experimental and
   settles the PATCH body and the delete request, which may then leave `enable_unofficial`.
3. **Edge Integration Cell targeting.** `TestAccEdgeIntegrationCell_securityAndPartnerDirectory`
   needs a tenant with an Edge Integration Cell. Passing it makes `runtime_location_id`
   supported for the tested resources; the deployment resources follow with their own test.
4. **Data types, message types, fault message types, service interfaces.** The `$metadata`
   defines all four with `SaveAsVersion`. For data types, reading, a create without content and
   `SaveAsVersion` work; every request with content failed with 500 ("map is null"). The next
   probe sends the content the API's own `$value` returns for an existing data type. If content
   can be written, the four resources reuse the shared design-time client.

## P2 — partly confirmed, more evidence first

5. **Security Content.** Certificate chains (upload media type), PGP keyrings (upload format;
   secret keyrings write-only) and OAuth2 client credential custom parameters (write path). The
   OAuth2 password and SAML bearer artifacts become user credential kinds if a probe finds them
   among `UserCredentials`.
6. **Classic virtual hosts.** The write API (`Configuration.svc/VirtualHostRequests`) is
   documented and the read schema confirmed; needs a key with
   `APIManagement.SelfService.Administrator` and a destructive-gated tenant test, since virtual
   hosts are tenant-wide.
7. **Data Space Integration.** Parse the DSIAPI OpenAPI document (`cmd/apidiscovery -spec-dir`)
   once it is available, then model only desired-state configuration (assets, policies,
   contract definitions); negotiations, agreements and transfers stay out of scope.
8. **Unofficial features.** `sapintegrationsuite_secure_parameter`, the access policy runtime
   assignments and the undocumented operations listed in Feature Support become supported only
   if SAP documents them. The weekly discovery run and the Hub checks watch for that.

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
