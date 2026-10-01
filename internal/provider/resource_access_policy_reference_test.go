package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

func accessPolicyReferenceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewAccessPolicyReferenceResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() produced diagnostics: %v", resp.Diagnostics)
	}
	return resp
}

func TestAccessPolicyReferenceResource_SchemaRequiredOptionalComputed(t *testing.T) {
	s := accessPolicyReferenceSchema(t).Schema

	cases := []struct {
		name                         string
		required, optional, computed bool
	}{
		{"id", false, false, true},
		{"access_policy_id", true, false, false},
		{"name", true, false, false},
		{"description", false, true, false},
		{"artifact_type", true, false, false},
		{"attribute", true, false, false},
		{"operator", true, false, false},
		{"value", true, false, false},
	}

	if len(s.Attributes) != len(cases) {
		t.Errorf("schema has %d attributes, want %d", len(s.Attributes), len(cases))
	}
	for _, c := range cases {
		attr, ok := s.Attributes[c.name]
		if !ok {
			t.Errorf("missing attribute %q", c.name)
			continue
		}
		if attr.IsRequired() != c.required || attr.IsOptional() != c.optional || attr.IsComputed() != c.computed {
			t.Errorf("%s: required/optional/computed = %v/%v/%v, want %v/%v/%v", c.name,
				attr.IsRequired(), attr.IsOptional(), attr.IsComputed(), c.required, c.optional, c.computed)
		}
	}
}

// No in-place update contract for ArtifactReferences has been confirmed, so
// every configurable attribute forces replacement and Update is unreachable.
func TestAccessPolicyReferenceResource_EveryConfigurableFieldRequiresReplace(t *testing.T) {
	s := accessPolicyReferenceSchema(t).Schema

	for _, name := range []string{"access_policy_id", "name", "description", "artifact_type", "attribute", "operator", "value"} {
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
}

func TestAccessPolicyReferenceResource_Update_AlwaysErrors(t *testing.T) {
	var resp resource.UpdateResponse
	NewAccessPolicyReferenceResource().Update(context.Background(), resource.UpdateRequest{}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Errorf("Update() did not produce a diagnostic; every field is RequiresReplace so Update should be unreachable")
	}
}

func TestAccessPolicyReferenceResource_ImportState(t *testing.T) {
	r := NewAccessPolicyReferenceResource().(resource.ResourceWithImportState)

	resp := &resource.ImportStateResponse{State: newTestState(t, accessPolicyReferenceSchema(t).Schema)}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "1901/55"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() produced diagnostics: %v", resp.Diagnostics)
	}

	var accessPolicyID, id types.String
	resp.Diagnostics.Append(resp.State.GetAttribute(context.Background(), pathRoot("access_policy_id"), &accessPolicyID)...)
	resp.Diagnostics.Append(resp.State.GetAttribute(context.Background(), pathRootID(), &id)...)
	if accessPolicyID.ValueString() != "1901" {
		t.Errorf("access_policy_id = %q, want %q", accessPolicyID.ValueString(), "1901")
	}
	if id.ValueString() != "1901/55" {
		t.Errorf("id = %q, want %q", id.ValueString(), "1901/55")
	}
}

func TestAccessPolicyReferenceResource_ImportState_InvalidID(t *testing.T) {
	r := NewAccessPolicyReferenceResource().(resource.ResourceWithImportState)

	for _, id := range []string{"not-composite", "1901/ref-1", "abc/55"} {
		resp := &resource.ImportStateResponse{State: newTestState(t, accessPolicyReferenceSchema(t).Schema)}
		r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("ImportState(%q) should produce a diagnostic", id)
		}
	}
}

func TestReferenceToModel_MapsWireFields(t *testing.T) {
	got := referenceToModel("1901", &cloudintegration.AccessPolicyReference{
		ID:                 "55",
		Name:               "utilities flows",
		Type:               "INTEGRATION_FLOW",
		ConditionAttribute: "Name",
		ConditionType:      "exactString",
		ConditionValue:     "Metering",
	})

	checks := map[string][2]string{
		"id":            {got.ID.ValueString(), "1901/55"},
		"name":          {got.Name.ValueString(), "utilities flows"},
		"artifact_type": {got.ArtifactType.ValueString(), "INTEGRATION_FLOW"},
		"attribute":     {got.Attribute.ValueString(), "Name"},
		"operator":      {got.Operator.ValueString(), "exactString"},
		"value":         {got.Value.ValueString(), "Metering"},
	}
	for name, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s = %q, want %q", name, c[0], c[1])
		}
	}
	if !got.Description.IsNull() {
		t.Errorf("description = %v, want null for an empty SAP description", got.Description)
	}
}

func TestReferenceIDFrom(t *testing.T) {
	if got := referenceIDFrom("1901/55"); got != "55" {
		t.Errorf("referenceIDFrom(%q) = %q, want %q", "1901/55", got, "55")
	}
}

func runStringValidator(v validator.String, value string) (bool, string) {
	resp := &validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("attr"),
		ConfigValue: types.StringValue(value),
	}, resp)
	if !resp.Diagnostics.HasError() {
		return false, ""
	}
	return true, resp.Diagnostics.Errors()[0].Detail()
}

func TestInt64StringValidator(t *testing.T) {
	for value, wantRejected := range map[string]bool{"1901": false, "0": false, "abc": true, "1901L": true, "1'": true} {
		if rejected, _ := runStringValidator(int64StringValidator{}, value); rejected != wantRejected {
			t.Errorf("%q: rejected = %v, want %v", value, rejected, wantRejected)
		}
	}
}
