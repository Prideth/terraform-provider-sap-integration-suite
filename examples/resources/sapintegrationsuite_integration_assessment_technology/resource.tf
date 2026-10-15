# Needs provider.integration_assessment. Changing a link in place also needs
# enable_unofficial = true.
# A technology of your own. SAP's technologies already exist on the tenant;
# look them up with the data source of the same name instead.
resource "sapintegrationsuite_integration_assessment_technology" "acme_esb" {
  name      = "ACME Enterprise Service Bus"
  vendor_id = sapintegrationsuite_integration_assessment_vendor.acme.id
}
