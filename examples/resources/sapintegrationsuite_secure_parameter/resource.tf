# Unofficial: needs enable_unofficial = true in the provider block. The API
# behind it is known only from the service's $metadata, not from SAP's
# documentation.
variable "custom_adapter_api_key" {
  type      = string
  sensitive = true
}

# Integration flows and scripts reference the value through the alias
# CUSTOM_ADAPTER_API_KEY.
resource "sapintegrationsuite_secure_parameter" "custom_adapter_api_key" {
  id          = "CUSTOM_ADAPTER_API_KEY"
  description = "API key of the custom adapter's backend"

  # secure_param_wo is write-only: Terraform never stores it in plan or state.
  # Change secure_param_wo_version whenever the value itself changes, so
  # Terraform redeploys the artifact with the new value.
  secure_param_wo         = var.custom_adapter_api_key
  secure_param_wo_version = "1"
}
