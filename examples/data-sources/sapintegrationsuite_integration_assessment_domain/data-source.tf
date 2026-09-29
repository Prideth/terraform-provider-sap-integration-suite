# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
variable "domain_name" {
  description = "Name of an integration domain, as the Integration Assessment UI lists it."
  type        = string
}

data "sapintegrationsuite_integration_assessment_domain" "selected" {
  name = var.domain_name
}

output "domain_id" {
  value = data.sapintegrationsuite_integration_assessment_domain.selected.id
}
