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

// testAccIAClient builds the provider's own Integration Assessment client
// from the acceptance test credentials, to read what a test needs from the
// tenant.
func testAccIAClient(t *testing.T) *integrationassessment.Client {
	t.Helper()
	httpClient, _, err := auth.Config{
		TokenURL:     os.Getenv("SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_TOKEN_URL"),
		ClientID:     os.Getenv("SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_CLIENT_ID"),
		ClientSecret: os.Getenv("SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_CLIENT_SECRET"),
	}.HTTPClient(context.Background(), http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	return integrationassessment.New(httpClient, os.Getenv("SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_ENTITIES_URL"))
}

// testAccIADeploymentModels returns the names of two of the tenant's
// deployment models, so the test needs no extra input. With only one model
// both names are the same and the deployment model change is not exercised.
func testAccIADeploymentModels(t *testing.T) (first, second string) {
	t.Helper()
	models, err := testAccIAClient(t).ListDeploymentModels(context.Background())
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

// iaTaxonomyNames are names of SAP's taxonomy entries on the tenant that
// identify exactly one entry each, so the lookups can find them.
type iaTaxonomyNames struct {
	domain, style, keyCharacteristic, keyCharacteristicValue, recommendationDegree string
}

// uniqueName returns the first name that occurs exactly once, or "".
func uniqueName(names []string) string {
	count := map[string]int{}
	for _, n := range names {
		count[n]++
	}
	for _, n := range names {
		if count[n] == 1 {
			return n
		}
	}
	return ""
}

// namesOf returns the name of every item.
func namesOf[T any](items []T, name func(T) string) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, name(item))
	}
	return names
}

// uniqueIn reports whether name occurs exactly once in names.
func uniqueIn(names []string, name string) bool {
	n := 0
	for _, x := range names {
		if x == name {
			n++
		}
	}
	return n == 1
}

// testAccIAKeyCharacteristicValue returns the names of a key characteristic
// and one of its values that the lookup can find, or "" for both.
func testAccIAKeyCharacteristicValue(t *testing.T, c *integrationassessment.Client) (characteristic, value string) {
	t.Helper()
	ctx := context.Background()
	characteristics, err := c.ListKeyCharacteristics(ctx)
	if err != nil {
		t.Fatalf("listing key characteristics: %v", err)
	}
	values, err := c.ListKeyCharacteristicValues(ctx)
	if err != nil {
		t.Fatalf("listing key characteristic values: %v", err)
	}
	characteristicNames := namesOf(characteristics, func(k integrationassessment.KeyCharacteristic) string { return k.Name })
	for _, k := range characteristics {
		if !uniqueIn(characteristicNames, k.Name) {
			continue
		}
		var valueNames []string
		for _, v := range values {
			if v.KeyCharacteristicID() == k.ID {
				valueNames = append(valueNames, v.Name)
			}
		}
		if v := uniqueName(valueNames); v != "" {
			return k.Name, v
		}
	}
	return "", ""
}

// testAccIATaxonomy reads the taxonomy entries the technology profile test
// links to, so the test needs no extra input.
func testAccIATaxonomy(t *testing.T) iaTaxonomyNames {
	t.Helper()
	ctx := context.Background()
	c := testAccIAClient(t)
	domains, err := c.ListDomains(ctx)
	if err != nil {
		t.Fatalf("listing domains: %v", err)
	}
	styles, err := c.ListStyles(ctx)
	if err != nil {
		t.Fatalf("listing styles: %v", err)
	}
	degrees, err := c.ListRecommendationDegrees(ctx)
	if err != nil {
		t.Fatalf("listing recommendation degrees: %v", err)
	}
	names := iaTaxonomyNames{
		domain:               uniqueName(namesOf(domains, func(d integrationassessment.Domain) string { return d.Name })),
		style:                uniqueName(namesOf(styles, func(s integrationassessment.Style) string { return s.Name })),
		recommendationDegree: uniqueName(namesOf(degrees, func(d integrationassessment.RecommendationDegree) string { return d.Name })),
	}
	names.keyCharacteristic, names.keyCharacteristicValue = testAccIAKeyCharacteristicValue(t, c)
	if names.domain == "" || names.style == "" || names.recommendationDegree == "" || names.keyCharacteristicValue == "" {
		t.Skipf("the tenant's taxonomy has no uniquely named entry for every lookup: %+v", names)
	}
	return names
}

// The technology profile: a technology linked to a domain and a style and
// rated on a key characteristic value, all through the lookups. Changing the
// rating's description replaces it, since the service has no update. Then
// an import of every association.
func TestAccIntegrationAssessment_technologyProfile(t *testing.T) {
	accgate.Require(t, accgate.IntegrationAssessment)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "true")
	tax := testAccIATaxonomy(t)
	name := testAccName()
	const (
		domain = "sapintegrationsuite_integration_assessment_technology_domain.test"
		style  = "sapintegrationsuite_integration_assessment_technology_style.test"
		rating = "sapintegrationsuite_integration_assessment_technology_key_characteristic.test"
	)
	config := func(description string) string {
		return fmt.Sprintf(`
data "sapintegrationsuite_integration_assessment_domain" "test" {
  name = %[2]q
}

data "sapintegrationsuite_integration_assessment_style" "test" {
  name = %[3]q
}

data "sapintegrationsuite_integration_assessment_key_characteristic_value" "test" {
  key_characteristic = %[4]q
  name               = %[5]q
}

data "sapintegrationsuite_integration_assessment_recommendation_degree" "test" {
  name = %[6]q
}

resource "sapintegrationsuite_integration_assessment_vendor" "test" {
  name = "%[1]s vendor"
}

resource "sapintegrationsuite_integration_assessment_technology" "test" {
  name      = "%[1]s technology"
  vendor_id = sapintegrationsuite_integration_assessment_vendor.test.id
}

resource "sapintegrationsuite_integration_assessment_technology_domain" "test" {
  technology_id = sapintegrationsuite_integration_assessment_technology.test.id
  domain_id     = data.sapintegrationsuite_integration_assessment_domain.test.id
}

resource "sapintegrationsuite_integration_assessment_technology_style" "test" {
  technology_id = sapintegrationsuite_integration_assessment_technology.test.id
  style_id      = data.sapintegrationsuite_integration_assessment_style.test.id
}

resource "sapintegrationsuite_integration_assessment_technology_key_characteristic" "test" {
  technology_id               = sapintegrationsuite_integration_assessment_technology.test.id
  key_characteristic_value_id = data.sapintegrationsuite_integration_assessment_key_characteristic_value.test.id
  recommendation_degree_id    = data.sapintegrationsuite_integration_assessment_recommendation_degree.test.id
  description                 = %[7]q
}
`, name, tax.domain, tax.style, tax.keyCharacteristic, tax.keyCharacteristicValue, tax.recommendationDegree, description)
	}
	importStep := func(address string) resource.TestStep {
		return resource.TestStep{ResourceName: address, ImportState: true, ImportStateVerify: true}
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("created"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(domain, "domain_id", "data.sapintegrationsuite_integration_assessment_domain.test", "id"),
					resource.TestCheckResourceAttrPair(style, "style_id", "data.sapintegrationsuite_integration_assessment_style.test", "id"),
					resource.TestCheckResourceAttrPair(rating, "key_characteristic_value_id",
						"data.sapintegrationsuite_integration_assessment_key_characteristic_value.test", "id"),
					resource.TestCheckResourceAttrPair(rating, "technology_id", "sapintegrationsuite_integration_assessment_technology.test", "id"),
					resource.TestCheckResourceAttr(rating, "description", "created"),
				),
			},
			{
				Config: config("replaced"),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(rating, plancheck.ResourceActionReplace),
					plancheck.ExpectResourceAction(domain, plancheck.ResourceActionNoop),
				}},
				Check: resource.TestCheckResourceAttr(rating, "description", "replaced"),
			},
			importStep(domain),
			importStep(style),
			importStep(rating),
		},
	})
}

// iaTaxonomyExpectations are the entries the taxonomy test looks up, with
// the values the data sources must return, as the client read them.
type iaTaxonomyExpectations struct {
	useCasePattern     integrationassessment.UseCasePattern
	integrationPattern integrationassessment.IntegrationPattern
	group              integrationassessment.KeyCharacteristicGroup
	determination      integrationassessment.DomainDetermination
}

// testAccIATaxonomyExpectations picks a uniquely named use case pattern,
// integration pattern and key characteristic group, and a domain
// determination whose pair of deployment models occurs once.
func testAccIATaxonomyExpectations(t *testing.T) iaTaxonomyExpectations {
	t.Helper()
	ctx := context.Background()
	c := testAccIAClient(t)
	var e iaTaxonomyExpectations
	var found [4]bool
	useCases, err := c.ListUseCasePatterns(ctx)
	if err != nil {
		t.Fatalf("listing use case patterns: %v", err)
	}
	names := namesOf(useCases, func(u integrationassessment.UseCasePattern) string { return u.Name })
	for _, u := range useCases {
		if uniqueIn(names, u.Name) {
			e.useCasePattern, found[0] = u, true
			break
		}
	}
	patterns, err := c.ListIntegrationPatterns(ctx)
	if err != nil {
		t.Fatalf("listing integration patterns: %v", err)
	}
	names = namesOf(patterns, func(p integrationassessment.IntegrationPattern) string { return p.Name })
	for _, p := range patterns {
		if uniqueIn(names, p.Name) {
			e.integrationPattern, found[1] = p, true
			break
		}
	}
	groups, err := c.ListKeyCharacteristicGroups(ctx)
	if err != nil {
		t.Fatalf("listing key characteristic groups: %v", err)
	}
	names = namesOf(groups, func(g integrationassessment.KeyCharacteristicGroup) string { return g.Name })
	for _, g := range groups {
		if uniqueIn(names, g.Name) {
			e.group, found[2] = g, true
			break
		}
	}
	determinations, err := c.ListDomainDeterminations(ctx)
	if err != nil {
		t.Fatalf("listing domain determinations: %v", err)
	}
	pairs := namesOf(determinations, func(d integrationassessment.DomainDetermination) string {
		return d.SourceDeploymentModelID() + "→" + d.TargetDeploymentModelID()
	})
	for i, d := range determinations {
		if d.SourceDeploymentModelID() != "" && uniqueIn(pairs, pairs[i]) {
			e.determination, found[3] = d, true
			break
		}
	}
	if found != [4]bool{true, true, true, true} {
		t.Skipf("the tenant's taxonomy has no unique entry for every lookup (found %v)", found)
	}
	return e
}

// The read-only ISA-M taxonomy lookups: use case pattern, integration
// pattern, key characteristic group and domain determination, compared with
// what the client reads from the same tenant.
func TestAccIntegrationAssessment_taxonomy(t *testing.T) {
	accgate.Require(t, accgate.IntegrationAssessment)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "true")
	e := testAccIATaxonomyExpectations(t)
	const (
		useCase       = "data.sapintegrationsuite_integration_assessment_use_case_pattern.test"
		pattern       = "data.sapintegrationsuite_integration_assessment_integration_pattern.test"
		group         = "data.sapintegrationsuite_integration_assessment_key_characteristic_group.test"
		determination = "data.sapintegrationsuite_integration_assessment_domain_determination.test"
	)
	config := fmt.Sprintf(`
data "sapintegrationsuite_integration_assessment_use_case_pattern" "test" {
  name = %q
}

data "sapintegrationsuite_integration_assessment_integration_pattern" "test" {
  name = %q
}

data "sapintegrationsuite_integration_assessment_key_characteristic_group" "test" {
  name = %q
}

data "sapintegrationsuite_integration_assessment_domain_determination" "test" {
  source_deployment_model_id = %q
  target_deployment_model_id = %q
}
`, e.useCasePattern.Name, e.integrationPattern.Name, e.group.Name,
		e.determination.SourceDeploymentModelID(), e.determination.TargetDeploymentModelID())
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(useCase, "id", e.useCasePattern.ID),
				resource.TestCheckResourceAttr(useCase, "style_id", e.useCasePattern.StyleID()),
				resource.TestCheckResourceAttr(pattern, "id", e.integrationPattern.ID),
				resource.TestCheckResourceAttr(pattern, "domain_id", e.integrationPattern.DomainID()),
				resource.TestCheckResourceAttr(pattern, "style_id", e.integrationPattern.StyleID()),
				resource.TestCheckResourceAttr(group, "id", e.group.ID),
				resource.TestCheckResourceAttr(determination, "id", e.determination.ID),
				resource.TestCheckResourceAttr(determination, "domain_id", e.determination.DomainID()),
			),
		}},
	})
}
