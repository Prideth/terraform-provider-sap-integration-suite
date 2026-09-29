---
page_title: "Integration Assessment"
subcategory: "Integration Assessment"
description: |-
  Maintaining the Integration Assessment landscape (vendors, applications, technologies and
  their instances) with Terraform: setup, the object model, status and what stays out of scope.
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

## What changes in place

Almost everything changes in place: names, an application's vendor (set, changed or removed),
an application instance's description, application and deployment model, a technology's vendor,
and a technology instance's name and deployment model. Only moving a technology instance to
another technology was not tested on a tenant, so it replaces the instance.
The service's limits apply: 20,000 applications, 20,000 application
instances, 50 technologies, 150 technology instances and 10,000 vendors per tenant.

## What stays out of scope

- **Requests and assessment results.** Business solution and interface requests follow a
  status workflow (`draft`, `new`, `in progress`, `completed`), and their results are reports.
  That is project state, not configuration.
- **Content import and export.** The Management API's `ImportContent` and `ExportContent`
  are one-shot transport actions.
- **The rest of the ISA-M taxonomy** (domains, styles, patterns, key characteristics) has no
  data source yet, and the domains, styles and key characteristics of a technology are not
  managed yet; maintain them in the UI.

## Importing existing objects

Every resource imports by the Id the service assigned. List existing objects in the UI or read
them with the data sources, then import them; see
[Importing Existing Content](importing-existing-content.md).
