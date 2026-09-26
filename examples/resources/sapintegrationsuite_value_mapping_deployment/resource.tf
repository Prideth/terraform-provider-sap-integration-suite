resource "sapintegrationsuite_value_mapping_deployment" "company_codes" {
  package_id      = sapintegrationsuite_integration_package.utilities.id
  mapping_id      = sapintegrationsuite_value_mapping.company_codes.mapping_id
  mapping_version = sapintegrationsuite_value_mapping.company_codes.version

  # New content keeps the version, so redeploy whenever the uploaded file changes.
  redeploy_triggers = {
    content = sapintegrationsuite_value_mapping.company_codes.content_hash
  }

  timeouts {
    create = "10m"
    update = "10m"
    delete = "5m"
  }
}
