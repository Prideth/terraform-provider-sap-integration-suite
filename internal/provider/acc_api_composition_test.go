package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

// envGraphDestination names a BTP destination of the API Composition
// subaccount that points at an OData service and carries the additional
// property IntegrationCell.Include = true. The test only reads through it.
const envGraphDestination = "SAP_INTEGRATION_SUITE_ACC_GRAPH_DESTINATION"

// The whole lifecycle of a business data graph over one existing
// destination: create (asynchronous, waits until SAP reports the graph
// active), change the excluded entities in place, import, destroy. This is
// the check the resource needs before it can leave experimental.
func TestAccBusinessDataGraph_basic(t *testing.T) {
	accgate.Require(t, accgate.APIComposition, envGraphDestination)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_EXPERIMENTAL", "true")
	id := testAccName()
	destination := os.Getenv(envGraphDestination)
	config := func(exclude string) string {
		return fmt.Sprintf(`
resource "sapintegrationsuite_business_data_graph" "test" {
  business_data_graph_identifier = %[1]q

  data_sources = [
    {
      name     = "tfacc"
      services = [{ destination_name = %[2]q }]
    },
  ]

  exclude = [%[3]q]

  timeouts {
    create = "30m"
    update = "30m"
  }
}
`, id, destination, exclude)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("sap.tfacc.DoesNotExist"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sapintegrationsuite_business_data_graph.test", "business_data_graph_identifier", id),
					resource.TestCheckResourceAttrSet("sapintegrationsuite_business_data_graph.test", "status"),
				),
			},
			{
				Config: config("sap.tfacc.AlsoDoesNotExist"),
				Check:  resource.TestCheckResourceAttr("sapintegrationsuite_business_data_graph.test", "exclude.0", "sap.tfacc.AlsoDoesNotExist"),
			},
			{
				ResourceName:            "sapintegrationsuite_business_data_graph.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts", "status_details", "log_messages"},
			},
		},
	})
}
