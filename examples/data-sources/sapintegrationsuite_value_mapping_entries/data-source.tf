data "sapintegrationsuite_value_mapping_entries" "countries" {
  value_mapping_id  = "CountryCodes"
  source_agency     = "ERP"
  source_identifier = "Country"
  target_agency     = "CRM"
  target_identifier = "CountryCode"
}

# The target values of every source value.
output "country_codes" {
  value = {
    for e in data.sapintegrationsuite_value_mapping_entries.countries.entries :
    e.source_value => e.target_value...
  }
}
