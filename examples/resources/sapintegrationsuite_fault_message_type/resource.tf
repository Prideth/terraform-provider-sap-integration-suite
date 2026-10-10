resource "sapintegrationsuite_fault_message_type" "order_rejected" {
  package_id            = sapintegrationsuite_integration_package.sales.id
  fault_message_type_id = "OrderRejected"
  name                  = "OrderRejected"
  namespace             = "urn:example:sales"
  description           = "Order could not be processed"
  data_type_id          = sapintegrationsuite_data_type.rejection_detail.data_type_id
}
