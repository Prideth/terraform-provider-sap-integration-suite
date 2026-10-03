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

0.4 adds the Integration Assessment resources and data sources, which are unofficial and
optional. To use them, set `enable_unofficial = true` and add the `integration_assessment` block
with the service key of an *Integration Assessment APIs* service instance; see the
[Integration Assessment guide](integration-assessment.md).

0.4 also contains the access policy checks of 0.3.1 and 0.3.2 (string lengths, reference values).
Coming from 0.3.0 or 0.3.1, read the next sections; nothing else needs to change.

## 0.3.1 to 0.3.2

0.3.2 only fixes defects; nothing is renamed and no state is migrated. Four more access policy
strings that SAP shortens silently are now checked while planning:

| Attribute | At most |
|---|---|
| `sapintegrationsuite_access_policy.role_name` | 200 characters |
| `sapintegrationsuite_access_policy_reference.name` | 50 characters |
| `sapintegrationsuite_access_policy_reference.description` | 200 characters |
| `sapintegrationsuite_access_policy_reference.value` | 150 characters |

Configurations within these limits need no change. A longer value fails `terraform plan` with an
error that names the limit and the length found. Such configurations never applied cleanly: SAP
stored only the first characters, and the apply ended with a value that differed from the plan.

- Shorten the value. A reference whose `value` lists many exact names is better written as one
  `regularExpression`, or split into several references.
- A policy or reference that an earlier apply left in SAP with a shortened value: if it is in the
  state, the next plan replaces it. If it is not, delete it in SAP, or import it and set the
  value to the text SAP kept.

## 0.3.0 to 0.3.1

0.3.1 only fixes defects; nothing is renamed and no state is migrated. Two kinds of configuration
that 0.3.0 accepted are now rejected while planning.

### Access policy descriptions longer than 200 characters

An access policy `description` longer than 200 characters is rejected. SAP keeps only the first
200 characters, and with 0.3.0 such an apply failed with "Provider produced inconsistent result
after apply" after the policy had been created, leaving it without its references.

If a description is longer, shorten it to 200 characters or fewer and run `terraform plan`
again. A policy that 0.3.0 already created with a truncated description:

- If it is in the state (the failed apply records it as tainted), the next plan replaces it.
  `terraform untaint <address>` keeps the existing policy instead; with
  `enable_unofficial = true` its description is then updated in place, otherwise set the
  description to the text SAP kept.
- If it is not in the state, import it with its numeric ID, or delete it in SAP, before applying;
  otherwise the create fails because the role name is already taken.

The references depending on the policy are created by the same apply.

### Access policy reference values

`artifact_type`, `attribute` and `operator` of `sapintegrationsuite_access_policy_reference`
accept only SAP's wire values, in the combinations SAP allows; the
[resource page](../resources/access_policy_reference.md) lists them. A UI label such as `MATCHES`
or `IntegrationPackage` fails the plan, and the error names the value to use, here
`regularExpression` and `INTEGRATION_PACKAGE`. Such configurations never worked: SAP rejected them
during apply, after the policy had been created. One exception: references to the eight artifact
types SAP accepts but does not document for access policies, such as `USER_CREDENTIAL`, worked
with 0.3.0 and now need `enable_unofficial = true` in the provider block. Existing references
are read and imported as SAP returns them.

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
