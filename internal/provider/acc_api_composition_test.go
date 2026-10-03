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

// envGraphNamespace is the namespace the test gives the destination's
// entities. SAP requires one for a custom OData service, which is what a
// test destination usually points at (for example a public OData sample
// service, which SAP's documentation allows with NoAuthentication for
// testing). Default: tfacc.custom.
const envGraphNamespace = "SAP_INTEGRATION_SUITE_ACC_GRAPH_NAMESPACE"

// The whole lifecycle of a business data graph over one existing
// destination: create (asynchronous, waits until SAP reports the graph
// active), change the excluded entities in place, import, destroy. This is
// the check that took the resource out of experimental (passed 2026-10-03).
func TestAccBusinessDataGraph_basic(t *testing.T) {
	accgate.Require(t, accgate.APIComposition, envGraphDestination)
	// The in-place update (PATCH) and the delete are unofficial operations.
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "true")
	id := testAccName()
	destination := os.Getenv(envGraphDestination)
	namespace := os.Getenv(envGraphNamespace)
	if namespace == "" {
		namespace = "tfacc.custom"
	}
	config := func(exclude string) string {
		return fmt.Sprintf(`
resource "sapintegrationsuite_business_data_graph" "test" {
  business_data_graph_identifier = %[1]q

  data_sources = [
    {
      name      = "tfacc"
      namespace = %[4]q
      services  = [{ destination_name = %[2]q }]
    },
  ]

  # The only data source leads for every entity of its namespace.
  locating_policy = {
    rules = [{ name = "%[4]s.*", leading = "tfacc" }]
  }

  exclude = [%[3]q]

  timeouts {
    create = "30m"
    update = "30m"
  }
}
`, id, destination, exclude, namespace)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(namespace + ".DoesNotExist"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sapintegrationsuite_business_data_graph.test", "business_data_graph_identifier", id),
					resource.TestCheckResourceAttrSet("sapintegrationsuite_business_data_graph.test", "status"),
				),
			},
			{
				Config: config(namespace + ".AlsoDoesNotExist"),
				Check:  resource.TestCheckResourceAttr("sapintegrationsuite_business_data_graph.test", "exclude.0", namespace+".AlsoDoesNotExist"),
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
