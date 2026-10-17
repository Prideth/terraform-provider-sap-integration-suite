package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

func testAccDataTypeXSD(id string, extraElement bool) string {
	extra := ""
	if extraElement {
		extra = `<xsd:element name="Customer" type="xsd:string" minOccurs="0"/>`
	}
	return fmt.Sprintf(`<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns="urn:tfacc:datatype" targetNamespace="urn:tfacc:datatype">
  <xsd:complexType name=%q>
    <xsd:sequence>
      <xsd:element name="OrderID" type="xsd:string"/>%s
    </xsd:sequence>
  </xsd:complexType>
</xsd:schema>`, id, extra)
}

func testAccDataTypeConfig(pkg, id, name, description, xsd, saveAs string) string {
	save := ""
	if saveAs != "" {
		save = fmt.Sprintf("  save_as_version = %q\n", saveAs)
	}
	return testAccPackageConfig(pkg, "data type test") + fmt.Sprintf(`
resource "sapintegrationsuite_data_type" "test" {
  package_id   = sapintegrationsuite_integration_package.test.id
  data_type_id = %[1]q
  name         = %[2]q
  namespace    = "urn:tfacc:datatype"
  description  = %[3]q
  xsd          = %[4]q
%[5]s}
`, id, name, description, xsd, save)
}

// The whole lifecycle of a data type: create, change the schema and the
// description in place, import, save a version, rename (a replacement:
// SAP ignores a name in the update, probe #73). It passed on 2026-10-03,
// when the resource needed enable_unofficial; since SAP's Integration
// Content API specification documents data types, it runs without it.
func TestAccDataType_basic(t *testing.T) {
	accgate.Require(t, accgate.CloudIntegration)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "")
	pkg, id := testAccName(), testAccName()
	const name = "sapintegrationsuite_data_type.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataTypeConfig(pkg, id, id, "tf-acc data type", testAccDataTypeXSD(id, false), ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", pkg+"/"+id),
					resource.TestCheckResourceAttr(name, "namespace", "urn:tfacc:datatype"),
					resource.TestCheckResourceAttr(name, "description", "tf-acc data type"),
					resource.TestCheckResourceAttrSet(name, "version"),
				),
			},
			{
				Config: testAccDataTypeConfig(pkg, id, id, "tf-acc data type changed", testAccDataTypeXSD(id, true), ""),
				Check:  resource.TestCheckResourceAttr(name, "description", "tf-acc data type changed"),
			},
			{
				ResourceName:            name,
				ImportState:             true,
				ImportStateIdFunc:       testAccAttrImportID(name, "id"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"xsd", "save_as_version"},
			},
			{
				Config: testAccDataTypeConfig(pkg, id, id, "tf-acc data type changed", testAccDataTypeXSD(id, true), "1.0.5"),
				Check:  resource.TestCheckResourceAttr(name, "version", "1.0.5"),
			},
			{
				Config: testAccDataTypeConfig(pkg, id, id+"Renamed", "tf-acc data type changed", testAccDataTypeXSD(id+"Renamed", true), "1.0.5"),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(name, plancheck.ResourceActionReplace),
				}},
				Check: resource.TestCheckResourceAttr(name, "name", id+"Renamed"),
			},
		},
	})
}
