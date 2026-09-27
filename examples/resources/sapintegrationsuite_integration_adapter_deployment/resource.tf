resource "sapintegrationsuite_integration_adapter_deployment" "sftp_extension" {
  adapter_id = sapintegrationsuite_integration_adapter.sftp_extension.id
}

# SAP documents that a custom adapter must be deployed before an
# integration flow consuming it is deployed. This provider does not infer
# that dependency automatically from integration flow content - declare it
# explicitly.
resource "sapintegrationsuite_integration_flow_deployment" "orders" {
  package_id   = sapintegrationsuite_integration_package.utilities.id
  flow_id      = sapintegrationsuite_integration_flow.orders.flow_id
  flow_version = sapintegrationsuite_integration_flow.orders.version

  depends_on = [
    sapintegrationsuite_integration_adapter_deployment.sftp_extension,
  ]
}
