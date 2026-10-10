---
page_title: "Provider: SAP Integration Suite"
description: |-
  Manage design-time content, deployments, security material, the Partner Directory, access
  policies and Classic API Management inside an existing SAP Integration Suite tenant.
---

# SAP Integration Suite Provider

The `sapintegrationsuite` provider manages what lives *inside* an SAP Integration Suite tenant:
integration packages and their artifacts, deployments to the Cloud Integration runtime,
externalized parameters, credentials and keystore entries, the Partner Directory, access
policies, and Classic API Management objects. It talks to SAP's public OData and REST APIs
only; it never calls the endpoints behind the SAP Integration Suite user interface.

New here? The [Getting Started guide](guides/getting-started.md) goes from a service key to a
deployed integration flow and its endpoint URL.

## What the provider manages

| Area | Resources | Credentials |
|---|---|---|
| Cloud Integration content | integration packages, integration flows, message mappings, value mappings, script collections, custom adapters, each with a separate `*_deployment` resource; externalized flow parameters; custom tags; number ranges | `oauth` block |
| Security material | user and OAuth2 client credentials, secure parameters, certificates, generated key pairs | `oauth` block |
| Partner Directory | string and binary parameters, alternative partners, authorized users, user credential parameters | `oauth` block |
| Access policies | policies and their artifact references | `oauth` block |
| Classic API Management | API providers, API products, certificate store references, key value maps | `api_management` block |
| API Composition | business data graphs | `api_composition` block |
| Integration Assessment | vendors, applications, application instances, technologies, technology instances, technology profiles (unofficial) | `integration_assessment` block |

Data sources exist for most of these, plus discovery data sources for deployed service
endpoints, keystore entries, partners, and the provider's own feature catalog.
[Feature Support](https://github.com/Prideth/terraform-provider-sap-integration-suite/blob/master/docs/feature-support.md)
lists every SAP Integration Suite capability with its status, including those the provider
deliberately does not manage and why.

## What the provider does not manage

The provider starts from a subaccount in which SAP Integration Suite is already subscribed and
its capabilities are activated. Everything around the tenant belongs to other tools:

| Object | Managed with |
|---|---|
| Subaccount, entitlements, the Integration Suite subscription | [SAP/btp](https://registry.terraform.io/providers/SAP/btp/latest) provider |
| Service instances and service keys (plans `api`, `apiportal-apiaccess`, `configuration`) | SAP/btp or [cloudfoundry/cloudfoundry](https://registry.terraform.io/providers/cloudfoundry/cloudfoundry/latest) provider |
| Role collections and their assignment to users | SAP/btp provider |
| BTP destinations (used by API providers and business data graphs) | SAP/btp provider or the BTP cockpit |
| Activating Integration Suite capabilities | the Integration Suite application (a manual step) |
| Edge Integration Cell installation on Kubernetes | SAP's Edge Lifecycle Management |
| Event Mesh queues and topics | outside this provider; Event Mesh has its own management API and tooling |
| Developer Hub | outside this provider; planned as a separate provider |

Runtime data is out of scope as well: message processing logs, message stores, data store
entries, variables and assessment results are not desired-state configuration.

The provider has no dependency on the SAP/btp provider. Pass values between the two through
ordinary Terraform references or variables.

## Requirements

- Terraform 1.5 or later. Resources with write-only secrets (`*_wo` attributes) need
  Terraform **1.11** or later; set `required_version = ">= 1.11"` if you use one of them.
- An SAP Integration Suite subscription (Cloud Foundry environment) with Cloud Integration
  activated.
- A service key for each API area you manage, see below.

## Authentication

Each API area has its own OAuth 2.0 client (client credentials grant). The provider never
reuses one client for another area, because SAP issues them from different services.

| Provider block | Service instance | Plan | Used for |
|---|---|---|---|
| `host` + `oauth` | Process Integration Runtime (`it-rt`) | `api` | Cloud Integration, Security Content, Partner Directory, access policies |
| `api_management` | API Management, API portal | `apiportal-apiaccess` | Classic API Management |
| `api_composition` | API Composition | `configuration` | business data graphs |
| `integration_assessment` | Integration Assessment APIs | `default` | the Integration Assessment landscape |

Where the values come from:

| Attribute | Plan `api` key | Plan `apiportal-apiaccess` key |
|---|---|---|
| `host` | `oauth.url` | `url` |
| `token_url` | `oauth.tokenurl` | `tokenUrl` |
| `client_id` | `oauth.clientid` | `clientId` |
| `client_secret` | `oauth.clientsecret` | `clientSecret` |

What a client may do is decided by the roles on its service instance, for example
`WorkspacePackagesEdit` for content or `AccessPoliciesEdit` for access policies. A missing role
shows up as `403 Forbidden`. The [Authorization and Roles guide](guides/authorization-and-roles.md)
lists the roles per resource family.

### Environment variables and precedence

Every attribute can come from an environment variable instead. A value set in the provider
block wins; an attribute that is unset or an empty string falls back to its variable.

| Attribute | Environment variable |
|---|---|
| `host` | `SAP_INTEGRATION_SUITE_HOST` |
| `oauth.token_url` | `SAP_INTEGRATION_SUITE_TOKEN_URL` |
| `oauth.client_id` | `SAP_INTEGRATION_SUITE_CLIENT_ID` |
| `oauth.client_secret` | `SAP_INTEGRATION_SUITE_CLIENT_SECRET` |
| `api_management.*` | `SAP_INTEGRATION_SUITE_API_MANAGEMENT_HOST`, `_TOKEN_URL`, `_CLIENT_ID`, `_CLIENT_SECRET` |
| `api_composition.*` | `SAP_INTEGRATION_SUITE_API_COMPOSITION_HOST`, `_TOKEN_URL`, `_CLIENT_ID`, `_CLIENT_SECRET`, `_USERNAME`, `_PASSWORD`, `_ORIGIN` |
| `integration_assessment.*` | `SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_ENTITIES_URL`, `_TOKEN_URL`, `_CLIENT_ID`, `_CLIENT_SECRET` |
| `enable_experimental` | `SAP_INTEGRATION_SUITE_ENABLE_EXPERIMENTAL` |
| `enable_unofficial` | `SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL` |
| `convert_ui_labels` | `SAP_INTEGRATION_SUITE_CONVERT_UI_LABELS` |

Missing Cloud Integration credentials do not fail the provider configuration; a resource or
data source that needs them fails with an error that says what is missing. The `api_management`
`api_composition` and `integration_assessment` blocks must have all four values or none; a
partial block fails at once. The optional user login of `api_composition` needs `username` and
`password` together.
The feature catalog data sources work without any credentials.

## Example usage

A minimal configuration that takes the credentials from the environment:

```terraform
terraform {
  required_providers {
    sapintegrationsuite = {
      source  = "Prideth/sap-integration-suite"
      version = "~> 0.6.0"
    }
  }
}

# Everything comes from the environment:
#   SAP_INTEGRATION_SUITE_HOST           oauth.url of the service key (plan "api")
#   SAP_INTEGRATION_SUITE_TOKEN_URL      oauth.tokenurl
#   SAP_INTEGRATION_SUITE_CLIENT_ID      oauth.clientid
#   SAP_INTEGRATION_SUITE_CLIENT_SECRET  oauth.clientsecret
provider "sapintegrationsuite" {}

resource "sapintegrationsuite_integration_package" "order_processing" {
  id          = "ORDER_PROCESSING"
  name        = "Order Processing"
  short_text  = "Order intake and confirmation"
  description = "Integration flows that receive sales orders from the web shop and confirm them to the customer."
}
```

A configuration for a pipeline that manages Cloud Integration content and Classic API
Management, with the service keys passed in as sensitive variables:

```terraform
terraform {
  # 1.11 is the first release with write-only attributes, which every
  # secret-bearing resource of this provider uses.
  required_version = ">= 1.11"

  required_providers {
    sapintegrationsuite = {
      source = "Prideth/sap-integration-suite"
      # A 0.x minor release may contain breaking changes; pin the minor
      # version and read the upgrade notes before raising it.
      version = "~> 0.6.0"
    }
  }
}

variable "integration_suite" {
  description = "The oauth section of the service key of a Process Integration Runtime instance with plan \"api\"."
  type = object({
    url          = string
    tokenurl     = string
    clientid     = string
    clientsecret = string
  })
  sensitive = true
}

variable "api_portal" {
  description = "Service key of the API Management, API portal instance with plan \"apiportal-apiaccess\"."
  type = object({
    url          = string
    tokenUrl     = string
    clientId     = string
    clientSecret = string
  })
  sensitive = true
}

provider "sapintegrationsuite" {
  # Cloud Integration, Security Content, Partner Directory and access
  # policies: one OAuth client from plan "api" of Process Integration Runtime.
  host = var.integration_suite.url

  oauth {
    token_url     = var.integration_suite.tokenurl
    client_id     = var.integration_suite.clientid
    client_secret = var.integration_suite.clientsecret
  }

  # Classic API Management uses its own client. Leave the block out when you
  # manage no API providers, products, proxies, certificate store references
  # or key value maps.
  api_management {
    host          = var.api_portal.url
    token_url     = var.api_portal.tokenUrl
    client_id     = var.api_portal.clientId
    client_secret = var.api_portal.clientSecret
  }

  # Both default to false. Turn a switch on only for the resources that need
  # it; see "Support status and opt-in switches" on this page.
  enable_experimental = false
  enable_unofficial   = false
}
```

Keep client secrets out of `.tf` files and out of version control. Pass them through a secret
store, `TF_VAR_` variables or the `SAP_INTEGRATION_SUITE_*` environment variables of the
pipeline. The provider never writes them to state, but Terraform stores the values of input
variables in saved plan files (`terraform plan -out`); the `SAP_INTEGRATION_SUITE_*`
environment variables are not stored anywhere.

## Support status and opt-in switches

Every resource and data source has a support status, listed on its page and in Feature
Support, which also lists the capabilities the provider does not implement:

- ✅ **Supported**: the full lifecycle is backed by an official SAP contract and was run on a
  tenant.
- 🟡 **Partial**: official, but some operations are missing because SAP does not offer them; the
  page says which.
- 👁️ **Read-only**: a data source only.
- 🧪 **Experimental**: an implementation exists, but its lifecycle has not yet been sufficiently
  validated against a real SAP tenant. The schema may still change. Needs
  `enable_experimental = true`.
- 🧭 **Unofficial**: validated on a tenant, but the SAP API contract behind it is not fully
  published or officially documented (it is known only from the service's `$metadata`), so SAP
  can change it without notice. Needs `enable_unofficial = true`.
- 🔬 **Research required**: a known capability whose public API coverage, lifecycle semantics or
  suitability for Terraform still needs investigation. There is no resource for it yet.
- ❌ **Unsupported**: investigated, and not implemented: SAP offers no usable public API or safe
  lifecycle, or the capability is out of the provider's scope.
- ↗️ **Separate provider**: belongs to another Terraform provider.

A few documented resources use one operation that SAP does not document, for example the
in-place description update of an access policy. `enable_unofficial` allows those operations too;
without it, a plan that needs one fails with an error that names it, and the resource stays
usable with its documented operations. The resource pages list these operations.

`convert_ui_labels` lets `sapintegrationsuite_access_policy_reference` accept the labels SAP's UI
shows (*Matches*, *Integration Flow*) and convert them to SAP's constants. It is experimental and
takes effect only together with `enable_experimental = true`; see the resource page for the
conversions it makes.

## Runtimes

SAP Integration Suite can run content in three places: the Cloud Integration runtime, the
Integration Cell (part of the current API Management), and Edge Integration Cells in your own
Kubernetes clusters.

- The `*_deployment` resources deploy to the **Cloud Integration runtime**. An integration flow
  whose externalized parameter `SAP_ProfileId` is `integrationcell` is deployed to the
  Integration Cell instead; the deployment resource detects this and stops with an
  explanation. Set `SAP_ProfileId` to `iflmap` with
  [`sapintegrationsuite_integration_flow_configuration`](resources/integration_flow_configuration.md).
- **Edge Integration Cell** targeting (`runtime_location_id`) exists in the schema but is not
  supported: it has never been run against a tenant with an Edge Integration Cell. Leave it
  unset. See the [Edge Integration Cell guide](guides/edge-integration-cell.md).

## Secrets and state

Secrets you supply (passwords, client secrets, secure parameter values) are **write-only**
attributes named `*_wo`. Terraform sends them to SAP but never writes them to the plan or
the state, and the provider never reads them back. Each has a companion `*_wo_version`, a
plain value such as a date or counter: change it to send a new secret, because Terraform cannot
see a change of the write-only value itself. Marking an attribute sensitive would only hide it
in the CLI output; write-only keeps it out of the state file.

The provider does not manage objects whose secret SAP generates and would have to be stored in
state. Key pairs are generated by SAP and their private key never leaves the tenant.

## Asynchronous operations and timeouts

Deployments, API proxy imports and business data graphs are processed asynchronously by SAP.
Their resources poll until SAP reports a final state and accept a `timeouts` block. A
deployment that does not start in time stays in state with its last status and is marked
tainted, so the next apply or destroy handles it instead of leaving it running untracked.

## Troubleshooting

| Symptom | Cause |
|---|---|
| `401 Unauthorized` on every request | Wrong token URL, client ID or secret, or a client from another service. Check the table under Authentication. |
| `403 Forbidden`, often with an empty body | The client lacks a role. See the [Authorization and Roles guide](guides/authorization-and-roles.md). |
| `context canceled` on the token URL | Provider 0.1.0. Upgrade to 0.2.0 or later. |
| `... is experimental` / `... is unofficial` | The resource or operation needs `enable_experimental` or `enable_unofficial`. |
| A deployment times out and the resource is tainted | SAP needed longer than `timeouts.create`, or the flow went to another runtime (`SAP_ProfileId`). The [Integration Content guide](guides/integration-content.md) lists typical deploy times. |
| `Could not update artifact ... Bundle-symbolicName` | The ZIP was exported under another artifact ID. The provider aligns flows and message mappings; for script collections and value mappings, fix `Bundle-SymbolicName` in the ZIP. |

Errors carry the HTTP status, SAP's error code and SAP's message, which usually name the
field or rule that failed. The provider writes no request log, and neither errors nor state
ever contain tokens or client secrets.

The [Troubleshooting guide](guides/troubleshooting.md) covers more cases, and
[Provider Upgrades](guides/provider-upgrades.md) describes what changed between versions.

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `api_composition` (Block, Optional) Credentials for API Composition's Configuration API, used only by sapintegrationsuite_business_data_graph. The API has its own region-specific host and OAuth client, from a service key of an API Composition service instance with plan "configuration"; the oauth and api_management credentials do not work there. Set host, token_url, client_id and client_secret together, or none. username and password add a key user's login through that client, which the Configuration API needed on a tenant. Each value can also come from a SAP_INTEGRATION_SUITE_API_COMPOSITION_* environment variable. (see [below for nested schema](#nestedblock--api_composition))
- `api_management` (Block, Optional) Optional, and independent of the oauth block above. Classic API Management (API Providers, API Products, Key Value Maps, Certificate Store References) authenticates against its own API Portal application URL and its own OAuth 2.0 client, generated from the apiportal-apiaccess service plan — never the Cloud Integration credentials configured above. Leave this entire block out if you do not use any sapintegrationsuite_api_provider, sapintegrationsuite_api_product, sapintegrationsuite_api_key_value_map, or sapintegrationsuite_api_management_certificate_store_reference resource or data source. All four values (or their SAP_INTEGRATION_SUITE_API_MANAGEMENT_* environment variable equivalents) must be supplied together, or all left unset. (see [below for nested schema](#nestedblock--api_management))
- `convert_ui_labels` (Boolean) Lets sapintegrationsuite_access_policy_reference accept the labels SAP's UI shows where SAP's API stores constants: "Matches" and "Equals" as operator, "name" or "Id" as attribute, and an artifact type label that spells exactly its constant, such as "Integration Flow" for INTEGRATION_FLOW. The provider sends the constant and shows each conversion as a plan warning; the state keeps your spelling. Only these proven conversions are made; other labels are still rejected with the value to use. Experimental: takes effect only together with enable_experimental = true. Off by default. Can also be set via the SAP_INTEGRATION_SUITE_CONVERT_UI_LABELS environment variable.
- `enable_experimental` (Boolean) Allows resources and data sources whose support status is "experimental": implemented on a documented API, but their lifecycle has not yet passed an acceptance test on a tenant, so behavior or schema may still change. Off by default; a configuration that uses one fails until this is true. Can also be set via the SAP_INTEGRATION_SUITE_ENABLE_EXPERIMENTAL environment variable. See docs/feature-support.md for which ones they are.
- `enable_unofficial` (Boolean) Allows resources and data sources whose support status is "unofficial": they work and were verified on a tenant, but SAP does not document the API behind them (it is known only from the service's $metadata), so SAP may change it without notice. It also allows the unofficial operations of otherwise documented resources, for example changing an access policy's description in place or deleting a number range. Off by default; a configuration or plan that needs one fails until this is true. Can also be set via the SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL environment variable.
- `host` (String) Base URL of the SAP Integration Suite tenant used for Cloud Integration APIs, for example https://<tenant>.it-cpi<...>.cfapps.<region>.hana.ondemand.com. Can also be set via the SAP_INTEGRATION_SUITE_HOST environment variable.
- `integration_assessment` (Block, Optional) Credentials for the Integration Assessment Entities API, used only by the sapintegrationsuite_integration_assessment_* resources and data sources. They come from a service key of a service instance of "Integration Assessment APIs" (plan default); the oauth, api_management and api_composition credentials do not work there. Set all four values, or none. Each can also come from a SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_* environment variable. (see [below for nested schema](#nestedblock--integration_assessment))
- `oauth` (Block, Optional) OAuth 2.0 client credentials used to authenticate against the SAP Integration Suite APIs. (see [below for nested schema](#nestedblock--oauth))

<a id="nestedblock--api_composition"></a>
### Nested Schema for `api_composition`

Optional:

- `client_id` (String) OAuth 2.0 client ID from the service key. Environment variable: SAP_INTEGRATION_SUITE_API_COMPOSITION_CLIENT_ID.
- `client_secret` (String, Sensitive) OAuth 2.0 client secret from the service key. Environment variable: SAP_INTEGRATION_SUITE_API_COMPOSITION_CLIENT_SECRET.
- `host` (String) Region-specific API Composition host from the service key, for example https://eu10.graph.sap. The provider appends /configuration/v1/sap.graph. Environment variable: SAP_INTEGRATION_SUITE_API_COMPOSITION_HOST.
- `origin` (String) Origin key of the user's identity provider in the subaccount (Security, Trust Configuration), sent as login_hint. Leave it out for the default identity provider. Environment variable: SAP_INTEGRATION_SUITE_API_COMPOSITION_ORIGIN.
- `password` (String, Sensitive) Password of username. Environment variable: SAP_INTEGRATION_SUITE_API_COMPOSITION_PASSWORD.
- `token_url` (String) Full OAuth 2.0 token endpoint URL, ending in /oauth/token. If the service key only has the authentication server URL, append /oauth/token. Environment variable: SAP_INTEGRATION_SUITE_API_COMPOSITION_TOKEN_URL.
- `username` (String) User with the role collection Graph.KeyUser. With username and password the provider requests the token with the password grant through the service key's client, so it carries the user's roles. The identity provider must accept passwords without a second factor. Set together with password. Environment variable: SAP_INTEGRATION_SUITE_API_COMPOSITION_USERNAME.


<a id="nestedblock--api_management"></a>
### Nested Schema for `api_management`

Optional:

- `client_id` (String) OAuth 2.0 client ID from the apiportal-apiaccess service key. Can also be set via the SAP_INTEGRATION_SUITE_API_MANAGEMENT_CLIENT_ID environment variable.
- `client_secret` (String, Sensitive) OAuth 2.0 client secret from the apiportal-apiaccess service key. Can also be set via the SAP_INTEGRATION_SUITE_API_MANAGEMENT_CLIENT_SECRET environment variable.
- `host` (String) Base URL of the API Portal application, for example https://<tenant>.prod-eu10.apiportal.cfapps.eu10.hana.ondemand.com, as returned by the apiportal-apiaccess service key's "url" field. Can also be set via the SAP_INTEGRATION_SUITE_API_MANAGEMENT_HOST environment variable.
- `token_url` (String) OAuth 2.0 token endpoint URL from the apiportal-apiaccess service key's "tokenUrl" field. Can also be set via the SAP_INTEGRATION_SUITE_API_MANAGEMENT_TOKEN_URL environment variable.


<a id="nestedblock--integration_assessment"></a>
### Nested Schema for `integration_assessment`

Optional:

- `client_id` (String) OAuth 2.0 client ID from the service key. Environment variable: SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_CLIENT_ID.
- `client_secret` (String, Sensitive) OAuth 2.0 client secret from the service key. Environment variable: SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_CLIENT_SECRET.
- `entities_url` (String) Service root of the Entities API, the service key's "entities" value, for example https://intas-api.cfapps.eu10.hana.ondemand.com/intas/entities/v1. Environment variable: SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_ENTITIES_URL.
- `token_url` (String) OAuth 2.0 token endpoint: the service key's "url" followed by /oauth/token. Environment variable: SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_TOKEN_URL.


<a id="nestedblock--oauth"></a>
### Nested Schema for `oauth`

Optional:

- `client_id` (String) OAuth 2.0 client ID. Can also be set via the SAP_INTEGRATION_SUITE_CLIENT_ID environment variable.
- `client_secret` (String, Sensitive) OAuth 2.0 client secret. Can also be set via the SAP_INTEGRATION_SUITE_CLIENT_SECRET environment variable.
- `token_url` (String) OAuth 2.0 token endpoint URL. Can also be set via the SAP_INTEGRATION_SUITE_TOKEN_URL environment variable.
