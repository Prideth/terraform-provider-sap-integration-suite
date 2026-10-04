# Service interfaces are experimental: SAP documents no request for them.
# Needs enable_experimental = true in the provider block.
resource "sapintegrationsuite_service_interface" "orders" {
  package_id           = sapintegrationsuite_integration_package.sales.id
  service_interface_id = "OrderService"
  name                 = "OrderService"
  namespace            = "urn:example:sales"
  description          = "Order intake"

  operation = [
    {
      # Synchronous: a request, a response and a fault.
      name                     = "CreateOrder"
      request_message_type_id  = sapintegrationsuite_message_type.order.message_type_id
      response_message_type_id = sapintegrationsuite_message_type.confirmation.message_type_id
      fault_message_type_ids   = [sapintegrationsuite_fault_message_type.rejection.fault_message_type_id]
    },
    {
      # Asynchronous: a request only.
      name                    = "CancelOrder"
      request_message_type_id = sapintegrationsuite_message_type.cancellation.message_type_id
    },
  ]
}
