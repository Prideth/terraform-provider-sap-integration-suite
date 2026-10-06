package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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

// An update deletes custom parameters set in SAP's UI (tenant check of
// 2026-10-04); the plan must say so, and stay quiet when there are none.
func TestOAuth2ClientCredentialResource_ModifyPlanWarnsAboutCustomParameters(t *testing.T) {
	for _, c := range []struct {
		name     string
		response string
		changed  bool
		warnings int
	}{
		{"update with parameters", `{"d": {"Name": "BACKEND_OAUTH", "CustomParameters": {"results": [{"Key": "resource", "Value": "x", "SendAsPartOf": "body"}]}}}`, true, 1},
		{"update without parameters", `{"d": {"Name": "BACKEND_OAUTH", "CustomParameters": {"results": []}}}`, true, 0},
		{"no change", `{"d": {"Name": "BACKEND_OAUTH", "CustomParameters": {"results": [{"Key": "resource", "Value": "x", "SendAsPartOf": "body"}]}}}`, false, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(c.response))
			}))
			defer server.Close()
			r := &oauth2ClientCredentialResource{client: securitycontent.New(http.DefaultClient, server.URL)}
			state := map[string]tftypes.Value{"id": str("BACKEND_OAUTH"), "client_id": str("one")}
			plan := map[string]tftypes.Value{"id": str("BACKEND_OAUTH"), "client_id": str("one")}
			if c.changed {
				plan["client_id"] = str("two")
			}
			diags := modifyPlan(t, r, state, plan, false)
			if diags.HasError() || diags.WarningsCount() != c.warnings {
				t.Fatalf("diagnostics = %v, want %d warning(s) and no error", diags, c.warnings)
			}
			if c.warnings > 0 && !strings.Contains(diags[0].Detail(), "1 custom parameter") {
				t.Errorf("detail %q does not name the count", diags[0].Detail())
			}
		})
	}
}
