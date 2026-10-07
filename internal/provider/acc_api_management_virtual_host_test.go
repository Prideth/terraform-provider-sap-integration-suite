package provider

import (
	"fmt"
	"os"
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

// envVirtualHostTrustStore names a truststore of the API portal that holds a
// client certificate (created in the UI: Configure, APIs, Certificates).
const envVirtualHostTrustStore = "SAP_INTEGRATION_SUITE_ACC_VIRTUAL_HOST_TRUST_STORE"

// A virtual host tfacc<random> with mutual TLS on the default domain:
// created with client authentication against the truststore, pointed at a
// certificate store reference to it, imported, switched off again and
// deleted. Needs the api_management key for the reference and the
// self-service key for the host.
func TestAccAPIManagementVirtualHost_mutualTLS(t *testing.T) {
	accgate.Require(t, accgate.APIManagementClassic, append([]string{envVirtualHostTrustStore}, selfServiceCredentials...)...)
	alias := testAccName()
	store := os.Getenv(envVirtualHostTrustStore)
	reference := fmt.Sprintf(`
resource "sapintegrationsuite_api_management_certificate_store_reference" "test" {
  name                   = %q
  certificate_store_name = %q
}
`, alias+"Ref", store)
	config := func(tls string) string {
		return reference + fmt.Sprintf(`
resource "sapintegrationsuite_api_management_virtual_host" "test" {
  alias = %q
  %s
}
`, alias, tls)
	}
	const address = "sapintegrationsuite_api_management_virtual_host.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(fmt.Sprintf("client_auth_enabled = true\n  trust_store = %q", store)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(address, "id"),
					resource.TestCheckResourceAttr(address, "client_auth_enabled", "true"),
					resource.TestCheckResourceAttr(address, "trust_store", store),
				),
			},
			{
				Config: config("client_auth_enabled = true\n  trust_store = \"ref://${sapintegrationsuite_api_management_certificate_store_reference.test.name}\""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "client_auth_enabled", "true"),
					resource.TestCheckResourceAttr(address, "trust_store", "ref://"+alias+"Ref"),
				),
			},
			{
				ResourceName:      address,
				ImportState:       true,
				ImportStateId:     alias,
				ImportStateVerify: true,
			},
			{
				Config: config(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "client_auth_enabled", "false"),
					resource.TestCheckNoResourceAttr(address, "trust_store"),
				),
			},
		},
	})
}

// An additional virtual host tfacc<random> on the default domain: created,
// renamed in place, imported by its alias and deleted. It adds a host name
// to the API portal and touches no other host.
func TestAccAPIManagementVirtualHost_basic(t *testing.T) {
	accgate.RequireService(t, accgate.APIManagementClassic, selfServiceCredentials...)
	alias := testAccName()
	config := func(alias string) string {
		return fmt.Sprintf(`
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
