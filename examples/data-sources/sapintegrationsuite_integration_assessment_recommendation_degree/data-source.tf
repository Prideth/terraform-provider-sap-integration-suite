# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
variable "recommendation_degree_name" {
  description = "Name of a recommendation degree, as the Integration Assessment UI lists it."
  type        = string
}

data "sapintegrationsuite_integration_assessment_recommendation_degree" "selected" {
  name = var.recommendation_degree_name
}

output "recommendation_degree_id" {
  value = data.sapintegrationsuite_integration_assessment_recommendation_degree.selected.id
}
