# Read a graph that is maintained elsewhere, for example in the
# Integration Suite UI, and check whether SAP processed it.
data "sapintegrationsuite_business_data_graph" "sales" {
  business_data_graph_identifier = "sales"
}

output "sales_graph_status" {
  value = data.sapintegrationsuite_business_data_graph.sales.status
}

output "sales_graph_data_sources" {
  value = [for ds in data.sapintegrationsuite_business_data_graph.sales.data_sources : ds.name]
}
