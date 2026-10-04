package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func userCredentialSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewUserCredentialResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() produced diagnostics: %v", resp.Diagnostics)
	}
	return resp
}

func TestUserCredentialResource_SchemaRequiredComputed(t *testing.T) {
	s := userCredentialSchema(t).Schema

	cases := []struct {
		name               string
		required, computed bool
	}{
		{"id", true, false},
		{"kind", false, true},
		{"description", false, false},
		{"user", true, false},
		{"company_id", false, false},
		{"password_wo", true, false},
		{"password_wo_version", true, false},
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

// TestUserCredentialResource_PasswordIsWriteOnlyAndSensitive is the safety
// property this whole resource depends on: password_wo must never be
// persisted to plan or state.
func TestUserCredentialResource_PasswordIsWriteOnlyAndSensitive(t *testing.T) {
	s := userCredentialSchema(t).Schema

	attr, ok := s.Attributes["password_wo"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("password_wo is not a StringAttribute")
	}
	if !attr.IsWriteOnly() {
		t.Error("password_wo must be WriteOnly")
	}
	if !attr.IsSensitive() {
		t.Error("password_wo must be Sensitive")
	}
}

// TestUserCredentialResource_IdentityFieldsForceReplace pins down that id
// (the credential's name/alias) and kind force replacement, while user,
// description, company_id, and password_wo_version do not — those are
// redeployed in place via Update, unlike
// sapintegrationsuite_partner_user_credential_parameter, which has no
// confirmed update path at all.
func TestUserCredentialResource_IdentityFieldsForceReplace(t *testing.T) {
	s := userCredentialSchema(t).Schema

	replaceFields := []string{"id", "kind"}
	for _, name := range replaceFields {
		attr, ok := s.Attributes[name].(schema.StringAttribute)
		if !ok {
			t.Fatalf("attribute %q is not a StringAttribute", name)
		}
		var mods []interface {
			Description(context.Context) string
		}
		for _, m := range attr.PlanModifiers {
			mods = append(mods, m)
		}
		if !hasRequiresReplace(mods) {
			t.Errorf("%s has no RequiresReplace plan modifier", name)
		}
	}

	mutableFields := []string{"user", "description", "company_id", "password_wo_version"}
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

func TestUserCredentialResource_ImportState(t *testing.T) {
	r := NewUserCredentialResource().(resource.ResourceWithImportState)

	resp := &resource.ImportStateResponse{State: newTestState(t, userCredentialSchema(t).Schema)}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "BACKEND_BASIC"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() produced diagnostics: %v", resp.Diagnostics)
	}

	var id types.String
	resp.Diagnostics.Append(resp.State.GetAttribute(context.Background(), pathRootID(), &id)...)
	if id.ValueString() != "BACKEND_BASIC" {
		t.Errorf("id = %q, want BACKEND_BASIC", id.ValueString())
	}
}

// SAP accepts only lowercase kinds; earlier releases documented
// "SuccessFactors" and "OpenConnectors", which SAP refuses at apply.
func TestUserCredentialResource_KindValidator(t *testing.T) {
	for _, c := range []struct {
		value string
		hint  string // "" = valid
	}{
		{"default", ""},
		{"successfactors", ""},
		{"openconnectors", ""},
		{"SuccessFactors", `Use "successfactors"`},
		{"OpenConnectors", `Use "openconnectors"`},
		{"Default", `Use "default"`},
		{"oauth2_password", "SAP accepts only"},
	} {
		req := validator.StringRequest{Path: path.Root("kind"), ConfigValue: types.StringValue(c.value)}
		var resp validator.StringResponse
		userCredentialKindValidator{}.ValidateString(context.Background(), req, &resp)
		if c.hint == "" {
			if resp.Diagnostics.HasError() {
				t.Errorf("%q: unexpected error %v", c.value, resp.Diagnostics)
			}
			continue
		}
		if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics[0].Detail(), c.hint) {
			t.Errorf("%q: diagnostics %v, want an error containing %q", c.value, resp.Diagnostics, c.hint)
		}
	}
}

func TestUserCredentialResource_ValidateConfig_SuccessFactorsNeedsCompanyID(t *testing.T) {
	r := NewUserCredentialResource().(resource.ResourceWithValidateConfig)
	for _, c := range []struct {
		kind, companyID string
		wantError       bool
	}{
		{"successfactors", "", true},
		{"successfactors", "COMPANY1", false},
		{"default", "", false},
	} {
		values := map[string]tftypes.Value{"kind": str(c.kind)}
		if c.companyID != "" {
			values["company_id"] = str(c.companyID)
		}
		raw, state := resourceObject(t, r, values)
		var resp resource.ValidateConfigResponse
		r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: state.Schema, Raw: raw}}, &resp)
		if resp.Diagnostics.HasError() != c.wantError {
			t.Errorf("kind %q, company_id %q: diagnostics %v, want error %v", c.kind, c.companyID, resp.Diagnostics, c.wantError)
		}
	}
}
