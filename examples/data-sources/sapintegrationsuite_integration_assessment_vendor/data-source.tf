# Unofficial: needs enable_unofficial = true and provider.integration_assessment.
# A vendor that already exists on the tenant, used without taking it over.
data "sapintegrationsuite_integration_assessment_vendor" "sap" {
  name = "SAP"
}

resource "sapintegrationsuite_integration_assessment_application" "s4hana" {
  name      = "SAP S/4HANA"
  vendor_id = data.sapintegrationsuite_integration_assessment_vendor.sap.id
}
