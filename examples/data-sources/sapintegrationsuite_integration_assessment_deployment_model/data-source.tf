# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
variable "deployment_model_name" {
  description = "Name of one of SAP's deployment models, as the Integration Assessment UI lists them."
  type        = string
}

data "sapintegrationsuite_integration_assessment_deployment_model" "selected" {
  name = var.deployment_model_name
}

output "deployment_model_id" {
  value = data.sapintegrationsuite_integration_assessment_deployment_model.selected.id
}
