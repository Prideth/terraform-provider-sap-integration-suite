---
page_title: "API Composition"
subcategory: "API Management"
description: |-
  Managing business data graphs through API Composition's Configuration API: credentials,
  the configuration model, asynchronous processing, and which parts of the API SAP documents.
---

# API Composition

API Composition, formerly called Graph, is a capability of API Management in SAP Integration Suite.
It combines the business systems of a landscape, such as S/4HANA, SAP Sales Cloud or custom
OData services, into one connected API. That API is a **business data graph**. Client
applications query it with one URL and one token, and API Composition decides which system
answers each request.

The provider manages the *configuration* of business data graphs through SAP's Configuration
API: which systems a graph uses, which system leads for which entity, and how keys translate
between systems. It does not call the graph's own data API. Querying data is the job of client
applications and is not configuration.

| Object | Resource / data source | Lifecycle |
|---|---|---|
| Business data graph | `sapintegrationsuite_business_data_graph` | Create, read, update, delete, import |

## Before you start

SAP's setup for API Composition has several steps outside the Configuration API. They must be
done before the first `apply`:

1. **Entitlement.** Add the **API Composition** service with plan **`configuration`** to the
   subaccount. SAP describes this plan as the one for configuring business data graphs with the
   Configuration API. Plan `integration-flow` of *Process Integration Runtime* is for client
   applications that *consume* a graph. It is not used here.
2. **Capability.** Activate API Composition (with Developer Hub) on the Integration Suite home page.
3. **Destinations.** Every system of the graph is reached through a BTP destination in the
   subaccount. SAP requires the additional property `IntegrationCell.Include = true` on each of
   them, otherwise API Composition does not offer them. The optional property
   `URL.ConnectionTimeoutInSeconds` sets a per-destination timeout.
4. **Credentials.** Create an instance of the API Composition service with plan `configuration`
   and a service key for it. The next section explains what to take from the key.

People who build graphs in the UI need the role collection `Graph.KeyUser` (role
`Graph_Key_User`); `Graph.Guest` gives read-only access to the UI and to the Configuration API.

## Credentials: a third, separate block

The Configuration API runs on its own region-specific host, for example `https://eu10.graph.sap`,
and uses its own OAuth client. The Cloud Integration credentials in `oauth` and the API Portal
credentials in `api_management` do not work there. The provider therefore has a separate,
optional block:

```hcl
provider "sapintegrationsuite" {
  host = var.integration_suite_host

  oauth {
    token_url     = var.integration_suite_token_url
    client_id     = var.integration_suite_client_id
    client_secret = var.integration_suite_client_secret
  }

  api_composition {
    host          = var.api_composition_host      # e.g. https://eu10.graph.sap
    token_url     = var.api_composition_token_url # full URL ending in /oauth/token
    client_id     = var.api_composition_client_id
    client_secret = var.api_composition_client_secret
  }
}
```

Each attribute can also come from an environment variable:
`SAP_INTEGRATION_SUITE_API_COMPOSITION_HOST`, `_TOKEN_URL`, `_CLIENT_ID` and `_CLIENT_SECRET`.
Set all four or none. Configurations without business data graphs leave the block out and are
not affected. The business data graph resource and data source report a clear error when the
block is missing.

**What to copy from the service key.** SAP does not document the service key of the
`configuration` plan field by field. For the consumption keys of *Process Integration Runtime*,
SAP lists `clientid`, `clientsecret`, `tokenurl` and `url`, and BTP service keys usually follow
that pattern. Take the client ID and secret, the token endpoint, and the API Composition host.
If the key only contains the authentication server's base URL, append `/oauth/token` to get
`token_url`. For `host`, use the scheme and host name only. The provider appends
`/configuration/v1/sap.graph`. On a test tenant, the key's `url` was the region host followed
by `/configuration`; leave that path out.

**Authorization: an open question.** SAP protects the Configuration API with the role
`Graph_Key_User` (`Graph_Guest` for reading only) and describes both only as roles that an
administrator assigns to people. SAP does not say how a service key of the `configuration` plan
obtains that role. On a test tenant (checked on 2026-09-29 and 2026-10-01), a token that the
key's client obtained with client credentials carried no scope except `uaa.resource`. The API
answered every request, even `$metadata`, with HTTP 403 and code 2707 ("You don't have
permission to access this resource. Please check your assigned roles in your SAP BTP
subaccount."). Whether a user token of a key user works instead has not been tested yet. Until
it is clear which token SAP expects, plan for this 403. When the provider receives it, its error
names the role and shows SAP's trace ID for a support ticket.

## The configuration model

The resource follows SAP's *Business Data Graph Configuration File*. That is the same JSON
document the Integration Suite UI generates and lets you download, so an existing graph
translates directly into HCL.

**Identifier.** `business_data_graph_identifier` is part of the URL client applications call.
SAP allows up to 20 lowercase alphanumeric characters separated by hyphens. The provider checks
this at plan time. Changing the identifier replaces the graph.

**Data sources.** Each entry in `data_sources` is one system. Its `name` is your choice. It
appears in key-based references in response payloads (for example `s4~1356`), so SAP recommends
short names. `services` lists the destinations. `path` lets one destination defined on a root URL
serve several services:

```hcl
data_sources = [
  {
    name = "s4"
    services = [
      { destination_name = "s4-root", path = "/odata/sap/API_BUSINESS_PARTNER" },
      { destination_name = "s4-root", path = "/odata4/sap/api_bank/srvd_a2x/sap/bank/0002" },
    ]
  },
  {
    name      = "custom"
    namespace = "company.custom" # only for custom services SAP does not know
    services  = [{ destination_name = "custom-odata" }]
  },
]
```

**Locating policy.** The locating policy tells API Composition which system to use for each
entity:

- A **rule** names the *leading* system for an entity (`sap.graph.Product`) or for a namespace
  with a trailing wildcard (`sap.s4.*`). A specific name beats a wildcard, so the order of rules
  does not matter. Every entity needs exactly one rule without cues.
- `local` lists systems whose key-based references stay in their own system. For example, when
  a Sales Cloud quote points at a product, the product is read from Sales Cloud, not from the
  leading S/4 system.
- **Cues** are labels a client can send with a request to select a different rule. Declare them
  in `locating_policy.cues` and reference them by name in a rule's `cues`. A cue may appear in
  only one rule per entity.
- `source_entity` is for composite custom entities whose parts come from different systems. Each
  part from a system other than the main one needs its own rule.
- **Key mappings** translate keys when two systems identify the same entity differently. Each
  mapping has a `foreign_key` side (the referencing attribute) and a `references` side (the key
  in the other system). SAP supports one attribute per side. An optional `strategy` with name
  `format` rewrites the value using an RE2 `match` with named groups and a `replace` pattern.

```hcl
locating_policy = {
  cues = [{ name = "emea", description = "European subsidiaries" }]

  key_mapping = [
    {
      foreign_key = { data_source = "c4c", entity_name = "sap.c4c.ProductCollection", attributes = ["ExternalID"] }
      references  = { data_source = "s4", entity_name = "sap.s4.A_Product", attributes = ["Product"] }
    },
  ]

  rules = [
    { name = "sap.s4.*", leading = "s4" },
    { name = "sap.graph.*", leading = "s4", local = ["c4c"] },
    { name = "sap.graph.Product", leading = "c4c", cues = ["emea"] },
  ]
}
```

**Exclude.** `exclude` removes mirrored entities from the graph's API, for example
`["sap.s4.A_Product", "sap.c4c.*"]`. Associations to them disappear as well. Custom entities can
still be built on them.

**Versions.** `graph_model_version` selects the version of API Composition's unified entity model.
When left out, SAP picks the current one, and the provider records it. `effective_graph_model_version`
shows what SAP actually applied.

**Empty lists.** Optional lists must be left out rather than set to `[]`. SAP returns an empty list
and a missing one the same way, so only the missing form survives a refresh without a diff. The
provider rejects `[]` at plan time.

## Asynchronous processing

SAP processes a new or changed graph in the background. According to SAP, the status is
`PROCESSING` first and then becomes `DEPLOYMENT_INITIATED` (success) or `FAILED`, with details
in `statusDetails` and `logMessages`.

Create and update wait for this. They poll with increasing intervals of up to ten seconds until
the status leaves `PROCESSING`, for at most 20 minutes by default. Adjust the limit with a
`timeouts` block:

```hcl
timeouts {
  create = "30m"
  update = "30m"
}
```

If SAP reports `FAILED`, the apply fails with SAP's `statusDetails` and log messages. The graph
still exists on SAP's side, so the provider records it in state, and Terraform marks it
**tainted**. The next apply replaces it. The same happens when the wait times out while the graph
is still processing. The most common cause of `FAILED` is a destination that API Composition
cannot reach or that lacks `IntegrationCell.Include`.

## What SAP documents and what the provider infers

SAP's page *Configuration API Specification and Usage* documents the service root
`/configuration/v1/sap.graph`, the property list, a complete create example (`POST
.../GraphConfiguration`), reading (`GET .../GraphConfiguration/{id}`), updating (`PATCH
.../GraphConfiguration/{id}`) and the status model. The field-level details of data sources, the
locating policy, cues and key mappings come from the *Business Data Graph Configuration File*
page. The Business Accelerator Hub lists the API as *Graph - Configuration*
(`Graph_ConfigurationAPI`) of type OData V4. It serves its metadata at
`/configuration/v1/sap.graph/$metadata`, but that document is only reachable with credentials
and is not published; fetching it with a service key of the `configuration` plan would confirm
the property names and the PATCH semantics. So far, a live system has answered only with the
403 described under "Credentials", so no part of this resource has been checked against SAP yet.

The following parts are the provider's own inference:

- **Update body.** SAP names the method and URL but shows no body. The provider sends the
  writable properties: identifier, versions, `exclude`, `dataSources` and `locatingPolicy`.
  `exclude` is always sent, as `[]` when empty, so removing it in HCL also removes it in SAP.
  Read-only properties and extensions are never sent.
- **Delete.** SAP says the API can delete graphs but documents no request. The provider sends
  `DELETE` to the graph's URL.

  Because SAP documents neither the update body nor the delete request, both are unofficial
  operations and need `enable_unofficial = true` in the provider block, in addition to
  `enable_experimental`. Without it, a graph can be created and read, but a plan that updates
  it in place, replaces it or destroys it fails with an error that names the operation.
- **Shape of the locating policy.** The property table calls it an "array of locating policies",
  but both SAP examples show a single object with `cues`, `keyMapping` and `rules`. The provider
  follows the examples.
- **Log messages.** SAP does not describe an entry of `logMessages`. The provider keeps each entry
  as the JSON text SAP returned.

## Limitations

- **Extensions** (custom entity projections) cannot be managed through the Configuration API.
  SAP says so explicitly. `extensions` is read-only, and updates leave SAP's extensions alone.
- **Cues on key mappings.** SAP describes key mappings scoped by cues but documents no property
  for them, so the provider does not support them.
- **OData containment** is described as a graph setting in SAP's configuration file page, but
  without a property name, so it cannot be set through this resource.
- The graph's data API, the API Composition Navigator in Developer Hub, and the service
  instances for client applications are outside this provider's scope.

## Import

Import a graph by its identifier:

```shell
terraform import sapintegrationsuite_business_data_graph.sales sales
```

The provider reads the graph from SAP, including attributes SAP filled itself, such as
`graph_model_version`. The data source `sapintegrationsuite_business_data_graph` reads a graph
without managing it, for example one maintained in the UI.
