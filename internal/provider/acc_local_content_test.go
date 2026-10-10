package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/samples"
)

// These tests need content that SAP's public samples do not have, taken from
// a directory of the tester's own exports (SAP_LOCAL_CONTENT_DIR). They skip
// without it. Script collections and value mappings do nothing at runtime on
// their own, so deploying them is safe; value mapping entries are replaced
// by made-up ones before upload.

func TestAccScriptCollection_localContent(t *testing.T) {
	accgate.Require(t, accgate.CloudIntegration, samples.EnvLocalContent)
	sc := samples.LocalArtifactOfType(t, "ScriptCollection")
	pkg, id := testAccName(), testAccName()
	path := testAccArtifactFile(t, sc.Content, id)
	config := func(deploy bool) string {
		c := testAccPackageConfig(pkg, "script collection test") + fmt.Sprintf(`
resource "sapintegrationsuite_script_collection" "test" {
  package_id           = sapintegrationsuite_integration_package.test.id
  script_collection_id = %[1]q
  name                 = "tf-acc %[1]s"
  content              = %[2]q
  content_hash         = filesha256(%[2]q)
}
`, id, path)
		if deploy {
			c += `
resource "sapintegrationsuite_script_collection_deployment" "test" {
  package_id                = sapintegrationsuite_integration_package.test.id
  script_collection_id      = sapintegrationsuite_script_collection.test.script_collection_id
  script_collection_version = sapintegrationsuite_script_collection.test.version

  timeouts {
    create = "15m"
    delete = "10m"
  }
}
`
		}
		return c
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(false),
				Check:  resource.TestCheckResourceAttrSet("sapintegrationsuite_script_collection.test", "version"),
			},
			{
				ResourceName:            "sapintegrationsuite_script_collection.test",
				ImportState:             true,
				ImportStateIdFunc:       testAccAttrImportID("sapintegrationsuite_script_collection.test", "id"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"content", "content_hash", "save_as_version"},
			},
			{
				Config: config(true),
				Check:  resource.TestCheckResourceAttr("sapintegrationsuite_script_collection_deployment.test", "status", "STARTED"),
			},
		},
	})
}

func TestAccValueMapping_localContent(t *testing.T) {
	accgate.Require(t, accgate.CloudIntegration, samples.EnvLocalContent)
	vm := samples.LocalArtifactOfType(t, "ValueMapping")
	synthetic, err := samples.SyntheticValueMapping(vm.Content, "tfacc")
	if err != nil {
		t.Fatal(err)
	}
	pkg, id := testAccName(), testAccName()
	path := testAccArtifactFile(t, synthetic, id)
	config := func(deploy bool) string {
		c := testAccPackageConfig(pkg, "value mapping test") + fmt.Sprintf(`
resource "sapintegrationsuite_value_mapping" "test" {
  package_id   = sapintegrationsuite_integration_package.test.id
  mapping_id   = %[1]q
  name         = "tf-acc %[1]s"
  content      = %[2]q
  content_hash = filesha256(%[2]q)
}
`, id, path)
		if deploy {
			c += `
resource "sapintegrationsuite_value_mapping_deployment" "test" {
  package_id      = sapintegrationsuite_integration_package.test.id
  mapping_id      = sapintegrationsuite_value_mapping.test.mapping_id
  mapping_version = sapintegrationsuite_value_mapping.test.version

  timeouts {
    create = "15m"
    delete = "10m"
  }
}
`
		}
		return c
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(false),
				Check:  resource.TestCheckResourceAttrSet("sapintegrationsuite_value_mapping.test", "version"),
			},
			{
				ResourceName:            "sapintegrationsuite_value_mapping.test",
				ImportState:             true,
				ImportStateIdFunc:       testAccAttrImportID("sapintegrationsuite_value_mapping.test", "id"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"content", "content_hash"},
			},
			{
				Config: config(true),
				Check:  resource.TestCheckResourceAttr("sapintegrationsuite_value_mapping_deployment.test", "status", "STARTED"),
			},
		},
	})
}

// save_as_version on a value mapping: the create labels the upload 2.0.0,
// a lower number relabels it in place, and an import reads the new label.
// SAP keeps one version, so nothing else changes. It passed on 2026-10-07
// behind enable_unofficial; since the Integration Content specification
// documents SaveAsVersion for value mappings, it runs without the switch.
func TestAccValueMapping_saveAsVersion(t *testing.T) {
	accgate.Require(t, accgate.CloudIntegration, samples.EnvLocalContent)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "")
	vm := samples.LocalArtifactOfType(t, "ValueMapping")
	synthetic, err := samples.SyntheticValueMapping(vm.Content, "tfacc")
	if err != nil {
		t.Fatal(err)
	}
	pkg, id := testAccName(), testAccName()
	path := testAccArtifactFile(t, synthetic, id)
	config := func(version string) string {
		return testAccPackageConfig(pkg, "value mapping version test") + fmt.Sprintf(`
resource "sapintegrationsuite_value_mapping" "test" {
  package_id      = sapintegrationsuite_integration_package.test.id
  mapping_id      = %[1]q
  name            = "tf-acc %[1]s"
  content         = %[2]q
  content_hash    = filesha256(%[2]q)
  save_as_version = %[3]q
}
`, id, path, version)
	}
	const address = "sapintegrationsuite_value_mapping.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("2.0.0"),
				Check:  resource.TestCheckResourceAttr(address, "version", "2.0.0"),
			},
			{
				Config: config("1.9.0"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
				},
				Check: resource.TestCheckResourceAttr(address, "version", "1.9.0"),
			},
			{
				ResourceName:            address,
				ImportState:             true,
				ImportStateIdFunc:       testAccAttrImportID(address, "id"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"content", "content_hash", "save_as_version"},
			},
		},
	})
}
