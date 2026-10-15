# Needs provider.integration_assessment.
resource "sapintegrationsuite_integration_assessment_key_characteristic_value" "eu_only" {
  name                  = "EU only"
  description           = "Message data stays in EU data centers"
  key_characteristic_id = sapintegrationsuite_integration_assessment_key_characteristic.data_residency.id
}

resource "sapintegrationsuite_integration_assessment_key_characteristic_value" "any_region" {
  name                  = "Any region"
  key_characteristic_id = sapintegrationsuite_integration_assessment_key_characteristic.data_residency.id
}
