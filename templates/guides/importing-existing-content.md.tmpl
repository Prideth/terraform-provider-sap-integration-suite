---
page_title: "Importing Existing Content"
subcategory: "Getting Started"
description: |-
  Bring packages, artifacts, deployments, security material and Partner Directory entries that
  already exist on a tenant under Terraform management: import IDs, discovery data sources and
  what an import cannot recover.
---

# Importing Existing Content

Most tenants already hold content when Terraform arrives. Every resource of this provider can
be imported, so existing objects can be adopted without deleting and re-creating them. This
guide lists the import IDs, the data sources that help find them, and what the first apply
after an import does.

## Import blocks or the CLI

With Terraform 1.5 or later, declare imports in configuration. They show up in the plan and
can be reviewed like any other change:

```terraform
import {
  to = sapintegrationsuite_integration_package.order_processing
  id = "ORDER_PROCESSING"
}

resource "sapintegrationsuite_integration_package" "order_processing" {
  id         = "ORDER_PROCESSING"
  name       = "Order Processing"
  short_text = "Order intake and confirmation"
}
```

`terraform import sapintegrationsuite_integration_package.order_processing ORDER_PROCESSING`
does the same from the command line. `terraform plan -generate-config-out=generated.tf`
writes a starting configuration for import blocks whose resource block does not exist yet;
review it before use, because it cannot contain file paths or secrets.

## Import IDs

| Resource | Import ID |
|---|---|
| `integration_package` | `<package_id>` |
| `integration_flow`, `message_mapping`, `script_collection`, `value_mapping` | `<package_id>/<artifact_id>` |
| `integration_flow_deployment`, `message_mapping_deployment`, `script_collection_deployment`, `value_mapping_deployment` | `<package_id>/<artifact_id>` |
| `integration_flow_configuration` | `<flow_id>/<flow_version>` |
| `integration_adapter` | `<package_id>/<adapter_id>` |
| `integration_adapter_deployment` | `<adapter_id>` |
| `custom_tag_configuration` | `CustomTags` |
| `number_range` | `<name>` (needs `enable_unofficial`) |
| `user_credential`, `oauth2_client_credential`, `secure_parameter` | `<name>` |
| `certificate`, `key_pair` | `<alias>` |
| `access_policy` | `<numeric policy ID>` |
| `access_policy_reference` | `<policy ID>/<reference ID>` |
| `partner_string_parameter`, `partner_binary_parameter`, `partner_user_credential_parameter` | `<partner_id>/<parameter_id>` |
| `partner_authorized_user` | `<user>` (lowercase) |
| `alternative_partner` | `<hex(agency)>/<hex(scheme)>/<hex(external_id)>` |
| `api_provider`, `api_product`, `api_management_certificate_store_reference` | `<name>` |
| `api_key_value_map` | `<name>/<scope>/<scope_id>` |
| `business_data_graph` | `<business_data_graph_identifier>` |
| `integration_assessment_vendor`, `_application`, `_application_instance`, `_technology`, `_technology_instance`, `_technology_domain`, `_technology_style`, `_technology_key_characteristic` | the UUID the service assigned |

The alternative partner uses hex encoding because agency, scheme and external ID may contain
`/`. `printf 'Sender_1' | xxd -p` prints the hex form of one part.

## Finding what to import

| Data source | Lists or finds |
|---|---|
| `sapintegrationsuite_integration_package` | one package by ID |
| `sapintegrationsuite_partners` | every partner ID in the Partner Directory |
| `sapintegrationsuite_partner_string_parameters` | every string parameter of one partner |
| `sapintegrationsuite_keystore_entries` | every keystore entry, with `owner` telling tenant entries from SAP's own |
| `sapintegrationsuite_access_policy` | a policy by `role_name`, which gives the numeric ID for the import |
| `sapintegrationsuite_access_policy_reference` | a reference with SAP's exact constants |
| `sapintegrationsuite_integration_adapter` | an adapter, including the `package_id` its import needs |
| `sapintegrationsuite_custom_tag_configuration` | the tenant's tag configuration |
| `sapintegrationsuite_api_providers` | every Classic API Management API provider |

Access policy IDs differ between tenants. Look them up by role name instead of copying numbers
between environments.

## What an import cannot recover

SAP does not return everything a configuration contains. The first plan after an import
therefore often shows a change even though nothing on the tenant differs:

- **Content files.** SAP returns no local file path and no hash for integration flows,
  mappings, script collections or adapters. After the import, `content` and
  `content_hash` are empty in state. If your configuration sets them, the first apply uploads
  that file once; make sure it is the content you want on the tenant. For message mappings and
  script collections, that upload is an in-place update and needs `enable_unofficial`; to adopt
  them without it, leave `content` and `content_hash` out until you change the content.
  Value mappings and adapters cannot be updated in place, so setting `content` after an import
  replaces them.
- **Secrets.** Passwords, client secrets and secure parameter values are write-only and never
  read back. After an import, the `*_wo_version` marker is empty, so the first apply is an
  in-place update that sends the configured secret. Have the correct secret in place before you
  apply.
- **Number range counters.** After an import, the first apply records `current_value_wo_version`
  without touching the counter. Change the marker once more to set the counter deliberately.
- **Key pairs.** Only the fields SAP returns are imported; the subject fields are filled from
  the subject DN. Generation parameters that SAP does not return (for example
  `signature_algorithm`) are trusted from the configuration. Any difference in a field that
  defines the key material forces a replacement, which generates a new key pair, so read the
  plan carefully.
- **Deployments.** A deployment is imported with the version SAP reports as deployed. If your
  configuration names another version, the first apply redeploys. `redeploy_triggers` is not
  known to SAP and starts empty, so a configuration that sets it also redeploys once.
- **Externalized parameters.** Importing `integration_flow_configuration` records every
  parameter the flow version has. List only the keys you want Terraform to manage. The first
  plan then shows the other keys leaving `parameters`; applying writes the listed keys again
  with their values and leaves the others unchanged on the tenant.

## Move state instead of re-importing

To rename a resource in configuration or move it into a module, use a `moved` block or
`terraform state mv`. Re-importing is not necessary and, for resources with secrets, would
send the secret again.

```terraform
moved {
  from = sapintegrationsuite_integration_flow.orders
  to   = module.order_processing.sapintegrationsuite_integration_flow.order_intake
}
```
