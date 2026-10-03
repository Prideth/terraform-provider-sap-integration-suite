package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

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

func testAccDataTypeConfig(pkg, id, description, xsd, saveAs string) string {
	save := ""
	if saveAs != "" {
		save = fmt.Sprintf("  save_as_version = %q\n", saveAs)
	}
	return testAccPackageConfig(pkg, "data type test") + fmt.Sprintf(`
resource "sapintegrationsuite_data_type" "test" {
  package_id   = sapintegrationsuite_integration_package.test.id
  data_type_id = %[1]q
  name         = %[1]q
  namespace    = "urn:tfacc:datatype"
  description  = %[2]q
  xsd          = %[3]q
%[4]s}
`, id, description, xsd, save)
}

// The whole lifecycle of a data type: create, change the schema and the
// description in place, import, save a version. SAP documents no request
// for data types; this test made the resource unofficial (passed 2026-10-03).
func TestAccDataType_basic(t *testing.T) {
	accgate.Require(t, accgate.CloudIntegration)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "true")
	pkg, id := testAccName(), testAccName()
	const name = "sapintegrationsuite_data_type.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataTypeConfig(pkg, id, "tf-acc data type", testAccDataTypeXSD(id, false), ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", pkg+"/"+id),
					resource.TestCheckResourceAttr(name, "namespace", "urn:tfacc:datatype"),
					resource.TestCheckResourceAttr(name, "description", "tf-acc data type"),
					resource.TestCheckResourceAttrSet(name, "version"),
				),
			},
			{
				Config: testAccDataTypeConfig(pkg, id, "tf-acc data type changed", testAccDataTypeXSD(id, true), ""),
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
				Config: testAccDataTypeConfig(pkg, id, "tf-acc data type changed", testAccDataTypeXSD(id, true), "1.0.5"),
				Check:  resource.TestCheckResourceAttr(name, "version", "1.0.5"),
			},
		},
	})
}
