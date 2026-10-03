package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// A tenant probe (2026-10-03) wrote access policy strings of 200 to 5000
// characters. SAP accepted all of them and stored only the first 200
// characters of a role name and of a reference description, 50 of a
// reference name and 150 of a reference value. These tests pin the plan-time
// checks that keep such values out of a request.

func validateLengthPolicy(t *testing.T, roleName string) []*tfprotov6.Diagnostic {
	t.Helper()
	return validateStringResourceConfig(t, "sapintegrationsuite_access_policy", map[string]*string{
		"id":          nil,
		"role_name":   ptr(roleName),
		"description": nil,
	})
}

func validateLengthReference(t *testing.T, name string, description *string, value string) []*tfprotov6.Diagnostic {
	t.Helper()
	return validateStringResourceConfig(t, "sapintegrationsuite_access_policy_reference", map[string]*string{
		"id":               nil,
		"access_policy_id": ptr("1901"),
		"name":             ptr(name),
		"description":      description,
		"artifact_type":    ptr("INTEGRATION_FLOW"),
		"attribute":        ptr("Name"),
		"operator":         ptr("exactString"),
		"value":            ptr(value),
	})
}

// checkLengthDiagnostics expects no error when wantSummary is empty, and
// otherwise exactly one error with that summary on the attribute.
func checkLengthDiagnostics(t *testing.T, diags []*tfprotov6.Diagnostic, wantSummary, attribute string) {
	t.Helper()
	errs := diagnosticsOf(diags, tfprotov6.DiagnosticSeverityError)
	if wantSummary == "" {
		for _, d := range errs {
			t.Errorf("unexpected error diagnostic: %s: %s", d.Summary, d.Detail)
		}
		return
	}
	if len(errs) != 1 {
		t.Fatalf("errors = %v, want exactly one", errs)
	}
	if errs[0].Summary != wantSummary {
		t.Errorf("summary = %q, want %q", errs[0].Summary, wantSummary)
	}
	if errs[0].Attribute == nil || !errs[0].Attribute.Equal(tftypes.NewAttributePath().WithAttributeName(attribute)) {
		t.Errorf("diagnostic attribute = %v, want %s", errs[0].Attribute, attribute)
	}
	if !strings.Contains(errs[0].Detail, "store only its first") {
		t.Errorf("detail does not explain the truncation: %s", errs[0].Detail)
	}
}

func TestAccessPolicyResource_RoleNameLength(t *testing.T) {
	cases := []struct {
		name, roleName, wantSummary string
	}{
		{"200 characters", strings.Repeat("R", 200), ""},
		{"200 non-ASCII characters", strings.Repeat("Ä", 200), ""},
		{"201 characters", strings.Repeat("R", 201), "Access policy role name is too long"},
		{"5000 characters", strings.Repeat("R", 5000), "Access policy role name is too long"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			checkLengthDiagnostics(t, validateLengthPolicy(t, c.roleName), c.wantSummary, "role_name")
		})
	}
}

func TestAccessPolicyReferenceResource_StringLengths(t *testing.T) {
	cases := []struct {
		name        string
		refName     string
		description *string
		value       string
		wantSummary string
		attribute   string
	}{
		{"all at the limit", strings.Repeat("n", 50), ptr(strings.Repeat("d", 200)), strings.Repeat("v", 150), "", ""},
		{"name 51", strings.Repeat("n", 51), nil, "SalesOrders", "Access policy reference name is too long", "name"},
		{"description 201", "Sales", ptr(strings.Repeat("d", 201)), "SalesOrders", "Access policy reference description is too long", "description"},
		{"value 151", "Sales", nil, strings.Repeat("v", 151), "Access policy reference value is too long", "value"},
		{"value 1000", "Sales", nil, strings.Repeat("v", 1000), "Access policy reference value is too long", "value"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			checkLengthDiagnostics(t, validateLengthReference(t, c.refName, c.description, c.value), c.wantSummary, c.attribute)
		})
	}
}
