package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/securitycontent"
)

func oauth2ClientCredentialSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewOAuth2ClientCredentialResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() produced diagnostics: %v", resp.Diagnostics)
	}
	return resp
}

func TestOAuth2ClientCredentialResource_SchemaRequiredComputed(t *testing.T) {
	s := oauth2ClientCredentialSchema(t).Schema

	cases := []struct {
		name               string
		required, computed bool
	}{
		{"id", true, false},
		{"description", false, false},
		{"token_service_url", true, false},
		{"client_id", true, false},
		{"scope", false, false},
		{"client_authentication", false, true},
		{"scope_content_type", false, true},
		{"resource", false, true},
		{"audience", false, true},
		{"client_secret_wo", true, false},
		{"client_secret_wo_version", true, false},
	}

	for _, c := range cases {
		attr, ok := s.Attributes[c.name]
		if !ok {
			t.Errorf("missing attribute %q", c.name)
			continue
		}
		if attr.IsRequired() != c.required {
			t.Errorf("%s.Required = %v, want %v", c.name, attr.IsRequired(), c.required)
		}
		if attr.IsComputed() != c.computed {
			t.Errorf("%s.Computed = %v, want %v", c.name, attr.IsComputed(), c.computed)
		}
	}
}

// TestOAuth2ClientCredentialResource_ClientSecretIsWriteOnlyAndSensitive is
// the safety property this whole resource depends on: client_secret_wo must
// never be persisted to plan or state.
func TestOAuth2ClientCredentialResource_ClientSecretIsWriteOnlyAndSensitive(t *testing.T) {
	s := oauth2ClientCredentialSchema(t).Schema

	attr, ok := s.Attributes["client_secret_wo"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("client_secret_wo is not a StringAttribute")
	}
	if !attr.IsWriteOnly() {
		t.Error("client_secret_wo must be WriteOnly")
	}
	if !attr.IsSensitive() {
		t.Error("client_secret_wo must be Sensitive")
	}
}

func TestOAuth2ClientCredentialResource_IdForcesReplaceButOthersDoNot(t *testing.T) {
	s := oauth2ClientCredentialSchema(t).Schema

	idAttr, ok := s.Attributes["id"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("id is not a StringAttribute")
	}
	var mods []interface {
		Description(context.Context) string
	}
	for _, m := range idAttr.PlanModifiers {
		mods = append(mods, m)
	}
	if !hasRequiresReplace(mods) {
		t.Error("id has no RequiresReplace plan modifier")
	}

	mutableFields := []string{"description", "token_service_url", "client_id", "scope", "client_secret_wo_version"}
	for _, name := range mutableFields {
		attr, ok := s.Attributes[name].(schema.StringAttribute)
		if !ok {
			t.Fatalf("attribute %q is not a StringAttribute", name)
		}
		if len(attr.PlanModifiers) != 0 {
			t.Errorf("%s should have no plan modifiers (must trigger in-place Update, not Replace)", name)
		}
	}
}

func TestOAuth2ClientCredentialResource_ImportState(t *testing.T) {
	r := NewOAuth2ClientCredentialResource().(resource.ResourceWithImportState)

	resp := &resource.ImportStateResponse{State: newTestState(t, oauth2ClientCredentialSchema(t).Schema)}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "BACKEND_OAUTH"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() produced diagnostics: %v", resp.Diagnostics)
	}

	var id types.String
	resp.Diagnostics.Append(resp.State.GetAttribute(context.Background(), pathRootID(), &id)...)
	if id.ValueString() != "BACKEND_OAUTH" {
		t.Errorf("id = %q, want BACKEND_OAUTH", id.ValueString())
	}
}

func TestOAuth2ClientCredentialResource_TokenRequestSettingsKeepPriorState(t *testing.T) {
	s := oauth2ClientCredentialSchema(t).Schema

	for _, name := range []string{"client_authentication", "scope_content_type", "resource", "audience"} {
		attr, ok := s.Attributes[name].(schema.StringAttribute)
		if !ok {
			t.Fatalf("attribute %q missing or not a StringAttribute", name)
		}
		if !attr.Optional || !attr.Computed {
			t.Errorf("%s must be Optional+Computed so SAP defaults and UI values are tracked", name)
		}
		keepsState := false
		for _, m := range attr.PlanModifiers {
			if m.Description(context.Background()) == stringplanmodifier.UseStateForUnknown().Description(context.Background()) {
				keepsState = true
			}
		}
		if !keepsState {
			t.Errorf("%s must use UseStateForUnknown so an omitted value is resent, not cleared, on PUT", name)
		}
	}
}

// customParametersValue builds the tftypes value of custom_parameters.
func customParametersValue(params ...[3]string) tftypes.Value {
	obj := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"key": tftypes.String, "value": tftypes.String, "send_as_part_of": tftypes.String}}
	var elems []tftypes.Value
	for _, p := range params {
		elems = append(elems, tftypes.NewValue(obj, map[string]tftypes.Value{"key": str(p[0]), "value": str(p[1]), "send_as_part_of": str(p[2])}))
	}
	return tftypes.NewValue(tftypes.Set{ElementType: obj}, elems)
}

// SAP takes custom parameters only on create and deletes them with every
// update (tenant check of 2026-10-04).
func TestOAuth2ClientCredentialResource_CustomParametersPlan(t *testing.T) {
	ci := securitycontent.New(http.DefaultClient, "https://tenant.example")
	params := customParametersValue([3]string{"resource", "https://graph.example", "body"})
	base := func(description string) map[string]tftypes.Value {
		return map[string]tftypes.Value{"id": str("OA"), "client_id": str("c"), "description": str(description), "custom_parameters": params}
	}

	// Creating with custom parameters needs enable_unofficial.
	off := modifyPlan(t, &oauth2ClientCredentialResource{client: ci}, nil, base("new"), false)
	if !unofficialOperationError(off) {
		t.Errorf("create without enable_unofficial: %v, want a refusal", off)
	}
	if on := modifyPlan(t, &oauth2ClientCredentialResource{client: ci, allowUnofficial: true}, nil, base("new"), false); on.HasError() {
		t.Errorf("create with enable_unofficial: %v", on)
	}
	if early := modifyPlan(t, &oauth2ClientCredentialResource{}, nil, base("new"), false); early.HasError() {
		t.Errorf("before the provider is configured: %v", early)
	}

	// Any change of a credential with custom parameters replaces it.
	r := &oauth2ClientCredentialResource{client: ci, allowUnofficial: true}
	_, empty := resourceObject(t, r, nil)
	stateRaw, _ := resourceObject(t, r, base("old"))
	planRaw, _ := resourceObject(t, r, base("new"))
	req := resource.ModifyPlanRequest{
		State: tfsdk.State{Schema: empty.Schema, Raw: stateRaw},
		Plan:  tfsdk.Plan{Schema: empty.Schema, Raw: planRaw},
	}
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(context.Background(), req, resp)
	if resp.Diagnostics.HasError() || len(resp.RequiresReplace) != 1 || !resp.RequiresReplace[0].Equal(path.Root("description")) {
		t.Errorf("update: requires replace %v, diagnostics %v; want description", resp.RequiresReplace, resp.Diagnostics)
	}

	// No change, no replacement.
	resp = &resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: empty.Schema, Raw: stateRaw}}
	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{State: req.State, Plan: tfsdk.Plan{Schema: empty.Schema, Raw: stateRaw}}, resp)
	if len(resp.RequiresReplace) != 0 || resp.Diagnostics.HasError() {
		t.Errorf("no change: %v, %v", resp.RequiresReplace, resp.Diagnostics)
	}
}

// Without custom_parameters, an update warns about parameters SAP holds.
func TestOAuth2ClientCredentialResource_WarnsAboutUnmanagedParameters(t *testing.T) {
	for _, c := range []struct {
		name     string
		response string
		warnings int
	}{
		{"parameters set in the UI", `{"d":{"Name":"OA","CustomParameters":{"results":[{"Key":"k","Value":"v","SendAsPartOf":"header"}]}}}`, 1},
		{"no parameters", `{"d":{"Name":"OA","CustomParameters":{"results":[]}}}`, 0},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(c.response))
		}))
		r := &oauth2ClientCredentialResource{client: securitycontent.New(http.DefaultClient, server.URL)}
		diags := modifyPlan(t, r,
			map[string]tftypes.Value{"id": str("OA"), "client_id": str("one")},
			map[string]tftypes.Value{"id": str("OA"), "client_id": str("two")}, false)
		server.Close()
		if diags.HasError() || diags.WarningsCount() != c.warnings {
			t.Errorf("%s: %v, want %d warning(s)", c.name, diags, c.warnings)
		}
	}
}
