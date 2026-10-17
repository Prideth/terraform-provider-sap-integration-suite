data "sapintegrationsuite_integration_package_custom_tags" "orders" {
  package_id = "OrderProcessing"
}

# Tag name -> values, for example to check that every package has an owner.
output "order_package_tags" {
  value = { for t in data.sapintegrationsuite_integration_package_custom_tags.orders.tags : t.name => t.values }
}
