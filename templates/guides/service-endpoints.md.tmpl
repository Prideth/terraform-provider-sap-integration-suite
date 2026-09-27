---
page_title: "Service Endpoints"
subcategory: "Cloud Integration"
description: |-
  What SAP's ServiceEndpoints discovery API is, why it has no matching Terraform resource, and
  how to use it alongside integration flow deployments.
---

# Service Endpoints

SAP Cloud Integration's **ServiceEndpoints** OData V2 API lets you discover the runtime URLs and
API definition links SAP has generated for the content currently deployed on a tenant. This
guide covers what [`data.sapintegrationsuite_service_endpoints`](../data-sources/service_endpoints.md)
does, why it is read-only by design, and the dependency pattern this provider expects you to use
alongside it.

## Why there is no `sapintegrationsuite_service_endpoint` resource

A Terraform resource needs something to create, update, and destroy. Service endpoints have none
of that: SAP generates them automatically the moment matching content (an integration flow, an
OData API artifact, and so on) is deployed, and removes them automatically when that content is
undeployed. There is no `POST`/`PUT`/`DELETE` operation for a service endpoint in SAP's public
API — only `GET`. Modeling this as a resource would mean inventing a fake lifecycle this
provider cannot actually implement, so it is a **read-only discovery data source**, matching the
Terraform Plugin Framework's own guidance for wrapping a read-only external data source.

The desired state that actually produces a service endpoint lives elsewhere, in the resources you
already use to deploy content:

```hcl
resource "sapintegrationsuite_integration_flow" "orders" {
  # ...
}

resource "sapintegrationsuite_integration_flow_deployment" "orders" {
  package_id   = sapintegrationsuite_integration_package.utilities.id
  flow_id      = sapintegrationsuite_integration_flow.orders.flow_id
  flow_version = sapintegrationsuite_integration_flow.orders.version
}
```

`data.sapintegrationsuite_service_endpoints` then discovers what that deployment actually exposed
at runtime — the entry point URLs, and links to any generated API definition documents.

## Why there is no single-endpoint data source either

SAP's own example requests filter `ServiceEndpoints` by `Name`, but this project could not
confirm that an integration artifact's name is guaranteed to identify at most one service
endpoint in every situation (for example, an artifact exposed through more than one adapter/
protocol at once). Rather than ship a `data.sapintegrationsuite_service_endpoint` singular
lookup that silently returns an arbitrary result if more than one match ever exists, this
provider only implements the collection data source, `sapintegrationsuite_service_endpoints`,
with `name` and `protocol` as optional filters. Filtering by `name` in practice usually narrows
the result to the single endpoint you expect — you just read `endpoints[0]` (or iterate) instead
of getting a single object back directly.

## Using it alongside a deployment

Because `data.sapintegrationsuite_service_endpoints` filters by `name`, not by a direct
Terraform reference to a deployment resource, Terraform has no automatic way to know it should
wait for a specific deployment before reading endpoints. Make that dependency explicit with
`depends_on`:

```hcl
data "sapintegrationsuite_service_endpoints" "orders" {
  name = "Order API"

  depends_on = [
    sapintegrationsuite_integration_flow_deployment.orders,
  ]
}
```

This provider does not attempt to infer this dependency automatically (for example by matching
`name` against a `sapintegrationsuite_integration_flow_deployment`'s configuration) — that kind
of implicit cross-resource guessing is exactly the sort of hidden behavior this provider avoids
elsewhere too. Declare it explicitly.

## Eventual consistency after a fresh deployment

This project could not confirm, from SAP's public documentation, a guaranteed delay (or the
absence of one) between an integration flow's deployment completing and its service endpoint
becoming visible through this API. If you see a deployment succeed but
`data.sapintegrationsuite_service_endpoints` return no matching entry immediately afterward,
re-running `terraform plan`/`apply` after a short wait is the practical mitigation. This provider
deliberately does not paper over that gap with a blind, fixed-duration retry loop inside the data
source itself: doing so would either be too short to help on a slow tenant or add pointless
latency to a request that succeeds instantly, and it would mask a genuine error (for example a
misspelled `name` filter) as if it were just eventual-consistency lag. If SAP's documentation is
found to confirm a specific, bounded propagation window in the future, a bounded, context-aware
retry could be added here.

## What this data source will never do

- It never invokes a discovered endpoint URL, and never tests connectivity to one.
- It never downloads the document a returned `api_definitions[].url` points to (WSDL, OpenAPI,
  EDMX, RAML). You get the link, not the content.
- Reading it never deploys, redeploys, or otherwise changes any runtime state — it only issues
  `GET` requests.

## Protocol values

SAP returns the protocol exactly as its own adapter-to-protocol mapping produces it, and this
provider passes that value through unchanged rather than reverse-mapping it back to an adapter
name (a SOAP adapter and an IDoc adapter, for example, both report `SOAP`, so the mapping is not
reversible in general). Values currently documented by SAP:

| Adapter | `protocol` value |
|---|---|
| SOAP | `SOAP` |
| IDoc | `SOAP` |
| OData V2 | `ODATAV2` |
| AS2 | `AS2` |
| AS4 | `AS4` |
| HTTPS | `REST` |

## Deterministic ordering

SAP does not document a guaranteed order for the `ServiceEndpoints` collection, or for the
`EntryPoints`/`ApiDefinitions` navigation properties nested inside each entry. This provider
sorts the top-level list by `name` then `protocol`, and each entry's `entry_points`/
`api_definitions` by their own fields, before writing Terraform state — so `terraform plan`
never reports a spurious diff caused purely by SAP returning the same data in a different order
between applies.
