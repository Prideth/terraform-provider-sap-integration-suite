# The runtime assignments data source is unofficial: it needs
# enable_unofficial = true in the provider block.
data "sapintegrationsuite_access_policy" "utilities" {
  role_name = "UTILITIES_ARCHITECT"
}

data "sapintegrationsuite_access_policy_runtime_assignments" "utilities" {
  access_policy_id = data.sapintegrationsuite_access_policy.utilities.id
}

# Runtimes where replication has not completed yet, with SAP's error text.
# transfer_status is passed through as SAP returns it; compare against the
# values you see in your own tenant.
output "utilities_policy_replication" {
  value = {
    for a in data.sapintegrationsuite_access_policy_runtime_assignments.utilities.assignments :
    a.runtime_location_id => {
      status = a.transfer_status
      errors = a.transfer_errors
      since  = a.status_updated_at
    }
  }
}
