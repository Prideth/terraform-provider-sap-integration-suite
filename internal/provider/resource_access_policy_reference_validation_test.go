package provider

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

// These tests send configurations through the provider's
// ValidateResourceConfig RPC, the call Terraform makes during validate and
// plan. A reference SAP would reject must fail there, before its policy is
// created; earlier releases accepted any non-empty string, so a wrong value
// failed only at apply time and left the policy without its references.

func validateReference(t *testing.T, artifactType, attribute, operator, value string) []*tfprotov6.Diagnostic {
	t.Helper()
	return validateStringResourceConfig(t, "sapintegrationsuite_access_policy_reference", map[string]*string{
		"id":               nil,
		"access_policy_id": ptr("1901"),
		"name":             ptr("tfacc reference"),
		"description":      nil,
		"artifact_type":    ptr(artifactType),
		"attribute":        ptr(attribute),
		"operator":         ptr(operator),
		"value":            ptr(value),
	})
}

func diagnosticsOf(diags []*tfprotov6.Diagnostic, severity tfprotov6.DiagnosticSeverity) []*tfprotov6.Diagnostic {
	var out []*tfprotov6.Diagnostic
	for _, d := range diags {
		if d.Severity == severity {
			out = append(out, d)
		}
	}
	return out
}

func requireNoErrors(t *testing.T, diags []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, d := range diagnosticsOf(diags, tfprotov6.DiagnosticSeverityError) {
		t.Errorf("unexpected error on %v: %s: %s", d.Attribute, d.Summary, d.Detail)
	}
}

// requireOneError checks that exactly one error is reported, on attribute,
// with the summary and every snippet in its detail.
func requireOneError(t *testing.T, diags []*tfprotov6.Diagnostic, attribute, summary string, snippets ...string) {
	t.Helper()
	errs := diagnosticsOf(diags, tfprotov6.DiagnosticSeverityError)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
	d := errs[0]
	if want := tftypes.NewAttributePath().WithAttributeName(attribute); d.Attribute == nil || !d.Attribute.Equal(want) {
		t.Errorf("error attribute = %v, want %s", d.Attribute, attribute)
	}
	if d.Summary != summary {
		t.Errorf("summary = %q, want %q", d.Summary, summary)
	}
	for _, s := range snippets {
		if !strings.Contains(d.Detail, s) {
			t.Errorf("detail %q does not contain %q", d.Detail, s)
		}
	}
}

// The combinations a tenant accepted and read back unchanged (2026-10-01),
// written out here independently of the matrix in the provider code.
func TestAccessPolicyReference_ValidCombinations(t *testing.T) {
	cases := [][3]string{
		{"INTEGRATION_FLOW", "Name", "exactString"},
		{"INTEGRATION_FLOW", "Name", "regularExpression"},
		{"INTEGRATION_FLOW", "ID", "exactString"},
		{"INTEGRATION_FLOW", "ID", "regularExpression"},
		{"INTEGRATION_PACKAGE", "Name", "exactString"},
		{"INTEGRATION_PACKAGE", "ID", "exactString"},
		{"MESSAGE_QUEUE", "Name", "regularExpression"},
		{"GLOBAL_VARIABLE", "Name", "exactString"},
		{"GLOBAL_DATA_STORE", "Name", "regularExpression"},
		{"ODATA_SERVICE", "ID", "regularExpression"},
		{"REST_API_PROVIDER", "Name", "exactString"},
		{"SOAP_API_PROVIDER", "Name", "exactString"},
		{"API_ARTIFACT", "ID", "exactString"},
		{"SCRIPT_COLLECTION", "Name", "exactString"},
		{"VALUE_MAPPING", "Name", "exactString"},
		{"MESSAGE_MAPPING", "Name", "exactString"},
		{"DATA_TYPE", "Name", "exactString"},
		{"MESSAGE_TYPE", "Name", "exactString"},
	}
	for _, c := range cases {
		t.Run(strings.Join(c[:], "/"), func(t *testing.T) {
			requireNoErrors(t, validateReference(t, c[0], c[1], c[2], "IFL_TFACC_ORDERS"))
		})
	}
}

func TestAccessPolicyReference_RejectsCombinationsSAPRejects(t *testing.T) {
	t.Run("integration package with regularExpression", func(t *testing.T) {
		requireOneError(t, validateReference(t, "INTEGRATION_PACKAGE", "Name", "regularExpression", "PKG_.*"),
			"operator", "Operator not supported for this artifact type", "only exactString for INTEGRATION_PACKAGE")
	})
	for _, typ := range []string{"MESSAGE_QUEUE", "GLOBAL_VARIABLE", "GLOBAL_DATA_STORE"} {
		t.Run(typ+" by ID", func(t *testing.T) {
			requireOneError(t, validateReference(t, typ, "ID", "exactString", "Q1"),
				"attribute", "Attribute not supported for this artifact type", typ+" references only by Name")
		})
	}
}

func TestAccessPolicyReference_RejectsUnknownArtifactTypes(t *testing.T) {
	cases := []struct{ value, hint string }{
		// Labels that spell their constant pass validation and are decided
		// in the plan (TestAccessPolicyReference_UILabels*); these are not
		// proven labels.
		{"OData API", `Use "ODATA_SERVICE"`},
		{"API", ""},
		{"foobar", ""},
	}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			snippets := []string{"Supported values: API_ARTIFACT, DATA_TYPE, GLOBAL_DATA_STORE,", "VALUE_MAPPING.",
				"With enable_unofficial = true also: AUTH2_AUTHORIZATION_CODE,", "can still be read and imported"}
			if c.hint != "" {
				snippets = append(snippets, c.hint)
			}
			diags := validateReference(t, c.value, "Name", "exactString", "X")
			requireOneError(t, diags, "artifact_type", "Unsupported access policy artifact type", snippets...)
			if c.hint == "" && strings.Contains(diagnosticsOf(diags, tfprotov6.DiagnosticSeverityError)[0].Detail, "Use \"") {
				t.Errorf("%q got a suggestion although none is known", c.value)
			}
		})
	}
}

func TestAccessPolicyReference_RejectsUnknownAttributes(t *testing.T) {
	cases := []struct{ value, hint string }{
		{"foobar", ""},
		{"Identifier", ""},
	}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			requireOneError(t, validateReference(t, "INTEGRATION_FLOW", c.value, "exactString", "X"),
				"attribute", "Unsupported access policy attribute", "Supported values: Name, ID.", c.hint)
		})
	}
}

func TestAccessPolicyReference_RejectsUILabelsAsOperators(t *testing.T) {
	cases := []struct{ value, hint string }{
		// Equals and Matches are proven UI labels: they pass validation and
		// are decided in the plan. Abbreviations are never converted.
		{"regex", `Use "regularExpression"`},
		{"regexp", `Use "regularExpression"`},
		{"regular_expression", `Use "regularExpression"`},
		{"foobar", ""},
	}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			requireOneError(t, validateReference(t, "INTEGRATION_FLOW", "Name", c.value, "X"),
				"operator", "Unsupported access policy operator",
				"Supported values: exactString, regularExpression.", "not UI labels", c.hint)
		})
	}
}

func TestAccessPolicyReference_RegularExpressionValue(t *testing.T) {
	cases := []struct {
		value       string
		wantError   bool
		wantWarning string // suggested pattern in the warning, "" for none
	}{
		{"^IFL_TFACC_.*$", false, ""},
		{"SALES_ORDERS_.*", false, ""},
		{`IFL_\d*`, false, ""},
		{"[A-Z]*_ORDERS", false, ""},
		{"(?<=IFL_)CORE", false, ""},         // lookbehind: valid in Java, unsupported in Go
		{`\p{Lower}+_ORDERS`, false, ""},     // Java's POSIX class, unknown to Go
		{`[\p{javaLowerCase}_]+`, false, ""}, // Java-specific class
		{`IFL_\h+`, false, ""},               // \h is Java only
		{"SALES_ORDERS_*", false, "SALES_ORDERS_.*"},
		{"*_ORDERS", true, ""},
		{"(SALES", true, ""},
		{"SALES)", true, ""},
		{"[A-Z", true, ""},
		{"[Z-A]", true, ""},
	}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			diags := validateReference(t, "INTEGRATION_FLOW", "Name", "regularExpression", c.value)
			if c.wantError {
				requireOneError(t, diags, "value", "Invalid regular expression", "Java regular expression")
				return
			}
			requireNoErrors(t, diags)
			warnings := diagnosticsOf(diags, tfprotov6.DiagnosticSeverityWarning)
			if c.wantWarning == "" {
				if len(warnings) != 0 {
					t.Errorf("unexpected warnings: %v", warnings)
				}
				return
			}
			if len(warnings) != 1 || !strings.Contains(warnings[0].Detail, `"`+c.wantWarning+`"`) {
				t.Errorf("warnings = %v, want one suggesting %q", warnings, c.wantWarning)
			}
		})
	}
}

// With exactString SAP takes the value literally, so regular expression
// characters are neither checked nor warned about.
func TestAccessPolicyReference_ExactStringValueIsLiteral(t *testing.T) {
	for _, value := range []string{"SALES_ORDERS_*", "*_ORDERS", "(IFL"} {
		diags := validateReference(t, "INTEGRATION_FLOW", "Name", "exactString", value)
		if len(diags) != 0 {
			t.Errorf("%q: diagnostics = %v, want none", value, diags)
		}
	}
}

// Every combination of the matrix is either valid or rejected with exactly
// one error, never with a confusing pile of them.
func TestAccessPolicyReference_MatrixIsConsistent(t *testing.T) {
	for _, typ := range referenceArtifactTypes {
		for _, attribute := range referenceAttributes {
			for _, operator := range referenceOperators {
				diags := validateReference(t, typ.WireValue, attribute, operator, "X")
				allowed := contains(typ.Attributes, attribute) && contains(typ.Operators, operator)
				errs := diagnosticsOf(diags, tfprotov6.DiagnosticSeverityError)
				if allowed != (len(errs) == 0) || len(errs) > 1 {
					t.Errorf("%s/%s/%s: allowed = %v, errors = %v", typ.WireValue, attribute, operator, allowed, errs)
				}
			}
		}
	}
}

// Read and import take whatever SAP returns, including values this provider
// version does not create, such as a type SAP adds later. Only configured
// values are validated, so an existing reference never fails a refresh.
func TestAccessPolicyReference_ReadKeepsValuesTheProviderDoesNotCreate(t *testing.T) {
	got := referenceToModel("1901", &cloudintegration.AccessPolicyReference{
		ID: "7", Name: "credentials", Type: "USER_CREDENTIAL", ConditionAttribute: "ID",
		ConditionType: "someFutureType", ConditionValue: "CRED_.*",
	})
	for name, pair := range map[string][2]string{
		"artifact_type": {got.ArtifactType.ValueString(), "USER_CREDENTIAL"},
		"attribute":     {got.Attribute.ValueString(), "ID"},
		"operator":      {got.Operator.ValueString(), "someFutureType"},
		"value":         {got.Value.ValueString(), "CRED_.*"},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s = %q, want %q", name, pair[0], pair[1])
		}
	}
}

// The table of supported combinations on the Registry page is written by
// hand; it must list exactly the matrix the provider enforces.
func TestAccessPolicyReference_DocsListTheMatrix(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "templates", "resources", "access_policy_reference.md.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]string{}
	for _, line := range strings.Split(string(src), "\n") {
		if strings.HasPrefix(line, "| `") && strings.Count(line, "|") == 6 {
			rows[strings.Trim(strings.Split(line, "|")[1], " `")] = line
		}
	}
	for _, typ := range referenceArtifactTypes {
		row, ok := rows[typ.WireValue]
		if !ok {
			t.Errorf("%s is missing from the table of supported combinations", typ.WireValue)
			continue
		}
		delete(rows, typ.WireValue)
		if nameOnly := len(typ.Attributes) == 1; nameOnly != strings.Contains(row, "`Name` only") {
			t.Errorf("%s: row %q does not match the attributes %v", typ.WireValue, row, typ.Attributes)
		}
		if exactOnly := len(typ.Operators) == 1; exactOnly != strings.Contains(row, "`exactString` only") {
			t.Errorf("%s: row %q does not match the operators %v", typ.WireValue, row, typ.Operators)
		}
		if typ.Unofficial != strings.HasSuffix(row, "| unofficial |") {
			t.Errorf("%s: row %q does not match Unofficial = %v", typ.WireValue, row, typ.Unofficial)
		}
	}
	for value := range rows {
		if strings.ToUpper(value) == value {
			t.Errorf("the table lists %s, which the provider does not accept", value)
		}
	}
}

// Types SAP accepts but does not document for access policies pass
// validation, so configurations that worked before keep their meaning, but a
// plan that creates such a reference needs enable_unofficial. Refreshing or
// destroying an existing one does not.
func TestAccessPolicyReference_UnofficialArtifactTypes(t *testing.T) {
	requireNoErrors(t, validateReference(t, "USER_CREDENTIAL", "ID", "regularExpression", "CRED_.*"))

	ci := cloudintegration.New(http.DefaultClient, "https://tenant.example")
	values := func(artifactType string) map[string]tftypes.Value {
		return map[string]tftypes.Value{
			"id": tftypes.NewValue(tftypes.String, tftypes.UnknownValue), "access_policy_id": str("1901"),
			"name": str("credentials"), "artifact_type": str(artifactType), "attribute": str("Name"),
			"operator": str("exactString"), "value": str("CRED_1"),
		}
	}
	existing := values("USER_CREDENTIAL")
	existing["id"] = str("1901/7")
	cases := []struct {
		name            string
		artifactType    string
		state           map[string]tftypes.Value
		replace, client bool
		allow, wantErr  bool
	}{
		{"create unofficial type", "USER_CREDENTIAL", nil, false, true, false, true},
		{"create unofficial type with enable_unofficial", "USER_CREDENTIAL", nil, false, true, true, false},
		{"replace into unofficial type", "SECURE_PARAMETER", existing, true, true, false, true},
		{"refresh existing unofficial reference", "USER_CREDENTIAL", existing, false, true, false, false},
		{"create documented type", "INTEGRATION_FLOW", nil, false, true, false, false},
		{"provider not configured yet", "USER_CREDENTIAL", nil, false, false, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &accessPolicyReferenceResource{allowUnofficial: c.allow}
			if c.client {
				r.client = ci
			}
			plan := values(c.artifactType)
			if c.state != nil {
				plan = c.state
				plan["artifact_type"] = str(c.artifactType)
			}
			diags := modifyPlan(t, r, c.state, plan, c.replace)
			if got := unofficialOperationError(diags); got != c.wantErr {
				t.Errorf("unofficial error = %v, want %v (%v)", got, c.wantErr, diags)
			}
		})
	}
}

func TestAccessPolicyReference_UILabelsPassValidation(t *testing.T) {
	for _, c := range [][3]string{
		{"Integration Flow", "Name", "Matches"},
		{"IntegrationPackage", "name", "EQUALS"},
		{"integration_flow", "Id", "equals"},
		{"user credential", "NAME", "matches"},
	} {
		t.Run(strings.Join(c[:], "/"), func(t *testing.T) {
			value := "X"
			if canonicalUILabel(uiLabelOperator, c[2]) == referenceOperatorRegex {
				value = "X.*"
			}
			requireNoErrors(t, validateReference(t, c[0], c[1], c[2], value))
		})
	}
	// The combination rules apply to what the labels stand for.
	requireOneError(t, validateReference(t, "Integration Package", "Name", "Matches", "PKG_.*"),
		"operator", "Operator not supported for this artifact type", "INTEGRATION_PACKAGE")
}

func TestConvertUILabel(t *testing.T) {
	cases := []struct {
		kind      uiLabelKind
		value     string
		want      string
		converted bool
	}{
		{uiLabelOperator, "Matches", referenceOperatorRegex, true},
		{uiLabelOperator, "EQUALS", referenceOperatorExact, true},
		{uiLabelOperator, "regularExpression", "", false},
		{uiLabelOperator, "regex", "", false},
		{uiLabelAttribute, "name", "Name", true},
		{uiLabelAttribute, "Id", "ID", true},
		{uiLabelAttribute, "Name", "", false},
		{uiLabelArtifactType, "Integration Flow", "INTEGRATION_FLOW", true},
		{uiLabelArtifactType, "Service Interface", "SERVICE_INTERFACE", true},
		{uiLabelArtifactType, "INTEGRATION_FLOW", "", false},
		// The UI's names for these constants are not proven.
		{uiLabelArtifactType, "API", "", false},
		{uiLabelArtifactType, "OData API", "", false},
		{uiLabelArtifactType, "REST API", "", false},
		{uiLabelArtifactType, "SOAP API", "", false},
	}
	for _, c := range cases {
		got, ok := convertUILabel(c.kind, c.value)
		if got != c.want || ok != c.converted {
			t.Errorf("convertUILabel(%v, %q) = %q, %v; want %q, %v", c.kind, c.value, got, ok, c.want, c.converted)
		}
	}
}

func TestUILabelConversion_Switches(t *testing.T) {
	labels := map[uiLabelKind]string{uiLabelArtifactType: "Integration Flow", uiLabelOperator: "Matches", uiLabelAttribute: "Name"}
	for _, c := range []struct {
		convert, experimental bool
		errors, warnings      int
		missing               []string
	}{
		{false, false, 2, 0, []string{"convert_ui_labels = true and enable_experimental = true"}},
		{true, false, 2, 0, []string{"set enable_experimental = true"}},
		{false, true, 2, 0, []string{"set convert_ui_labels = true"}},
		{true, true, 0, 2, nil},
	} {
		var diags diag.Diagnostics
		ok := uiLabelConversion(c.convert, c.experimental, labels, &diags)
		if ok != (c.errors == 0) || diags.ErrorsCount() != c.errors || diags.WarningsCount() != c.warnings {
			t.Fatalf("convert=%v experimental=%v: ok %v, %d errors, %d warnings", c.convert, c.experimental, ok, diags.ErrorsCount(), diags.WarningsCount())
		}
		for _, d := range diags {
			for _, m := range c.missing {
				if !strings.Contains(d.Detail(), m) {
					t.Errorf("detail %q does not name %q", d.Detail(), m)
				}
			}
			if !strings.Contains(d.Detail(), "INTEGRATION_FLOW") && !strings.Contains(d.Detail(), "regularExpression") {
				t.Errorf("detail %q does not name the constant", d.Detail())
			}
		}
	}
}

func TestUILabelEquivalentAndSpelling(t *testing.T) {
	m := uiLabelEquivalent{kind: uiLabelOperator}
	for _, c := range []struct {
		state, plan, want string
	}{
		{"regularExpression", "Matches", "regularExpression"}, // after an import: no diff
		{"Matches", "regularExpression", "Matches"},
		{"exactString", "Matches", "Matches"}, // a real change stays a change
	} {
		req := planmodifier.StringRequest{StateValue: types.StringValue(c.state), PlanValue: types.StringValue(c.plan)}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		m.PlanModifyString(context.Background(), req, resp)
		if resp.PlanValue.ValueString() != c.want {
			t.Errorf("state %q, plan %q: planned %q, want %q", c.state, c.plan, resp.PlanValue.ValueString(), c.want)
		}
	}
	if got := keepSpelling(uiLabelArtifactType, types.StringValue("Integration Flow"), "INTEGRATION_FLOW"); got.ValueString() != "Integration Flow" {
		t.Errorf("keepSpelling kept %q", got.ValueString())
	}
	if got := keepSpelling(uiLabelArtifactType, types.StringValue("Integration Flow"), "VALUE_MAPPING"); got.ValueString() != "VALUE_MAPPING" {
		t.Errorf("keepSpelling must show a different stored value, got %q", got.ValueString())
	}
}
