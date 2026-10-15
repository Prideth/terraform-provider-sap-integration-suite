# The agency pair must exist in the value mapping's content; SAP cannot create
# a pair through the API. Destroying the resource leaves the entry in place.
resource "sapintegrationsuite_value_mapping_entry" "germany" {
  value_mapping_id  = sapintegrationsuite_value_mapping.countries.mapping_id
  source_agency     = "ERP"
  source_identifier = "Country"
  target_agency     = "CRM"
  target_identifier = "CountryCode"
  source_value      = "DE"
  target_value      = "276"
}

# A changed entry reaches the runtime only with a new deployment.
resource "sapintegrationsuite_value_mapping_deployment" "countries" {
  package_id      = sapintegrationsuite_value_mapping.countries.package_id
  mapping_id      = sapintegrationsuite_value_mapping.countries.mapping_id
  mapping_version = sapintegrationsuite_value_mapping.countries.version
  redeploy_triggers = {
    germany = "${sapintegrationsuite_value_mapping_entry.germany.source_value}=${sapintegrationsuite_value_mapping_entry.germany.target_value}"
  }
}
