terraform {
  required_providers {
    sapintegrationsuite = {
      source  = "Prideth/sap-integration-suite"
      version = "~> 0.1"
    }
  }
}

# Credentials can also be supplied via the SAP_INTEGRATION_SUITE_HOST,
# SAP_INTEGRATION_SUITE_TOKEN_URL, SAP_INTEGRATION_SUITE_CLIENT_ID, and
# SAP_INTEGRATION_SUITE_CLIENT_SECRET environment variables instead of the
# attributes below.
provider "sapintegrationsuite" {
  host = var.integration_suite_host

  # Both off by default. Resources and data sources whose status is
  # "experimental" (lifecycle not yet verified on a tenant) or "unofficial"
  # (works, but SAP does not document the API) refuse to run until the
  # matching switch is true, so nobody uses them by accident.
  # enable_unofficial also allows the undocumented operations of documented
  # resources, such as an in-place update of a message mapping's content or
  # deleting a number range. See docs/feature-support.md, "Contract
  # sources", for the list. Environment variables:
  # SAP_INTEGRATION_SUITE_ENABLE_EXPERIMENTAL, SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL.
  enable_experimental = false
  enable_unofficial   = false

  oauth {
    token_url     = var.integration_suite_token_url
    client_id     = var.integration_suite_client_id
    client_secret = var.integration_suite_client_secret
  }

  # Optional, and independent of the oauth block above - Classic API
  # Management (API Providers, API Proxies, API Products, Key Value Maps)
  # authenticates with its own API Portal application URL and OAuth 2.0
  # client (the apiportal-apiaccess service plan). Leave this entire block
  # out if you do not use any sapintegrationsuite_api_provider,
  # sapintegrationsuite_api_product, sapintegrationsuite_api_key_value_map,
  # or sapintegrationsuite_api_management_certificate_store_reference
  # resource or data source. All four values (or their
  # SAP_INTEGRATION_SUITE_API_MANAGEMENT_* environment variable
  # equivalents) must be supplied together, or all left unset.
  api_management {
    host          = var.api_management_host
    token_url     = var.api_management_token_url
    client_id     = var.api_management_client_id
    client_secret = var.api_management_client_secret
  }

  # Optional. Only sapintegrationsuite_business_data_graph uses this block.
  # The values come from a service key of an API Composition service
  # instance with plan "configuration"; the credentials above do not work
  # for the Configuration API. Set all four values or none.
  api_composition {
    host          = var.api_composition_host
    token_url     = var.api_composition_token_url
    client_id     = var.api_composition_client_id
    client_secret = var.api_composition_client_secret
  }
}
