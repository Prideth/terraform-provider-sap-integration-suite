# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
variable "integration_pattern_name" {
  description = "Name of an integration pattern, as the Integration Assessment UI lists it."
  type        = string
}

data "sapintegrationsuite_integration_assessment_integration_pattern" "selected" {
  name = var.integration_pattern_name
}

output "integration_pattern_domain_and_style" {
  value = {
    domain_id = data.sapintegrationsuite_integration_assessment_integration_pattern.selected.domain_id
    style_id  = data.sapintegrationsuite_integration_assessment_integration_pattern.selected.style_id
  }
}
