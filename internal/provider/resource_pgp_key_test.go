package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/securitycontent"
)

// The fixture is a public key gpg generated for the tests.
const pgpFixtureFingerprint = "D8DE705CC863FED0B2A56EB6AB8AF9015AD46E69"

func pgpFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../client/securitycontent/testdata/pgp-public-rsa.asc")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestPGPPublicKey_ReplacesOnlyForAnotherKey(t *testing.T) {
	r := &pgpPublicKeyResource{}
	key := pgpFixture(t)
	for _, c := range []struct {
		name, stateFingerprint, plan string
		replace                      bool
	}{
		{"same key, other line endings", pgpFixtureFingerprint, strings.ReplaceAll(key, "\r\n", "\n"), false},
		{"first plan after an import", pgpFixtureFingerprint, key, false},
		{"another key", "0000000000000000000000000000000000000000", key, true},
	} {
		raw, empty := resourceObject(t, r, map[string]tftypes.Value{"key_id": str(pgpFixtureFingerprint[24:]), "fingerprint": str(c.stateFingerprint)})
		req := planmodifier.StringRequest{
			State:     tfsdk.State{Schema: empty.Schema, Raw: raw},
			PlanValue: types.StringValue(c.plan),
		}
		resp := &stringplanmodifier.RequiresReplaceIfFuncResponse{}
		pgpOtherKey(context.Background(), req, resp)
		if resp.RequiresReplace != c.replace {
			t.Errorf("%s: replace = %v, want %v", c.name, resp.RequiresReplace, c.replace)
		}
	}
}

func TestPGPPublicKey_ValidateAndPlan(t *testing.T) {
	r := &pgpPublicKeyResource{}
	ctx := context.Background()
	raw, empty := resourceObject(t, r, map[string]tftypes.Value{"public_key": str("not a key")})
	var vr resource.ValidateConfigResponse
	r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: empty.Schema, Raw: raw}}, &vr)
	if !vr.Diagnostics.HasError() {
		t.Error("garbage passed validation")
	}

	raw, empty = resourceObject(t, r, map[string]tftypes.Value{"public_key": str(pgpFixture(t))})
	plan := tfsdk.Plan{Schema: empty.Schema, Raw: raw}
	resp := &resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{State: empty, Plan: plan}, resp)
	var m pgpPublicKeyModel
	resp.Plan.Get(ctx, &m)
	if m.KeyID.ValueString() != pgpFixtureFingerprint[24:] || m.Fingerprint.ValueString() != pgpFixtureFingerprint {
		t.Errorf("planned key_id %q, fingerprint %q", m.KeyID.ValueString(), m.Fingerprint.ValueString())
	}
}

func TestPGPPublicKey_DeleteKeepsAKeyWithASecretPart(t *testing.T) {
	for _, c := range []struct {
		keyType string
		deleted bool
	}{{"Public", true}, {"Secret,Public", false}} {
		var deleted bool
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				deleted = true
			}
			_, _ = w.Write([]byte(`{"d":{"Id":"AB8AF9015AD46E69","KeyId":"AB8AF9015AD46E69","Type":"` + c.keyType + `"}}`))
		}))
		r := &pgpPublicKeyResource{client: securitycontent.New(http.DefaultClient, server.URL)}
		raw, empty := resourceObject(t, r, map[string]tftypes.Value{"key_id": str("AB8AF9015AD46E69")})
		resp := &resource.DeleteResponse{}
		r.Delete(context.Background(), resource.DeleteRequest{State: tfsdk.State{Schema: empty.Schema, Raw: raw}}, resp)
		server.Close()
		if deleted != c.deleted || resp.Diagnostics.HasError() || (resp.Diagnostics.WarningsCount() == 1) == c.deleted {
			t.Errorf("type %s: deleted %v, diagnostics %v", c.keyType, deleted, resp.Diagnostics)
		}
	}
}

func TestPGPSecretKey_ValidateRefusesAPublicKey(t *testing.T) {
	r := &pgpSecretKeyResource{}
	raw, empty := resourceObject(t, r, map[string]tftypes.Value{"secret_key_wo": str(pgpFixture(t))})
	var resp resource.ValidateConfigResponse
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: empty.Schema, Raw: raw}}, &resp)
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics[0].Summary(), "Public key") {
		t.Errorf("diagnostics = %v", resp.Diagnostics)
	}
}

func TestPGPSecretKey_IsWriteOnly(t *testing.T) {
	r := NewPGPSecretKeyResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	for _, name := range []string{"secret_key_wo", "passphrase_wo"} {
		a := resp.Schema.Attributes[name]
		if !a.IsWriteOnly() || !a.IsSensitive() {
			t.Errorf("%s must be write-only and sensitive", name)
		}
	}
}
