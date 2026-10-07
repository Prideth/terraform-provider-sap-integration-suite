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

# A host for partners that must present a client certificate (mutual TLS).
# The truststore "partner-clients" holds their certificates or those of the
# CAs that issued them; it is created in the SAP Integration Suite UI
# (Configure, APIs, Certificates). The reference lets you rotate to a new
# truststore later without touching the host.
resource "sapintegrationsuite_api_management_certificate_store_reference" "partner_clients" {
  name                   = "partner-clients-current"
  certificate_store_name = "partner-clients"
}

resource "sapintegrationsuite_api_management_virtual_host" "partners" {
  alias               = "partner-apis"
  client_auth_enabled = true
  trust_store         = "ref://${sapintegrationsuite_api_management_certificate_store_reference.partner_clients.name}"
}
