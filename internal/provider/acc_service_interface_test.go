package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

// testAccServiceInterfaceConfig builds a package with two data types, a
// request and a response message type, a fault message type and a service
// interface whose operations are given as HCL.
func testAccServiceInterfaceConfig(pkg, dtReq, dtResp, mtReq, mtResp, fault, si, description, operations, saveAs string) string {
	save := ""
	if saveAs != "" {
		save = fmt.Sprintf("  save_as_version = %q\n", saveAs)
	}
	return testAccPackageConfig(pkg, "service interface test") + fmt.Sprintf(`
resource "sapintegrationsuite_data_type" "request" {
  package_id   = sapintegrationsuite_integration_package.test.id
  data_type_id = %[1]q
  name         = %[1]q
  namespace    = "urn:tfacc:mt"
  xsd          = %[8]q
}

resource "sapintegrationsuite_data_type" "response" {
  package_id   = sapintegrationsuite_integration_package.test.id
  data_type_id = %[2]q
  name         = %[2]q
  namespace    = "urn:tfacc:mt"
  xsd          = %[9]q
}

resource "sapintegrationsuite_message_type" "request" {
  package_id      = sapintegrationsuite_integration_package.test.id
  message_type_id = %[3]q
  name            = %[3]q
  namespace       = "urn:tfacc:mt"
  data_type_id    = sapintegrationsuite_data_type.request.data_type_id
}

resource "sapintegrationsuite_message_type" "response" {
  package_id      = sapintegrationsuite_integration_package.test.id
  message_type_id = %[4]q
  name            = %[4]q
  namespace       = "urn:tfacc:mt"
  data_type_id    = sapintegrationsuite_data_type.response.data_type_id
}

resource "sapintegrationsuite_fault_message_type" "fault" {
  package_id            = sapintegrationsuite_integration_package.test.id
  fault_message_type_id = %[5]q
  name                  = %[5]q
  namespace             = "urn:tfacc:mt"
  data_type_id          = sapintegrationsuite_data_type.request.data_type_id
}

resource "sapintegrationsuite_service_interface" "test" {
  package_id           = sapintegrationsuite_integration_package.test.id
  service_interface_id = %[6]q
  name                 = %[6]q
  namespace            = "urn:tfacc:si"
  description          = %[7]q
  operation            = %[10]s
%[11]s}
`, dtReq, dtResp, mtReq, mtResp, fault, si, description, testAccTypeXSD(dtReq), testAccTypeXSD(dtResp), operations, save)
}

const (
	// One asynchronous operation: request and fault.
	testAccSIAsync = `[
    {
      name                    = "Submit"
      request_message_type_id = sapintegrationsuite_message_type.request.message_type_id
      fault_message_type_ids  = [sapintegrationsuite_fault_message_type.fault.fault_message_type_id]
    },
  ]`
	// The same operation made synchronous, and a second, asynchronous one.
	testAccSISync = `[
    {
      name                     = "Submit"
      request_message_type_id  = sapintegrationsuite_message_type.request.message_type_id
      response_message_type_id = sapintegrationsuite_message_type.response.message_type_id
      fault_message_type_ids   = [sapintegrationsuite_fault_message_type.fault.fault_message_type_id]
    },
    {
      name                    = "Notify"
      request_message_type_id = sapintegrationsuite_message_type.request.message_type_id
    },
  ]`
)

// Service interfaces: create with an asynchronous operation, switch it to
// synchronous and add a second operation in place, import, save a version.
// It passed on 2026-10-04, including a synchronous operation written
// through the API, when service interfaces needed enable_unofficial; since
// SAP's Integration Content API specification documents them, it runs
// without it.
func TestAccServiceInterface_basic(t *testing.T) {
	accgate.Require(t, accgate.CloudIntegration)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "")
	pkg, dtReq, dtResp, mtReq, mtResp, fault, si := testAccName(), testAccName(), testAccName(), testAccName(), testAccName(), testAccName(), testAccName()
	const name = "sapintegrationsuite_service_interface.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceInterfaceConfig(pkg, dtReq, dtResp, mtReq, mtResp, fault, si, "tf-acc interface", testAccSIAsync, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "operation.#", "1"),
					resource.TestCheckResourceAttr(name, "operation.0.name", "Submit"),
					resource.TestCheckResourceAttr(name, "operation.0.request_message_type_id", mtReq),
					resource.TestCheckNoResourceAttr(name, "operation.0.response_message_type_id"),
					resource.TestCheckResourceAttr(name, "operation.0.fault_message_type_ids.0", fault),
					resource.TestCheckResourceAttr(name, "description", "tf-acc interface"),
					resource.TestCheckResourceAttrSet(name, "version"),
				),
			},
			{
				Config: testAccServiceInterfaceConfig(pkg, dtReq, dtResp, mtReq, mtResp, fault, si, "tf-acc interface changed", testAccSISync, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "operation.#", "2"),
					resource.TestCheckResourceAttr(name, "operation.0.response_message_type_id", mtResp),
					resource.TestCheckResourceAttr(name, "operation.1.name", "Notify"),
					resource.TestCheckNoResourceAttr(name, "operation.1.response_message_type_id"),
					resource.TestCheckResourceAttr(name, "description", "tf-acc interface changed"),
				),
			},
			{
				ResourceName:            name,
				ImportState:             true,
				ImportStateIdFunc:       testAccAttrImportID(name, "id"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"save_as_version"},
			},
			{
				Config: testAccServiceInterfaceConfig(pkg, dtReq, dtResp, mtReq, mtResp, fault, si, "tf-acc interface changed", testAccSISync, "1.0.3"),
				Check:  resource.TestCheckResourceAttr(name, "version", "1.0.3"),
			},
		},
	})
}
