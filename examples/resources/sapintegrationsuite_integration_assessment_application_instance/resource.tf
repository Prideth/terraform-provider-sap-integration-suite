# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
variable "deployment_model_name" {
  description = "Name of one of SAP's deployment models, as the Integration Assessment UI lists them."
  type        = string
}

data "sapintegrationsuite_integration_assessment_deployment_model" "selected" {
  name = var.deployment_model_name
}

resource "sapintegrationsuite_integration_assessment_application_instance" "warehouse_prd" {
  name                = "ACME Warehouse Management PRD"
  description         = "Production system in the Frankfurt data center"
  application_id      = sapintegrationsuite_integration_assessment_application.warehouse.id
  deployment_model_id = data.sapintegrationsuite_integration_assessment_deployment_model.selected.id
}
