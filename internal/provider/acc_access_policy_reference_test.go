package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/auth"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
	sapthttp "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/http"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

// One synthetic policy with an exact and a regular-expression reference.
// SAP must store and return the wire values unchanged: INTEGRATION_FLOW,
// Name, exactString or regularExpression, and the value as written. The
// import step reads both references back with GET and compares every
// attribute. Neither value matches an existing integration flow.
func TestAccAccessPolicyReference_wireValues(t *testing.T) {
	accgate.Require(t, accgate.SecurityContent)
	role := testAccName()
	pattern := "^IFL_TFACC_" + strings.ToUpper(strings.TrimPrefix(role, "tfacc")) + "_.*$"
	config := fmt.Sprintf(`
resource "sapintegrationsuite_access_policy" "test" {
  role_name   = %[1]q
  description = "tfacc reference wire values"
}

resource "sapintegrationsuite_access_policy_reference" "exact" {
  access_policy_id = sapintegrationsuite_access_policy.test.id
  name             = "tfacc exact"
  artifact_type    = "INTEGRATION_FLOW"
  attribute        = "Name"
  operator         = "exactString"
  value            = %[1]q
}

resource "sapintegrationsuite_access_policy_reference" "regex" {
  access_policy_id = sapintegrationsuite_access_policy.test.id
  name             = "tfacc regex"
  artifact_type    = "INTEGRATION_FLOW"
  attribute        = "Name"
  operator         = "regularExpression"
  value            = %[2]q
}
`, role, pattern)
	check := func(name, operator, value string) resource.TestCheckFunc {
		address := "sapintegrationsuite_access_policy_reference." + name
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(address, "artifact_type", "INTEGRATION_FLOW"),
			resource.TestCheckResourceAttr(address, "attribute", "Name"),
			resource.TestCheckResourceAttr(address, "operator", operator),
			resource.TestCheckResourceAttr(address, "value", value),
		)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					check("exact", "exactString", role),
					check("regex", "regularExpression", pattern),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				RefreshState: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					check("exact", "exactString", role),
					check("regex", "regularExpression", pattern),
				),
			},
			{ResourceName: "sapintegrationsuite_access_policy_reference.exact", ImportState: true, ImportStateVerify: true},
			{ResourceName: "sapintegrationsuite_access_policy_reference.regex", ImportState: true, ImportStateVerify: true},
		},
	})
}

// The provider's compatibility matrix must agree with SAP: every combination
// it accepts is created and read back unchanged, every combination it
// rejects is rejected by SAP as well, and SAP still names each accepted
// artifact type when it refuses an unknown one. A failure means SAP changed
// its rules. The references match no artifact and are deleted at once.
func TestAccAccessPolicyReference_matrixMatchesSAP(t *testing.T) {
	accgate.Require(t, accgate.SecurityContent)
	ctx := context.Background()
	client := testAccCloudIntegrationClient(t)

	policy, err := client.CreateAccessPolicy(ctx, cloudintegration.AccessPolicy{RoleName: testAccName(), Description: "tfacc reference matrix"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.DeleteAccessPolicy(context.Background(), policy.ID); err != nil {
			t.Errorf("deleting policy %s: %v", policy.ID, err)
		}
	})
	value := testAccName()

	_, err = client.CreateAccessPolicyReference(ctx, policy.ID, cloudintegration.AccessPolicyReference{
		Name: "tfacc matrix", Type: "TFACC_UNKNOWN", ConditionAttribute: referenceAttributeName,
		ConditionType: referenceOperatorExact, ConditionValue: value,
	})
	if err == nil {
		t.Fatal("SAP accepted the artifact type TFACC_UNKNOWN")
	}
	for _, typ := range referenceArtifactTypes {
		if !strings.Contains(err.Error(), typ.WireValue) {
			t.Errorf("SAP's list of artifact types no longer contains %s: %v", typ.WireValue, err)
		}
	}

	for _, typ := range referenceArtifactTypes {
		for _, attribute := range referenceAttributes {
			for _, operator := range referenceOperators {
				want := contains(typ.Attributes, attribute) && contains(typ.Operators, operator)
				ref, err := client.CreateAccessPolicyReference(ctx, policy.ID, cloudintegration.AccessPolicyReference{
					Name: "tfacc matrix", Type: typ.WireValue, ConditionAttribute: attribute,
					ConditionType: operator, ConditionValue: value,
				})
				combination := typ.WireValue + "/" + attribute + "/" + operator
				switch {
				case err != nil && want:
					t.Errorf("%s: provider accepts it, SAP rejected it: %v", combination, err)
				case err == nil && !want:
					t.Errorf("%s: provider rejects it, SAP accepted it", combination)
				}
				if err != nil {
					continue
				}
				if ref.Type != typ.WireValue || ref.ConditionAttribute != attribute || ref.ConditionType != operator || ref.ConditionValue != value {
					t.Errorf("%s: SAP returned %s/%s/%s %q", combination, ref.Type, ref.ConditionAttribute, ref.ConditionType, ref.ConditionValue)
				}
				if err := client.DeleteAccessPolicyReference(ctx, ref.ID); err != nil {
					t.Errorf("%s: deleting reference %s: %v", combination, ref.ID, err)
				}
			}
		}
	}
}

// testAccCloudIntegrationClient builds the Cloud Integration client from the
// same environment variables the provider reads.
func testAccCloudIntegrationClient(t *testing.T) *cloudintegration.Client {
	t.Helper()
	httpClient, invalidate, err := auth.Config{
		TokenURL:     os.Getenv("SAP_INTEGRATION_SUITE_TOKEN_URL"),
		ClientID:     os.Getenv("SAP_INTEGRATION_SUITE_CLIENT_ID"),
		ClientSecret: os.Getenv("SAP_INTEGRATION_SUITE_CLIENT_SECRET"),
	}.HTTPClient(context.Background(), http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	return cloudintegration.New(sapthttp.New(sapthttp.Config{Transport: httpClient, InvalidateToken: invalidate}),
		os.Getenv("SAP_INTEGRATION_SUITE_HOST"))
}

// UI labels (convert_ui_labels, 0.7.0): without the switches a reference
// written with the UI's labels fails in the plan and nothing is created;
// with convert_ui_labels and enable_experimental the provider sends SAP's
// constants, the state keeps the labels, and an import reads the constants
// back without a plan difference.
func TestAccAccessPolicyReference_uiLabels(t *testing.T) {
	accgate.Require(t, accgate.SecurityContent)
	t.Setenv("SAP_INTEGRATION_SUITE_CONVERT_UI_LABELS", "")
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_EXPERIMENTAL", "")
	role := testAccName()
	pattern := "^IFL_TFACC_" + strings.ToUpper(strings.TrimPrefix(role, "tfacc")) + "_.*$"
	resources := fmt.Sprintf(`
resource "sapintegrationsuite_access_policy" "test" {
  role_name   = %[1]q
  description = "tfacc reference UI labels"
}

resource "sapintegrationsuite_access_policy_reference" "label" {
  access_policy_id = sapintegrationsuite_access_policy.test.id
  name             = "tfacc label"
  artifact_type    = "Integration Flow"
  attribute        = "name"
  operator         = "Matches"
  value            = %[2]q
}
`, role, pattern)
	const switches = `
provider "sapintegrationsuite" {
  convert_ui_labels   = true
  enable_experimental = true
}
`
	const address = "sapintegrationsuite_access_policy_reference.label"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      resources,
				ExpectError: regexp.MustCompile(`UI label instead of SAP's constant`),
			},
			{
				Config: switches + resources,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "artifact_type", "Integration Flow"),
					resource.TestCheckResourceAttr(address, "attribute", "name"),
					resource.TestCheckResourceAttr(address, "operator", "Matches"),
					resource.TestCheckResourceAttr(address, "value", pattern),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config:       switches + resources,
				ResourceName: address,
				ImportState:  true,
				// SAP stored the constants.
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					for _, s := range states {
						if s.Attributes["value"] != pattern {
							continue
						}
						for attr, want := range map[string]string{"artifact_type": "INTEGRATION_FLOW", "attribute": "Name", "operator": "regularExpression"} {
							if got := s.Attributes[attr]; got != want {
								return fmt.Errorf("imported %s = %q, want %q", attr, got, want)
							}
						}
						return nil
					}
					return fmt.Errorf("the imported reference was not found")
				},
			},
		},
	})
}
