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
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
)

func businessDataGraphSchema(t *testing.T) schema.Schema {
	t.Helper()
	r := NewBusinessDataGraphResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() produced diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// businessDataGraphPlan builds a plan for Create from a model, the way
// Terraform would after planning: computed values unknown, timeouts null.
func businessDataGraphPlan(t *testing.T, s schema.Schema, m businessDataGraphModel) tfsdk.Plan {
	t.Helper()
	m.ID = types.StringUnknown()
	m.SchemaVersion = types.StringUnknown()
	m.GraphModelVersion = types.StringUnknown()
	m.EffectiveGraphModelVersion = types.StringUnknown()
	m.Status = types.StringUnknown()
	m.StatusDetails = types.StringUnknown()
	m.Timeouts = timeouts.Value{Object: types.ObjectNull(map[string]attr.Type{
		"create": types.StringType,
		"update": types.StringType,
	})}
	plan := tfsdk.Plan{Schema: s, Raw: newTestState(t, s).Raw}
	if diags := plan.Set(context.Background(), m); diags.HasError() {
		t.Fatalf("building plan: %v", diags)
	}
	return plan
}

func sampleBusinessDataGraph() businessDataGraphModel {
	return businessDataGraphModel{
		BusinessDataGraphIdentifier: types.StringValue("my-bdg"),
		DataSources: []businessDataSourceModel{{
			Name:      types.StringValue("s4"),
			Namespace: types.StringNull(),
			Services: []businessDataSourceServiceModel{{
				DestinationName: types.StringValue("s4-business-partner"),
				Path:            types.StringNull(),
			}},
		}},
		LocatingPolicy: &locatingPolicyModel{
			Cues: []locatingCueModel{{Name: types.StringValue("emea"), Description: types.StringNull()}},
			Rules: []locatingRuleModel{{
				Name:         types.StringValue("sap.s4.*"),
				Leading:      types.StringValue("s4"),
				Cues:         []string{"emea"},
				SourceEntity: types.StringNull(),
			}},
		},
	}
}

func TestBusinessDataGraphResource_Schema_IdentifierIsRequiresReplace(t *testing.T) {
	s := businessDataGraphSchema(t)

	attr, ok := s.Attributes["business_data_graph_identifier"].(schema.StringAttribute)
	if !ok {
		t.Fatal("business_data_graph_identifier is not a schema.StringAttribute")
	}
	if !attr.Required {
		t.Error("business_data_graph_identifier must be Required")
	}
	if len(attr.PlanModifiers) == 0 {
		t.Error("business_data_graph_identifier has no plan modifier; expected RequiresReplace since this resource addresses every operation by this value")
	}
}

func TestBusinessDataGraphResource_Schema_DataSourcesAndLocatingPolicyAreRequired(t *testing.T) {
	s := businessDataGraphSchema(t)

	if ds, ok := s.Attributes["data_sources"].(schema.ListNestedAttribute); !ok || !ds.Required {
		t.Error("data_sources must be a Required schema.ListNestedAttribute")
	}
	if lp, ok := s.Attributes["locating_policy"].(schema.SingleNestedAttribute); !ok || !lp.Required {
		t.Error("locating_policy must be a Required schema.SingleNestedAttribute")
	}
	if ext, ok := s.Attributes["extensions"].(schema.ListAttribute); !ok || ext.Optional || !ext.Computed {
		t.Error("extensions must be read-only: SAP's Configuration API does not manage extensions")
	}
}

func TestBusinessDataGraphIDPattern(t *testing.T) {
	for id, want := range map[string]bool{
		"my-bdg":        true,
		"abc-xyz-123mn": true,
		"bdg1":          true,
		"My-BDG":        false,
		"-bdg":          false,
		"bdg-":          false,
		"my_bdg":        false,
		"my--bdg":       false,
	} {
		if got := businessDataGraphIDPattern.MatchString(id); got != want {
			t.Errorf("%q: match = %v, want %v", id, got, want)
		}
	}
}

func TestBusinessDataGraphResource_Configure_RequiresAPICompositionClient(t *testing.T) {
	r := NewBusinessDataGraphResource().(resource.ResourceWithConfigure)
	var resp resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: &Data{}}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a configuration error when APICompositionHTTPClient is nil")
	}
}

// SAP may return empty lists for properties the configuration left out;
// they must come back as null, or Terraform reports an inconsistent result.
func TestBusinessDataGraphFromClient_EmptyListsBecomeNull(t *testing.T) {
	got := businessDataGraphFromClient(&apicomposition.GraphConfiguration{
		BusinessDataGraphIdentifier: "my-bdg",
		Exclude:                     []string{},
		LocatingPolicy: apicomposition.LocatingPolicy{
			Cues:  []apicomposition.LocatingCue{},
			Rules: []apicomposition.LocatingRule{{Name: "sap.s4.*", Leading: "s4", Local: []string{}, Cues: []string{}}},
		},
	})
	if got.Exclude != nil {
		t.Errorf("exclude = %#v, want nil", got.Exclude)
	}
	if got.LocatingPolicy.Cues != nil {
		t.Errorf("cues = %#v, want nil", got.LocatingPolicy.Cues)
	}
	if r := got.LocatingPolicy.Rules[0]; r.Local != nil || r.Cues != nil {
		t.Errorf("rule = %#v, want nil local and cues", r)
	}
}

func TestBusinessDataGraphResource_Create_PollsUntilDeploymentInitiated(t *testing.T) {
	var posted map[string]json.RawMessage
	getCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &posted); err != nil {
				t.Errorf("decoding POST body: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"businessDataGraphIdentifier": "my-bdg", "status": "PROCESSING"}`))
		case http.MethodGet:
			getCount++
			_, _ = w.Write([]byte(`{"businessDataGraphIdentifier": "my-bdg", "schemaVersion": "1.0", "graphModelVersion": "1.0.0",
				"status": "DEPLOYMENT_INITIATED", "exclude": [], "extensions": [],
				"dataSources": [{"name": "s4", "services": [{"destinationName": "s4-business-partner", "path": ""}]}],
				"locatingPolicy": {"cues": [{"name": "emea"}], "rules": [{"name": "sap.s4.*", "leading": "s4", "cues": ["emea"]}]}}`))
		}
	}))
	defer server.Close()

	r := &businessDataGraphResource{client: apicomposition.New(http.DefaultClient, server.URL)}
	s := businessDataGraphSchema(t)
	ctx := context.Background()

	resp := &resource.CreateResponse{State: newTestState(t, s)}
	r.Create(ctx, resource.CreateRequest{Plan: businessDataGraphPlan(t, s, sampleBusinessDataGraph())}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create() produced diagnostics: %v", resp.Diagnostics)
	}
	if getCount == 0 {
		t.Error("expected Create to poll GET at least once to leave PROCESSING")
	}
	if got := string(posted["locatingPolicy"]); got != `{"cues":[{"name":"emea"}],"rules":[{"name":"sap.s4.*","leading":"s4","cues":["emea"]}]}` {
		t.Errorf("locatingPolicy sent as %s", got)
	}
	for _, key := range []string{"schemaVersion", "graphModelVersion"} {
		if _, ok := posted[key]; ok {
			t.Errorf("%s was sent although it was not configured", key)
		}
	}

	var got businessDataGraphModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.ID.ValueString() != "my-bdg" {
		t.Errorf("id = %q, want my-bdg", got.ID.ValueString())
	}
	if got.Status.ValueString() != apicomposition.StatusDeploymentInitiated {
		t.Errorf("status = %q, want DEPLOYMENT_INITIATED", got.Status.ValueString())
	}
	if got.GraphModelVersion.ValueString() != "1.0.0" {
		t.Errorf("graph_model_version = %q, want the value SAP filled in", got.GraphModelVersion.ValueString())
	}
}

// A graph SAP failed to process still exists; it must land in state so
// Terraform can taint it instead of losing track of it.
func TestBusinessDataGraphResource_Create_FailedKeepsState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		_, _ = w.Write([]byte(`{"businessDataGraphIdentifier": "my-bdg", "status": "FAILED", "statusDetails": "destination s4-business-partner not found",
			"dataSources": [{"name": "s4", "services": [{"destinationName": "s4-business-partner"}]}],
			"locatingPolicy": {"rules": [{"name": "sap.s4.*", "leading": "s4"}]}}`))
	}))
	defer server.Close()

	r := &businessDataGraphResource{client: apicomposition.New(http.DefaultClient, server.URL)}
	s := businessDataGraphSchema(t)
	ctx := context.Background()

	resp := &resource.CreateResponse{State: newTestState(t, s)}
	r.Create(ctx, resource.CreateRequest{Plan: businessDataGraphPlan(t, s, sampleBusinessDataGraph())}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error for status FAILED")
	}

	var got businessDataGraphModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.ID.ValueString() != "my-bdg" || got.Status.ValueString() != apicomposition.StatusFailed {
		t.Errorf("state = id %q status %q, want the failed graph recorded", got.ID.ValueString(), got.Status.ValueString())
	}
}

func TestBusinessDataGraphErrorDetail_ForbiddenNamesTheRole(t *testing.T) {
	err := &apierror.Error{StatusCode: http.StatusForbidden, Code: "2707", Message: "check your assigned roles", RequestID: "trace-1"}
	detail := businessDataGraphErrorDetail(err)
	for _, want := range []string{"HTTP 403", `"2707"`, "SAP trace ID: trace-1", "Graph_Key_User"} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail does not contain %q:\n%s", want, detail)
		}
	}
}

func TestBusinessDataGraphErrorDetail_OtherErrorsHaveNoRoleHint(t *testing.T) {
	detail := businessDataGraphErrorDetail(&apierror.Error{StatusCode: http.StatusBadRequest, Message: "invalid locating policy"})
	if strings.Contains(detail, "Graph_Key_User") || strings.Contains(detail, "trace ID") {
		t.Errorf("unexpected hint in a 400 without trace ID:\n%s", detail)
	}
}
