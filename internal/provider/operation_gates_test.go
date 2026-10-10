package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apicomposition"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
	sapthttp "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/http"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/features"
)

// unofficialOperationError reports whether the diagnostics contain the error
// requireUnofficialOperation adds.
func unofficialOperationError(diags diag.Diagnostics) bool {
	for _, d := range diags.Errors() {
		if strings.Contains(d.Summary(), ": ") && strings.HasSuffix(d.Summary(), " is unofficial") {
			return true
		}
	}
	return false
}

// resourceObject builds a raw value for a resource's schema: every attribute
// null except the given ones.
func resourceObject(t *testing.T, r resource.Resource, values map[string]tftypes.Value) (tftypes.Value, tfsdk.State) {
	t.Helper()
	ctx := context.Background()
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	if sr.Diagnostics.HasError() {
		t.Fatalf("Schema() produced diagnostics: %v", sr.Diagnostics)
	}
	objType := sr.Schema.Type().TerraformType(ctx).(tftypes.Object)
	attrs := map[string]tftypes.Value{}
	for name, typ := range objType.AttributeTypes {
		if v, ok := values[name]; ok {
			attrs[name] = v
		} else {
			attrs[name] = tftypes.NewValue(typ, nil)
		}
	}
	return tftypes.NewValue(objType, attrs), tfsdk.State{Schema: sr.Schema, Raw: tftypes.NewValue(objType, nil)}
}

func str(v string) tftypes.Value { return tftypes.NewValue(tftypes.String, v) }

// modifyPlan runs a resource's ModifyPlan. A nil state or plan map means a
// create or a destroy.
func modifyPlan(t *testing.T, r resource.Resource, state, plan map[string]tftypes.Value, replace bool) diag.Diagnostics {
	t.Helper()
	_, empty := resourceObject(t, r, nil)
	s := empty.Schema
	stateRaw, planRaw := empty.Raw, empty.Raw
	if state != nil {
		stateRaw, _ = resourceObject(t, r, state)
	}
	if plan != nil {
		planRaw, _ = resourceObject(t, r, plan)
	}
	req := resource.ModifyPlanRequest{
		State:  tfsdk.State{Schema: s, Raw: stateRaw},
		Plan:   tfsdk.Plan{Schema: s, Raw: planRaw},
		Config: tfsdk.Config{Schema: s, Raw: planRaw},
	}
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}
	if replace {
		resp.RequiresReplace = path.Paths{path.Root("id")}
	}
	r.(resource.ResourceWithModifyPlan).ModifyPlan(context.Background(), req, resp)
	return resp.Diagnostics
}

// TestOperationGates_Plan checks every plan that needs an undocumented
// operation: refused without enable_unofficial, allowed with it, and never
// refused while the provider is not configured yet.
func TestOperationGates_Plan(t *testing.T) {
	ci := cloudintegration.New(http.DefaultClient, "https://tenant.example")
	ac := apicomposition.New(http.DefaultClient, "https://tenant.example")
	type gated interface {
		resource.Resource
		resource.ResourceWithModifyPlan
	}
	newVM := func(c, a bool) gated {
		r := &valueMappingResource{allowUnofficial: a}
		if c {
			r.client = ci
		}
		return r
	}
	newAP := func(c, a bool) gated {
		r := &accessPolicyResource{allowUnofficial: a}
		if c {
			r.client = ci
		}
		return r
	}
	newBDG := func(c, a bool) gated {
		r := &businessDataGraphResource{allowUnofficial: a}
		if c {
			r.client = ac
		}
		return r
	}
	newNR := func(c, a bool) gated {
		r := &numberRangeResource{allowUnofficial: a}
		if c {
			r.client = ci
		}
		return r
	}
	cases := []struct {
		name        string
		newResource func(configured, allow bool) gated
		state, plan map[string]tftypes.Value
		replace     bool
		wantRefusal bool
	}{
		{
			name:        "value mapping create with save_as_version",
			newResource: newVM,
			plan:        map[string]tftypes.Value{"id": str("P/VM"), "save_as_version": str("1.0.1")},
			wantRefusal: true,
		},
		{
			name:        "value mapping create without save_as_version",
			newResource: newVM,
			plan:        map[string]tftypes.Value{"id": str("P/VM")},
		},
		{
			name:        "value mapping relabel in place",
			newResource: newVM,
			state:       map[string]tftypes.Value{"id": str("P/VM"), "save_as_version": str("1.0.1")},
			plan:        map[string]tftypes.Value{"id": str("P/VM"), "save_as_version": str("1.0.2")},
			wantRefusal: true,
		},
		{
			name:        "value mapping save_as_version removed",
			newResource: newVM,
			state:       map[string]tftypes.Value{"id": str("P/VM"), "save_as_version": str("1.0.1")},
			plan:        map[string]tftypes.Value{"id": str("P/VM")},
		},
		{
			name:        "value mapping replacement keeping save_as_version",
			newResource: newVM,
			state:       map[string]tftypes.Value{"id": str("P/VM"), "name": str("old"), "save_as_version": str("1.0.1")},
			plan:        map[string]tftypes.Value{"name": str("new"), "save_as_version": str("1.0.1")},
			replace:     true,
			wantRefusal: true,
		},
		{
			name:        "access policy description update",
			newResource: newAP,
			state:       map[string]tftypes.Value{"id": str("1"), "description": str("old")},
			plan:        map[string]tftypes.Value{"id": str("1"), "description": str("new")},
			wantRefusal: true,
		},
		{
			name:        "access policy replacement",
			newResource: newAP,
			state:       map[string]tftypes.Value{"id": str("1"), "role_name": str("old")},
			plan:        map[string]tftypes.Value{"role_name": str("new")},
			replace:     true,
		},
		{
			name:        "business data graph update",
			newResource: newBDG,
			state:       map[string]tftypes.Value{"id": str("sap.graph.a")},
			plan:        map[string]tftypes.Value{"id": str("sap.graph.b")},
			wantRefusal: true,
		},
		{
			name:        "business data graph destroy",
			newResource: newBDG,
			state:       map[string]tftypes.Value{"id": str("sap.graph.a")},
			wantRefusal: true,
		},
		{
			name:        "business data graph replacement",
			newResource: newBDG,
			state:       map[string]tftypes.Value{"id": str("sap.graph.a")},
			plan:        map[string]tftypes.Value{"id": str("sap.graph.b")},
			replace:     true,
			wantRefusal: true,
		},
		{
			name:        "number range update keeping the counter",
			newResource: newNR,
			state:       map[string]tftypes.Value{"id": str("NR"), "description": str("old"), "current_value_wo_version": str("v1")},
			plan:        map[string]tftypes.Value{"id": str("NR"), "description": str("new"), "current_value_wo_version": str("v1")},
			wantRefusal: true,
		},
		{
			name:        "number range update setting the counter",
			newResource: newNR,
			state:       map[string]tftypes.Value{"id": str("NR"), "description": str("old"), "current_value_wo_version": str("v1")},
			plan:        map[string]tftypes.Value{"id": str("NR"), "description": str("new"), "current_value_wo_version": str("v2")},
		},
		{
			name:        "number range destroy",
			newResource: newNR,
			state:       map[string]tftypes.Value{"id": str("NR")},
			wantRefusal: true,
		},
		{
			name:        "number range create",
			newResource: newNR,
			plan:        map[string]tftypes.Value{"name": str("NR"), "current_value_wo_version": str("v1")},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			off := modifyPlan(t, tc.newResource(true, false), tc.state, tc.plan, tc.replace)
			if got := unofficialOperationError(off); got != tc.wantRefusal {
				t.Errorf("without enable_unofficial: refused = %v, want %v (%v)", got, tc.wantRefusal, off)
			}
			if on := modifyPlan(t, tc.newResource(true, true), tc.state, tc.plan, tc.replace); on.HasError() {
				t.Errorf("with enable_unofficial: %v", on)
			}
			if early := modifyPlan(t, tc.newResource(false, false), tc.state, tc.plan, tc.replace); early.HasError() {
				t.Errorf("before the provider is configured: %v", early)
			}
		})
	}
}

// TestOperationGates_MatchCatalog keeps the operation gates in step with the
// catalog. A resource is operation-gated when it has an allowUnofficial field,
// which Configure must fill from enable_unofficial. Every gated type belongs
// to a feature that lists undocumented operations, and every feature that
// lists some has at least one gated type, so an operation cannot be listed
// without being switched off, nor switched off without being listed.
// operationGatedTypes configures every resource with both switches on and
// returns the types that have an allowUnofficial field, checking that each
// has a ModifyPlan and that Configure fills the field.
func operationGatedTypes(t *testing.T) map[string]bool {
	t.Helper()
	ctx := context.Background()
	p := &sapIntegrationSuiteProvider{version: "test"}
	data := &Data{
		Host:                           "https://tenant.example",
		HTTPClient:                     &sapthttp.Client{},
		APIManagementClassicHost:       "https://apim.example",
		APIManagementClassicHTTPClient: &sapthttp.Client{},
		APICompositionHost:             "https://graph.example",
		APICompositionHTTPClient:       &sapthttp.Client{},
		EnableExperimental:             true,
		EnableUnofficial:               true,
	}
	gated := map[string]bool{}
	for _, newResource := range p.Resources(ctx) {
		r := newResource()
		var meta resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "sapintegrationsuite"}, &meta)
		field := reflect.ValueOf(r).Elem().FieldByName("allowUnofficial")
		if !field.IsValid() {
			continue
		}
		gated[meta.TypeName] = true
		if _, ok := r.(resource.ResourceWithModifyPlan); !ok {
			t.Errorf("%s has allowUnofficial but no ModifyPlan, so a refused operation fails only at apply", meta.TypeName)
		}
		rc := r.(resource.ResourceWithConfigure)
		var resp resource.ConfigureResponse
		rc.Configure(ctx, resource.ConfigureRequest{ProviderData: data}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("%s: Configure: %v", meta.TypeName, resp.Diagnostics)
		}
		if !field.Bool() {
			t.Errorf("%s: Configure does not set allowUnofficial from enable_unofficial", meta.TypeName)
		}
	}

	return gated
}

func TestOperationGates_MatchCatalog(t *testing.T) {
	gated := operationGatedTypes(t)

	listed := map[string]bool{}
	for _, f := range features.Catalog {
		if len(f.UndocumentedOperations) == 0 {
			continue
		}
		hasGate := false
		for _, rt := range f.ResourceTypes {
			listed[rt] = true
			hasGate = hasGate || gated[rt]
		}
		if !hasGate {
			t.Errorf("feature %s lists undocumented operations %q, but none of its resources gates them behind enable_unofficial",
				f.Key, f.UndocumentedOperations)
		}
	}
	for typeName := range gated {
		if !listed[typeName] {
			t.Errorf("%s gates operations behind enable_unofficial, but no catalog feature of it lists undocumented operations", typeName)
		}
	}
}

// numberRangeServer records every request and answers like a tenant.
func numberRangeServer(t *testing.T) (*httptest.Server, *[]string, *[]byte) {
	t.Helper()
	var requests []string
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method)
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(numberRangeTestEntity))
			return
		}
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(numberRangeTestEntity))
	}))
	t.Cleanup(server.Close)
	return server, &requests, &body
}

// Without enable_unofficial a number range uses only what SAP documents: the
// create posts without checking for an existing name, and the state comes
// from the plan and the counter that was sent.
func TestNumberRangeResource_DocumentedOnly_Create(t *testing.T) {
	server, requests, _ := numberRangeServer(t)
	r := &numberRangeResource{client: cloudintegration.New(http.DefaultClient, server.URL)}
	s := numberRangeSchema(t).Schema
	ctx := context.Background()
	v1 := "v1"

	config := numberRangeStateRaw(t, s, &v1)
	obj := map[string]tftypes.Value{}
	if err := config.As(&obj); err != nil {
		t.Fatal(err)
	}
	obj["current_value_wo"] = str("7")
	configWithCounter := tftypes.NewValue(config.Type(), obj)

	resp := &resource.CreateResponse{State: newNumberRangeTestState(t, s)}
	r.Create(ctx, resource.CreateRequest{
		Plan:   tfsdk.Plan{Schema: s, Raw: config},
		Config: tfsdk.Config{Schema: s, Raw: configWithCounter},
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create() produced diagnostics: %v", resp.Diagnostics)
	}
	if strings.Join(*requests, ",") != http.MethodPost {
		t.Errorf("requests = %v, want only the documented POST", *requests)
	}
	var got numberRangeModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.CurrentValue.ValueString() != "7" || got.ID.ValueString() != "MyRange" {
		t.Errorf("state = id %q, current_value %q; want MyRange and the sent counter 7", got.ID.ValueString(), got.CurrentValue.ValueString())
	}
	if !got.DeployedOn.IsNull() {
		t.Error("deployed_on should stay null when the number range cannot be read")
	}
}

func TestNumberRangeResource_DocumentedOnly_ReadKeepsState(t *testing.T) {
	server, requests, _ := numberRangeServer(t)
	r := &numberRangeResource{client: cloudintegration.New(http.DefaultClient, server.URL)}
	s := numberRangeSchema(t).Schema
	v1 := "v1"

	prior := tfsdk.State{Schema: s, Raw: numberRangeStateRaw(t, s, &v1)}
	resp := &resource.ReadResponse{State: prior}
	r.Read(context.Background(), resource.ReadRequest{State: prior}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() produced diagnostics: %v", resp.Diagnostics)
	}
	if len(*requests) != 0 {
		t.Errorf("requests = %v, want none", *requests)
	}
	if !resp.State.Raw.Equal(prior.Raw) {
		t.Error("state changed without a read")
	}
}

// An update that changes current_value_wo_version sends the configured
// counter with the documented PUT and reads nothing.
func TestNumberRangeResource_DocumentedOnly_UpdateWithCounter(t *testing.T) {
	server, requests, body := numberRangeServer(t)
	r := &numberRangeResource{client: cloudintegration.New(http.DefaultClient, server.URL)}
	s := numberRangeSchema(t).Schema
	ctx := context.Background()
	v1, v2 := "v1", "v2"

	plan := numberRangeStateRaw(t, s, &v2)
	obj := map[string]tftypes.Value{}
	if err := plan.As(&obj); err != nil {
		t.Fatal(err)
	}
	obj["current_value_wo"] = str("500")
	config := tftypes.NewValue(plan.Type(), obj)

	resp := &resource.UpdateResponse{State: newNumberRangeTestState(t, s)}
	r.Update(ctx, resource.UpdateRequest{
		State:  tfsdk.State{Schema: s, Raw: numberRangeStateRaw(t, s, &v1)},
		Plan:   tfsdk.Plan{Schema: s, Raw: plan},
		Config: tfsdk.Config{Schema: s, Raw: config},
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update() produced diagnostics: %v", resp.Diagnostics)
	}
	if strings.Join(*requests, ",") != http.MethodPut {
		t.Errorf("requests = %v, want only the documented PUT", *requests)
	}
	if got := sentCurrentValue(*body); got != "500" {
		t.Errorf("PUT CurrentValue = %q, want 500", got)
	}
	var got numberRangeModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.CurrentValue.ValueString() != "500" {
		t.Errorf("current_value = %q, want the sent 500", got.CurrentValue.ValueString())
	}
}

// Update, Delete and Import refuse at apply time too, in case the plan was
// made before the provider was configured.
func TestNumberRangeResource_DocumentedOnly_Refusals(t *testing.T) {
	server, requests, _ := numberRangeServer(t)
	r := &numberRangeResource{client: cloudintegration.New(http.DefaultClient, server.URL)}
	s := numberRangeSchema(t).Schema
	ctx := context.Background()
	v1 := "v1"
	raw := numberRangeStateRaw(t, s, &v1)

	update := &resource.UpdateResponse{State: newNumberRangeTestState(t, s)}
	r.Update(ctx, resource.UpdateRequest{
		State:  tfsdk.State{Schema: s, Raw: raw},
		Plan:   tfsdk.Plan{Schema: s, Raw: raw},
		Config: tfsdk.Config{Schema: s, Raw: raw},
	}, update)
	if !unofficialOperationError(update.Diagnostics) {
		t.Errorf("Update keeping the counter: %v, want the unofficial refusal", update.Diagnostics)
	}

	var del resource.DeleteResponse
	r.Delete(ctx, resource.DeleteRequest{State: tfsdk.State{Schema: s, Raw: raw}}, &del)
	if !unofficialOperationError(del.Diagnostics) {
		t.Errorf("Delete: %v, want the unofficial refusal", del.Diagnostics)
	}

	imp := &resource.ImportStateResponse{State: newNumberRangeTestState(t, s)}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "MyRange"}, imp)
	if !unofficialOperationError(imp.Diagnostics) {
		t.Errorf("ImportState: %v, want the unofficial refusal", imp.Diagnostics)
	}
	if len(*requests) != 0 {
		t.Errorf("requests = %v, want none", *requests)
	}
}
