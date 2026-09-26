package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestAPIProxyResource_Configure_RequiresAPIManagementClient(t *testing.T) {
	r := NewAPIProxyResource().(resource.ResourceWithConfigure)
	var resp resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: &Data{}}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a configuration error when APIManagementClassicHTTPClient is nil")
	}
}

// name, content and content_hash are the only user inputs; every other
// attribute comes from SAP.
func TestAPIProxyResource_Schema(t *testing.T) {
	var resp resource.SchemaResponse
	NewAPIProxyResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	required := map[string]bool{"name": true, "content": true, "content_hash": true}
	for name, attr := range resp.Schema.Attributes {
		if attr.IsRequired() != required[name] {
			t.Errorf("%s: required = %v", name, attr.IsRequired())
		}
		if !required[name] && !attr.IsComputed() {
			t.Errorf("%s must be computed", name)
		}
	}
	for name := range required {
		a, ok := resp.Schema.Attributes[name].(schema.StringAttribute)
		if !ok || len(a.PlanModifiers) == 0 {
			t.Errorf("%s must force replacement", name)
		}
	}
}

// A changed bundle replaces the proxy, except on the first apply after an
// import, when the state has no content yet.
func TestReplaceUnlessImported(t *testing.T) {
	m := replaceUnlessImported("test")
	for _, tc := range []struct {
		name  string
		state types.String
		want  bool
	}{
		{"changed bundle", types.StringValue("old.zip"), true},
		{"after import", types.StringNull(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{StateValue: tc.state, PlanValue: types.StringValue("new.zip"), ConfigValue: types.StringValue("new.zip")}
			req.State.Raw = nonNullRaw()
			req.Plan.Raw = nonNullRaw()
			var resp planmodifier.StringResponse
			resp.PlanValue = req.PlanValue
			m.PlanModifyString(context.Background(), req, &resp)
			if resp.RequiresReplace != tc.want {
				t.Errorf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.want)
			}
		})
	}
}

// nonNullRaw is a present (non-null) plan or state, so that RequiresReplaceIf
// neither treats the request as a create nor as a destroy.
func nonNullRaw() tftypes.Value {
	return tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, map[string]tftypes.Value{})
}
