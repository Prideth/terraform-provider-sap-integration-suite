package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

// testAccCopySource is a small SAP standard package of the Discover catalog
// (one flow, which the test configures but never deploys). The tenant must not
// hold it in Design: the resource refuses to take over an existing copy.
const testAccCopySource = "ERPtoSuccessFactorsEmployeeCentralCostCenter"

// testAccCopyFlow is the one integration flow of testAccCopySource.
const testAccCopyFlow = "com.sap.ODTFINCC.fin2ec.SAPToSFSFCostCenterReplication.v1"

// Copies of standard packages: a copy with a configured parameter, an
// upgrade in place that keeps the parameter (OVERWRITE_MERGE), an import, an
// additional copy with a suffix, and an upgrade of that copy, which replaces
// it. Destroy deletes both copies.
func TestAccIntegrationPackageCopy_basic(t *testing.T) {
	accgate.Require(t, accgate.CloudIntegration)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_EXPERIMENTAL", "")
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "")
	suffix := "tfacc" + testAccName()[5:]
	const (
		copied     = "sapintegrationsuite_integration_package_copy.test"
		additional = "sapintegrationsuite_integration_package_copy.extra"
		configured = "sapintegrationsuite_integration_flow_configuration.test"
	)
	config := func(trigger, extraTrigger string) string {
		c := fmt.Sprintf(`
resource "sapintegrationsuite_integration_package_copy" "test" {
  source_package_id = %[1]q
  upgrade_trigger   = %[2]q
}

# Fixed values, not the copy's computed flows: those are unknown during an
# upgrade, which would make Terraform set the parameter again and hide
# whether SAP kept it.
resource "sapintegrationsuite_integration_flow_configuration" "test" {
  flow_id      = %[3]q
  flow_version = "active"
  parameters = {
    SFSF_EC_Logon_Credentials_Name = "tfacc-credential"
  }

  depends_on = [sapintegrationsuite_integration_package_copy.test]
}
`, testAccCopySource, trigger, testAccCopyFlow)
		if extraTrigger != "" {
			c += fmt.Sprintf(`
resource "sapintegrationsuite_integration_package_copy" "extra" {
  source_package_id = %[1]q
  suffix            = %[2]q
  upgrade_trigger   = %[3]q
}
`, testAccCopySource, suffix, extraTrigger)
		}
		return c
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("1", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(copied, "package_id", testAccCopySource),
					resource.TestCheckResourceAttr(copied, "vendor", "SAP"),
					resource.TestCheckResourceAttr(copied, "mode", "READ_ONLY"),
					resource.TestCheckResourceAttrSet(copied, "version"),
					resource.TestCheckResourceAttrSet(copied, "integration_flows.0.id"),
					resource.TestCheckResourceAttr(configured, "parameters.SFSF_EC_Logon_Credentials_Name", "tfacc-credential"),
				),
			},
			{
				// OVERWRITE_MERGE keeps the parameter: the configuration shows no
				// drift after the upgrade.
				Config:           config("2", ""),
				ConfigPlanChecks: expectInPlace(copied),
				Check:            resource.TestCheckResourceAttr(configured, "parameters.SFSF_EC_Logon_Credentials_Name", "tfacc-credential"),
			},
			{
				ResourceName:            copied,
				ImportState:             true,
				ImportStateId:           testAccCopySource,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"upgrade_trigger"},
			},
			{
				Config: config("2", "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(additional, "package_id", testAccCopySource+"."+suffix),
					resource.TestMatchResourceAttr(additional, "name", regexp.MustCompile(`\.`+suffix+`$`)),
					resource.TestMatchResourceAttr(additional, "integration_flows.0.id", regexp.MustCompile(`\.`+suffix+`$`)),
				),
			},
			{
				// SAP cannot overwrite a suffixed copy: the upgrade replaces it.
				Config: config("2", "2"),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(additional, plancheck.ResourceActionReplace),
					plancheck.ExpectResourceAction(copied, plancheck.ResourceActionNoop),
				}},
				Check: resource.TestCheckResourceAttr(additional, "package_id", testAccCopySource+"."+suffix),
			},
			{
				ResourceName:            additional,
				ImportState:             true,
				ImportStateId:           testAccCopySource + "/" + suffix,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"upgrade_trigger"},
			},
		},
	})
}
