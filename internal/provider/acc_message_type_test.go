package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

func testAccMessageTypeConfig(pkg, dt1, dt2, mt, fmt2, useDT, description, saveAs string) string {
	save := ""
	if saveAs != "" {
		save = fmt.Sprintf("  save_as_version = %q\n", saveAs)
	}
	return testAccPackageConfig(pkg, "message type test") + fmt.Sprintf(`
resource "sapintegrationsuite_data_type" "one" {
  package_id   = sapintegrationsuite_integration_package.test.id
  data_type_id = %[1]q
  name         = %[1]q
  namespace    = "urn:tfacc:mt"
  xsd          = %[6]q
}

resource "sapintegrationsuite_data_type" "two" {
  package_id   = sapintegrationsuite_integration_package.test.id
  data_type_id = %[2]q
  name         = %[2]q
  namespace    = "urn:tfacc:mt"
  xsd          = %[7]q
}

resource "sapintegrationsuite_message_type" "test" {
  package_id      = sapintegrationsuite_integration_package.test.id
  message_type_id = %[3]q
  name            = %[3]q
  namespace       = "urn:tfacc:mt"
  description     = %[8]q
  data_type_id    = sapintegrationsuite_data_type.%[5]s.data_type_id
%[9]s}

resource "sapintegrationsuite_fault_message_type" "test" {
  package_id            = sapintegrationsuite_integration_package.test.id
  fault_message_type_id = %[4]q
  name                  = %[4]q
  namespace             = "urn:tfacc:mt"
  description           = %[8]q
  data_type_id          = sapintegrationsuite_data_type.%[5]s.data_type_id
}
`, dt1, dt2, mt, fmt2, useDT, testAccTypeXSD(dt1), testAccTypeXSD(dt2), description, save)
}

func testAccTypeXSD(name string) string {
	return fmt.Sprintf(`<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns="urn:tfacc:mt" targetNamespace="urn:tfacc:mt">
  <xsd:complexType name=%q><xsd:sequence><xsd:element name="ID" type="xsd:string"/></xsd:sequence></xsd:complexType>
</xsd:schema>`, name)
}

// Message types and fault message types on data types: create, switch the
// data type and the description in place, import, save a version. SAP
// documents no request for either.
func TestAccMessageType_basic(t *testing.T) {
	accgate.Require(t, accgate.CloudIntegration)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_EXPERIMENTAL", "true")
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "true")
	pkg, dt1, dt2, mt, fm := testAccName(), testAccName(), testAccName(), testAccName(), testAccName()
	const mtName, fmName = "sapintegrationsuite_message_type.test", "sapintegrationsuite_fault_message_type.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMessageTypeConfig(pkg, dt1, dt2, mt, fm, "one", "tf-acc message", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mtName, "data_type_id", dt1),
					resource.TestCheckResourceAttr(mtName, "description", "tf-acc message"),
					resource.TestCheckResourceAttrSet(mtName, "version"),
					resource.TestCheckResourceAttr(fmName, "data_type_id", dt1),
				),
			},
			{
				Config: testAccMessageTypeConfig(pkg, dt1, dt2, mt, fm, "two", "tf-acc message changed", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mtName, "data_type_id", dt2),
					resource.TestCheckResourceAttr(mtName, "description", "tf-acc message changed"),
					resource.TestCheckResourceAttr(fmName, "data_type_id", dt2),
					resource.TestCheckResourceAttr(fmName, "description", "tf-acc message changed"),
				),
			},
			{
				ResourceName:            mtName,
				ImportState:             true,
				ImportStateIdFunc:       testAccAttrImportID(mtName, "id"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"save_as_version"},
			},
			{
				ResourceName:            fmName,
				ImportState:             true,
				ImportStateIdFunc:       testAccAttrImportID(fmName, "id"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"save_as_version"},
			},
			{
				Config: testAccMessageTypeConfig(pkg, dt1, dt2, mt, fm, "two", "tf-acc message changed", "1.0.3"),
				Check:  resource.TestCheckResourceAttr(mtName, "version", "1.0.3"),
			},
		},
	})
}
