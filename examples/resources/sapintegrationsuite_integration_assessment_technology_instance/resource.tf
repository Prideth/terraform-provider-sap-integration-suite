# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
# Every change creates a new instance; an update was not tested.
resource "sapintegrationsuite_integration_assessment_technology_instance" "acme_esb_prd" {
  name                = "ACME ESB PRD"
  technology_id       = sapintegrationsuite_integration_assessment_technology.acme_esb.id
  deployment_model_id = data.sapintegrationsuite_integration_assessment_deployment_model.selected.id
}
