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
   and a service key for it, and pick the user the provider logs in as. The next section
   explains what to take from the key and why the user is needed.

People who build graphs in the UI need the role collection `Graph.KeyUser` (role
`Graph_Key_User`); `Graph.Guest` gives read-only access to the UI and to the Configuration API.
The user the provider logs in as needs `Graph.KeyUser` as well.

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

    # A user with the role collection Graph.KeyUser; see "Logging in as a key user".
    username = var.api_composition_username
    password = var.api_composition_password
    origin   = var.api_composition_origin # only for an identity provider other than the default
  }
}
```

Each attribute can also come from an environment variable:
`SAP_INTEGRATION_SUITE_API_COMPOSITION_HOST`, `_TOKEN_URL`, `_CLIENT_ID`, `_CLIENT_SECRET`,
`_USERNAME`, `_PASSWORD` and `_ORIGIN`. Set the first four together, or none. Configurations
without business data graphs leave the block out and are not affected. The business data graph
resource and data source report a clear error when the block is missing.

**What to copy from the service key.** SAP does not document the service key of the
`configuration` plan field by field. For the consumption keys of *Process Integration Runtime*,
SAP lists `clientid`, `clientsecret`, `tokenurl` and `url`, and BTP service keys usually follow
that pattern. Take the client ID and secret, the token endpoint, and the API Composition host.
If the key only contains the authentication server's base URL, append `/oauth/token` to get
`token_url`. For `host`, use the scheme and host name only. The provider appends
`/configuration/v1/sap.graph`. On a test tenant, the key's `url` was the region host followed
by `/configuration`; leave that path out.

### Logging in as a key user

The Configuration API accepts only tokens that carry its scope `config`. SAP grants that scope
with the role `Graph_Key_User` (`Graph_Guest` for reading only) and describes both only as roles
that an administrator assigns to people. SAP does not say how the client of a `configuration`
service key obtains the scope on its own, and on a test tenant it did not:

- A token that the key's client requested with **client credentials** carried no scope except
  `uaa.resource`. The API answered every request, even `$metadata`, with HTTP 403 and code 2707
  ("You don't have permission to access this resource. Please check your assigned roles in your
  SAP BTP subaccount."). Checked on 2026-09-29, 2026-10-01 and 2026-10-03.
- A token that the same client requested for a user with the role collection `Graph.KeyUser`,
  with the **password grant**, carried the scope `config`. The API returned its service document
  and `$metadata` (2026-10-03).

With `username` and `password`, the provider requests its tokens the second way: the service
key's client authenticates itself as before, and the token carries the user's roles. Without
them, it uses client credentials, which only works if your tenant grants that client the scope.
The API's `$metadata` declares client credentials and the authorization code flow as its
security schemes; the password grant is a feature of the SAP BTP token service, so it is
verified on a tenant but not part of SAP's description of this API.

What the user login needs:

- **A user with `Graph.KeyUser`.** Prefer a technical user that exists only for Terraform over a
  person's account, because every change in SAP is then made in that user's name.
- **An identity provider that accepts passwords.** The password grant cannot answer a second
  factor or a browser login. Users of SAP ID service work; users of SAP Cloud Identity Services
  work when no second factor is enforced for them.
- **`origin` for an identity provider other than the default.** The provider sends the origin
  key as `login_hint`, so the token service checks the password against that identity provider.
  The origin key is shown in the subaccount under *Security* > *Trust Configuration*.

A wrong password answers with `invalid_grant` "User authentication failed." from the token
service. Repeated failures can lock the user, so check `username`, `password` and `origin` before
running `apply` again. When the API itself answers 403, the provider's error names the missing
scope and the user login, and shows SAP's trace ID for a support ticket.

## The configuration model

The resource follows SAP's *Business Data Graph Configuration File*. That is the same JSON
document the Integration Suite UI generates and lets you download, so an existing graph
translates directly into HCL.

**Identifier.** `business_data_graph_identifier` is part of the URL client applications call.
SAP allows up to 20 lowercase alphanumeric characters separated by hyphens. The provider checks
this at plan time. Changing the identifier replaces the graph.

**Data sources.** Each entry in `data_sources` is one system. Its `name` is your choice. It
appears in key-based references in response payloads (for example `s4~1356`), so SAP recommends
short names. `services` lists the destinations. A destination serves only one data source: when two
data sources named the same destination on a tenant, the first one stayed empty. `path` lets one
destination defined on a root URL serve several services:

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
  only one rule per entity. Every cue needs a `description`: SAP rejects a cue without one, so the
  provider requires it.
- `source_entity` is for composite custom entities whose parts come from different systems. Each
  part from a system other than the main one needs its own rule.
- **Key mappings** translate keys when two systems identify the same entity differently. Each
  mapping has a `foreign_key` side (the referencing attribute) and a `references` side (the key
  in the other system). SAP supports one attribute per side. An optional `strategy` with name
  `format` rewrites the value using an RE2 `match` with named groups and a `replace` pattern.
  A key mapping with `cues` applies only to requests with those cues; SAP then requires rules
  with the same cue for the entities of both sides ("No rule matches declared 'foreignKey'"
  otherwise).

**Settings named only in the `$metadata`.** SAP's pages describe OData containment and
cue-scoped key mappings, but the property names come only from the API's `$metadata`, and the
descriptions are not on SAP's pages at all. Setting any of these needs `enable_unofficial = true`:

| Attribute | Meaning |
|---|---|
| `description` | Description of the graph. |
| `odata_containment` | Whether contained entities are reached only through their parent entity. SAP enables it by default. |
| `locating_policy.description` | Description of the locating policy. |
| `locating_policy.key_mapping[].cues` | Cues that select a key mapping. |

`description` and `odata_containment` keep SAP's value when you leave them out, so removing them
from the configuration changes nothing in SAP. An acceptance test created a graph with the first
three, changed them in place and imported it on a tenant (October 2026). SAP accepted and
evaluated the cues of a key mapping in the same test series; a complete round trip needs two
destinations and has not run yet.

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
(`Graph_ConfigurationAPI`) of type OData V4.

The API's `$metadata` is not published, but a key user's token read it from a test tenant on
2026-10-03, and the provider checks its requests against that document. It confirms every
property name the provider sends and reads, including the shape of the locating policy (a single
object) and of key mappings. It also settles three details SAP's pages leave open:

- A graph can only be read by its identifier. The collection itself is declared not readable,
  and `GET .../GraphConfiguration` answered HTTP 405. The data source therefore needs the
  identifier; there is no lookup by other attributes.
- `extensions` is a list of objects with a `name`, not a list of strings. The provider shows the
  names.
- A graph has a `deleted` flag. The provider treats a graph that SAP marks as deleted like one
  that no longer exists: a refresh removes it from state.

The following parts are still the provider's own inference:

- **Update body.** SAP names the method and URL but shows no body. The provider sends the
  writable properties: identifier, versions, `exclude`, `dataSources` and `locatingPolicy`.
  `exclude` is always sent, as `[]` when empty, so removing it in HCL also removes it in SAP.
  Read-only properties and extensions are never sent. The `$metadata` confirms the property
  names but not how SAP applies a `PATCH`.
- **Delete.** SAP says the API can delete graphs but documents no request. The provider sends
  `DELETE` to the graph's URL.

  Because SAP documents neither the update body nor the delete request, both are unofficial
  operations and need `enable_unofficial = true` in the provider block. Without it, a graph can
  be created and read, but a plan that updates it in place, replaces it or destroys it fails with
  an error that names the operation.
- **Log messages.** The `$metadata` types an entry of `logMessages` as `level`, `message` and
  `code`. The provider keeps each entry as the JSON text SAP returned.

An acceptance test created a graph over a custom OData destination, changed it in place, imported
it and deleted it on a tenant (October 2026), so the update body and the delete request work as
described, although SAP does not document them.

## Limitations

- **Extensions** (custom entity projections) cannot be managed through this resource. SAP's
  documentation says the Configuration API does not manage them, although the `$metadata`
  declares an `Extension` entity set. `extensions` is read-only, and updates leave SAP's
  extensions alone.
- **Entity names of custom services.** API Composition names the entities of a custom OData
  service after its entity sets, prefixed with the data source's namespace (for example
  `company.custom.Categories`). SAP does not document this; it showed in a tenant's validation
  messages.
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
