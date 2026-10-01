resource "sapintegrationsuite_access_policy_reference" "metering_flow" {
  access_policy_id = sapintegrationsuite_access_policy.utilities.id

  name        = "Metering flow"
  description = "The metering integration flow of the utilities package"

  artifact_type = "INTEGRATION_FLOW"
  attribute     = "Name"
  operator      = "exactString"
  value         = "Metering"
}

# "Matches" in the UI is the wire value "regularExpression". The value is a
# Java regular expression: ".*" stands for any characters, so this matches
# every integration flow whose name starts with SALES_ORDERS_.
resource "sapintegrationsuite_access_policy_reference" "core_its_flows" {
  access_policy_id = sapintegrationsuite_access_policy.utilities.id

  name          = "Sales order integration flows"
  artifact_type = "INTEGRATION_FLOW"
  attribute     = "Name"
  operator      = "regularExpression"
  value         = "^SALES_ORDERS_.*$"
}
