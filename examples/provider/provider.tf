terraform {
  # 1.11 is the first release with write-only attributes, which every
  # secret-bearing resource of this provider uses.
  required_version = ">= 1.11"

  required_providers {
    sapintegrationsuite = {
      source = "Prideth/sap-integration-suite"
      # A 0.x minor release may contain breaking changes; pin the minor
      # version and read the upgrade notes before raising it.
      version = "~> 0.6.0"
    }
  }
}

variable "integration_suite" {
  description = "The oauth section of the service key of a Process Integration Runtime instance with plan \"api\"."
  type = object({
    url          = string
    tokenurl     = string
    clientid     = string
    clientsecret = string
  })
  sensitive = true
}

variable "api_portal" {
  description = "Service key of the API Management, API portal instance with plan \"apiportal-apiaccess\"."
  type = object({
    url          = string
    tokenUrl     = string
    clientId     = string
    clientSecret = string
  })
  sensitive = true
}

provider "sapintegrationsuite" {
  # Cloud Integration, Security Content, Partner Directory and access
  # policies: one OAuth client from plan "api" of Process Integration Runtime.
  host = var.integration_suite.url

  oauth {
    token_url     = var.integration_suite.tokenurl
    client_id     = var.integration_suite.clientid
    client_secret = var.integration_suite.clientsecret
  }

  # Classic API Management uses its own client. Leave the block out when you
  # manage no API providers, products, proxies, certificate store references
  # or key value maps.
  api_management {
    host          = var.api_portal.url
    token_url     = var.api_portal.tokenUrl
    client_id     = var.api_portal.clientId
    client_secret = var.api_portal.clientSecret
  }

  # Both default to false. Turn a switch on only for the resources that need
  # it; see "Support status and opt-in switches" on this page.
  enable_experimental = false
  enable_unofficial   = false
}
