package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/samples"
)

// Content for these tests comes from SAP's public samples (see
// internal/testutil/samples). Only samples marked Deployable are deployed.

func testAccPackageConfig(id, description string) string {
	return fmt.Sprintf(`
resource "sapintegrationsuite_integration_package" "test" {
  id          = %[1]q
  name        = "tf-acc %[1]s"
  short_text  = "Terraform acceptance test"
  description = %[2]q
}
`, id, description)
}

func TestAccIntegrationPackage_basic(t *testing.T) {
	id := testAccName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPackageConfig(id, "created"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sapintegrationsuite_integration_package.test", "id", id),
					resource.TestCheckResourceAttr("sapintegrationsuite_integration_package.test", "description", "created"),
				),
			},
			{
				// Update reads the package first and sends its other fields
				// back, because SAP's PUT replaces the entity.
				Config: testAccPackageConfig(id, "updated"),
				Check:  resource.TestCheckResourceAttr("sapintegrationsuite_integration_package.test", "description", "updated"),
			},
			{
				ResourceName:      "sapintegrationsuite_integration_package.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccFlowConfig(pkg, flow, path, extra string) string {
	return testAccPackageConfig(pkg, "flow test") + fmt.Sprintf(`
resource "sapintegrationsuite_integration_flow" "test" {
  package_id   = sapintegrationsuite_integration_package.test.id
  flow_id      = %[1]q
  name         = "tf-acc %[1]s"
  content      = %[2]q
  content_hash = filesha256(%[2]q)
}
%[3]s`, flow, path, extra)
}

// A flow from the SAP codejam export: create, change its content to another
// flow of the same export (same ID in the MANIFEST), import, and set one of
// its externalized parameters. Nothing is deployed.
func TestAccIntegrationFlow_sampleContent(t *testing.T) {
	pkg, flow := testAccName(), testAccName()
	v1 := testAccArtifactFile(t, testAccExportFlow(t, "codejam-package-export", "Request Employee Dependants - Exercise 05"), flow)
	v2 := testAccArtifactFile(t, testAccExportFlow(t, "codejam-package-export", "Request Employee Dependants - Exercise 06"), flow)
	configuration := `
resource "sapintegrationsuite_integration_flow_configuration" "test" {
  flow_id      = sapintegrationsuite_integration_flow.test.flow_id
  flow_version = sapintegrationsuite_integration_flow.test.version
  parameters = {
    european_countries = "DE,AT"
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowConfig(pkg, flow, v1, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sapintegrationsuite_integration_flow.test", "id", pkg+"/"+flow),
					resource.TestCheckResourceAttrSet("sapintegrationsuite_integration_flow.test", "version"),
				),
			},
			{
				Config: testAccFlowConfig(pkg, flow, v2, ""),
				Check:  resource.TestCheckResourceAttrSet("sapintegrationsuite_integration_flow.test", "version"),
			},
			{
				ResourceName:            "sapintegrationsuite_integration_flow.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"content", "content_hash", "save_as_version"},
			},
			{
				Config: testAccFlowConfig(pkg, flow, v2, configuration),
				Check: resource.TestCheckResourceAttr("sapintegrationsuite_integration_flow_configuration.test",
					"parameters.european_countries", "DE,AT"),
			},
		},
	})
}

// save_as_version: SAP answers the function import with 200 and no body; the
// provider must read the saved version back.
func TestAccIntegrationFlow_saveAsVersion(t *testing.T) {
	pkg, flow := testAccName(), testAccName()
	path := testAccArtifactFile(t, testAccExportFlow(t, "codejam-package-export", "Request Employee Dependants - Exercise 05"), flow)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPackageConfig(pkg, "save as version") + fmt.Sprintf(`
resource "sapintegrationsuite_integration_flow" "test" {
  package_id      = sapintegrationsuite_integration_package.test.id
  flow_id         = %[1]q
  name            = "tf-acc %[1]s"
  content         = %[2]q
  content_hash    = filesha256(%[2]q)
  save_as_version = "1.2.3"
}
`, flow, path),
				Check: resource.TestCheckResourceAttr("sapintegrationsuite_integration_flow.test", "version", "1.2.3"),
			},
		},
	})
}

// The e-bite Event Hub flow has only an HTTPS sender and no receiver, so
// deploying it starts nothing. Destroy undeploys it.
func TestAccIntegrationFlowDeployment_sample(t *testing.T) {
	pkg, flow := testAccName(), testAccName()
	path := testAccArtifactFile(t, testAccExportFlow(t, "eventhub-package-export", "ReceiveEvents_SAPCloudApplicationEventHub"), flow)
	// Tenants with Integration Cell give new flows SAP_ProfileId =
	// "integrationcell"; the flow then never reaches the Cloud Integration
	// runtime. Point it at iflmap first, as the configuration guide shows.
	deployment := `
resource "sapintegrationsuite_integration_flow_configuration" "test" {
  flow_id      = sapintegrationsuite_integration_flow.test.flow_id
  flow_version = sapintegrationsuite_integration_flow.test.version
  parameters = {
    SAP_ProfileId = "iflmap"
  }
}

resource "sapintegrationsuite_integration_flow_deployment" "test" {
  package_id        = sapintegrationsuite_integration_package.test.id
  flow_id           = sapintegrationsuite_integration_flow.test.flow_id
  flow_version      = sapintegrationsuite_integration_flow.test.version
  redeploy_triggers = sapintegrationsuite_integration_flow_configuration.test.parameters

  timeouts {
    create = "5m"
    delete = "5m"
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowConfig(pkg, flow, path, deployment),
				Check:  resource.TestCheckResourceAttr("sapintegrationsuite_integration_flow_deployment.test", "status", "STARTED"),
			},
		},
	})
}

func testAccMappingConfig(pkg, mapping, path string, deploy bool) string {
	config := testAccPackageConfig(pkg, "mapping test") + fmt.Sprintf(`
resource "sapintegrationsuite_message_mapping" "test" {
  package_id   = sapintegrationsuite_integration_package.test.id
  mapping_id   = %[1]q
  name         = "tf-acc %[1]s"
  content      = %[2]q
  content_hash = filesha256(%[2]q)
}
`, mapping, path)
	if deploy {
		config += `
resource "sapintegrationsuite_message_mapping_deployment" "test" {
  package_id      = sapintegrationsuite_integration_package.test.id
  mapping_id      = sapintegrationsuite_message_mapping.test.mapping_id
  mapping_version = sapintegrationsuite_message_mapping.test.version

  timeouts {
    create = "5m"
    delete = "5m"
  }
}
`
	}
	return config
}

// Message mappings from SAP's spend analysis sample: create, replace the
// content with another mapping under the same ID, import, then deploy.
func TestAccMessageMapping_sample(t *testing.T) {
	pkg, mapping := testAccName(), testAccName()
	v1 := testAccArtifactFile(t, samples.Get(t, "spend-account-dim-map"), mapping)
	v2 := testAccArtifactFile(t, samples.Get(t, "spend-supplier-dim-map"), mapping)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMappingConfig(pkg, mapping, v1, false),
				Check:  resource.TestCheckResourceAttrSet("sapintegrationsuite_message_mapping.test", "version"),
			},
			{
				Config: testAccMappingConfig(pkg, mapping, v2, false),
				Check:  resource.TestCheckResourceAttrSet("sapintegrationsuite_message_mapping.test", "version"),
			},
			{
				ResourceName:            "sapintegrationsuite_message_mapping.test",
				ImportState:             true,
				ImportStateIdFunc:       testAccAttrImportID("sapintegrationsuite_message_mapping.test", "id"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"content", "content_hash", "save_as_version"},
			},
			{
				Config: testAccMappingConfig(pkg, mapping, v2, true),
				Check:  resource.TestCheckResourceAttr("sapintegrationsuite_message_mapping_deployment.test", "status", "STARTED"),
			},
		},
	})
}

// testAccAttrImportID imports by the value of one attribute of a resource.
func testAccAttrImportID(name, attr string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return "", fmt.Errorf("%s not in state", name)
		}
		return rs.Primary.Attributes[attr], nil
	}
}
