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

// envGraphDestination2 names a second destination like envGraphDestination,
// for tests with two data sources. API Composition uses a destination for
// only one data source: with the same destination twice, the second data
// source took it and the first one stayed empty (2026-10-03).
const envGraphDestination2 = "SAP_INTEGRATION_SUITE_ACC_GRAPH_DESTINATION_2"

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

// graphTestNamespace is the namespace the acceptance tests give the
// destination's entities (envGraphNamespace, default tfacc.custom).
func graphTestNamespace() string {
	if ns := os.Getenv(envGraphNamespace); ns != "" {
		return ns
	}
	return "tfacc.custom"
}

// The settings named only in the $metadata: description, odata_containment
// and the locating policy's description. Created, changed in place and
// imported; every value must read back as configured.
func TestAccBusinessDataGraph_metadataSettings(t *testing.T) {
	accgate.Require(t, accgate.APIComposition, envGraphDestination)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "true")
	id := testAccName()
	destination := os.Getenv(envGraphDestination)
	namespace := graphTestNamespace()
	config := func(description string, containment bool, policyDescription string) string {
		return fmt.Sprintf(`
resource "sapintegrationsuite_business_data_graph" "test" {
  business_data_graph_identifier = %[1]q
  description                    = %[3]q
  odata_containment              = %[4]t

  data_sources = [
    {
      name      = "tfacc"
      namespace = %[2]q
      services  = [{ destination_name = %[5]q }]
    },
  ]

  locating_policy = {
    description = %[6]q
    rules       = [{ name = "%[2]s.*", leading = "tfacc" }]
  }

  timeouts {
    create = "30m"
    update = "30m"
  }
}
`, id, namespace, description, containment, destination, policyDescription)
	}
	const name = "sapintegrationsuite_business_data_graph.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("tfacc graph", false, "tfacc policy"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "description", "tfacc graph"),
					resource.TestCheckResourceAttr(name, "odata_containment", "false"),
					resource.TestCheckResourceAttr(name, "locating_policy.description", "tfacc policy"),
				),
			},
			{
				Config: config("tfacc graph changed", true, "tfacc policy changed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "description", "tfacc graph changed"),
					resource.TestCheckResourceAttr(name, "odata_containment", "true"),
					resource.TestCheckResourceAttr(name, "locating_policy.description", "tfacc policy changed"),
				),
			},
			{
				ResourceName:            name,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts", "status_details", "log_messages"},
			},
		},
	})
}

// A key mapping scoped by a cue, between two data sources with their own
// destinations, both pointing at the same OData sample service. SAP requires
// a description on every cue and, for a cue-scoped key mapping, locating
// rules with the same cue for both sides (HTTP 400 "No rule matches declared
// 'foreignKey'" otherwise, 2026-10-03). API Composition names a custom
// service's entities after its entity sets (namespace.Categories). The test
// checks that SAP accepts the cues and returns them; it does not query the
// graph.
func TestAccBusinessDataGraph_keyMappingCues(t *testing.T) {
	accgate.Require(t, accgate.APIComposition, envGraphDestination, envGraphDestination2)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "true")
	id := testAccName()
	destination := os.Getenv(envGraphDestination)
	destination2 := os.Getenv(envGraphDestination2)
	namespace := graphTestNamespace()
	config := fmt.Sprintf(`
resource "sapintegrationsuite_business_data_graph" "test" {
  business_data_graph_identifier = %[1]q

  data_sources = [
    { name = "tfacc", namespace = %[2]q, services = [{ destination_name = %[3]q }] },
    { name = "tfacc2", namespace = "%[2]s2", services = [{ destination_name = %[4]q }] },
  ]

  locating_policy = {
    cues = [{ name = "tfacc", description = "tfacc cue" }]
    key_mapping = [
      {
        cues        = ["tfacc"]
        foreign_key = { data_source = "tfacc", entity_name = "%[2]s.Products", attributes = ["CategoryID"] }
        references  = { data_source = "tfacc2", entity_name = "%[2]s2.Categories", attributes = ["CategoryID"] }
      },
    ]
    rules = [
      { name = "%[2]s.*", leading = "tfacc" },
      { name = "%[2]s2.*", leading = "tfacc2" },
      { name = "%[2]s.*", leading = "tfacc", cues = ["tfacc"] },
      { name = "%[2]s2.*", leading = "tfacc2", cues = ["tfacc"] },
    ]
  }

  timeouts {
    create = "30m"
  }
}
`, id, namespace, destination, destination2)
	const name = "sapintegrationsuite_business_data_graph.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "locating_policy.key_mapping.0.cues.#", "1"),
					resource.TestCheckResourceAttr(name, "locating_policy.key_mapping.0.cues.0", "tfacc"),
				),
			},
			{
				ResourceName:            name,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts", "status_details", "log_messages"},
			},
		},
	})
}
