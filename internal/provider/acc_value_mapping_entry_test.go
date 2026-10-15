package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/samples"
)

// Value mapping entries: a synthetic value mapping with the
// agency pair tfacc_source/code_tfacc -> tfacc_target/code_tfacc (A -> 1,
// B -> 2) gets the entry X -> 9, which the data source lists and the
// deployment's redeploy_triggers follow; a target change fails without
// enable_unofficial and changes the entry in place with it; then an import.
// Destroy leaves the entry in the value mapping, which is deleted with the
// package anyway. It passed on 2026-10-15 with enable_experimental, which
// made the resource partial; it runs without the switch since.
func TestAccValueMappingEntry_basic(t *testing.T) {
	accgate.Require(t, accgate.CloudIntegration, samples.EnvLocalContent)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_EXPERIMENTAL", "")
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "")
	vm := samples.LocalArtifactOfType(t, "ValueMapping")
	synthetic, err := samples.SyntheticValueMapping(vm.Content, "tfacc")
	if err != nil {
		t.Fatal(err)
	}
	pkg, id := testAccName(), testAccName()
	path := testAccArtifactFile(t, synthetic, id)
	const (
		entry    = "sapintegrationsuite_value_mapping_entry.test"
		entries  = "data.sapintegrationsuite_value_mapping_entries.test"
		deployed = "sapintegrationsuite_value_mapping_deployment.test"
	)
	config := func(target string, unofficial bool) string {
		c := ""
		if unofficial {
			c = `
provider "sapintegrationsuite" {
  enable_unofficial = true
}
`
		}
		return c + testAccPackageConfig(pkg, "value mapping entry test") + fmt.Sprintf(`
resource "sapintegrationsuite_value_mapping" "test" {
  package_id   = sapintegrationsuite_integration_package.test.id
  mapping_id   = %[1]q
  name         = "tf-acc %[1]s"
  content      = %[2]q
  content_hash = filesha256(%[2]q)
}

resource "sapintegrationsuite_value_mapping_entry" "test" {
  value_mapping_id  = sapintegrationsuite_value_mapping.test.mapping_id
  source_agency     = "tfacc_source"
  source_identifier = "code_tfacc"
  target_agency     = "tfacc_target"
  target_identifier = "code_tfacc"
  source_value      = "X"
  target_value      = %[3]q
}

data "sapintegrationsuite_value_mapping_entries" "test" {
  value_mapping_id  = sapintegrationsuite_value_mapping_entry.test.value_mapping_id
  source_agency     = "tfacc_source"
  source_identifier = "code_tfacc"
  target_agency     = "tfacc_target"
  target_identifier = "code_tfacc"
}

resource "sapintegrationsuite_value_mapping_deployment" "test" {
  package_id      = sapintegrationsuite_integration_package.test.id
  mapping_id      = sapintegrationsuite_value_mapping.test.mapping_id
  mapping_version = sapintegrationsuite_value_mapping.test.version
  redeploy_triggers = {
    entry = "${sapintegrationsuite_value_mapping_entry.test.source_value}=${sapintegrationsuite_value_mapping_entry.test.target_value}"
  }

  timeouts {
    create = "15m"
    delete = "10m"
  }
}
`, id, path, target)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("9", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(entry, "entry_id"),
					resource.TestCheckResourceAttr(entries, "entries.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs(entries, "entries.*", map[string]string{"source_value": "X", "target_value": "9"}),
					resource.TestCheckResourceAttr(deployed, "status", "STARTED"),
				),
				// The entry turns the value mapping into a Draft after the
				// deployment read its version, so the next plan deploys it.
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config("9", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(deployed, "mapping_version", "Draft"),
					resource.TestCheckResourceAttr(deployed, "status", "STARTED"),
				),
			},
			{
				Config:      config("8", false),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`entry change in place \(UpsertValMaps with ValMapId\)\s+is\s+unofficial`),
			},
			{
				Config:           config("8", true),
				ConfigPlanChecks: expectInPlace(entry, deployed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(entry, "target_value", "8"),
					resource.TestCheckTypeSetElemNestedAttrs(entries, "entries.*", map[string]string{"source_value": "X", "target_value": "8"}),
					resource.TestCheckResourceAttr(deployed, "status", "STARTED"),
				),
			},
			{
				ResourceName:      entry,
				ImportState:       true,
				ImportStateIdFunc: testAccAttrImportID(entry, "id"),
				ImportStateVerify: true,
			},
		},
	})
}
