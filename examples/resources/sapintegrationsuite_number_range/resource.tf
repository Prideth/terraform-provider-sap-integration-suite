# SAP documents only Create (POST) and Update (PUT) for Number Ranges.
# Reading by name, delete and import were verified on a tenant but are
# unofficial and need enable_unofficial = true in the provider block.
# Without it, a refresh keeps the state, every update must change
# current_value_wo_version, and destroy and import are refused. See
# docs/guides/runtime-stores-and-number-ranges.md before adopting this
# resource.
resource "sapintegrationsuite_number_range" "invoice_numbers" {
  name        = "InvoiceNumbers"
  min_value   = "0"
  max_value   = "999999"
  description = "Interchange numbers for outbound EDIFACT invoices"
  rotate      = true

  # Zero-pads the displayed value to 6 digits (e.g. "000042").
  field_length = "6"

  # current_value_wo is only pushed to SAP when current_value_wo_version
  # changes from what is already in state. An ordinary apply that only
  # changes description/min_value/max_value/rotate/field_length never
  # touches the runtime counter, even though current_value_wo must still be
  # set on every apply (it is a write-only attribute Terraform never
  # stores).
  current_value_wo         = "0"
  current_value_wo_version = "initial"
}

# To deliberately (re)set the counter later, bump the version marker and
# provide the new value - this is the only thing that causes
# current_value_wo to actually be sent to SAP again:
#
# resource "sapintegrationsuite_number_range" "invoice_numbers" {
#   # ...unchanged static configuration...
#   current_value_wo         = "1000"
#   current_value_wo_version = "manual-correction-2026-01"
# }
