# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
variable "use_case_pattern_name" {
  description = "Name of a use case pattern, as the Integration Assessment UI lists it."
  type        = string
}

data "sapintegrationsuite_integration_assessment_use_case_pattern" "selected" {
  name = var.use_case_pattern_name
}

output "use_case_pattern_style_id" {
  description = "The integration style the pattern refines."
  value       = data.sapintegrationsuite_integration_assessment_use_case_pattern.selected.style_id
}
