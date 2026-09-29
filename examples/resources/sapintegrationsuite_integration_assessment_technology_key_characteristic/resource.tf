# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
# Every change creates a new rating; the service refuses to update one.
resource "sapintegrationsuite_integration_assessment_technology_key_characteristic" "acme_esb" {
  technology_id               = sapintegrationsuite_integration_assessment_technology.acme_esb.id
  key_characteristic_value_id = data.sapintegrationsuite_integration_assessment_key_characteristic_value.selected.id
  recommendation_degree_id    = data.sapintegrationsuite_integration_assessment_recommendation_degree.selected.id
  description                 = "Rated by the integration architecture board"
}
