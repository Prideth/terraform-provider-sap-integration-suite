variable "backend_client_secret" {
  type      = string
  sensitive = true
}

resource "sapintegrationsuite_oauth2_client_credential" "backend" {
  id                = "BACKEND_OAUTH"
  token_service_url = "https://auth.example.com/oauth/token"
  client_id         = "integration-client"
  scope             = "read write"

  # client_secret_wo is write-only: Terraform never stores it in plan or
  # state. SAP requires re-entering the client secret on every edit, so this
  # provider resends it on every apply that touches this resource, not only
  # when client_secret_wo_version changes.
  client_secret_wo         = var.backend_client_secret
  client_secret_wo_version = "1"
}

# A token service that needs extra parameters. custom_parameters needs
# enable_unofficial = true; SAP takes the parameters only when the
# credential is created, so every change of this resource replaces it.
resource "sapintegrationsuite_oauth2_client_credential" "graph" {
  id                = "GRAPH_OAUTH"
  token_service_url = "https://login.example.com/oauth2/v2.0/token"
  client_id         = "integration-client"

  custom_parameters = [
    { key = "resource", value = "https://graph.example.com", send_as_part_of = "body" },
  ]

  client_secret_wo         = var.backend_client_secret
  client_secret_wo_version = "1"
}
