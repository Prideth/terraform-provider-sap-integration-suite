resource "sapintegrationsuite_integration_flow_deployment" "orders" {
  package_id   = sapintegrationsuite_integration_package.utilities.id
  flow_id      = sapintegrationsuite_integration_flow.orders.flow_id
  flow_version = sapintegrationsuite_integration_flow.orders.version
}

# Every visible service endpoint, unfiltered.
data "sapintegrationsuite_service_endpoints" "all" {}

# Only the endpoint(s) generated for one specific deployed integration flow.
# depends_on is required here: this data source has no way to know it should
# wait for the deployment above, since service endpoints are discovered by
# name, not by a Terraform reference to the deployment resource itself.
data "sapintegrationsuite_service_endpoints" "orders" {
  name = "Order API"

  depends_on = [
    sapintegrationsuite_integration_flow_deployment.orders,
  ]
}

# Only SOAP endpoints, across every deployed artifact.
data "sapintegrationsuite_service_endpoints" "soap" {
  protocol = "SOAP"
}

output "order_api_entry_point_urls" {
  value = [
    for endpoint in data.sapintegrationsuite_service_endpoints.orders.endpoints :
    [for entry_point in endpoint.entry_points : entry_point.url]
  ]
}
