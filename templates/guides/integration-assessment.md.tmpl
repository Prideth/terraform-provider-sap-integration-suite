---
page_title: "Integration Assessment"
subcategory: "Integration Assessment"
description: |-
  Maintaining the Integration Assessment landscape (vendors, applications, technologies, their
  instances and technology profiles) with Terraform: setup, the object model, status and what
  stays out of scope.
---

# Integration Assessment

Integration Assessment implements SAP's Integration Solution Advisory Methodology (ISA-M). Its
assessments recommend integration technologies for interfaces between the systems of a
landscape, so the landscape has to be described first: which applications exist, where their
instances run, and which technologies are available. That description changes slowly and is
often kept in several places. Managing it with Terraform keeps it in one reviewed source.

## Status

The resources and data sources of this capability are **unofficial** and need
`enable_unofficial = true`. SAP documents the Integration Assessment APIs, their entities and
their per-tenant limits, and lists the Entities API on the Business Accelerator Hub, but the
field-level specification there needs an SAP login. The provider's requests follow the service's
`$metadata`, committed as a snapshot and compared with the live service by the API discovery,
and every operation was verified on a tenant in September 2026. They become supported once the
specification confirms them.

## Setup

Integration Assessment is its own BTP service with its own credentials:

1. Subscribe to Integration Assessment in the subaccount (SAP/btp provider or cockpit).
2. Create a service instance of **Integration Assessment APIs**, plan `default`, and a service
   key. The key has `entities`, `management`, `clientid`, `clientsecret` and `url`.
3. Configure the provider:

```terraform
provider "sapintegrationsuite" {
  enable_unofficial = true

  integration_assessment {
    entities_url  = var.integration_assessment.entities
    token_url     = "${var.integration_assessment.url}/oauth/token"
    client_id     = var.integration_assessment.clientid
    client_secret = var.integration_assessment.clientsecret
  }
}
```

The block is independent of `oauth`: a configuration that only maintains the landscape needs
no Cloud Integration credentials. The values can also come from the
`SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_*` environment variables.

## The landscape model

| Object | Resource | Links to |
|---|---|---|
| Vendor | [`integration_assessment_vendor`](../resources/integration_assessment_vendor.md) | — |
| Application | [`integration_assessment_application`](../resources/integration_assessment_application.md) | a vendor (optional) |
| Application instance | [`integration_assessment_application_instance`](../resources/integration_assessment_application_instance.md) | an application and a deployment model |
| Technology | [`integration_assessment_technology`](../resources/integration_assessment_technology.md) | a vendor |
| Technology instance | [`integration_assessment_technology_instance`](../resources/integration_assessment_technology_instance.md) | a technology and a deployment model |

Every object gets a UUID from the service. Configurations never write Ids literally: they
reference the resources, or look up objects that already exist by name with the
[vendor](../data-sources/integration_assessment_vendor.md),
[technology](../data-sources/integration_assessment_technology.md) and
[deployment model](../data-sources/integration_assessment_deployment_model.md) data sources.
SAP delivers its own technologies and the deployment models; look them up instead of creating
them.

```terraform
variable "deployment_model_name" {
  type = string
}

data "sapintegrationsuite_integration_assessment_deployment_model" "selected" {
  name = var.deployment_model_name
}

resource "sapintegrationsuite_integration_assessment_vendor" "acme" {
  name = "ACME Logistics"
}

resource "sapintegrationsuite_integration_assessment_application" "warehouse" {
  name      = "ACME Warehouse Management"
  vendor_id = sapintegrationsuite_integration_assessment_vendor.acme.id
}

resource "sapintegrationsuite_integration_assessment_application_instance" "warehouse_prd" {
  name                = "ACME Warehouse Management PRD"
  application_id      = sapintegrationsuite_integration_assessment_application.warehouse.id
  deployment_model_id = data.sapintegrationsuite_integration_assessment_deployment_model.selected.id
}
```

## Technology profiles

The assessment compares technologies by their profile: the integration domains and styles a
technology serves, and how strongly it meets each key characteristic value. For your own
technologies you maintain that profile with three association resources. They link a technology
to entries of SAP's taxonomy, which you look up by name:

| Association | Resource | Taxonomy lookup |
|---|---|---|
| Domain | [`integration_assessment_technology_domain`](../resources/integration_assessment_technology_domain.md) | [domain](../data-sources/integration_assessment_domain.md) |
| Style | [`integration_assessment_technology_style`](../resources/integration_assessment_technology_style.md) | [style](../data-sources/integration_assessment_style.md) |
| Key characteristic rating | [`integration_assessment_technology_key_characteristic`](../resources/integration_assessment_technology_key_characteristic.md) | [key characteristic value](../data-sources/integration_assessment_key_characteristic_value.md) and [recommendation degree](../data-sources/integration_assessment_recommendation_degree.md) |

A complete profile for your own technology: one domain, one style and one key characteristic
rating. The names are variables because they are SAP's taxonomy names as the Integration
Assessment UI shows them; the Ids behind them differ between tenants.

```terraform
variable "profile" {
  description = "Taxonomy names, as the Integration Assessment UI lists them."
  type = object({
    domain                   = string
    style                    = string
    key_characteristic       = string
    key_characteristic_value = string
    recommendation_degree    = string
  })
}

data "sapintegrationsuite_integration_assessment_domain" "selected" {
  name = var.profile.domain
}

data "sapintegrationsuite_integration_assessment_style" "selected" {
  name = var.profile.style
}

data "sapintegrationsuite_integration_assessment_key_characteristic_value" "selected" {
  key_characteristic = var.profile.key_characteristic
  name               = var.profile.key_characteristic_value
}

data "sapintegrationsuite_integration_assessment_recommendation_degree" "selected" {
  name = var.profile.recommendation_degree
}

resource "sapintegrationsuite_integration_assessment_vendor" "acme" {
  name = "ACME Logistics"
}

resource "sapintegrationsuite_integration_assessment_technology" "acme_esb" {
  name      = "ACME ESB"
  vendor_id = sapintegrationsuite_integration_assessment_vendor.acme.id
}

resource "sapintegrationsuite_integration_assessment_technology_domain" "acme_esb" {
  technology_id = sapintegrationsuite_integration_assessment_technology.acme_esb.id
  domain_id     = data.sapintegrationsuite_integration_assessment_domain.selected.id
}

resource "sapintegrationsuite_integration_assessment_technology_style" "acme_esb" {
  technology_id = sapintegrationsuite_integration_assessment_technology.acme_esb.id
  style_id      = data.sapintegrationsuite_integration_assessment_style.selected.id
}

resource "sapintegrationsuite_integration_assessment_technology_key_characteristic" "acme_esb" {
  technology_id               = sapintegrationsuite_integration_assessment_technology.acme_esb.id
  key_characteristic_value_id = data.sapintegrationsuite_integration_assessment_key_characteristic_value.selected.id
  recommendation_degree_id    = data.sapintegrationsuite_integration_assessment_recommendation_degree.selected.id
  description                 = "Rated by the integration architecture board"
}
```

A technology usually has several domains, styles and ratings; declare one resource per entry,
for example with `for_each` over a map of names.

The service has no update for these associations, so every change replaces one: a rating with a
new description or degree is deleted and created again. That is quick and has no side effects
on the technology.

## The ISA-M taxonomy

SAP ships the taxonomy of the Integration Solution Advisory Methodology as reference content.
The provider reads it with data sources and never manages it:

| Taxonomy entry | Data source | Found by |
|---|---|---|
| Deployment model | [`integration_assessment_deployment_model`](../data-sources/integration_assessment_deployment_model.md) | name |
| Domain | [`integration_assessment_domain`](../data-sources/integration_assessment_domain.md) | name |
| Style | [`integration_assessment_style`](../data-sources/integration_assessment_style.md) | name |
| Use case pattern | [`integration_assessment_use_case_pattern`](../data-sources/integration_assessment_use_case_pattern.md) | name; returns the style it refines |
| Integration pattern | [`integration_assessment_integration_pattern`](../data-sources/integration_assessment_integration_pattern.md) | name; returns its domain and style |
| Key characteristic group | [`integration_assessment_key_characteristic_group`](../data-sources/integration_assessment_key_characteristic_group.md) | name |
| Key characteristic value | [`integration_assessment_key_characteristic_value`](../data-sources/integration_assessment_key_characteristic_value.md) | key characteristic name and value name |
| Recommendation degree | [`integration_assessment_recommendation_degree`](../data-sources/integration_assessment_recommendation_degree.md) | name |
| Domain determination | [`integration_assessment_domain_determination`](../data-sources/integration_assessment_domain_determination.md) | source and target deployment model |

Names are compared exactly, including case, and must identify one entry. A key characteristic
value needs its key characteristic's name too, because value names repeat. A domain
determination has no name; it says which domain applies between two deployment models, so it is
found by that pair.

## What changes in place

Almost everything changes in place: names, an application's vendor (set, changed or removed),
an application instance's description, application and deployment model, a technology's vendor,
and a technology instance's name and deployment model. Only moving a technology instance to
another technology was not tested on a tenant, so it replaces the instance.
The service's limits apply: 20,000 applications, 20,000 application
instances, 50 technologies, 150 technology instances and 10,000 vendors per tenant.

## What stays out of scope

- **Requests and assessment results.** Business solution and interface requests, their line
  items, the technology decisions on them, the integration and message flows they describe and
  the interface request report follow a status workflow (`draft`, `new`, `in progress`,
  `completed`). Terraform manages desired configuration; an assessment request and its status
  are the state of a piece of work, not configuration, so none of these entities has a resource.
- **Content import and export.** The Management API's `ImportContent` and `ExportContent`
  are one-shot transport actions.
- **The taxonomy itself.** Domains, styles, patterns, key characteristics and the other ISA-M
  entries are SAP's reference content. They are looked up, never created or changed.

## Importing existing objects

Every resource imports by the Id the service assigned. List existing objects in the UI or read
them with the data sources, then import them; see
[Importing Existing Content](importing-existing-content.md).
