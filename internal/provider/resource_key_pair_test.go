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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/securitycontent"
)

func keyPairSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewKeyPairResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() produced diagnostics: %v", resp.Diagnostics)
	}
	return resp
}

func TestKeyPairResource_SchemaRequiredComputed(t *testing.T) {
	s := keyPairSchema(t).Schema

	cases := []struct {
		name               string
		required, computed bool
	}{
		{"id", false, true},
		{"alias", true, false},
		{"key_type", false, true},
		{"signature_algorithm", false, false},
		{"key_size", false, true},
		{"key_algorithm_parameter", false, false},
		{"common_name", true, false},
		{"country", true, false},
		{"valid_not_before", false, true},
		{"valid_not_after", false, true},
		{"public_key_openssh", false, true},
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

	// No field for a private key must exist anywhere on this schema.
	for name := range s.Attributes {
		if name == "public_key_openssh" {
			continue
		}
		if containsString([]string{"private_key", "private_key_pem", "private_key_base64"}, name) {
			t.Errorf("schema must never expose a private key attribute, found %q", name)
		}
	}
}

func keyPairConfigValue(objType tftypes.Object, overrides map[string]tftypes.Value) tftypes.Value {
	base := map[string]tftypes.Value{
		"id":                      tftypes.NewValue(tftypes.String, nil),
		"alias":                   tftypes.NewValue(tftypes.String, "my-keypair"),
		"runtime_location_id":     tftypes.NewValue(tftypes.String, nil),
		"key_type":                tftypes.NewValue(tftypes.String, "RSA"),
		"signature_algorithm":     tftypes.NewValue(tftypes.String, nil),
		"key_size":                tftypes.NewValue(tftypes.Number, 2048),
		"key_algorithm_parameter": tftypes.NewValue(tftypes.String, nil),
		"common_name":             tftypes.NewValue(tftypes.String, "cn.example.invalid"),
		"organization_unit":       tftypes.NewValue(tftypes.String, nil),
		"organization":            tftypes.NewValue(tftypes.String, nil),
		"locality":                tftypes.NewValue(tftypes.String, nil),
		"state":                   tftypes.NewValue(tftypes.String, nil),
		"country":                 tftypes.NewValue(tftypes.String, "DE"),
		"email":                   tftypes.NewValue(tftypes.String, nil),
		"valid_not_before":        tftypes.NewValue(tftypes.String, nil),
		"valid_not_after":         tftypes.NewValue(tftypes.String, nil),
		"public_key_openssh":      tftypes.NewValue(tftypes.String, nil),
	}
	for k, v := range overrides {
		base[k] = v
	}
	return tftypes.NewValue(objType, base)
}

func TestKeyPairResource_ValidateConfig_RSARequiresKeySize(t *testing.T) {
	r := NewKeyPairResource().(resource.ResourceWithValidateConfig)
	s := keyPairSchema(t).Schema
	ctx := context.Background()
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	raw := keyPairConfigValue(objType, map[string]tftypes.Value{
		"key_size": tftypes.NewValue(tftypes.Number, nil),
	})

	req := resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: s, Raw: raw}}
	var resp resource.ValidateConfigResponse
	r.ValidateConfig(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("ValidateConfig() did not reject RSA without key_size")
	}
}

func TestKeyPairResource_ValidateConfig_ECRequiresSizeOrCurve(t *testing.T) {
	r := NewKeyPairResource().(resource.ResourceWithValidateConfig)
	s := keyPairSchema(t).Schema
	ctx := context.Background()
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	raw := keyPairConfigValue(objType, map[string]tftypes.Value{
		"key_type": tftypes.NewValue(tftypes.String, "EC"),
		"key_size": tftypes.NewValue(tftypes.Number, nil),
	})

	req := resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: s, Raw: raw}}
	var resp resource.ValidateConfigResponse
	r.ValidateConfig(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("ValidateConfig() did not reject EC with neither key_size nor key_algorithm_parameter")
	}
}

func TestKeyPairResource_ValidateConfig_ECWithCurveIsValid(t *testing.T) {
	r := NewKeyPairResource().(resource.ResourceWithValidateConfig)
	s := keyPairSchema(t).Schema
	ctx := context.Background()
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	raw := keyPairConfigValue(objType, map[string]tftypes.Value{
		"key_type":                tftypes.NewValue(tftypes.String, "EC"),
		"key_size":                tftypes.NewValue(tftypes.Number, nil),
		"key_algorithm_parameter": tftypes.NewValue(tftypes.String, "secp256r1"),
	})

	req := resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: s, Raw: raw}}
	var resp resource.ValidateConfigResponse
	r.ValidateConfig(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("ValidateConfig() unexpectedly rejected EC with key_algorithm_parameter: %v", resp.Diagnostics)
	}
}

func TestKeyPairResource_ValidateConfig_ECKeySizeOutOfRange(t *testing.T) {
	r := NewKeyPairResource().(resource.ResourceWithValidateConfig)
	s := keyPairSchema(t).Schema
	ctx := context.Background()
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	raw := keyPairConfigValue(objType, map[string]tftypes.Value{
		"key_type": tftypes.NewValue(tftypes.String, "EC"),
		"key_size": tftypes.NewValue(tftypes.Number, 4096),
	})

	req := resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: s, Raw: raw}}
	var resp resource.ValidateConfigResponse
	r.ValidateConfig(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("ValidateConfig() did not reject an EC key_size outside SAP's documented 112-571 range")
	}
}

func TestKeyPairResource_ValidateConfig_SignatureAlgorithmMustMatchKeyType(t *testing.T) {
	r := NewKeyPairResource().(resource.ResourceWithValidateConfig)
	s := keyPairSchema(t).Schema
	ctx := context.Background()
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	raw := keyPairConfigValue(objType, map[string]tftypes.Value{
		"key_type":            tftypes.NewValue(tftypes.String, "RSA"),
		"signature_algorithm": tftypes.NewValue(tftypes.String, "SHA-256/DSA"),
	})

	req := resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: s, Raw: raw}}
	var resp resource.ValidateConfigResponse
	r.ValidateConfig(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("ValidateConfig() did not reject a DSA signature_algorithm with key_type RSA")
	}
}

func TestKeyPairResource_ValidateConfig_ValidRSAConfig(t *testing.T) {
	r := NewKeyPairResource().(resource.ResourceWithValidateConfig)
	s := keyPairSchema(t).Schema
	ctx := context.Background()
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	raw := keyPairConfigValue(objType, nil)

	req := resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: s, Raw: raw}}
	var resp resource.ValidateConfigResponse
	r.ValidateConfig(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("ValidateConfig() unexpectedly rejected a valid RSA config: %v", resp.Diagnostics)
	}
}

func newKeyPairTestState(t *testing.T, s schema.Schema) tfsdk.State {
	t.Helper()
	return tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
}

func TestKeyPairResource_Create(t *testing.T) {
	var genBody []byte
	getCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			genBody = body
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/KeystoreEntries('6d792d6b657970616972')":
			getCalls++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"d":{"Hexalias":"6d792d6b657970616972","Alias":"my-keypair","KeyType":"RSA","KeySize":2048,"ValidNotBefore":"2026-01-01T00:00:00Z","ValidNotAfter":"2029-01-01T00:00:00Z"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/KeystoreEntries('6d792d6b657970616972')/Sshkey/$value":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ssh-rsa AAAAB3NzaC1yc2E test\n"))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	r := &keyPairResource{client: securitycontent.New(http.DefaultClient, server.URL)}
	s := keyPairSchema(t).Schema
	ctx := context.Background()
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	planRaw := keyPairConfigValue(objType, nil)
	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: planRaw}}
	resp := &resource.CreateResponse{State: newKeyPairTestState(t, s)}

	r.Create(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create() produced diagnostics: %v", resp.Diagnostics)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(genBody, &decoded); err != nil {
		t.Fatalf("decoding generation request body: %v", err)
	}
	if decoded["Alias"] != "my-keypair" {
		t.Errorf("Alias = %v, want my-keypair", decoded["Alias"])
	}
	if decoded["CommonName"] != "cn.example.invalid" {
		t.Errorf("CommonName = %v", decoded["CommonName"])
	}

	var got keyPairModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.ID.ValueString() != "my-keypair" {
		t.Errorf("id = %q, want my-keypair", got.ID.ValueString())
	}
	if got.PublicKeyOpenSSH.IsNull() {
		t.Error("public_key_openssh should be populated for an RSA key pair")
	}
	if got.KeySize.ValueInt64() != 2048 {
		t.Errorf("key_size = %d, want 2048", got.KeySize.ValueInt64())
	}
}

func TestKeyPairResource_Delete_SubmitsOnlyItsOwnAlias(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = body
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	r := &keyPairResource{client: securitycontent.New(http.DefaultClient, server.URL)}
	s := keyPairSchema(t).Schema
	ctx := context.Background()
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	stateRaw := keyPairConfigValue(objType, map[string]tftypes.Value{
		"id":                  tftypes.NewValue(tftypes.String, "tf-acc-keypair-only"),
		"alias":               tftypes.NewValue(tftypes.String, "tf-acc-keypair-only"),
		"runtime_location_id": tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.DeleteRequest{State: tfsdk.State{Schema: s, Raw: stateRaw}}
	resp := &resource.DeleteResponse{}

	r.Delete(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete() produced diagnostics: %v", resp.Diagnostics)
	}

	want := `{"Aliases":"tf-acc-keypair-only"}`
	if string(gotBody) != want {
		t.Errorf("delete request body = %s, want %s", gotBody, want)
	}
}

func TestKeyPairResource_ImportState_UsesAlias(t *testing.T) {
	r := NewKeyPairResource().(resource.ResourceWithImportState)
	s := keyPairSchema(t).Schema

	resp := &resource.ImportStateResponse{State: newKeyPairTestState(t, s)}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "imported-alias"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() produced diagnostics: %v", resp.Diagnostics)
	}

	var alias types.String
	resp.Diagnostics.Append(resp.State.GetAttribute(context.Background(), pathRoot("alias"), &alias)...)
	if alias.ValueString() != "imported-alias" {
		t.Errorf("alias = %q, want imported-alias", alias.ValueString())
	}
}

// After an import, SAP has not told us the signature algorithm, so state
// holds null. Configuring it must record the value, not generate a new key
// pair; a real change of a known value still replaces.
func TestKeyPair_GenerationParameterReplaceOnlyWhenKnown(t *testing.T) {
	ctx := context.Background()
	existing := tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, map[string]tftypes.Value{})
	check := func(prior types.String, planned string) bool {
		req := planmodifier.StringRequest{
			State:      tfsdk.State{Raw: existing},
			Plan:       tfsdk.Plan{Raw: existing},
			StateValue: prior,
			PlanValue:  types.StringValue(planned),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		requiresReplaceUnlessUnknownBefore().PlanModifyString(ctx, req, resp)
		return resp.RequiresReplace
	}
	if check(types.StringNull(), "SHA-256/RSA") {
		t.Error("an imported key pair would be regenerated just for recording signature_algorithm")
	}
	if !check(types.StringValue("SHA-512/RSA"), "SHA-256/RSA") {
		t.Error("changing a known signature_algorithm must generate a new key pair")
	}
}
