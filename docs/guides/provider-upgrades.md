---
page_title: "Provider Upgrades"
subcategory: "Getting Started"
description: |-
  What to change in configuration and state when moving between provider versions, one
  upgrade at a time.
---

# Provider Upgrades

The provider is on a 0.x release line. A minor release (0.2 to 0.3) may contain breaking
schema or lifecycle changes; a patch release only fixes defects. Pin the minor version:

```terraform
terraform {
  required_providers {
    sapintegrationsuite = {
      source  = "Prideth/sap-integration-suite"
      version = "~> 0.5.0"
    }
  }
}
```

Upgrade one minor version at a time. Run `terraform init -upgrade`, then `terraform plan`, and
read the plan before applying: every change below that forces a replacement is marked. The
complete list of changes is in the
[CHANGELOG](https://github.com/Prideth/terraform-provider-sap-integration-suite/blob/master/CHANGELOG.md).

## 0.4 to 0.5

Nothing to change, and no state migration. 0.5 only adds Integration Assessment types: the
technology profile resources (technology domain, style and key characteristic rating) and
read-only lookups for the whole ISA-M taxonomy. They are unofficial and optional, like the rest of
Integration Assessment; see the [Integration Assessment guide](integration-assessment.md).

## 0.3 to 0.4

Nothing to change. 0.4 only adds the Integration Assessment resources and data sources, which
are unofficial and optional. To use them, set `enable_unofficial = true` and add the
`integration_assessment` block with the service key of an *Integration Assessment APIs* service
instance; see the [Integration Assessment guide](integration-assessment.md).

## 0.2 to 0.3

### Experimental and unofficial resources need a switch

These types fail with "... is experimental" or "... is unofficial" until the provider block
enables them:

| Type | Switch |
|---|---|
| `sapintegrationsuite_business_data_graph` (resource and data source) | `enable_experimental = true` |
| `sapintegrationsuite_secure_parameter` | `enable_unofficial = true` |
| `sapintegrationsuite_access_policy_runtime_assignments` (data source) | `enable_unofficial = true` |

```terraform
provider "sapintegrationsuite" {
  enable_experimental = true
  enable_unofficial   = true
}
```

State and existing objects are not touched.

### Undocumented operations need `enable_unofficial`

Without the switch, a plan that needs one of these operations fails before anything is sent:

- an in-place update of a message mapping or script collection (new content or name);
- `save_as_version` when creating a script collection;
- a changed `description` of an access policy;
- an in-place update, a replacement or a destroy of a business data graph.

Number ranges keep working, but only with the create and update SAP documents. A refresh no
longer reads the number range, every update must change `current_value_wo_version`, and
destroy, replacement and import are refused. To keep the 0.2 behavior, set
`enable_unofficial = true`. To stop managing a number range without deleting it, run
`terraform state rm`.

## 0.1 to 0.2

0.1.0 could not authenticate under Terraform, so no SAP object can have been created or
imported with it. The upgrade changes configuration only. Attributes that were removed are
dropped from state on the first refresh.

| Resource or data source | Change |
|---|---|
| `sapintegrationsuite_integration_package` | Add `short_text` (required). In-place update. |
| `sapintegrationsuite_access_policy_reference` | Add `name` (required). Write `artifact_type = "INTEGRATION_FLOW"` instead of `"IntegrationFlow"` and `operator = "exactString"` instead of `"EQUALS"`. |
| `sapintegrationsuite_integration_adapter` and data source | Remove `type` and `application`. |
| `sapintegrationsuite_service_endpoints` | Read `api_definitions[*].name` instead of `api_definitions[*].type`. |
| `sapintegrationsuite_partner_authorized_user` and data source | Write `user` in lowercase. |
| `sapintegrationsuite_api_product` | Set `api_proxy_names` (required). Every attribute now forces a new product. |
| `sapintegrationsuite_access_policy` and data source | Remove references to `reconciliation_status`. |
| Keystore entry data sources | `valid_not_before` and `valid_not_after` are RFC 3339 timestamps. |

Before and after, for an access policy reference:

```terraform
# 0.1
resource "sapintegrationsuite_access_policy_reference" "order_flows" {
  access_policy_id = sapintegrationsuite_access_policy.order_team.id
  artifact_type    = "IntegrationFlow"
  attribute        = "Name"
  operator         = "EQUALS"
  value            = "ORDER_INTAKE"
}

# 0.2
resource "sapintegrationsuite_access_policy_reference" "order_flows" {
  access_policy_id = sapintegrationsuite_access_policy.order_team.id
  name             = "Order intake flow"
  artifact_type    = "INTEGRATION_FLOW"
  attribute        = "Name"
  operator         = "exactString"
  value            = "ORDER_INTAKE"
}
```
