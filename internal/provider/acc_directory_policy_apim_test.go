package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Partner Directory: every entry belongs to a partner ID of its own
// (tfacc<random>), which SAP creates with the first parameter and removes
// with the last.
func TestAccPartnerDirectory_basic(t *testing.T) {
	pid := testAccName()
	user := testAccName() // authorized users are tenant-wide and must be lowercase
	xml := filepath.Join(t.TempDir(), "routing.xml")
	if err := os.WriteFile(xml, []byte(`<routing><receiver>tfacc</receiver></routing>`), 0o600); err != nil {
		t.Fatal(err)
	}
	xmlPath := filepath.ToSlash(xml)
	config := func(value string) string {
		return fmt.Sprintf(`
resource "sapintegrationsuite_partner_string_parameter" "test" {
  partner_id   = %[1]q
  parameter_id = "tfacc_string"
  value        = %[3]q
}

resource "sapintegrationsuite_partner_binary_parameter" "test" {
  partner_id   = sapintegrationsuite_partner_string_parameter.test.partner_id
  parameter_id = "tfacc_binary"
  content_type = "xml"
  content      = %[4]q
  content_hash = filesha256(%[4]q)
}

resource "sapintegrationsuite_alternative_partner" "test" {
  partner_id  = sapintegrationsuite_partner_string_parameter.test.partner_id
  agency      = "tfacc_agency"
  scheme      = "tfacc_scheme"
  external_id = %[1]q
}

resource "sapintegrationsuite_partner_authorized_user" "test" {
  partner_id = sapintegrationsuite_partner_string_parameter.test.partner_id
  user       = %[2]q
}

resource "sapintegrationsuite_partner_user_credential_parameter" "test" {
  partner_id          = sapintegrationsuite_partner_string_parameter.test.partner_id
  parameter_id        = "tfacc_credential"
  user                = "tfacc-user"
  password_wo         = "tfacc-not-a-real-password"
  password_wo_version = "1"
}
`, pid, user, value, xmlPath)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("first"), Check: resource.TestCheckResourceAttr("sapintegrationsuite_partner_string_parameter.test", "value", "first")},
			{Config: config("second"), Check: resource.TestCheckResourceAttr("sapintegrationsuite_partner_string_parameter.test", "value", "second")},
			{ResourceName: "sapintegrationsuite_partner_string_parameter.test", ImportState: true, ImportStateVerify: true},
			{
				ResourceName: "sapintegrationsuite_partner_binary_parameter.test", ImportState: true, ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"content", "content_hash"},
			},
			{ResourceName: "sapintegrationsuite_alternative_partner.test", ImportState: true, ImportStateVerify: true},
			{ResourceName: "sapintegrationsuite_partner_authorized_user.test", ImportState: true, ImportStateVerify: true},
			{
				ResourceName: "sapintegrationsuite_partner_user_credential_parameter.test", ImportState: true, ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"password_wo_version"},
			},
		},
	})
}

// An access policy for a made-up role, with one reference; the description
// update uses PATCH (the tenant rejects PUT with 501).
func TestAccAccessPolicy_withReference(t *testing.T) {
	role := testAccName()
	config := func(description string) string {
		return fmt.Sprintf(`
resource "sapintegrationsuite_access_policy" "test" {
  role_name   = %[1]q
  description = %[2]q
}

resource "sapintegrationsuite_access_policy_reference" "test" {
  access_policy_id = sapintegrationsuite_access_policy.test.id
  name             = "tfacc reference"
  artifact_type    = "INTEGRATION_FLOW"
  attribute        = "Name"
  operator         = "exactString"
  value            = %[1]q
}
`, role, description)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("created"), Check: resource.TestCheckResourceAttr("sapintegrationsuite_access_policy.test", "role_name", role)},
			{Config: config("updated"), Check: resource.TestCheckResourceAttr("sapintegrationsuite_access_policy.test", "description", "updated")},
			{ResourceName: "sapintegrationsuite_access_policy.test", ImportState: true, ImportStateVerify: true},
			{ResourceName: "sapintegrationsuite_access_policy_reference.test", ImportState: true, ImportStateVerify: true},
		},
	})
}

// testAccAPIManagementPreCheck skips unless the API portal credentials are
// set.
func testAccAPIManagementPreCheck(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"SAP_INTEGRATION_SUITE_API_MANAGEMENT_HOST", "SAP_INTEGRATION_SUITE_API_MANAGEMENT_TOKEN_URL",
		"SAP_INTEGRATION_SUITE_API_MANAGEMENT_CLIENT_ID", "SAP_INTEGRATION_SUITE_API_MANAGEMENT_CLIENT_SECRET",
	} {
		if os.Getenv(name) == "" {
			t.Skipf("%s is not set", name)
		}
	}
}

// A key value map scoped to a proxy name that does not exist; SAP accepts
// that, and nothing outside the map is touched. Entries force a new map.
func TestAccAPIKeyValueMap_basic(t *testing.T) {
	name := testAccName()
	config := func(value string) string {
		return fmt.Sprintf(`
resource "sapintegrationsuite_api_key_value_map" "test" {
  name     = %[1]q
  scope    = "APIPROXY"
  scope_id = %[1]q
  entries = [
    { key = "tfacc", value = %[2]q },
  ]
}
`, name, value)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccAPIManagementPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("one"), Check: resource.TestCheckResourceAttr("sapintegrationsuite_api_key_value_map.test", "entries.0.value", "one")},
			{Config: config("two"), Check: resource.TestCheckResourceAttr("sapintegrationsuite_api_key_value_map.test", "entries.0.value", "two")},
			{ResourceName: "sapintegrationsuite_api_key_value_map.test", ImportState: true, ImportStateVerify: true},
		},
	})
}

// An API product needs an existing API proxy; name one in
// SAP_INTEGRATION_SUITE_ACC_API_PROXY. The product is replace-only.
func TestAccAPIProduct_basic(t *testing.T) {
	proxy := os.Getenv("SAP_INTEGRATION_SUITE_ACC_API_PROXY")
	if proxy == "" {
		t.Skip("SAP_INTEGRATION_SUITE_ACC_API_PROXY is not set")
	}
	name := testAccName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccAPIManagementPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "sapintegrationsuite_api_product" "test" {
  name            = %[1]q
  title           = "tfacc product"
  status_code     = "DRAFT"
  api_proxy_names = [%[2]q]
  additional_properties = [
    { name = "tfacc", value = "one" },
  ]
}
`, name, proxy),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sapintegrationsuite_api_product.test", "is_published", "false"),
					resource.TestCheckResourceAttr("sapintegrationsuite_api_product.test", "api_proxy_names.0", proxy),
				),
			},
			{ResourceName: "sapintegrationsuite_api_product.test", ImportState: true, ImportStateVerify: true},
		},
	})
}
