resource "sapintegrationsuite_script_collection_deployment" "shared" {
  package_id                = sapintegrationsuite_integration_package.utilities.id
  script_collection_id      = sapintegrationsuite_script_collection.shared.script_collection_id
  script_collection_version = sapintegrationsuite_script_collection.shared.version

  # New content keeps the version, so redeploy whenever the uploaded file changes.
  redeploy_triggers = {
    content = sapintegrationsuite_script_collection.shared.content_hash
  }

  timeouts {
    create = "10m"
    update = "10m"
    delete = "5m"
  }
}
