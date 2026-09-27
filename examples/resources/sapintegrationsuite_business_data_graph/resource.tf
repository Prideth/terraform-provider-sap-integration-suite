# Experimental: needs enable_experimental = true in the provider block.
# Updating a graph in place and destroying it also need
# enable_unofficial = true, because SAP documents neither request.
# A graph over an S/4HANA system and SAP Sales Cloud. Both destinations
# must exist in the subaccount and carry the additional property
# IntegrationCell.Include = true. Needs provider.api_composition.
resource "sapintegrationsuite_business_data_graph" "sales" {
  business_data_graph_identifier = "sales"

  data_sources = [
    {
      name = "s4"
      # One root destination, several OData services below it.
      services = [
        { destination_name = "s4-root", path = "/odata/sap/API_BUSINESS_PARTNER" },
        { destination_name = "s4-root", path = "/odata/sap/API_PRODUCT_SRV" },
      ]
    },
    {
      name     = "c4c"
      services = [{ destination_name = "c4c-odata" }]
    },
  ]

  locating_policy = {
    # Sales Cloud refers to S/4 products through its own ExternalID field.
    key_mapping = [
      {
        foreign_key = {
          data_source = "c4c"
          entity_name = "sap.c4c.ProductCollection"
          attributes  = ["ExternalID"]
        }
        references = {
          data_source = "s4"
          entity_name = "sap.s4.A_Product"
          attributes  = ["Product"]
        }
      },
    ]

    rules = [
      { name = "sap.s4.*", leading = "s4" },
      { name = "sap.c4c.*", leading = "c4c" },
      { name = "sap.graph.*", leading = "s4", local = ["c4c"] },
      { name = "sap.graph.SalesQuote", leading = "c4c", local = ["s4"] },
    ]
  }

  # Keep Sales Cloud's own entities out of the graph's API.
  exclude = ["sap.c4c.*"]

  # SAP processes the graph asynchronously; the default wait is 20 minutes.
  timeouts {
    create = "30m"
  }
}
