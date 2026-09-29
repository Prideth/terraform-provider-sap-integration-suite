# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
# Every change creates a new association; the service has no update for it.
resource "sapintegrationsuite_integration_assessment_technology_style" "acme_esb" {
  technology_id = sapintegrationsuite_integration_assessment_technology.acme_esb.id
  style_id      = data.sapintegrationsuite_integration_assessment_style.selected.id
}
