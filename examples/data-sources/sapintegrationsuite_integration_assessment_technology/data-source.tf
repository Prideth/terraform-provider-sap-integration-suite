# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
variable "technology_name" {
  description = "Exact name of one of the technologies the tenant lists."
  type        = string
}

data "sapintegrationsuite_integration_assessment_technology" "selected" {
  name = var.technology_name
}

resource "sapintegrationsuite_integration_assessment_technology_instance" "prd" {
  name                = "Integration technology PRD"
  technology_id       = data.sapintegrationsuite_integration_assessment_technology.selected.id
  deployment_model_id = data.sapintegrationsuite_integration_assessment_deployment_model.selected.id
}
