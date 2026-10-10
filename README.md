# Terraform Provider for SAP Integration Suite

A Terraform provider for the content **inside an existing** SAP Integration
Suite tenant: Cloud Integration packages, artifacts and deployments,
externalized parameters, security material, the Partner Directory, access
policies and Classic API Management, the Integration Assessment landscape
(unofficial), and API Composition business data graphs.

> **This project is an independent open-source Terraform provider and is
> not an official SAP product**, unless and until SAP formally adopts or
> publishes it.

## Status

The provider is on a 0.x release line: a minor release may contain breaking
changes, each with upgrade steps in [`CHANGELOG.md`](CHANGELOG.md). Pin the
minor version. The practitioner documentation is on the
[Terraform Registry](https://registry.terraform.io/providers/Prideth/sap-integration-suite/latest/docs);
[`ROADMAP.md`](ROADMAP.md) lists planned work.

**Repository naming**: the GitHub repository, Go module, provider binary
name, and registry manifest all use the final naming
(`github.com/Prideth/terraform-provider-sap-integration-suite`,
`Prideth/sap-integration-suite`, `sapintegrationsuite`).

**Branch model**: `master` is this repository's permanent stable/default
branch, and `dev` is the permanent integration/development branch. All
feature work starts from `dev` on a short-lived `feature/<name>` branch
and merges back into `dev`:

```
dev → feature/<name> → dev
```

`dev` is only promoted to `master` as an explicit, separate
stabilization/release step — never automatically as part of merging a
feature. See `CONTRIBUTING.md` for the full workflow.

## Provider scope

This provider manages **content and capabilities inside** an Integration
Suite tenant. It does **not** create BTP subaccounts, entitlements, the
Integration Suite subscription, generic BTP destinations, role collections,
or anything else owned by the SAP BTP control plane — use the official
[`SAP/btp`](https://registry.terraform.io/providers/SAP/btp/latest)
provider for that. See [`docs/provider-scope.md`](docs/provider-scope.md)
and [`docs/provider-boundaries.md`](docs/provider-boundaries.md) for the
full picture.

```
SAP/btp                              Prideth/sap-integration-suite
  Subaccount                           Integration packages, artifacts
  Entitlements                         Deployments, externalized parameters
  Integration Suite subscription  -->  Credentials, certificates, key pairs
  Service instances / bindings         Partner Directory
  Destinations                         Access policies
  Role collections / assignments       Classic API Management
```

Current API Management (API artifacts, Integration Cell) has no public API
yet, and Edge Integration Cell targeting is not supported.

## Feature Support

Every resource and data source here is backed by a currently documented,
SAP-supported public API — see
[`docs/sap-api-references.md`](docs/sap-api-references.md) for the exact
API behind each one, and
[`docs/api-capability-matrix.md`](docs/api-capability-matrix.md) /
[`docs/provisioning-capability-matrix.md`](docs/provisioning-capability-matrix.md)
for the full discovery behind what is and is not implemented yet.

The table below is this provider's complete, canonical feature-support
dashboard — every Integration Suite capability area this project has
evaluated, supported or not, grouped by area.

<!-- BEGIN GENERATED FEATURE SUPPORT -->

Generated from `internal/features/catalog.go` by `go run ./cmd/gendocs -readme` — do not hand-edit the table below; regenerate it instead (`make docs` does this automatically). See [`docs/feature-support.md`](docs/feature-support.md) for the full per-operation matrix and every feature's detailed limitations.

Legend: ✅ Supported · 🟡 Partial · 👁️ Read-only · 🧪 Experimental · 🧭 Unofficial · 🔬 Research required · ❌ Unsupported · ↗️ Separate provider. The [status legend](docs/feature-support.md#status-legend) defines each one.

### Cloud Integration

| Feature | Status | Terraform Support |
|---|:---:|---|
| Archiving Configuration | ❌ | No safe Terraform lifecycle confirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| Custom Tag Configuration | 🟡 | Resource + Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Data Store | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Data Store Entry | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Data Type | ✅ | Resource |
| Design-Time Artifact Versioning | 🟡 | Resource — see [feature-support.md](docs/feature-support.md#all-features) |
| Integration Adapter | 🟡 | Resource + Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Integration Adapter Deployment | 🟡 | Resource — see [feature-support.md](docs/feature-support.md#all-features) |
| Integration Flow | ✅ | Resource |
| Integration Flow Configuration | ✅ | Resource |
| Integration Flow Deployment | ✅ | Resource |
| Integration Package | ✅ | Resource + Data Source |
| Message Mapping | ✅ | Resource + Data Source |
| Message Mapping Deployment | ✅ | Resource |
| Message Processing Logs | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Message Store Entries / JMS Resources | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Message Type | ✅ | Resource |
| Number Range | ✅ | Resource |
| Script Collection | ✅ | Resource + Data Source |
| Script Collection Deployment | ✅ | Resource |
| Service Endpoints | 👁️ | Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Service Interface | ✅ | Resource |
| Value Mapping | 🟡 | Resource + Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Value Mapping Deployment | ✅ | Resource |
| Value Mapping Entry | ❌ | No safe Terraform lifecycle confirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| Variable | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |

### Security & Access Policies

| Feature | Status | Terraform Support |
|---|:---:|---|
| Access Policy | ✅ | Resource + Data Source |
| Access Policy Reference | ✅ | Resource + Data Source |
| Certificate | ✅ | Resource |
| Certificate Chain | 🧭 | Resource — see [feature-support.md](docs/feature-support.md#all-features) |
| Certificate-User Mapping | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| Key Pair | 🟡 | Resource — see [feature-support.md](docs/feature-support.md#all-features) |
| Keystore Entry | 👁️ | Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Known Hosts (SSH) | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| OAuth2 Client Credential | 🟡 | Resource + Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| OAuth2 Password Credentials | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| OAuth2 SAML Bearer Assertion | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| PGP Keys | 🧭 | Resource — see [feature-support.md](docs/feature-support.md#all-features) |
| Secure Parameter | 🧭 | Resource — see [feature-support.md](docs/feature-support.md#all-features) |
| SSH Key | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| User Credential | 🟡 | Resource + Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Security Material Where-Used | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |

### Partner Directory

| Feature | Status | Terraform Support |
|---|:---:|---|
| Alternative Partner | ✅ | Resource + Data Source |
| Partner Directory Authorized User | ✅ | Resource + Data Source |
| Partner Directory Binary Parameter | ✅ | Resource + Data Source |
| Partner | 👁️ | Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Partner Directory String Parameter | ✅ | Resource + Data Source |
| Partner Directory User Credential Parameter | ✅ | Resource |

### Classic API Management

| Feature | Status | Terraform Support |
|---|:---:|---|
| API Product (classic API Management) | ✅ | Resource + Data Source |
| API Provider (classic API Management) | 🟡 | Resource + Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| API Proxy (classic API Management) | 🔬 | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| API Proxy Deployment (classic API Management) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| API Management Application and Developer (Classic) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| API Management Cache Resource (Classic) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| API Management Certificate Store and Certificate (Classic) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| Certificate Store Reference (classic API Management) | ✅ | Resource + Data Source |
| API Management Key Value Map across API Proxies (Classic) | 🔬 | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| Key Value Map (classic API Management) | 🟡 | Resource + Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Policy (classic API Management) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| API Management Policy Template (Classic) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| API Management Product Access Control (Classic) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| API Management Rate Plan (Classic) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| API Management Virtual Host (Classic) | 🟡 | Resource — see [feature-support.md](docs/feature-support.md#all-features) |

### Current API Management / API Artifacts

| Feature | Status | Terraform Support |
|---|:---:|---|
| API Artifact — Current API Management | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| API Artifact Deployment — Integration Cell | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| API Artifact Policy | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| MCP Server | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| Reusable API Artifact | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| Runtime Profile | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |

### API Composition

| Feature | Status | Terraform Support |
|---|:---:|---|
| API Composition Business Data Graph | ✅ | Resource + Data Source |

### Integration Cell

| Feature | Status | Terraform Support |
|---|:---:|---|
| Integration Cell Runtime | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| Integration Cell Virtual Host | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |

### Edge Integration Cell

| Feature | Status | Terraform Support |
|---|:---:|---|
| Edge Integration Cell Access Policy Replication | 🧭 | Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Edge Integration Cell Runtime Targeting | 🔬 | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| Edge Integration Cell Local API Access | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Edge Integration Cell Registration | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| Edge Integration Cell Runtime Operations | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |

### Integration Assessment

| Feature | Status | Terraform Support |
|---|:---:|---|
| Integration Assessment Requests and Assessment Workflow | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Integration Assessment Landscape Configuration | 🧭 | Resource + Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Integration Assessment Master Data | 🧭 | Data Source — see [feature-support.md](docs/feature-support.md#all-features) |

### Capability Provisioning

| Feature | Status | Terraform Support |
|---|:---:|---|
| Current API Management Capability Activation | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| API Management Capability Activation | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| Cloud Integration Capability Activation | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| Edge Integration Cell Capability Activation | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| Integration Cell Capability Activation | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |

### Additional Integration Suite Capabilities

| Feature | Status | Terraform Support |
|---|:---:|---|
| Data Space Integration | 🔬 | Research required — see [feature-support.md](docs/feature-support.md#all-features) |
| Developer Hub | ↗️ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Event Mesh | ↗️ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Integration Advisor Design-Time Content | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| Integration Advisor Runtime Artifact Injection | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Migration Assessment Extraction and Scenario Evaluation | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Migration Assessment Source System | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| OData Provisioning | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| Open Connectors | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Trading Partner Management Agreement | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| Trading Partner Management Agreement Template | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| Trading Partner Management Company Profile | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| Trading Partner Management Partner Directory Generation | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Trading Partner Management Partner Profile | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |

24 supported · 11 partial · 3 read-only · 0 experimental · 6 unofficial · 4 research required · 49 unsupported · 2 separate provider, out of 99 evaluated Integration Suite features.

<!-- END GENERATED FEATURE SUPPORT -->

The catalog behind this table is also queryable directly from Terraform,
with no SAP host or credentials required — useful for scripting a
compliance check or a support-status gate in CI:

```hcl
data "sapintegrationsuite_provider_features" "all" {}

output "supported_features" {
  value = [
    for feature in data.sapintegrationsuite_provider_features.all.features :
    feature.key
    if feature.support_status == "supported"
  ]
}

data "sapintegrationsuite_provider_feature" "value_mapping" {
  key = "cloud_integration.value_mapping"
}
```

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.5,
  or >= 1.11 if you use any attribute ending in `_wo` (write-only secrets
  and the number range counter): `sapintegrationsuite_user_credential`,
  `sapintegrationsuite_oauth2_client_credential`,
  `sapintegrationsuite_secure_parameter`,
  `sapintegrationsuite_partner_user_credential_parameter`,
  `sapintegrationsuite_api_provider` and `sapintegrationsuite_number_range`.
  The provider does not enforce a `required_version`; set one in your
  configuration.
- An SAP Integration Suite tenant with Cloud Integration activated.
- A service key of Process Integration Runtime, plan `api`, with the roles
  your resources need; see the
  [Authorization and Roles guide](docs/guides/authorization-and-roles.md).

## Installation

```hcl
terraform {
  required_providers {
    sapintegrationsuite = {
      source  = "Prideth/sap-integration-suite"
      version = "~> 0.6.0"
    }
  }
}
```

## Authentication

```hcl
provider "sapintegrationsuite" {
  host = var.integration_suite_host

  oauth {
    token_url     = var.integration_suite_token_url
    client_id     = var.integration_suite_client_id
    client_secret = var.integration_suite_client_secret
  }
}
```

Or via environment variables:

```shell
export SAP_INTEGRATION_SUITE_HOST="https://<tenant>.it-cpi<...>.cfapps.<region>.hana.ondemand.com"
export SAP_INTEGRATION_SUITE_TOKEN_URL="https://<tenant>.authentication.<region>.hana.ondemand.com/oauth/token"
export SAP_INTEGRATION_SUITE_CLIENT_ID="..."
export SAP_INTEGRATION_SUITE_CLIENT_SECRET="..."
```

## Example

```hcl
resource "sapintegrationsuite_integration_package" "utilities" {
  id          = "UTILITIES"
  name        = "Utilities Integration"
  short_text  = "Utilities integration content"
  description = "Integration content for the utilities line of business"
}

resource "sapintegrationsuite_integration_flow" "metering" {
  package_id = sapintegrationsuite_integration_package.utilities.id
  flow_id    = "metering"
  name       = "Metering"

  content      = "${path.module}/iflows/metering.zip"
  content_hash = filesha256("${path.module}/iflows/metering.zip")
}

resource "sapintegrationsuite_integration_flow_deployment" "metering" {
  package_id   = sapintegrationsuite_integration_package.utilities.id
  flow_id      = sapintegrationsuite_integration_flow.metering.flow_id
  flow_version = sapintegrationsuite_integration_flow.metering.version

  # Uploading new content keeps the version; redeploy when it changes.
  redeploy_triggers = {
    content = sapintegrationsuite_integration_flow.metering.content_hash
  }
}
```

The [Getting Started guide](docs/guides/getting-started.md) walks through the
whole path from a service key to a deployed flow and its endpoint URL.

See [`examples/greenfield`](examples/greenfield) for a full new-landscape
example and [`examples/brownfield`](examples/brownfield) for importing an
existing one.

## Import

Every resource is importable using its documented ID format, for example:

```shell
terraform import sapintegrationsuite_integration_package.utilities UTILITIES
terraform import sapintegrationsuite_integration_flow.metering UTILITIES/metering
terraform import sapintegrationsuite_access_policy.utilities <access-policy-id>
```

See each resource's page under `docs/resources/` for its exact ID format.

## Development

```shell
go build ./...
go test ./...
make lint   # golangci-lint
make docs   # regenerate docs/ from schema + examples
```

## Testing

- **Unit tests** (`go test ./...`) run without credentials and cover the
  HTTP client, OAuth token handling, OData V2 request/pagination/error
  handling, and the API clients, using `httptest`.
- **Acceptance tests** exercise a real tenant. They run with `TF_ACC=1`,
  credentials, and either `SAP_INTEGRATION_SUITE_ACC_ALL=1` or the gate of a
  capability, for example `SAP_INTEGRATION_SUITE_ACC_CLOUD_INTEGRATION=1`;
  tests without their gate or credentials skip and say which variable is
  missing. Destructive tests additionally need
  `SAP_INTEGRATION_SUITE_ACC_DESTRUCTIVE=1`. `make accplan` shows what would
  run:

  ```shell
  make testacc-all
  ```

  See "Running the acceptance tests against a tenant" in
  [`CONTRIBUTING.md`](CONTRIBUTING.md).

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md). Every new resource must trace back
to a public SAP API, with its contract source recorded in the feature
catalog: SAP documentation, an official API specification, SAP's own tooling,
or only the service's `$metadata` (then it is unofficial or experimental) —
see [`docs/sap-api-references.md`](docs/sap-api-references.md) for the pattern.

## Roadmap

See [`ROADMAP.md`](ROADMAP.md).

## Known limitations

Each resource page on the Registry lists its own limitations, separated into
what SAP's API does not offer and what the provider does not implement. The
most important ones:

- **Capability activation** (Cloud Integration, API Management, Integration
  Cell, Edge Integration Cell) has no public API; it stays a manual step. See
  [`docs/provisioning-capability-matrix.md`](docs/provisioning-capability-matrix.md).
- **Content files cannot be imported.** SAP returns no file for integration
  flows, mappings, script collections or adapters; the first
  apply after an import uploads the configured file. See
  [Importing Existing Content](docs/guides/importing-existing-content.md).
- **Replace-only resources.** Value mappings, integration adapters, API
  products, API providers and key value maps have no documented update, so
  every change replaces them. For API products that drops the subscriptions
  of the old product.
- **Undocumented operations are opt-in.** The content update of message
  mappings and script collections, reading and deleting number ranges, the
  description update of access policies and updating or deleting business
  data graphs work on a tenant but are not documented by SAP; they need
  `enable_unofficial = true`.
- **Value mapping entries** are not managed individually: SAP's entry
  functions dropped values and deleted nothing in tenant tests. The value
  mapping's content is the unit of change.
- **Current API Management** (API artifacts, MCP servers, Integration Cell,
  its virtual hosts and runtime profiles) has no public API as of September
  2026. See [`docs/guides/current-api-management.md`](docs/guides/current-api-management.md).
- **Edge Integration Cell targeting** (`runtime_location_id`) is not
  supported; leave the attribute unset.
- **Secrets are never read back.** Passwords, client secrets and secure
  parameter values are write-only, so a secret changed outside Terraform is
  not detected. Change the matching `*_wo_version` to send a new one.
- **Partner Directory data is unencrypted** in SAP; keep secrets in user
  credential parameters, not in string or binary parameters.

## API support matrix

See [`docs/api-capability-matrix.md`](docs/api-capability-matrix.md) and
[`docs/provisioning-capability-matrix.md`](docs/provisioning-capability-matrix.md).

## Disclaimer

This project is an independent open-source Terraform provider for SAP
Integration Suite. It is not an official SAP product, and SAP has not
endorsed or certified it, unless SAP formally adopts or publishes it in the
future. It uses SAP's public APIs and never the endpoints behind the SAP
Integration Suite user interface. Where a feature relies on a part of a public
API that SAP does not document, the feature is marked unofficial and stays
switched off until `enable_unofficial` is set; see
[`docs/feature-support.md`](docs/feature-support.md) and
[`docs/sap-api-references.md`](docs/sap-api-references.md).

## License

[Apache License 2.0](LICENSE)
