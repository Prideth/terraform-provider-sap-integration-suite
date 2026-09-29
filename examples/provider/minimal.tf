terraform {
  required_providers {
    sapintegrationsuite = {
      source  = "Prideth/sap-integration-suite"
      version = "~> 0.4.0"
    }
  }
}

# Everything comes from the environment:
#   SAP_INTEGRATION_SUITE_HOST           oauth.url of the service key (plan "api")
#   SAP_INTEGRATION_SUITE_TOKEN_URL      oauth.tokenurl
#   SAP_INTEGRATION_SUITE_CLIENT_ID      oauth.clientid
#   SAP_INTEGRATION_SUITE_CLIENT_SECRET  oauth.clientsecret
provider "sapintegrationsuite" {}

resource "sapintegrationsuite_integration_package" "order_processing" {
  id          = "ORDER_PROCESSING"
  name        = "Order Processing"
  short_text  = "Order intake and confirmation"
  description = "Integration flows that receive sales orders from the web shop and confirm them to the customer."
}
