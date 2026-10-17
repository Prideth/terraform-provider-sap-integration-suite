package provider

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

// Custom tag values of a package: two values on a tag of the tenant's
// configuration (the first optional tag without permitted values, in name
// order; the test never changes the tenant-wide configuration), read back in
// order and by the data source; a package update that keeps them; a change
// in place; an import; a value with "|" refused at plan time; and the tag
// cleared once the resource is gone while the package stays. Needs at least
// one such tag on the tenant.
func TestAccIntegrationPackageCustomTag_basic(t *testing.T) {
	accgate.Require(t, accgate.CloudIntegration)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_EXPERIMENTAL", "")
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "")
	pkg := testAccName()
	const (
		tag  = "sapintegrationsuite_integration_package_custom_tag.test"
		tags = "data.sapintegrationsuite_integration_package_custom_tags.test"
	)
	base := func(description string) string {
		return testAccPackageConfig(pkg, description) + `
data "sapintegrationsuite_custom_tag_configuration" "tenant" {}

locals {
  tag = sort([
    for t in data.sapintegrationsuite_custom_tag_configuration.tenant.tags : t.name
    if !t.mandatory && try(length(t.permitted_values), 0) == 0
  ])[0]
}
`
	}
	withTag := func(description string, values ...string) string {
		return base(description) + fmt.Sprintf(`
resource "sapintegrationsuite_integration_package_custom_tag" "test" {
  package_id = sapintegrationsuite_integration_package.test.id
  name       = local.tag
  values     = ["%s"]
}

data "sapintegrationsuite_integration_package_custom_tags" "test" {
  package_id = sapintegrationsuite_integration_package_custom_tag.test.package_id
}
`, strings.Join(values, `", "`))
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withTag("created", "tfacc-b", "tfacc-a"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(tag, "id", regexp.MustCompile("^"+pkg+"/.+")),
					resource.TestCheckResourceAttr(tag, "values.#", "2"),
					resource.TestCheckResourceAttr(tag, "values.0", "tfacc-b"),
					resource.TestCheckResourceAttr(tag, "values.1", "tfacc-a"),
					resource.TestCheckTypeSetElemNestedAttrs(tags, "tags.*", map[string]string{
						"values.#": "2", "values.0": "tfacc-b", "values.1": "tfacc-a",
					}),
				),
			},
			{
				// The package PUT keeps the tag: no drift after it.
				Config:           withTag("updated", "tfacc-b", "tfacc-a"),
				ConfigPlanChecks: expectInPlace("sapintegrationsuite_integration_package.test"),
				Check:            resource.TestCheckResourceAttr(tag, "values.#", "2"),
			},
			{
				Config:           withTag("updated", "tfacc-c, with comma"),
				ConfigPlanChecks: expectInPlace(tag),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(tag, "values.#", "1"),
					resource.TestCheckResourceAttr(tag, "values.0", "tfacc-c, with comma"),
					resource.TestCheckTypeSetElemNestedAttrs(tags, "tags.*", map[string]string{
						"values.#": "1", "values.0": "tfacc-c, with comma",
					}),
				),
			},
			{
				ResourceName:      tag,
				ImportState:       true,
				ImportStateIdFunc: testAccAttrImportID(tag, "id"),
				ImportStateVerify: true,
			},
			{
				Config:      withTag("updated", "tfacc-a|tfacc-b"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`contains "\|"`),
			},
			{
				// Destroying the resource clears the tag; the package stays.
				Config: base("updated"),
			},
			{
				Config: base("updated") + `
data "sapintegrationsuite_integration_package_custom_tags" "after" {
  package_id = sapintegrationsuite_integration_package.test.id
}

output "cleared" {
  value = [for t in data.sapintegrationsuite_integration_package_custom_tags.after.tags : length(t.values) if t.name == local.tag][0]
}
`,
				Check: resource.TestCheckOutput("cleared", "0"),
			},
		},
	})
}
