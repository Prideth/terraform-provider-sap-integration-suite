# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
variable "key_characteristic_group_name" {
  description = "Name of a key characteristic group, as the Integration Assessment UI lists it."
  type        = string
}

data "sapintegrationsuite_integration_assessment_key_characteristic_group" "selected" {
  name = var.key_characteristic_group_name
}

output "key_characteristic_group_id" {
  value = data.sapintegrationsuite_integration_assessment_key_characteristic_group.selected.id
}
