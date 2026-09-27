package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

func numberRangeSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewNumberRangeResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() produced diagnostics: %v", resp.Diagnostics)
	}
	return resp
}

func TestNumberRangeResource_SchemaRequiredComputed(t *testing.T) {
	s := numberRangeSchema(t).Schema

	cases := []struct {
		name               string
		required, computed bool
	}{
		{"id", false, true},
		{"name", true, false},
		{"min_value", true, false},
		{"max_value", true, false},
		{"description", false, true},
		{"rotate", true, false},
		{"field_length", false, true},
		{"current_value_wo", true, false},
		{"current_value_wo_version", true, false},
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

	nameAttr, ok := s.Attributes["name"].(schema.StringAttribute)
	if !ok || len(nameAttr.PlanModifiers) == 0 {
		t.Error("name must carry a RequiresReplace plan modifier")
	}

	currentValueAttr, ok := s.Attributes["current_value_wo"].(schema.StringAttribute)
	if !ok || !currentValueAttr.WriteOnly {
		t.Error("current_value_wo must be WriteOnly")
	}
}

func TestNumberRangeResource_ValidateConfig_RejectsMinGreaterThanMax(t *testing.T) {
	r := NewNumberRangeResource().(resource.ResourceWithValidateConfig)
	s := numberRangeSchema(t).Schema
	ctx := context.Background()

	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	raw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                       tftypes.NewValue(tftypes.String, nil),
		"name":                     tftypes.NewValue(tftypes.String, "MyRange"),
		"min_value":                tftypes.NewValue(tftypes.String, "9999"),
		"max_value":                tftypes.NewValue(tftypes.String, "0"),
		"description":              tftypes.NewValue(tftypes.String, nil),
		"rotate":                   tftypes.NewValue(tftypes.Bool, true),
		"field_length":             tftypes.NewValue(tftypes.String, nil),
		"current_value_wo":         tftypes.NewValue(tftypes.String, nil),
		"current_value_wo_version": tftypes.NewValue(tftypes.String, "v1"),
		"current_value":            tftypes.NewValue(tftypes.String, nil),
		"deployed_by":              tftypes.NewValue(tftypes.String, nil),
		"deployed_on":              tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: s, Raw: raw}}
	var resp resource.ValidateConfigResponse
	r.ValidateConfig(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("ValidateConfig() did not reject min_value > max_value")
	}
}

func TestNumberRangeResource_ValidateConfig_AcceptsValidRange(t *testing.T) {
	r := NewNumberRangeResource().(resource.ResourceWithValidateConfig)
	s := numberRangeSchema(t).Schema
	ctx := context.Background()

	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	raw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                       tftypes.NewValue(tftypes.String, nil),
		"name":                     tftypes.NewValue(tftypes.String, "MyRange"),
		"min_value":                tftypes.NewValue(tftypes.String, "0"),
		"max_value":                tftypes.NewValue(tftypes.String, "9999"),
		"description":              tftypes.NewValue(tftypes.String, nil),
		"rotate":                   tftypes.NewValue(tftypes.Bool, true),
		"field_length":             tftypes.NewValue(tftypes.String, nil),
		"current_value_wo":         tftypes.NewValue(tftypes.String, nil),
		"current_value_wo_version": tftypes.NewValue(tftypes.String, "v1"),
		"current_value":            tftypes.NewValue(tftypes.String, nil),
		"deployed_by":              tftypes.NewValue(tftypes.String, nil),
		"deployed_on":              tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: s, Raw: raw}}
	var resp resource.ValidateConfigResponse
	r.ValidateConfig(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("ValidateConfig() unexpectedly produced diagnostics: %v", resp.Diagnostics)
	}
}

func newNumberRangeTestState(t *testing.T, s schema.Schema) tfsdk.State {
	t.Helper()
	return tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
}

// TestNumberRangeResource_Update_SendsLiveCounterWhenVersionUnchanged is the
// regression test for the counter-preservation contract. SAP rejects a PUT
// without CurrentValue (500, tenant test September 2026), so an update that
// only changes "description" must send the counter SAP currently holds (42
// in numberRangeTestEntity), never a value from the configuration.
func TestNumberRangeResource_Update_SendsLiveCounterWhenVersionUnchanged(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(numberRangeTestEntity))
			return
		}
		body, _ := io.ReadAll(r.Body)
		gotBody = body
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	r := &numberRangeResource{client: cloudintegration.New(http.DefaultClient, server.URL), allowUnofficial: true}
	s := numberRangeSchema(t).Schema
	ctx := context.Background()
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	stateRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                       tftypes.NewValue(tftypes.String, "MyRange"),
		"name":                     tftypes.NewValue(tftypes.String, "MyRange"),
		"min_value":                tftypes.NewValue(tftypes.String, "0"),
		"max_value":                tftypes.NewValue(tftypes.String, "9999"),
		"description":              tftypes.NewValue(tftypes.String, "old description"),
		"rotate":                   tftypes.NewValue(tftypes.Bool, true),
		"field_length":             tftypes.NewValue(tftypes.String, "4"),
		"current_value_wo":         tftypes.NewValue(tftypes.String, nil),
		"current_value_wo_version": tftypes.NewValue(tftypes.String, "v1"),
		"current_value":            tftypes.NewValue(tftypes.String, nil),
		"deployed_by":              tftypes.NewValue(tftypes.String, nil),
		"deployed_on":              tftypes.NewValue(tftypes.String, nil),
	})
	planRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                       tftypes.NewValue(tftypes.String, "MyRange"),
		"name":                     tftypes.NewValue(tftypes.String, "MyRange"),
		"min_value":                tftypes.NewValue(tftypes.String, "0"),
		"max_value":                tftypes.NewValue(tftypes.String, "9999"),
		"description":              tftypes.NewValue(tftypes.String, "new description"),
		"rotate":                   tftypes.NewValue(tftypes.Bool, true),
		"field_length":             tftypes.NewValue(tftypes.String, "4"),
		"current_value_wo":         tftypes.NewValue(tftypes.String, nil),
		"current_value_wo_version": tftypes.NewValue(tftypes.String, "v1"),
		"current_value":            tftypes.NewValue(tftypes.String, nil),
		"deployed_by":              tftypes.NewValue(tftypes.String, nil),
		"deployed_on":              tftypes.NewValue(tftypes.String, nil),
	})
	configRaw := planRaw

	req := resource.UpdateRequest{
		State:  tfsdk.State{Schema: s, Raw: stateRaw},
		Plan:   tfsdk.Plan{Schema: s, Raw: planRaw},
		Config: tfsdk.Config{Schema: s, Raw: configRaw},
	}
	resp := &resource.UpdateResponse{State: newNumberRangeTestState(t, s)}

	r.Update(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update() produced diagnostics: %v", resp.Diagnostics)
	}

	if got := sentCurrentValue(gotBody); got != "42" {
		t.Errorf("PUT CurrentValue = %q, want the live counter 42 (body %s)", got, gotBody)
	}
}

// TestNumberRangeResource_Update_SendsCurrentValueWhenVersionChanges proves
// the deliberate-reset path: bumping current_value_wo_version does cause
// CurrentValue to be sent, mirroring this provider's credential-rotation
// pattern.
func TestNumberRangeResource_Update_SendsCurrentValueWhenVersionChanges(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(numberRangeTestEntity))
			return
		}
		body, _ := io.ReadAll(r.Body)
		gotBody = body
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	r := &numberRangeResource{client: cloudintegration.New(http.DefaultClient, server.URL), allowUnofficial: true}
	s := numberRangeSchema(t).Schema
	ctx := context.Background()
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	stateRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                       tftypes.NewValue(tftypes.String, "MyRange"),
		"name":                     tftypes.NewValue(tftypes.String, "MyRange"),
		"min_value":                tftypes.NewValue(tftypes.String, "0"),
		"max_value":                tftypes.NewValue(tftypes.String, "9999"),
		"description":              tftypes.NewValue(tftypes.String, "d"),
		"rotate":                   tftypes.NewValue(tftypes.Bool, true),
		"field_length":             tftypes.NewValue(tftypes.String, "4"),
		"current_value_wo":         tftypes.NewValue(tftypes.String, nil),
		"current_value_wo_version": tftypes.NewValue(tftypes.String, "v1"),
		"current_value":            tftypes.NewValue(tftypes.String, nil),
		"deployed_by":              tftypes.NewValue(tftypes.String, nil),
		"deployed_on":              tftypes.NewValue(tftypes.String, nil),
	})
	planRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                       tftypes.NewValue(tftypes.String, "MyRange"),
		"name":                     tftypes.NewValue(tftypes.String, "MyRange"),
		"min_value":                tftypes.NewValue(tftypes.String, "0"),
		"max_value":                tftypes.NewValue(tftypes.String, "9999"),
		"description":              tftypes.NewValue(tftypes.String, "d"),
		"rotate":                   tftypes.NewValue(tftypes.Bool, true),
		"field_length":             tftypes.NewValue(tftypes.String, "4"),
		"current_value_wo":         tftypes.NewValue(tftypes.String, nil),
		"current_value_wo_version": tftypes.NewValue(tftypes.String, "v2"),
		"current_value":            tftypes.NewValue(tftypes.String, nil),
		"deployed_by":              tftypes.NewValue(tftypes.String, nil),
		"deployed_on":              tftypes.NewValue(tftypes.String, nil),
	})
	configRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                       tftypes.NewValue(tftypes.String, nil),
		"name":                     tftypes.NewValue(tftypes.String, "MyRange"),
		"min_value":                tftypes.NewValue(tftypes.String, "0"),
		"max_value":                tftypes.NewValue(tftypes.String, "9999"),
		"description":              tftypes.NewValue(tftypes.String, "d"),
		"rotate":                   tftypes.NewValue(tftypes.Bool, true),
		"field_length":             tftypes.NewValue(tftypes.String, "4"),
		"current_value_wo":         tftypes.NewValue(tftypes.String, "500"),
		"current_value_wo_version": tftypes.NewValue(tftypes.String, "v2"),
		"current_value":            tftypes.NewValue(tftypes.String, nil),
		"deployed_by":              tftypes.NewValue(tftypes.String, nil),
		"deployed_on":              tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.UpdateRequest{
		State:  tfsdk.State{Schema: s, Raw: stateRaw},
		Plan:   tfsdk.Plan{Schema: s, Raw: planRaw},
		Config: tfsdk.Config{Schema: s, Raw: configRaw},
	}
	resp := &resource.UpdateResponse{State: newNumberRangeTestState(t, s)}

	r.Update(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update() produced diagnostics: %v", resp.Diagnostics)
	}

	if got := sentCurrentValue(gotBody); got != "500" {
		t.Errorf("PUT CurrentValue = %q, want the configured 500 after a version change (body %s)", got, gotBody)
	}
}

// sentCurrentValue returns the CurrentValue a PUT body carried, or "" if
// it had none.
func sentCurrentValue(body []byte) string {
	var decoded struct {
		CurrentValue string `json:"CurrentValue"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return ""
	}
	return decoded.CurrentValue
}

// numberRangeTestEntity is a GET response in the shape a tenant returned in
// September 2026: every value a string, the date as /Date(ms)/.
const numberRangeTestEntity = `{"d":{"Name":"MyRange","Description":"from SAP","MaxValue":"9999","MinValue":"0","Rotate":"true","CurrentValue":"42","FieldLength":"4","DeployedBy":"sb-client","DeployedOn":"\/Date(1790410291173)\/"}}`

func numberRangeStateRaw(t *testing.T, s schema.Schema, version *string) tftypes.Value {
	t.Helper()
	objType := s.Type().TerraformType(context.Background()).(tftypes.Object)
	var v interface{}
	if version != nil {
		v = *version
	}
	return tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                       tftypes.NewValue(tftypes.String, "MyRange"),
		"name":                     tftypes.NewValue(tftypes.String, "MyRange"),
		"min_value":                tftypes.NewValue(tftypes.String, "0"),
		"max_value":                tftypes.NewValue(tftypes.String, "9999"),
		"description":              tftypes.NewValue(tftypes.String, "d"),
		"rotate":                   tftypes.NewValue(tftypes.Bool, true),
		"field_length":             tftypes.NewValue(tftypes.String, "4"),
		"current_value_wo":         tftypes.NewValue(tftypes.String, nil),
		"current_value_wo_version": tftypes.NewValue(tftypes.String, v),
		"current_value":            tftypes.NewValue(tftypes.String, nil),
		"deployed_by":              tftypes.NewValue(tftypes.String, nil),
		"deployed_on":              tftypes.NewValue(tftypes.String, nil),
	})
}

// Read reports what SAP holds, so a description changed in the UI shows up
// as drift, and the live counter appears in current_value.
func TestNumberRangeResource_Read_DetectsDrift(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/NumberRanges('MyRange')" || r.URL.RawQuery != "" {
			t.Errorf("request = %s?%s, want a bare GET by key", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(numberRangeTestEntity))
	}))
	defer server.Close()

	r := &numberRangeResource{client: cloudintegration.New(http.DefaultClient, server.URL), allowUnofficial: true}
	s := numberRangeSchema(t).Schema
	ctx := context.Background()
	v1 := "v1"

	resp := &resource.ReadResponse{State: newNumberRangeTestState(t, s)}
	r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: numberRangeStateRaw(t, s, &v1)}}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() produced diagnostics: %v", resp.Diagnostics)
	}

	var got numberRangeModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.Description.ValueString() != "from SAP" {
		t.Errorf("description = %q, want the value SAP reports", got.Description.ValueString())
	}
	if got.CurrentValue.ValueString() != "42" {
		t.Errorf("current_value = %q, want 42", got.CurrentValue.ValueString())
	}
	if got.CurrentValueWOVersion.ValueString() != "v1" {
		t.Errorf("current_value_wo_version = %q, want it kept from state", got.CurrentValueWOVersion.ValueString())
	}
	if got.DeployedOn.ValueString() == "" {
		t.Error("deployed_on is empty, want the converted /Date(ms)/ value")
	}
}

func TestNumberRangeResource_Read_RemovesDeletedRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"Not Found","message":{"lang":"en","value":"not found"}}}`))
	}))
	defer server.Close()

	r := &numberRangeResource{client: cloudintegration.New(http.DefaultClient, server.URL), allowUnofficial: true}
	s := numberRangeSchema(t).Schema
	v1 := "v1"

	resp := &resource.ReadResponse{State: tfsdk.State{Schema: s, Raw: numberRangeStateRaw(t, s, &v1)}}
	r.Read(context.Background(), resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: numberRangeStateRaw(t, s, &v1)}}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() produced diagnostics: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("state should be removed when SAP reports 404")
	}
}

func TestNumberRangeResource_Delete_SendsDelete(t *testing.T) {
	var method, path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	r := &numberRangeResource{client: cloudintegration.New(http.DefaultClient, server.URL), allowUnofficial: true}
	s := numberRangeSchema(t).Schema
	v1 := "v1"

	var resp resource.DeleteResponse
	r.Delete(context.Background(), resource.DeleteRequest{State: tfsdk.State{Schema: s, Raw: numberRangeStateRaw(t, s, &v1)}}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete() produced diagnostics: %v", resp.Diagnostics)
	}
	if method != http.MethodDelete || path != "/api/v1/NumberRanges('MyRange')" {
		t.Errorf("request = %s %s, want DELETE NumberRanges('MyRange')", method, path)
	}
}

func TestNumberRangeResource_ImportState_SetsName(t *testing.T) {
	r := &numberRangeResource{allowUnofficial: true}
	s := numberRangeSchema(t).Schema
	ctx := context.Background()

	resp := &resource.ImportStateResponse{State: newNumberRangeTestState(t, s)}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "MyRange"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() produced diagnostics: %v", resp.Diagnostics)
	}
	var name types.String
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, pathRoot("name"), &name)...)
	if name.ValueString() != "MyRange" {
		t.Errorf("name = %q, want MyRange", name.ValueString())
	}
}

// After an import the marker is null. The first apply must record it and
// send the live counter back, or adopting a number range in use would reset
// its counter to whatever the configuration says.
func TestNumberRangeResource_Update_AfterImportKeepsCounter(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(numberRangeTestEntity))
			return
		}
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	r := &numberRangeResource{client: cloudintegration.New(http.DefaultClient, server.URL), allowUnofficial: true}
	s := numberRangeSchema(t).Schema
	ctx := context.Background()
	v1 := "v1"

	config := numberRangeStateRaw(t, s, &v1)
	req := resource.UpdateRequest{
		State:  tfsdk.State{Schema: s, Raw: numberRangeStateRaw(t, s, nil)},
		Plan:   tfsdk.Plan{Schema: s, Raw: numberRangeStateRaw(t, s, &v1)},
		Config: tfsdk.Config{Schema: s, Raw: config},
	}
	resp := &resource.UpdateResponse{State: newNumberRangeTestState(t, s)}
	r.Update(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update() produced diagnostics: %v", resp.Diagnostics)
	}
	if got := sentCurrentValue(gotBody); got != "42" {
		t.Errorf("first apply after import sent CurrentValue %q, want the live counter 42 (body %s)", got, gotBody)
	}
	var got numberRangeModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.CurrentValueWOVersion.ValueString() != "v1" {
		t.Errorf("current_value_wo_version = %q, want the marker recorded", got.CurrentValueWOVersion.ValueString())
	}
}

// SAP does not document what a create on an existing name does, so Create
// stops instead of posting.
func TestNumberRangeResource_Create_RefusesExistingName(t *testing.T) {
	posted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posted = true
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(numberRangeTestEntity))
	}))
	defer server.Close()

	r := &numberRangeResource{client: cloudintegration.New(http.DefaultClient, server.URL), allowUnofficial: true}
	s := numberRangeSchema(t).Schema
	v1 := "v1"

	raw := numberRangeStateRaw(t, s, &v1)
	resp := &resource.CreateResponse{State: newNumberRangeTestState(t, s)}
	r.Create(context.Background(), resource.CreateRequest{
		Plan:   tfsdk.Plan{Schema: s, Raw: raw},
		Config: tfsdk.Config{Schema: s, Raw: raw},
	}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected Create to fail for an existing number range")
	}
	if posted {
		t.Error("Create sent a POST for an existing name")
	}
}
