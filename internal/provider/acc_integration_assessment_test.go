package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/auth"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/integrationassessment"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

// testAccIADeploymentModel returns the name of one of the tenant's
// deployment models, read through the provider's own client, so the test
// needs no extra input.
func testAccIADeploymentModel(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	httpClient, _, err := auth.Config{
		TokenURL:     os.Getenv("SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_TOKEN_URL"),
		ClientID:     os.Getenv("SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_CLIENT_ID"),
		ClientSecret: os.Getenv("SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_CLIENT_SECRET"),
	}.HTTPClient(ctx, http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	models, err := integrationassessment.New(httpClient, os.Getenv("SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_ENTITIES_URL")).ListDeploymentModels(ctx)
	if err != nil {
		t.Fatalf("listing deployment models: %v", err)
	}
	if len(models) == 0 {
		t.Skip("the tenant has no deployment models")
	}
	return models[0].Name
}

// The landscape lifecycle: a vendor with an application, an application
// instance, a technology and a technology instance; then renames, the
// application's vendor removed and the instance's description changed in
// place; then an import of every object. Destroy deletes them in dependency
// order.
func TestAccIntegrationAssessment_landscape(t *testing.T) {
	accgate.Require(t, accgate.IntegrationAssessment)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "true")
	model := testAccIADeploymentModel(t)
	name := testAccName()
	config := func(suffix, description string, withVendor bool) string {
		vendorLink := ""
		if withVendor {
			vendorLink = "vendor_id = sapintegrationsuite_integration_assessment_vendor.test.id"
		}
		return fmt.Sprintf(`
data "sapintegrationsuite_integration_assessment_deployment_model" "test" {
  name = %[4]q
}

resource "sapintegrationsuite_integration_assessment_vendor" "test" {
  name = "%[1]s vendor%[2]s"
}

resource "sapintegrationsuite_integration_assessment_application" "test" {
  name = "%[1]s application%[2]s"
  %[5]s
}

resource "sapintegrationsuite_integration_assessment_application_instance" "test" {
  name                = "%[1]s instance%[2]s"
  description         = %[3]q
  application_id      = sapintegrationsuite_integration_assessment_application.test.id
  deployment_model_id = data.sapintegrationsuite_integration_assessment_deployment_model.test.id
}

resource "sapintegrationsuite_integration_assessment_technology" "test" {
  name      = "%[1]s technology%[2]s"
  vendor_id = sapintegrationsuite_integration_assessment_vendor.test.id
}

resource "sapintegrationsuite_integration_assessment_technology_instance" "test" {
  name                = "%[1]s technology instance"
  technology_id       = sapintegrationsuite_integration_assessment_technology.test.id
  deployment_model_id = data.sapintegrationsuite_integration_assessment_deployment_model.test.id
}

data "sapintegrationsuite_integration_assessment_vendor" "test" {
  name = sapintegrationsuite_integration_assessment_vendor.test.name
}
`, name, suffix, description, model, vendorLink)
	}
	importStep := func(address string) resource.TestStep {
		return resource.TestStep{ResourceName: address, ImportState: true, ImportStateVerify: true}
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("", "created", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("sapintegrationsuite_integration_assessment_application.test", "vendor_id",
						"sapintegrationsuite_integration_assessment_vendor.test", "id"),
					resource.TestCheckResourceAttrPair("data.sapintegrationsuite_integration_assessment_vendor.test", "id",
						"sapintegrationsuite_integration_assessment_vendor.test", "id"),
					resource.TestCheckResourceAttr("sapintegrationsuite_integration_assessment_application_instance.test", "description", "created"),
				),
			},
			{
				Config: config(" renamed", "updated", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sapintegrationsuite_integration_assessment_vendor.test", "name", name+" vendor renamed"),
					resource.TestCheckNoResourceAttr("sapintegrationsuite_integration_assessment_application.test", "vendor_id"),
					resource.TestCheckResourceAttr("sapintegrationsuite_integration_assessment_application_instance.test", "description", "updated"),
					resource.TestCheckResourceAttr("sapintegrationsuite_integration_assessment_technology.test", "name", name+" technology renamed"),
				),
			},
			importStep("sapintegrationsuite_integration_assessment_vendor.test"),
			importStep("sapintegrationsuite_integration_assessment_application.test"),
			importStep("sapintegrationsuite_integration_assessment_application_instance.test"),
			importStep("sapintegrationsuite_integration_assessment_technology.test"),
			importStep("sapintegrationsuite_integration_assessment_technology_instance.test"),
		},
	})
}
