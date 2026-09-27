# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
resource "sapintegrationsuite_integration_assessment_application" "warehouse" {
  name      = "ACME Warehouse Management"
  vendor_id = sapintegrationsuite_integration_assessment_vendor.acme.id
}
