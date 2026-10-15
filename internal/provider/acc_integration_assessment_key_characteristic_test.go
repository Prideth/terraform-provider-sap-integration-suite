package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

// iaKeyCharacteristicStep is what one step of the key characteristic test
// changes.
type iaKeyCharacteristicStep struct {
	suffix      string // appended to every name
	description string // of the group, the key characteristic and the first value
	isSingle    bool
}

// testAccIAKeyCharacteristicsConfig is a group of its own with a key
// characteristic and its values, a technology rated on a value, and the
// first value looked up again through the data source. A Multi key
// characteristic gets two values and the rating is on the second; a Single
// one gets one value, because SAP refuses a second (400 V170), and the
// rating is on it.
func testAccIAKeyCharacteristicsConfig(name, degree string, s iaKeyCharacteristicStep) string {
	second, rated := `
resource "sapintegrationsuite_integration_assessment_key_characteristic_value" "second" {
  name                  = "`+name+` value 2"
  key_characteristic_id = sapintegrationsuite_integration_assessment_key_characteristic.test.id
}
`, "second"
	if s.isSingle {
		second, rated = "", "test"
	}
	return fmt.Sprintf(`
resource "sapintegrationsuite_integration_assessment_key_characteristic_group" "test" {
  name        = "%[1]s group%[2]s"
  description = %[3]q
}

resource "sapintegrationsuite_integration_assessment_key_characteristic" "test" {
  name                        = "%[1]s characteristic%[2]s"
  description                 = %[3]q
  is_single                   = %[4]t
  key_characteristic_group_id = sapintegrationsuite_integration_assessment_key_characteristic_group.test.id
}

resource "sapintegrationsuite_integration_assessment_key_characteristic_value" "test" {
  name                  = "%[1]s value%[2]s"
  description           = %[3]q
  key_characteristic_id = sapintegrationsuite_integration_assessment_key_characteristic.test.id
}

%[6]s
data "sapintegrationsuite_integration_assessment_key_characteristic_value" "test" {
  key_characteristic = sapintegrationsuite_integration_assessment_key_characteristic.test.name
  name               = sapintegrationsuite_integration_assessment_key_characteristic_value.test.name
}

data "sapintegrationsuite_integration_assessment_recommendation_degree" "test" {
  name = %[5]q
}

resource "sapintegrationsuite_integration_assessment_vendor" "test" {
  name = "%[1]s vendor"
}

resource "sapintegrationsuite_integration_assessment_technology" "test" {
  name      = "%[1]s technology"
  vendor_id = sapintegrationsuite_integration_assessment_vendor.test.id
}

resource "sapintegrationsuite_integration_assessment_technology_key_characteristic" "test" {
  technology_id               = sapintegrationsuite_integration_assessment_technology.test.id
  key_characteristic_value_id = sapintegrationsuite_integration_assessment_key_characteristic_value.%[7]s.id
  recommendation_degree_id    = data.sapintegrationsuite_integration_assessment_recommendation_degree.test.id
}
`, name, s.suffix, s.description, s.isSingle, degree, second, rated)
}

// Custom key characteristics: a group of its own, a Multi key
// characteristic in it with two values, and a technology rated on one of
// them; then names and descriptions changed in place; then the type
// changed to Single with one value, which replaces the key characteristic
// and with it its value and the rating, because SAP refuses to update the
// type; then an import of the
// group, the key characteristic and a value. Destroy deletes
// the rating, the values, the key characteristic and the group in that
// order. It passed on 2026-10-15 with enable_experimental, which made the
// three resources supported; it runs without the switch since.
func TestAccIntegrationAssessment_keyCharacteristics(t *testing.T) {
	accgate.Require(t, accgate.IntegrationAssessment)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_EXPERIMENTAL", "")
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "")
	degree := testAccIATaxonomy(t).recommendationDegree
	name := testAccName()
	config := func(s iaKeyCharacteristicStep) string { return testAccIAKeyCharacteristicsConfig(name, degree, s) }
	const (
		group  = "sapintegrationsuite_integration_assessment_key_characteristic_group.test"
		kc     = "sapintegrationsuite_integration_assessment_key_characteristic.test"
		value  = "sapintegrationsuite_integration_assessment_key_characteristic_value.test"
		rating = "sapintegrationsuite_integration_assessment_technology_key_characteristic.test"
	)
	importStep := func(address string) resource.TestStep {
		return resource.TestStep{ResourceName: address, ImportState: true, ImportStateVerify: true}
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(iaKeyCharacteristicStep{description: "created"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(kc, "key_characteristic_group_id", group, "id"),
					resource.TestCheckResourceAttr(kc, "is_single", "false"),
					resource.TestCheckResourceAttrPair(value, "key_characteristic_id", kc, "id"),
					resource.TestCheckResourceAttrPair("data.sapintegrationsuite_integration_assessment_key_characteristic_value.test", "id", value, "id"),
					resource.TestCheckResourceAttrPair(rating, "key_characteristic_value_id",
						"sapintegrationsuite_integration_assessment_key_characteristic_value.second", "id"),
					resource.TestCheckResourceAttr(group, "description", "created"),
				),
			},
			{
				Config:           config(iaKeyCharacteristicStep{suffix: " renamed", description: "updated"}),
				ConfigPlanChecks: expectInPlace(group, kc, value),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(group, "name", name+" group renamed"),
					resource.TestCheckResourceAttr(group, "description", "updated"),
					resource.TestCheckResourceAttr(kc, "name", name+" characteristic renamed"),
					resource.TestCheckResourceAttr(value, "name", name+" value renamed"),
					resource.TestCheckResourceAttr(value, "description", "updated"),
				),
			},
			{
				Config: config(iaKeyCharacteristicStep{suffix: " renamed", description: "updated", isSingle: true}),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(kc, plancheck.ResourceActionReplace),
					plancheck.ExpectResourceAction(value, plancheck.ResourceActionReplace),
					plancheck.ExpectResourceAction(group, plancheck.ResourceActionNoop),
				}},
				Check: resource.TestCheckResourceAttr(kc, "is_single", "true"),
			},
			importStep(group),
			importStep(kc),
			importStep(value),
		},
	})
}
