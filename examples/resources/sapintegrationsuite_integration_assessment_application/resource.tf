# Needs provider.integration_assessment. Changing a link in place also needs
# enable_unofficial = true.
resource "sapintegrationsuite_integration_assessment_application" "warehouse" {
  name      = "ACME Warehouse Management"
  vendor_id = sapintegrationsuite_integration_assessment_vendor.acme.id
}
