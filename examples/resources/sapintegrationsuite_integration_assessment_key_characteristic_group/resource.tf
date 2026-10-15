# Experimental: needs enable_experimental = true and provider.integration_assessment.
resource "sapintegrationsuite_integration_assessment_key_characteristic_group" "compliance" {
  name        = "Compliance"
  description = "Criteria from the company's integration guidelines"
}
