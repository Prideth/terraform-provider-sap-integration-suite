# Requires provider.api_management to be configured. EXPERIMENTAL.
# The ZIP is an API proxy bundle as the API portal exports it; its
# APIProxy/Orders_v1.xml descriptor must declare the same name. A new
# content_hash deletes the proxy and imports the new bundle.
resource "sapintegrationsuite_api_proxy" "orders" {
  name         = "Orders_v1"
  content      = "${path.module}/proxies/Orders_v1.zip"
  content_hash = filesha256("${path.module}/proxies/Orders_v1.zip")
}

# Products link proxies by name.
resource "sapintegrationsuite_api_product" "orders" {
  name            = "Orders"
  title           = "Orders"
  api_proxy_names = [sapintegrationsuite_api_proxy.orders.name]
}
