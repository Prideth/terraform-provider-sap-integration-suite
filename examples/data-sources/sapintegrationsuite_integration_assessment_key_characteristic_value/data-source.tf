# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
variable "key_characteristic_name" {
  description = "Name of a key characteristic, as the Integration Assessment UI lists it."
  type        = string
}

variable "key_characteristic_value_name" {
  description = "Name of one of that key characteristic's values."
  type        = string
}

data "sapintegrationsuite_integration_assessment_key_characteristic_value" "selected" {
  key_characteristic = var.key_characteristic_name
  name               = var.key_characteristic_value_name
}

output "key_characteristic_value_id" {
  value = data.sapintegrationsuite_integration_assessment_key_characteristic_value.selected.id
}
