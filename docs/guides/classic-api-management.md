---
page_title: "Classic API Management"
subcategory: "API Management"
description: |-
  API Providers, API Proxies (experimental), API Products, Key Value Maps, and Certificate Store
  References for Classic API Management — what this provider manages, what it deliberately does
  not, and exactly what evidence every decision rests on.
---

# Classic API Management

Classic API Management is SAP Integration Suite's original API gateway product: API Providers
(backend connections), API Proxies (the deployable gateway artifact, with its own mediation
policy engine), API Products (subscribable bundles of proxies), and Key Value Maps (runtime
configuration lookups), managed through the **API Portal**'s own `Management.svc` OData API. It
is a distinct product model from `docs/guides/current-api-management.md` (API Artifacts,
Integration Cell) — the two are never mixed in this provider's terminology, code, or credentials,
even where their concepts sound similar.

## Two credential sets, never shared

Classic API Management authenticates against the API Portal application, using the
`apiportal-apiaccess` service plan — a completely separate OAuth 2.0 client from the one that
authenticates Cloud Integration and Current API Management calls. Configure it with its own
provider block:

```hcl
provider "sapintegrationsuite" {
  host = var.integration_suite_host

  oauth {
    token_url     = var.integration_suite_token_url
    client_id     = var.integration_suite_client_id
    client_secret = var.integration_suite_client_secret
  }

  api_management {
    host          = var.api_management_host
    token_url     = var.api_management_token_url
    client_id     = var.api_management_client_id
    client_secret = var.api_management_client_secret
  }
}
```

Or the equivalent `SAP_INTEGRATION_SUITE_API_MANAGEMENT_HOST` / `_TOKEN_URL` / `_CLIENT_ID` /
`_CLIENT_SECRET` environment variables. Both `oauth` and `api_management` are optional and
independent: a configuration that never touches Classic API Management resources works exactly
as before with `api_management` left out entirely, and every Classic API Management resource and
data source fails with a specific, actionable error (not a generic authentication failure) if
`api_management` is absent or only partially configured.

To obtain these credentials: create a service instance of the **API Management, API portal**
plan `apiportal-apiaccess` in the SAP BTP Cockpit (Cloud Foundry environment), assign it the
`APIPortal.Administrator` role (or `APIPortal.Guest` for read-only access), and generate a
service key — its `url`, `tokenUrl`, `clientId`, and `clientSecret` fields map directly to the
four `api_management` attributes above. See SAP's own "Accessing API Management APIs
Programmatically" documentation for the full procedure; this provider does not automate creating
that service instance or key, the same boundary it already applies to every other credential set
it consumes.

## What this provider manages today

| Object | Resource / Data Source | Lifecycle |
|---|---|---|
| API Provider | `sapintegrationsuite_api_provider` | Create, Read, Delete — no Update |
| API Product | `sapintegrationsuite_api_product` | Create, Read, Delete — no Update |
| Certificate Store Reference | `sapintegrationsuite_api_management_certificate_store_reference` | Full CRUD |
| Key Value Map | `sapintegrationsuite_api_key_value_map` | Create, Read, Delete — no Update, unencrypted only |

Every one of these is confirmed field-for-field against SAP's own official "SAP API Management
Standalone Service" user guide (its worked Create/Update/Delete request and response bodies),
not inferred from UI screenshots or guessed from field labels — see `docs/sap-api-references.md`
for the full evidence trail. Where a tenant test in September 2026 showed that the service
behaves differently from that guide, as it does for API products, the provider follows the
tenant.

### API Provider

The backend connection definition an API Proxy targets — host, port, SSL/authentication
settings.

```hcl
resource "sapintegrationsuite_api_provider" "backend" {
  name      = "ES5_1"
  title     = "ES5"
  host      = "sapes5.sapdevcenter.com"
  port      = 443
  use_ssl   = true
  trust_all = false

  auth_type   = "BASIC"
  user_name   = "backend-user"
  password_wo = var.backend_password
}
```

**Only the "Internet" connection type is supported** (a direct host/port connection). SAP
documents three further connection types — On Premise (via Cloud Connector), Open Connectors,
and Cloud Integration — each with its own field set, described in SAP's UI documentation in
enough detail to understand the workflow, but without a confirmed field-level JSON mapping this
provider could find in any reachable primary source. Configure those connection types through
the SAP Integration Suite UI instead.

**No Update exists.** SAP's own official Piper `apiProviderUpload` pipeline step — SAP's tooling,
not a third party — documents that this API "only supports create operation" and that an existing
provider must be deleted and recreated to change anything. Every attribute on
`sapintegrationsuite_api_provider` is therefore `RequiresReplace`: this provider does not invent
an unconfirmed PUT/PATCH to make this resource feel more complete than the underlying API
actually is.

**Eventual consistency.** SAP's documentation states, verbatim: "When you create, update, or
delete an API provider using the API, changes may not be immediately reflected in subsequent GET
API provider requests. API provider data is cached to reduce calls to the destination service,
which can result in stale data being returned for a short period after a modification. The
updated data is typically available within approximately 20 seconds." This resource's Create
polls with bounded, jittered exponential backoff (never a fixed `sleep(20s)`) until the new
provider becomes visible, so a chained `apply` referencing it right away does not race this
window — but a `terraform plan` refresh or `terraform import` run moments after a manual change
elsewhere may still occasionally need a retry.

### API Product

A bundle of API proxies that application developers subscribe to, with optional custom
attributes and request quotas.

```hcl
resource "sapintegrationsuite_api_product" "sample" {
  name        = "SampleProduct"
  title       = "Sample Product"
  status_code = "PUBLISHED"

  api_proxy_names = ["SampleAPI"]

  additional_properties = [
    { name = "team", value = "integration-platform" },
  ]
}
```

**A product cannot be changed, only replaced.** On a tenant in September 2026, SAP answered
`PUT`, `PATCH` and `MERGE` on an existing product with `405 UPDATE operation not supported on
APIProduct entity`. SAP's older user guide still shows a `PUT` example, but the service no
longer accepts it. Every argument of `sapintegrationsuite_api_product` therefore forces a new
product: Terraform deletes the old one and creates the new one. Applications subscribed to the
old product lose that subscription, so read a plan that replaces a product before you apply
it, and consider `lifecycle { prevent_destroy = true }` for products with subscribers.

**At least one proxy is required.** SAP refuses a product without a linked proxy ("At least one
API Proxy should be linked to an API Product"). The proxies must already exist: manage them with
`sapintegrationsuite_api_proxy` (experimental, see below) or create them in the SAP Integration
Suite UI.

**Status.** SAP needs a `status_code` on create; without one its create fails with an internal
error. The provider sends `PUBLISHED` unless you set something else. `DRAFT` also works and
creates an unpublished product (`is_published` comes back `false`). Leave `is_published` out
unless you have a reason to set it: SAP derives it from the status.

**Additional properties** are sent inside the create request. SAP does not accept them on their
own (`405 CREATE operation not supported on APIProductAdditionalProperty entity`), and each one
must carry the product's name as `entityId`, which the provider fills in for you. SAP documents
limits of 255 characters for a name, 1024 for a value, and 18 attributes per product; the
provider does not check those itself, since SAP may change them.

### API Proxy (experimental)

The deployable gateway artifact — proxy and target endpoints, mediation policies, resources —
managed as the ZIP bundle the API portal exports, in the same spirit as
`sapintegrationsuite_integration_flow`: opaque content tracked by its hash, not every field of
the proxy XML as Terraform attributes.

```hcl
resource "sapintegrationsuite_api_proxy" "orders" {
  name         = "Orders_v1"
  content      = "${path.module}/proxies/Orders_v1.zip"
  content_hash = filesha256("${path.module}/proxies/Orders_v1.zip")
}
```

**How it talks to SAP.** The provider uploads the bundle exactly the way SAP's own API Management
Client SDK 3.0.6 does: a `POST` to `/apiportal/api/1.0/Transport.svc/APIProxies` with the ZIP as
raw `application/octet-stream` bytes. It then reads the proxy from
`Management.svc/APIProxies('<name>')`, waiting until the API portal returns it, and deletes it
there. The Business Accelerator Hub lists this Transport API ("API Portal - Transport (CF)",
"Export and Import API Proxy via zip bundle") as an official API; its specification is behind an
SAP login, which is why the SDK is the reference.

**Why it is experimental.** The lifecycle has not yet passed an acceptance test on a tenant.
SAP's documentation says a proxy imported this way is deployed by default, and `state` shows
what SAP reports, but nothing public says whether importing a changed bundle over an existing
proxy replaces it cleanly. The resource therefore never does that: **a new `content_hash`
deletes the proxy and imports the new bundle.** The proxy is unavailable in between, and a
product that includes it loses the link, so read such plans before applying them.

**The bundle names the proxy.** SAP takes the name from the bundle's `APIProxy/<name>.xml`
descriptor. The provider reads it before uploading and stops with an error if it differs from
`name`, so a copied bundle cannot silently create a proxy under another name. The base path in
`APIProxyEndPoint/*.xml` must be free on the virtual host.

**Dependencies.** A target endpoint that references an API provider needs that provider on the
tenant first (`depends_on` a `sapintegrationsuite_api_provider`); SAP's sample repository
documents that the import fails otherwise. A bundle with a target URL (`provider_id` `NONE`)
has no such dependency.

**Import.** `terraform import sapintegrationsuite_api_proxy.orders Orders_v1`. SAP does not return
the bundle file, so the first apply after an import only records `content` and `content_hash`;
the provider assumes the file matches the proxy on the tenant. Any later hash change replaces it
as described above.

### Certificate Store Reference

A named pointer to an already-existing keystore or truststore, used so a virtual host's TLS
configuration can be repointed at a new store — for certificate rotation — without editing the
virtual host itself.

```hcl
resource "sapintegrationsuite_api_management_certificate_store_reference" "backend_trust" {
  name                   = "backend-trust-reference"
  certificate_store_name = "backend-truststore"
}
```

This is the best-confirmed resource in this whole phase: SAP's official documentation shows
complete, verbatim Create, Read, Update, and Delete request/response bodies and error codes for
this exact entity. It does not create the keystore or truststore itself, or upload any
certificate material — SAP documents that as a UI-only operation ("Upload the PKCS12/PFX file")
with no accompanying REST API anywhere found, so `certificate_store_name` must reference a store
that already exists. This is a distinct remote object from Cloud Integration's
`sapintegrationsuite_certificate` / `sapintegrationsuite_key_pair`, which manage actual keystore
entries under a completely different API and service — the two are never conflated, even though
both involve TLS material.

### Key Value Map

A named collection of key/value pairs scoped to an API proxy, environment, or organization,
readable at runtime through the Key Value Map Operations policy.

```hcl
resource "sapintegrationsuite_api_key_value_map" "oc_instance_token" {
  name     = "apim.oc.instance.token"
  scope    = "APIPROXY"
  scope_id = "SampleAPI"

  entries = [
    { key = "default", value = var.open_connectors_instance_token },
  ]
}
```

**No Update exists (as implemented here), and entries are RequiresReplace.** SAP's documentation
confirms a full Create/Update-entries/Delete UI lifecycle for Key Value Maps, but only Create's
REST payload is shown verbatim anywhere reachable — this provider does not guess at an
unconfirmed per-entry `PUT`/`POST`/`DELETE` contract, so changing anything about a map's entries
replaces the whole resource.

**Encrypted maps are not supported.** SAP's `isEncrypted` field is real and documented (a
per-map checkbox in the UI), but this provider could not confirm whether `GET` returns an
encrypted entry's plaintext value back, a masked placeholder, or nothing at all — and guessing
wrong would either leak a secret into Terraform state or produce permanent diffs. This resource
always sends `isEncrypted = false` and its `ValidateConfig` rejects `encrypted = true` outright.
Store secret values through unencrypted maps only until SAP's encrypted-entry read-back behavior
is confirmed, or manage them outside Terraform.

## What this provider deliberately does not manage, in this phase

### API Proxy Deployment

A separate deployment resource, like `sapintegrationsuite_integration_flow_deployment`, has no
documented API to build on: SAP documents that a proxy transported or exported, individually or
as part of a product, "by default gets imported to the target in the deployed state", and no
public call deploys or undeploys an existing proxy on its own. `sapintegrationsuite_api_proxy`
deploys by importing and shows the resulting `state`.

### Policy

Individual mediation policies (`VerifyAPIKey`, `Quota`, `AssignMessage`, and the 30-plus others
SAP documents) are confirmed to be XML content embedded inside the API Proxy ZIP bundle itself —
a `<policies>` element in the proxy's root XML referencing named files under a `Policy/` folder —
not an independently addressable OData entity with its own Create/Read/Update/Delete. There is no
`sapintegrationsuite_api_proxy_policy` resource candidate here: policies would be managed as part
of the proxy's own opaque content, exactly like Cloud Integration's design-time artifacts, as
`sapintegrationsuite_api_proxy` does. This provider does not attempt to reproduce SAP's
entire policy schema catalog as nested Terraform blocks.

### Virtual Hosts

Classic virtual hosts, the default-domain aliases and custom domains under which proxies are
reachable, have a documented API. It works with requests rather than entities: every change is a
`POST` to `/apiportal/operations/1.0/Configuration.svc/VirtualHostRequests` with an `operation`
of `CREATE`, `UPDATE` or `DELETE`, the subaccount subdomain as `accountId`, the alias or custom
domain as `virtualHostUrl`, and for custom domains or mutual TLS the keystore, key alias and
truststore. It needs a service key with the `APIManagement.SelfService.Administrator` role, which
is a different role from the `APIPortal.Administrator` the other Classic resources use.

What is missing is the read side. SAP only says that `Management.svc/VirtualHosts` returns the
`virtualHostId`, and documents none of that entity's other fields. Without them the provider
could create a virtual host but never notice when it changes or disappears, so it waits for the
entity's schema (`Management.svc/$metadata` from an API Portal tenant would settle it). Keep in
mind that SAP refuses to delete a virtual host while any proxy, deployed, draft or in an old
revision, still references it, or while it is the default.

### Monetization, Rate Plans, and analytics

Not evaluated in this phase. Classic API Management's monetization/rate-plan features and its
API analytics dashboards are operational/reporting concerns rather than desired-state
configuration by their nature — the same category this provider already excludes for Cloud
Integration's Message Processing Logs and Edge Integration Cell's local monitoring APIs (see
`docs/provider-scope.md`) — and were not researched further given that established precedent.

## Developer Hub stays a separate concern

Classic API Management's own documentation occasionally cross-references Developer Hub
Application and Subscription entities (for example custom-attribute examples that happen to use
a Developer Hub endpoint). None of that is implemented here: Developer Hub is out of this
provider's scope entirely, planned as its own separate Terraform provider — see
`docs/provider-scope.md`.

## Import and drift

`sapintegrationsuite_api_product` and
`sapintegrationsuite_api_management_certificate_store_reference` support `terraform import` by
name (`terraform import sapintegrationsuite_api_product.sample SampleProduct`).
`sapintegrationsuite_api_provider` also supports import (its `id` is its `name`), but every
subsequent change to an imported provider replaces it, per the no-Update limitation above. The
same holds for an imported API product: import reads its linked proxies and additional
properties from SAP, so the configuration has to match them exactly, or the next apply
replaces the product.
`sapintegrationsuite_api_key_value_map` imports via its composite key,
`"<name>/<scope>/<scope_id>"` (`terraform import sapintegrationsuite_api_key_value_map.oc_instance_token apim.oc.instance.token/APIPROXY/SampleAPI`).

Drift detection reads back every attribute this provider writes, except: `password_wo` on
`sapintegrationsuite_api_provider` (write-only by design, never returned by `GET`, and this
resource has no Update to reconcile it against anyway). For `sapintegrationsuite_api_product`,
SAP's product response only links to the proxies and properties, so the provider reads them
through `APIProducts('<name>')/apiProxies` and `APIProducts('<name>')/additionalProperties`. A
proxy order that differs from your configuration is not reported as a change.
