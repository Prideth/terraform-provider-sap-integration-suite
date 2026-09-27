---
page_title: "Current API Management and Integration Cell"
subcategory: "API Management"
description: |-
  SAP's API-centric model on Integration Cell (API artifacts, MCP servers, runtime profiles,
  virtual hosts, policies), what SAP exposes through public APIs as of September 2026, and what
  that means for a Terraform setup.
---

# Current API Management and Integration Cell

SAP Integration Suite runs two API management models side by side, and this provider keeps them
strictly apart:

- **Current API Management**: API artifacts and MCP servers designed inside integration
  packages and deployed to Integration Cell or Edge Integration Cell. SAP calls the approach
  *API-centric integration*, and it is where SAP's product work goes. This guide covers it.
- **Classic API Management**: API providers, API proxies, API products and key value maps in
  the API Portal, with their own runtime and their own public API. The provider already
  manages part of it; see the [Classic API Management guide](classic-api-management.md).

If you are looking for `sapintegrationsuite_api_artifact`, it does not exist, and the reason
is SAP's API surface, not a gap in this provider's backlog. As of September 2026 SAP has not
published a public API for any object in the current model. The rest of this guide explains
how that was established, what each object is, and how to structure a Terraform setup around
the parts that have to stay manual for now.

## Status by object

| Object | Public API | What this means for Terraform |
|---|---|---|
| API artifact (REST, SOAP, OData) | None | Create and version in the UI |
| Reusable API artifact | None | Same as API artifact |
| API artifact deployment | None | Deploy in the UI; runtime targeting is chosen there |
| Policies and mediation | None | Edited inside the artifact in the UI |
| MCP server | None | Create, configure and deploy in the UI |
| Runtime profile | None | Enable or disable under *Settings* > *Integrations* |
| Integration Cell activation | None | One-time bootstrap under *Settings* > *Runtime* |
| Integration Cell runtime status and configuration | None | *Monitor* > *Integrations and APIs* |
| Virtual host | None | *Monitor* > *Integrations and APIs* > *Virtual Host* |

"None" is a precise statement here. SAP exposes these operations only through the Integration
Suite UI, and this provider does not call the UI's internal endpoints. Those endpoints are
unversioned, undocumented and can change with any release, and a provider built on them
would break in ways nobody could anticipate.

## How the conclusion was reached

An earlier research pass reached the same verdict. Because SAP shipped a lot of new
functionality in 2026, the September 2026 re-audit did not rely on it and checked every
channel where a public API would show up:

1. **The Integration Content API.** If API artifacts were ordinary integration artifacts with
   a new type, they would appear in this API's resource table next to integration flows,
   value mappings and script collections. The current page (published 2026-07-10) lists no
   API artifact, MCP server, runtime profile or virtual host resource.
2. **SAP's API documentation index pages.** The *API Documentation* pages for Cloud
   Integration and API Management point to exactly two Business Accelerator Hub packages:
   `CloudIntegrationAPI`, and the Classic API Portal package `APIMgmt`.
3. **The lifecycle documentation.** Roughly sixty current Help pages cover creating, copying,
   versioning, reverting, deleting and deploying API artifacts, creating MCP servers from
   every supported source, virtual hosts, runtime profiles and Integration Cell activation.
   Every one is a UI procedure. None names an endpoint, an entity set or a Business
   Accelerator Hub page.
4. **SAP's API Management Client SDK.** Version 3.0.0 was announced on 2026-09-20, and 3.0.6
   is on Maven Central. It wraps only Classic API Portal endpoints
   (`/apiportal/api/1.0/Management.svc`, `ContentArchive.svc`): API proxies, products, key
   value maps, providers and policy templates. It has nothing for Integration Cell.
5. **SAP's own CI/CD tooling.** `SAP/cicd-actions-for-sap-integration-suite`, actively
   developed in 2026, automates integration packages, integration flows, the Partner
   Directory and access policies. It has no action for API artifacts or MCP servers.
6. **What's New.** Every 2026 API Management entry about Integration Cell describes UI
   functionality: API-centric integration, the MCP Gateway, reusable APIs, simplified artifact
   creation, trace data, remote MCP servers.
7. **The Business Accelerator Hub.** The hub's catalog service lists every package and its APIs
   without a login. On 2026-09-27 the APIs of all 173 packages related to Integration Suite,
   API Management, Edge Integration Cell, graphs and BTP (308 APIs in total) were searched for
   API management, API artifact, Integration Cell, runtime profile, virtual host, MCP, API
   deployment, API policy, reusable API, proxy, transport and gateway. The only matches are the
   Classic API Portal APIs (including Transport and Virtual Host Request), the Edge Integration
   Cell's local APIs and Event Mesh.
8. **A tenant's own `$metadata`.** The Cloud Integration OData service of a Cloud Foundry
   tenant (131 entity sets in the committed snapshot) has no entity type for API artifacts,
   MCP servers, virtual hosts or runtime profiles. Its `APIDefinitions` set holds the API
   definition links of service endpoints, not API artifacts. The API discovery
   (`cmd/apidiscovery`) compares that snapshot with a live tenant and reports any new entity
   set, so an API artifact entity would surface there first.
9. **How the objects move today.** API artifacts and MCP servers are content of an integration
   package: a package export lists them as artifacts of type `API`, and SAP's transport
   documentation moves them with the package (export and import, SAP Cloud Transport
   Management, CTS+). The documented package import (`POST IntegrationPackages` with the
   base64 `PackageContent`, optionally `?Overwrite=true`) is therefore the only public path by
   which they reach a tenant, and it treats the package as one opaque archive.

The only page whose title suggests otherwise, *Accessing API Management APIs
Programmatically*, is about the Classic API Portal's `apiportal-apiaccess` service plan.

## The objects, and how they would map to Terraform

Knowing the model helps with manual administration today, and it records the design this
provider would follow if SAP publishes an API.

```text
BTP subaccount
│
├── API Management capability        bootstrap, UI only
├── Integration Cell                 bootstrap, UI only
│   └── Virtual hosts                UI only
│
└── Integration package              sapintegrationsuite_integration_package
    ├── API artifact                 UI only
    │   ├── type: REST / SOAP / OData / reusable
    │   ├── policies and mediation
    │   ├── runtime profile: Integration Cell or Edge Integration Cell
    │   └── deployment → runtime + virtual host
    └── MCP server                   UI only
        ├── source: API artifact / HTTP endpoint / RFC / remote MCP server / Classic API proxy
        └── deployment → Integration Cell + virtual host
```

### API artifacts

An API artifact is the complete definition of an API: its endpoints, the policies that run on
each request, its security scheme and its backend. It lives inside an integration package, the
same container that holds integration flows. You can create one from a URL or OpenAPI
specification, from a BTP destination, from a discoverable system such as SAP S/4HANA Cloud,
or by referring to a Classic API provider.

The **API state** field (Alpha, Beta, Active) is a label for consumers and has nothing to do
with deployment. A future resource would keep it as a plain attribute and would never treat
`Active` as "deployed".

**Versioning** works differently from integration flows. Several versions of an API artifact
can coexist at design time and at runtime, only one version is active at a time, and
reverting makes an older version active again without deleting the newer ones. Terraform manages a single desired
state per resource, so a future `sapintegrationsuite_api_artifact` would most likely manage
the active version and treat older versions as history, similar to how this provider handles
integration flow versions.

**Reusable API artifacts** are internal building blocks: no external endpoint, callable only
through the API Direct adapter, and not allowed to call other reusable APIs. They are a variant
of the same object, so the intended design is a type attribute on the API artifact resource
rather than a second resource type.

### Policies

Authentication, quota, spike arrest, threat protection, transformations and external callouts
are configured as policies inside an API artifact. SAP Help gives them no lifecycle of their
own. If an API appears, the design depends on what it returns. Policies embedded in the
artifact definition would be an attribute of the artifact. Only independently addressable
policies with their own create and delete calls would justify a separate resource.

### MCP servers

MCP servers, new in 2026, expose APIs as tools for AI agents through the Model Context
Protocol. You can build one from a deployed API artifact, from any HTTP endpoint with an
OpenAPI specification, from an RFC-enabled SAP backend, from a remote MCP server (since
September 2026), or from a Classic API proxy. You then choose which operations become tools
(up to 30 per server since August 2026), configure authentication, and deploy the server to
Integration Cell. The same virtual host rules apply as for API artifacts.

Making an MCP server discoverable through a Developer Hub product is a Developer Hub concern.
That will belong to the separate Developer Hub provider, even if this provider gains MCP
server support.

### Runtime profiles

A runtime profile tells the designer which platform an artifact targets: Cloud Integration,
Integration Cell, Edge Integration Cell or a specific SAP Process Orchestration release. Once
an API artifact exists, its profile is fixed. The one exception is Edge Integration Cell,
where you may pick a different target Edge Integration Cell when editing or deploying. A
future resource would therefore mark the profile `RequiresReplace`, so moving an artifact from
Integration Cell to Edge Integration Cell would create a new artifact instead of silently
failing.

SAP's *Runtime Profiles* reference page still has no row for Integration Cell, even though
the artifact creation dialogs offer it. This guide records the inconsistency and does not try
to resolve it.

### Integration Cell

It helps to split "Integration Cell" into separate concerns, because an API could appear for
one without the others:

- **Activation** is a one-time step under *Settings* > *Runtime*. It is the kind of bootstrap
  step this provider leaves manual for every capability.
- **Runtime discovery and status** appear only in *Monitor* > *Integrations and APIs* with
  the Integration Cell runtime selected.
- **Runtime configuration**, such as trace log level for APIs (August 2026), is set in the
  same UI.
- **Deployment targeting** means choosing the runtime profile and virtual host at
  deployment time. It is part of each artifact's deployment, not a standalone object.

None of the four has a public API today.

### Virtual hosts

A virtual host is the host name under which deployed APIs and MCP servers are reachable. A
tenant starts with a default virtual host under SAP's default domain. An administrator
(`PI_Administrator`) can add more, for example to separate development and production
traffic. SAP caps a tenant at 11 virtual hosts and enforces a few other rules:

- The name `Default` is reserved, and the host alias is at most 22 characters (letters,
  digits and hyphens, not starting with a hyphen).
- The default virtual host cannot be deleted, and neither can a virtual host used by a
  deployed API. A host used only by undeployed APIs can be deleted. Those APIs fall back to
  the default host at their next deployment.

SAP also distinguishes the **design-time** virtual host, stored on the artifact, from the
**deployment-time** virtual host chosen in the deploy dialog. They can differ. An artifact
designed against `api-dev.example.com` and deployed with `api-prod.example.com` keeps showing
the development host in its design-time settings, while the endpoint in the monitor uses the
production host. A future Terraform model would keep the two in separate attributes on
separate resources (artifact and deployment) and would treat the default virtual host as
read-only rather than something `terraform destroy` could touch.

### BTP destinations

API artifacts and MCP servers can reach their backend through a BTP destination. Destinations
are subaccount objects and belong to the [SAP/btp](https://registry.terraform.io/providers/SAP/btp/latest)
provider (`btp_subaccount_destination` and related resources), never to this one. Two SAP
prerequisites are easy to miss when you manage destinations in code: Integration Cell only
sees destinations that carry the **label** `IntegrationCell.Include` with the value `true`,
and only the proxy types `Internet` and `OnPremise` are supported.

## What you can do with Terraform today

The current model's objects stay manual, but the surrounding setup does not have to be:

- **Integration packages** that contain your API artifacts and MCP servers can be managed
  with `sapintegrationsuite_integration_package`. Terraform then owns the package and its
  metadata, and your team adds the API content in the UI. Be careful with that split:
  destroying the package resource deletes the package on SAP's side, and SAP does not
  document that UI-created content inside it survives. Consider `prevent_destroy` on
  packages that hold hand-built API artifacts.
- **Access policies** can protect API artifacts. SAP added API artifacts to access policies
  in June 2025, and the artifact type appears as *API* in the UI. See the
  [Access Policies guide](access-policies.md) for how to find the wire constant for that type
  and attach a reference with `sapintegrationsuite_access_policy_reference`.
- **Destinations**, roles and role collections for the people and systems involved belong in
  the SAP/btp provider, next to this one.
- **Classic API Management** objects that an MCP server or API artifact builds on, such as an
  API provider or an API proxy exposed as an MCP server, can be managed with the Classic
  resources where this provider supports them.

The bootstrap steps (Integration Suite subscription, API Management capability, Integration
Cell activation, and at least one virtual host if you need more than the default) remain
manual, as they are for every capability this provider touches.

## When this changes

The verdict describes SAP's published API surface in September 2026, not a judgment about
the product. The channels to watch are the Integration Content API resource table, the
Business Accelerator Hub packages `CloudIntegrationAPI` and `APIMgmt`, the API Management
Client SDK release notes, and SAP's CI/CD tooling. If any of them starts to cover API
artifacts, MCP servers or virtual hosts, the model above is the design this provider would
implement. The feature catalog entries under `api_gateway.*` and `integration_cell.*`
(see the `sapintegrationsuite_provider_features` data source) track each object separately.
