package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

// envRuntimeLocation names the Edge Integration Cell the test addresses: the
// runtime location ID SAP shows in the monitoring URL after selecting the
// Edge Integration Cell as runtime.
const envRuntimeLocation = "SAP_INTEGRATION_SUITE_RUNTIME_LOCATION_ID"

// Security material and Partner Directory entries on an Edge Integration
// Cell, through the /location/<id>/api/v1 service root SAP Help documents.
// It checks create, in-place update, read and import with a location:<id>/
// import ID, and that both objects live on the edge runtime only. Until it
// passes on a tenant with an Edge Integration Cell, runtime_location_id stays
// unsupported.
func TestAccEdgeIntegrationCell_securityAndPartnerDirectory(t *testing.T) {
	accgate.Require(t, accgate.EdgeIntegrationCell)
	loc := os.Getenv(envRuntimeLocation)
	name, pid := testAccName(), testAccName()
	config := func(description, value string) string {
		return fmt.Sprintf(`
resource "sapintegrationsuite_user_credential" "edge" {
  runtime_location_id = %[1]q
  id                  = %[2]q
  description         = %[4]q
  user                = "tfacc-user"
  password_wo         = "tfacc-not-a-real-password"
  password_wo_version = "1"
}

resource "sapintegrationsuite_partner_string_parameter" "edge" {
  runtime_location_id = %[1]q
  partner_id          = %[3]q
  parameter_id        = "tfacc_string"
  value               = %[5]q
}
`, loc, name, pid, description, value)
	}
	locatedID := func(resourceName string, id func(*terraform.ResourceState) string) resource.ImportStateIdFunc {
		return func(s *terraform.State) (string, error) {
			rs, ok := s.RootModule().Resources[resourceName]
			if !ok {
				return "", fmt.Errorf("%s not in state", resourceName)
			}
			return importLocationPrefix + loc + "/" + id(rs), nil
		}
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("created", "first"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sapintegrationsuite_user_credential.edge", "runtime_location_id", loc),
					resource.TestCheckResourceAttr("sapintegrationsuite_partner_string_parameter.edge", "runtime_location_id", loc),
				),
			},
			{
				Config: config("updated", "second"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sapintegrationsuite_user_credential.edge", "description", "updated"),
					resource.TestCheckResourceAttr("sapintegrationsuite_partner_string_parameter.edge", "value", "second"),
				),
			},
			{
				ResourceName:      "sapintegrationsuite_user_credential.edge",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: locatedID("sapintegrationsuite_user_credential.edge", func(rs *terraform.ResourceState) string {
					return rs.Primary.Attributes["id"]
				}),
				ImportStateVerifyIgnore: []string{"password_wo_version"},
			},
			{
				ResourceName:      "sapintegrationsuite_partner_string_parameter.edge",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: locatedID("sapintegrationsuite_partner_string_parameter.edge", func(rs *terraform.ResourceState) string {
					return rs.Primary.Attributes["partner_id"] + "/" + rs.Primary.Attributes["parameter_id"]
				}),
			},
		},
	})
}
