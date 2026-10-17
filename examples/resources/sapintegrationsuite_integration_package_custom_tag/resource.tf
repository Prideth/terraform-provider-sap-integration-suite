# The tag must exist in the tenant's custom tag configuration. Depending on
# the configuration resource makes Terraform define the tag first.
resource "sapintegrationsuite_integration_package_custom_tag" "owner" {
  package_id = sapintegrationsuite_integration_package.orders.id
  name       = "Owner"
  values     = ["Finance"]

  depends_on = [sapintegrationsuite_custom_tag_configuration.governance]
}

# Several values keep their order.
resource "sapintegrationsuite_integration_package_custom_tag" "business_unit" {
  package_id = sapintegrationsuite_integration_package.orders.id
  name       = "BusinessUnit"
  values     = ["Sales", "Transport"]

  depends_on = [sapintegrationsuite_custom_tag_configuration.governance]
}
