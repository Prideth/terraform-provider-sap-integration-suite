# No SAP host or OAuth credentials are needed for this data source: it
# answers entirely from this provider version's built-in feature catalog.
data "sapintegrationsuite_provider_features" "all" {}

output "supported_features" {
  value = [
    for f in data.sapintegrationsuite_provider_features.all.features :
    f.key
    if f.support_status == "supported"
  ]
}

output "unsupported_features" {
  value = [
    for f in data.sapintegrationsuite_provider_features.all.features :
    {
      feature = f.key
      reason  = f.support_reason
    }
    if f.support_status == "unsupported"
  ]
}

output "features_under_research" {
  value = [
    for f in data.sapintegrationsuite_provider_features.all.features :
    f.key
    if f.support_status == "research_required"
  ]
}
