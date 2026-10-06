package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

// selfServiceCredentials are the variables of provider.api_management_self_service:
// a key with the role APIManagement.SelfService.Administrator.
var selfServiceCredentials = []string{
	"SAP_INTEGRATION_SUITE_API_MANAGEMENT_SELF_SERVICE_HOST",
	"SAP_INTEGRATION_SUITE_API_MANAGEMENT_SELF_SERVICE_TOKEN_URL",
	"SAP_INTEGRATION_SUITE_API_MANAGEMENT_SELF_SERVICE_CLIENT_ID",
	"SAP_INTEGRATION_SUITE_API_MANAGEMENT_SELF_SERVICE_CLIENT_SECRET",
}

// An additional virtual host tfacc<random> on the default domain: created,
// renamed in place, imported by its alias and deleted. It adds a host name
// to the API portal and touches no other host.
func TestAccAPIManagementVirtualHost_basic(t *testing.T) {
	accgate.RequireService(t, accgate.APIManagementClassic, selfServiceCredentials...)
	alias := testAccName()
	config := func(alias string) string {
		return fmt.Sprintf(`
provider "sapintegrationsuite" {
  enable_experimental = true
}

resource "sapintegrationsuite_api_management_virtual_host" "test" {
  alias = %q
}
`, alias)
	}
	const address = "sapintegrationsuite_api_management_virtual_host.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(alias),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(address, "id"),
					resource.TestMatchResourceAttr(address, "host_name", regexp.MustCompile(`^`+alias+`\.`)),
					resource.TestCheckResourceAttr(address, "port", "443"),
					resource.TestCheckResourceAttr(address, "default", "false"),
				),
			},
			{
				Config: config(alias + "-2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "alias", alias+"-2"),
					resource.TestMatchResourceAttr(address, "host_name", regexp.MustCompile(`^`+alias+`-2\.`)),
				),
			},
			{
				ResourceName:      address,
				ImportState:       true,
				ImportStateId:     alias + "-2",
				ImportStateVerify: true,
			},
		},
	})
}
