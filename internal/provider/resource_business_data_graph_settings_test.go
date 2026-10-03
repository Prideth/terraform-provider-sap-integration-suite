package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apicomposition"
)

// The settings below are named only in the Configuration API's $metadata:
// description, odata_containment, locating_policy.description and the cues
// of a key mapping. They are sent only when configured, read back, and
// setting them needs enable_unofficial.

// businessDataGraphConfig is m as Terraform passes the configuration: every
// attribute that is not set, computed ones included, is null.
func businessDataGraphConfig(t *testing.T, s schema.Schema, m businessDataGraphModel) tfsdk.Config {
	t.Helper()
	m.ID = types.StringNull()
	m.EffectiveGraphModelVersion = types.StringNull()
	m.Status = types.StringNull()
	m.StatusDetails = types.StringNull()
	m.Extensions = types.ListNull(types.StringType)
	m.LogMessages = types.ListNull(types.StringType)
	m.Timeouts = timeouts.Value{Object: types.ObjectNull(map[string]attr.Type{
		"create": types.StringType,
		"update": types.StringType,
	})}
	state := tfsdk.State{Schema: s, Raw: newTestState(t, s).Raw}
	if diags := state.Set(context.Background(), m); diags.HasError() {
		t.Fatalf("building config: %v", diags)
	}
	return tfsdk.Config{Schema: s, Raw: state.Raw}
}

func graphWithSettings() businessDataGraphModel {
	m := sampleBusinessDataGraph()
	m.Description = types.StringValue("Sales landscape")
	m.ODataContainment = types.BoolValue(false)
	m.LocatingPolicy.Description = types.StringValue("S/4 leads")
	m.LocatingPolicy.KeyMapping = []keyMappingModel{{
		Cues: []string{"emea"},
		ForeignKey: &keyMappingSideModel{
			DataSource: types.StringValue("s4"), EntityName: types.StringValue("sap.s4.A_SalesOrder"),
			Attributes: []string{"SoldToParty"},
		},
		References: &keyMappingSideModel{
			DataSource: types.StringValue("s4"), EntityName: types.StringValue("sap.s4.A_BusinessPartner"),
			Attributes: []string{"BusinessPartner"},
		},
	}}
	return m
}

// graphSettingsServer answers like the tenant and records the POST body.
func graphSettingsServer(t *testing.T, posted *map[string]json.RawMessage) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, posted); err != nil {
				t.Errorf("decoding POST body: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		_, _ = w.Write([]byte(`{"businessDataGraphIdentifier": "my-bdg", "status": "DEPLOYMENT_INITIATED",
			"description": "Sales landscape", "odataContainment": false,
			"dataSources": [{"name": "s4", "services": [{"destinationName": "s4-business-partner"}]}],
			"locatingPolicy": {"description": "S/4 leads", "cues": [{"name": "emea"}],
				"keyMapping": [{"cues": ["emea"],
					"foreignKey": {"dataSource": "s4", "entityName": "sap.s4.A_SalesOrder", "attributes": ["SoldToParty"]},
					"references": {"dataSource": "s4", "entityName": "sap.s4.A_BusinessPartner", "attributes": ["BusinessPartner"]}}],
				"rules": [{"name": "sap.s4.*", "leading": "s4", "cues": ["emea"]}]}}`))
	}))
}

func TestBusinessDataGraphResource_Create_SendsAndReadsMetadataSettings(t *testing.T) {
	var posted map[string]json.RawMessage
	server := graphSettingsServer(t, &posted)
	defer server.Close()

	r := &businessDataGraphResource{client: apicomposition.New(http.DefaultClient, server.URL), allowUnofficial: true}
	s := businessDataGraphSchema(t)
	ctx := context.Background()
	m := graphWithSettings()

	resp := &resource.CreateResponse{State: newTestState(t, s)}
	r.Create(ctx, resource.CreateRequest{Plan: businessDataGraphPlan(t, s, m), Config: businessDataGraphConfig(t, s, m)}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create() produced diagnostics: %v", resp.Diagnostics)
	}

	for key, want := range map[string]string{"description": `"Sales landscape"`, "odataContainment": `false`} {
		if got := string(posted[key]); got != want {
			t.Errorf("POST %s = %s, want %s", key, got, want)
		}
	}
	policy := string(posted["locatingPolicy"])
	for _, want := range []string{`"description":"S/4 leads"`, `"keyMapping":[{"cues":["emea"]`} {
		if !strings.Contains(policy, want) {
			t.Errorf("locatingPolicy %s does not contain %s", policy, want)
		}
	}

	var got businessDataGraphModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.Description.ValueString() != "Sales landscape" || got.ODataContainment.IsNull() || got.ODataContainment.ValueBool() {
		t.Errorf("state description = %v, odata_containment = %v", got.Description, got.ODataContainment)
	}
	if got.LocatingPolicy.Description.ValueString() != "S/4 leads" || strings.Join(got.LocatingPolicy.KeyMapping[0].Cues, ",") != "emea" {
		t.Errorf("state locating policy = %+v", got.LocatingPolicy)
	}
}

// Left out, the settings are not sent, and SAP's values land in state.
func TestBusinessDataGraphResource_Create_OmitsUnsetMetadataSettings(t *testing.T) {
	var posted map[string]json.RawMessage
	server := graphSettingsServer(t, &posted)
	defer server.Close()

	r := &businessDataGraphResource{client: apicomposition.New(http.DefaultClient, server.URL)}
	s := businessDataGraphSchema(t)
	ctx := context.Background()
	m := sampleBusinessDataGraph()

	resp := &resource.CreateResponse{State: newTestState(t, s)}
	r.Create(ctx, resource.CreateRequest{Plan: businessDataGraphPlan(t, s, m), Config: businessDataGraphConfig(t, s, m)}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create() without the settings and without enable_unofficial failed: %v", resp.Diagnostics)
	}
	for _, key := range []string{"description", "odataContainment"} {
		if _, ok := posted[key]; ok {
			t.Errorf("%s was sent although it was not configured", key)
		}
	}
	// The cue's own description is required; only the policy's must be absent.
	if strings.HasPrefix(string(posted["locatingPolicy"]), `{"description"`) {
		t.Errorf("locating policy description sent although not configured: %s", posted["locatingPolicy"])
	}
	var got businessDataGraphModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.Description.ValueString() != "Sales landscape" {
		t.Errorf("description = %v, want SAP's value in state", got.Description)
	}
}

func TestBusinessDataGraphResource_MetadataSettingsNeedEnableUnofficial(t *testing.T) {
	s := businessDataGraphSchema(t)
	ctx := context.Background()
	cases := map[string]func(*businessDataGraphModel){
		"description":       func(m *businessDataGraphModel) { m.Description = types.StringValue("x") },
		"odata_containment": func(m *businessDataGraphModel) { m.ODataContainment = types.BoolValue(true) },
		"policy description": func(m *businessDataGraphModel) {
			m.LocatingPolicy.Description = types.StringValue("x")
		},
		"key mapping cues": func(m *businessDataGraphModel) {
			*m = graphWithSettings()
			m.Description = types.StringNull()
			m.ODataContainment = types.BoolNull()
			m.LocatingPolicy.Description = types.StringNull()
		},
	}
	for name, set := range cases {
		t.Run(name, func(t *testing.T) {
			m := sampleBusinessDataGraph()
			set(&m)
			r := &businessDataGraphResource{client: apicomposition.New(http.DefaultClient, "https://graph.example")}

			// ModifyPlan stops the plan ...
			var planResp resource.ModifyPlanResponse
			planResp.Plan = businessDataGraphPlan(t, s, m)
			r.ModifyPlan(ctx, resource.ModifyPlanRequest{
				Config: businessDataGraphConfig(t, s, m), Plan: businessDataGraphPlan(t, s, m),
				State: newTestState(t, s),
			}, &planResp)
			if !planResp.Diagnostics.HasError() || !strings.Contains(planResp.Diagnostics.Errors()[0].Detail(), "enable_unofficial") {
				t.Fatalf("ModifyPlan diagnostics = %v, want the enable_unofficial error", planResp.Diagnostics)
			}

			// ... and Create refuses as well, before any request.
			createResp := &resource.CreateResponse{State: newTestState(t, s)}
			r.Create(ctx, resource.CreateRequest{Plan: businessDataGraphPlan(t, s, m), Config: businessDataGraphConfig(t, s, m)}, createResp)
			if !createResp.Diagnostics.HasError() {
				t.Fatal("Create accepted a $metadata-only setting without enable_unofficial")
			}
		})
	}
}
