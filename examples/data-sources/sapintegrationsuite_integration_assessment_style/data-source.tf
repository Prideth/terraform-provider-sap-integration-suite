# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
variable "style_name" {
  description = "Name of an integration style, as the Integration Assessment UI lists it."
  type        = string
}

data "sapintegrationsuite_integration_assessment_style" "selected" {
  name = var.style_name
}

output "style_id" {
  value = data.sapintegrationsuite_integration_assessment_style.selected.id
}
