# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
# name and deployment_model_id change in place; a new technology_id creates a new instance.
resource "sapintegrationsuite_integration_assessment_technology_instance" "acme_esb_prd" {
  name                = "ACME ESB PRD"
  technology_id       = sapintegrationsuite_integration_assessment_technology.acme_esb.id
  deployment_model_id = data.sapintegrationsuite_integration_assessment_deployment_model.selected.id
}
