# Terraform Provider for SAP Integration Suite

A Terraform provider for configuring, provisioning, and administering
content and capabilities **inside an already-provisioned** SAP Integration
Suite tenant — Cloud Integration, Access Policies, Classic API Management,
and (as their public APIs are confirmed) SAP's current API Management
model (API Artifacts, Integration Cell), and Edge Integration Cell.

> **This project is an independent open-source Terraform provider and is
> not an official SAP product**, unless and until SAP formally adopts or
> publishes it.

## Status

Pre-release, under active development toward `v0.1.0`. Schemas may still
change. See `ROADMAP.md` for what is planned and `CHANGELOG.md` for what
has landed.

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
  Subaccount                           Access Policies
  Entitlements                         Integration Packages
  Integration Suite subscription  -->  Integration Flows
  Service instances / bindings         Integration Flow Deployments
  Destinations                         (Current API Management,
  Role collections / assignments        Integration Cell, Edge Integration
                                         Cell as their public APIs are
                                         confirmed)
```

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

Legend: ✅ Supported · ⚠️ Partial support / important limitations · 👁️ Read-only / data source only · 🧪 Experimental · ❌ Unsupported / not implemented · ↗️ Planned as a separate Terraform provider

### Cloud Integration

| Feature | Status | Terraform Support |
|---|:---:|---|
| Archiving Configuration | ❌ | No safe Terraform lifecycle confirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| Custom Tag Configuration | ⚠️ | Resource + Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Data Store | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Data Store Entry | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Data Type | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| Design-Time Artifact Versioning | ⚠️ | Resource — see [feature-support.md](docs/feature-support.md#all-features) |
| Integration Adapter | ⚠️ | Resource + Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Integration Adapter Deployment | ⚠️ | Resource — see [feature-support.md](docs/feature-support.md#all-features) |
| Integration Flow | ✅ | Resource |
| Integration Flow Configuration | ✅ | Resource |
| Integration Flow Deployment | ✅ | Resource |
| Integration Package | ✅ | Resource + Data Source |
| Message Mapping | ✅ | Resource + Data Source |
| Message Mapping Deployment | ✅ | Resource |
| Message Processing Logs | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Message Store Entries / JMS Resources | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Message Type | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| Number Range | ✅ | Resource |
| Script Collection | ✅ | Resource + Data Source |
| Script Collection Deployment | ✅ | Resource |
| Service Endpoints | 👁️ | Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Service Interface | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| Value Mapping | ⚠️ | Resource + Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Value Mapping Deployment | ✅ | Resource |
| Value Mapping Entry | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| Variable | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |

### Security & Access Policies

| Feature | Status | Terraform Support |
|---|:---:|---|
| Access Policy | ✅ | Resource + Data Source |
| Access Policy Reference | ✅ | Resource + Data Source |
| Certificate | ✅ | Resource |
| Certificate Chain | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| Certificate-User Mapping | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| Key Pair | ⚠️ | Resource — see [feature-support.md](docs/feature-support.md#all-features) |
| Keystore Entry | 👁️ | Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Known Hosts (SSH) | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| OAuth2 Client Credential | ⚠️ | Resource + Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| OAuth2 Password Credentials | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| OAuth2 SAML Bearer Assertion | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| PGP Keyrings | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| Secure Parameter | ✅ | Resource |
| SSH Key | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| User Credential | ⚠️ | Resource + Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
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
| API Provider (classic API Management) | ⚠️ | Resource + Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| API Proxy (classic API Management) | 🧪 | Resource — see [feature-support.md](docs/feature-support.md#all-features) |
| API Proxy Deployment (classic API Management) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| API Management Application and Developer (Classic) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| API Management Cache Resource (Classic) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| API Management Certificate Store and Certificate (Classic) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| Certificate Store Reference (classic API Management) | ✅ | Resource + Data Source |
| API Management Key Value Map across API Proxies (Classic) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| Key Value Map (classic API Management) | ⚠️ | Resource + Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Policy (classic API Management) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| API Management Policy Template (Classic) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| API Management Product Access Control (Classic) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| API Management Rate Plan (Classic) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| API Management Virtual Host (Classic) | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |

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
| API Composition Business Data Graph | 🧪 | Resource + Data Source — see [feature-support.md](docs/feature-support.md#all-features) |

### Integration Cell

| Feature | Status | Terraform Support |
|---|:---:|---|
| Integration Cell Runtime | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| Integration Cell Virtual Host | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |

### Edge Integration Cell

| Feature | Status | Terraform Support |
|---|:---:|---|
| Edge Integration Cell Access Policy Replication | 👁️ | Data Source — see [feature-support.md](docs/feature-support.md#all-features) |
| Edge Integration Cell Runtime Targeting | ❌ | Planned — API details unconfirmed — see [feature-support.md](docs/feature-support.md#all-features) |
| Edge Integration Cell Local API Access | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Edge Integration Cell Registration | ❌ | No suitable public API — see [feature-support.md](docs/feature-support.md#all-features) |
| Edge Integration Cell Runtime Operations | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |

### Integration Assessment

| Feature | Status | Terraform Support |
|---|:---:|---|
| Integration Assessment Requests and Assessment Workflow | ❌ | Out of scope — see [feature-support.md](docs/feature-support.md#all-features) |
| Integration Assessment Landscape Configuration | ❌ | Research required — see [feature-support.md](docs/feature-support.md#all-features) |
| Integration Assessment Master Data | ❌ | Research required — see [feature-support.md](docs/feature-support.md#all-features) |

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
| Data Space Integration | ❌ | Research required — see [feature-support.md](docs/feature-support.md#all-features) |
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

21 supported · 10 partial · 4 read-only · 2 experimental · 60 unsupported · 2 planned as a separate provider, out of 99 evaluated Integration Suite features.

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
  or >= 1.11 if you use any resource with a write-only (`_wo`) secret
  attribute — currently `sapintegrationsuite_partner_user_credential_parameter`,
  `sapintegrationsuite_user_credential`, and
  `sapintegrationsuite_oauth2_client_credential`. This provider does not
  enforce a `required_version` constraint itself; set one in your own
  configuration if you rely on write-only attributes.
- An SAP Integration Suite tenant with Cloud Integration activated
- An OAuth 2.0 client credentials service key with the Integration Content /
  Security Content API scopes; Partner Directory resources additionally
  require the `AuthGroup_TenantPartnerDirectoryConfigurator` role
  (or `AuthGroup_Administrator`) — see `docs/guides/partner-directory.md`

## Installation

```hcl
terraform {
  required_providers {
    sapintegrationsuite = {
      source  = "Prideth/sap-integration-suite"
      version = "~> 0.1"
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
  package_id = sapintegrationsuite_integration_package.utilities.id
  flow_id    = sapintegrationsuite_integration_flow.metering.flow_id
}
```

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
- **Acceptance tests** exercise a real tenant and only run with `TF_ACC=1`
  and `SAP_INTEGRATION_SUITE_*` credentials set:

  ```shell
  TF_ACC=1 go test -v -timeout 60m ./...
  ```

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md). Every new resource must trace back
to an officially documented, SAP-supported public API — see
[`docs/sap-api-references.md`](docs/sap-api-references.md) for the pattern.

## Roadmap

See [`ROADMAP.md`](ROADMAP.md).

## Known limitations

- No public API for Integration Suite capability activation (Cloud
  Integration, API Management, Integration Cell, Edge Integration Cell) was
  found; activation stays a manual, one-time bootstrap step. See
  `docs/provisioning-capability-matrix.md`.
- `sapintegrationsuite_integration_flow`, `sapintegrationsuite_value_mapping`,
  `sapintegrationsuite_message_mapping`, and
  `sapintegrationsuite_script_collection`'s `content`/`content_hash`
  cannot be populated by `terraform import`, since SAP does not return a
  local file path for an existing design-time artifact; apply a matching
  configuration after import to bring content under management.
- `sapintegrationsuite_value_mapping` has no in-place update: changing
  `name`, `content`, or `content_hash` replaces the resource (creates a new
  artifact, then deletes the old one) rather than calling an unverified
  `PUT`. SAP separately documents a distinct
  `ValueMappingDesigntimeArtifactSaveAsVersion` action this provider does
  not yet use; implementing true in-place update through it is deferred to
  v0.2.x. See `docs/sap-api-references.md`.
  `sapintegrationsuite_message_mapping` does not share this limitation — it
  has a confirmed in-place update via `PUT`, on different, entity-specific
  evidence (see `docs/sap-api-references.md`).
- `sapintegrationsuite_message_mapping` is the reusable, package-level
  message mapping artifact, not the inline/local message mapping step
  configurable directly inside an integration flow — see
  `docs/resource-design.md` for the distinction.
- Individual value mapping entries (`UpsertValMaps`, `UpdateDefaultValMap`,
  `DeleteValMaps`) are not yet manageable through this provider — only the
  design-time artifact as a whole. See `docs/resource-design.md` for why.
- Whether Delete removes only the active version or every version of the
  artifact is unconfirmed for `sapintegrationsuite_value_mapping`,
  `sapintegrationsuite_message_mapping`, and
  `sapintegrationsuite_script_collection` — see `docs/sap-api-references.md`.
- Current API Management (API Artifacts, Integration Cell, Virtual Hosts,
  Runtime Profiles), Classic API Management, and Edge Integration Cell
  resources are not implemented: no public API was found for any current
  API Management object after a thorough research pass — see
  `docs/guides/current-api-management.md` and `ROADMAP.md`.
- There is no `sapintegrationsuite_partner` resource: SAP documents no
  confirmed create operation for Partner Directory `Partners`, and
  deleting one is documented as cascading to every entity that belongs to
  it. Use `data.sapintegrationsuite_partner` / `data.sapintegrationsuite_partners`
  for discovery instead. See `docs/guides/partner-directory.md`.
- `sapintegrationsuite_partner_user_credential_parameter` has no in-place
  update and never reads a password back from SAP — a permanent property
  of its security model. See `docs/guides/partner-directory.md`.
- `sapintegrationsuite_user_credential` and
  `sapintegrationsuite_oauth2_client_credential` never read a password or
  client secret back from SAP — the same permanent property, though unlike
  the Partner Directory credential these two do have a confirmed in-place
  update (a full redeploy). Only a handful of fields are exposed for
  `sapintegrationsuite_oauth2_client_credential` (grant type placement,
  client authentication mode, resource, audience, and custom parameters are
  not yet implemented). See `docs/guides/security-content.md`.
- Most Security Content artifact types remain unimplemented: keystore
  entries, certificates, key pairs, SSH keys, certificate chains, secure
  parameters, and known hosts are deferred pending confirmation of their
  exact OData `$metadata`. Certificate-to-user mapping has no public API
  for the Cloud Foundry environment this provider targets (Neo-only). OAuth2
  Authorization Code is out of scope on safety grounds (requires
  interactive human authorization). See `docs/guides/security-content.md`
  and `docs/feature-support.md`.
- Partner Directory data (string and binary parameters) is stored
  unencrypted by SAP; do not store secrets there.

## API support matrix

See [`docs/api-capability-matrix.md`](docs/api-capability-matrix.md) and
[`docs/provisioning-capability-matrix.md`](docs/provisioning-capability-matrix.md).

## Disclaimer

This project is an independent open-source Terraform provider for SAP
Integration Suite. It is not an official SAP product, and SAP has not
endorsed or certified it, unless SAP formally adopts or publishes it in the
future. It only uses officially documented, SAP-supported public APIs — see
`docs/sap-api-references.md`.

## License

[Apache License 2.0](LICENSE)
