# Message types are unofficial: SAP documents no request for them. Needs
# enable_unofficial = true in the provider block.
resource "sapintegrationsuite_message_type" "order" {
  package_id      = sapintegrationsuite_integration_package.sales.id
  message_type_id = "OrderMessage"
  name            = "OrderMessage" # also the root element; cannot change in place
  namespace       = "urn:example:sales"
  description     = "Sales order message"
  data_type_id    = sapintegrationsuite_data_type.order.data_type_id
}
