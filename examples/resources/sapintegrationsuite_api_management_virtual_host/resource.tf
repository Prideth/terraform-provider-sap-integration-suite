# Needs the api_management_self_service block (a key with the role
# APIManagement.SelfService.Administrator) in the provider block.

# An additional host name for production APIs: prod-apis.<tenant domain>.
resource "sapintegrationsuite_api_management_virtual_host" "prod" {
  alias = "prod-apis"
}

# The full host name, for DNS documentation or API consumers.
output "prod_api_host" {
  value = sapintegrationsuite_api_management_virtual_host.prod.host_name
}
