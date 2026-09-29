# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
variable "source_deployment_model_name" {
  description = "Deployment model the integration starts from, as the Integration Assessment UI lists it."
  type        = string
}

variable "target_deployment_model_name" {
  description = "Deployment model the integration goes to."
  type        = string
}

data "sapintegrationsuite_integration_assessment_deployment_model" "source" {
  name = var.source_deployment_model_name
}

data "sapintegrationsuite_integration_assessment_deployment_model" "target" {
  name = var.target_deployment_model_name
}

data "sapintegrationsuite_integration_assessment_domain_determination" "selected" {
  source_deployment_model_id = data.sapintegrationsuite_integration_assessment_deployment_model.source.id
  target_deployment_model_id = data.sapintegrationsuite_integration_assessment_deployment_model.target.id
}

output "determined_domain_id" {
  value = data.sapintegrationsuite_integration_assessment_domain_determination.selected.domain_id
}
