# Experimental: needs enable_experimental = true and provider.integration_assessment.
resource "sapintegrationsuite_integration_assessment_key_characteristic" "data_residency" {
  name                        = "Data residency"
  description                 = "Where the integration technology processes and stores message data"
  key_characteristic_group_id = sapintegrationsuite_integration_assessment_key_characteristic_group.compliance.id
}

# A Single key characteristic: a question answered with yes or no.
resource "sapintegrationsuite_integration_assessment_key_characteristic" "audit_log" {
  name                        = "Audit log"
  is_single                   = true
  key_characteristic_group_id = sapintegrationsuite_integration_assessment_key_characteristic_group.compliance.id
}
