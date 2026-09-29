# Requires provider.api_management to be configured. The proxies in
# api_proxy_names must already exist: create them in the SAP Integration
# Suite UI or transport them there.
# SAP cannot change a product after it is created, so changing any
# argument here makes Terraform delete the product and create a new one.
# Applications subscribed to the old product lose their subscription.
resource "sapintegrationsuite_api_product" "sample" {
  name        = "SampleProduct"
  title       = "Sample Product"
  description = "Sample product bundling the SampleAPI proxy"
  status_code = "PUBLISHED"

  api_proxy_names = ["SampleAPI"]

  additional_properties = [
    {
      name  = "team"
      value = "integration-platform"
    },
  ]
}
