package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/auth"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/integrationassessment"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

// testAccIADeploymentModels returns the names of two of the tenant's
// deployment models, read through the provider's own client, so the test
// needs no extra input. With only one model both names are the same and the
// deployment model change is not exercised.
func testAccIADeploymentModels(t *testing.T) (first, second string) {
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
	first, second = models[0].Name, models[0].Name
	if len(models) > 1 {
		second = models[1].Name
	}
	return first, second
}

// iaLandscapeStep is what one step of the landscape test changes.
type iaLandscapeStep struct {
	suffix      string // appended to the names that change in place
	description string // of the application instance
	withVendor  bool   // whether the first application has a vendor
	moved       bool   // links point at the second vendor, application and deployment model
}

// testAccIALandscapeConfig is the landscape of one test step.
func testAccIALandscapeConfig(name, model, otherModel string, s iaLandscapeStep) string {
	vendorLink := ""
	if s.withVendor {
		vendorLink = "vendor_id = sapintegrationsuite_integration_assessment_vendor.test.id"
	}
	target := "test"
	if s.moved {
		target = "second"
	}
	return fmt.Sprintf(`
data "sapintegrationsuite_integration_assessment_deployment_model" "test" {
  name = %[4]q
}

data "sapintegrationsuite_integration_assessment_deployment_model" "second" {
  name = %[6]q
}

resource "sapintegrationsuite_integration_assessment_vendor" "test" {
  name = "%[1]s vendor%[2]s"
}

resource "sapintegrationsuite_integration_assessment_vendor" "second" {
  name = "%[1]s vendor 2"
}

resource "sapintegrationsuite_integration_assessment_application" "test" {
  name = "%[1]s application%[2]s"
  %[5]s
}

resource "sapintegrationsuite_integration_assessment_application" "second" {
  name = "%[1]s application 2"
}

resource "sapintegrationsuite_integration_assessment_application_instance" "test" {
  name                = "%[1]s instance%[2]s"
  description         = %[3]q
  application_id      = sapintegrationsuite_integration_assessment_application.%[7]s.id
  deployment_model_id = data.sapintegrationsuite_integration_assessment_deployment_model.%[7]s.id
}

resource "sapintegrationsuite_integration_assessment_technology" "test" {
  name      = "%[1]s technology%[2]s"
  vendor_id = sapintegrationsuite_integration_assessment_vendor.%[7]s.id
}

resource "sapintegrationsuite_integration_assessment_technology_instance" "test" {
  name                = "%[1]s technology instance%[2]s"
  technology_id       = sapintegrationsuite_integration_assessment_technology.test.id
  deployment_model_id = data.sapintegrationsuite_integration_assessment_deployment_model.%[7]s.id
}

data "sapintegrationsuite_integration_assessment_vendor" "test" {
  name = sapintegrationsuite_integration_assessment_vendor.test.name
}
`, name, s.suffix, s.description, model, vendorLink, otherModel, target)
}

// expectInPlace checks that each address is planned as an update, not a
// replacement.
func expectInPlace(addresses ...string) resource.ConfigPlanChecks {
	var checks []plancheck.PlanCheck
	for _, a := range addresses {
		checks = append(checks, plancheck.ExpectResourceAction(a, plancheck.ResourceActionUpdate))
	}
	return resource.ConfigPlanChecks{PreApply: checks}
}

// The landscape lifecycle: two vendors with an application each, an
// application instance, a technology and a technology instance; then
// renames, the application's vendor removed and the instance's description
// changed in place; then the instance moved to the other application and
// deployment model, the technology to the other vendor and the technology
// instance to the other deployment model, all in place; then an import of
// every object. Destroy deletes them in dependency order.
func TestAccIntegrationAssessment_landscape(t *testing.T) {
	accgate.Require(t, accgate.IntegrationAssessment)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "true")
	model, otherModel := testAccIADeploymentModels(t)
	name := testAccName()
	config := func(s iaLandscapeStep) string { return testAccIALandscapeConfig(name, model, otherModel, s) }
	const (
		application  = "sapintegrationsuite_integration_assessment_application.test"
		instance     = "sapintegrationsuite_integration_assessment_application_instance.test"
		technology   = "sapintegrationsuite_integration_assessment_technology.test"
		techInstance = "sapintegrationsuite_integration_assessment_technology_instance.test"
		vendor       = "sapintegrationsuite_integration_assessment_vendor.test"
	)
	importStep := func(address string) resource.TestStep {
		return resource.TestStep{ResourceName: address, ImportState: true, ImportStateVerify: true}
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(iaLandscapeStep{description: "created", withVendor: true}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(application, "vendor_id", vendor, "id"),
					resource.TestCheckResourceAttrPair("data.sapintegrationsuite_integration_assessment_vendor.test", "id", vendor, "id"),
					resource.TestCheckResourceAttr(instance, "description", "created"),
				),
			},
			{
				Config:           config(iaLandscapeStep{suffix: " renamed", description: "updated"}),
				ConfigPlanChecks: expectInPlace(instance, technology, techInstance),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(vendor, "name", name+" vendor renamed"),
					resource.TestCheckNoResourceAttr(application, "vendor_id"),
					resource.TestCheckResourceAttr(instance, "description", "updated"),
					resource.TestCheckResourceAttr(technology, "name", name+" technology renamed"),
					resource.TestCheckResourceAttr(techInstance, "name", name+" technology instance renamed"),
				),
			},
			{
				Config:           config(iaLandscapeStep{suffix: " renamed", description: "updated", moved: true}),
				ConfigPlanChecks: expectInPlace(instance, technology),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(instance, "application_id",
						"sapintegrationsuite_integration_assessment_application.second", "id"),
					resource.TestCheckResourceAttrPair(instance, "deployment_model_id",
						"data.sapintegrationsuite_integration_assessment_deployment_model.second", "id"),
					resource.TestCheckResourceAttrPair(technology, "vendor_id",
						"sapintegrationsuite_integration_assessment_vendor.second", "id"),
					resource.TestCheckResourceAttrPair(techInstance, "deployment_model_id",
						"data.sapintegrationsuite_integration_assessment_deployment_model.second", "id"),
				),
			},
			importStep(vendor),
			importStep(application),
			importStep(instance),
			importStep(technology),
			importStep(techInstance),
		},
	})
}
