# A standard package from the Discover catalog, copied into Design. Raise
# upgrade_trigger, for example to the version Discover shows, to bring the
# copy to SAP's current content; the flows keep their configuration.
resource "sapintegrationsuite_integration_package_copy" "cost_center" {
  source_package_id = "ERPtoSuccessFactorsEmployeeCentralCostCenter"
  upgrade_trigger   = "11.0.0"
}

# The copied flows are configured and deployed like any other flow.
resource "sapintegrationsuite_integration_flow_configuration" "cost_center" {
  flow_id      = sapintegrationsuite_integration_package_copy.cost_center.integration_flows[0].id
  flow_version = "active"
  parameters = {
    SFSF_EC_Logon_Credentials_Name = "SFSF_EC_CREDENTIALS"
  }
}

# An additional copy next to the original, for example to try a variant.
# SAP appends ".<suffix>" to the package ID, its name and the flow IDs.
resource "sapintegrationsuite_integration_package_copy" "cost_center_test" {
  source_package_id = "ERPtoSuccessFactorsEmployeeCentralCostCenter"
  suffix            = "test"
}
