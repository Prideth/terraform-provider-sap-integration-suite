package provider

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/securitycontent"
)

func generateTestCertPEM(t *testing.T, commonName string) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("creating certificate: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func certificateSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewCertificateResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() produced diagnostics: %v", resp.Diagnostics)
	}
	return resp
}

func TestCertificateResource_SchemaRequiredComputed(t *testing.T) {
	s := certificateSchema(t).Schema

	cases := []struct {
		name               string
		required, computed bool
	}{
		{"id", false, true},
		{"alias", true, false},
		{"certificate", true, false},
		{"certificate_sha256", false, true},
		{"subject_dn", false, true},
		{"issuer_dn", false, true},
		{"serial_number", false, true},
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

	certAttr, ok := s.Attributes["certificate"].(schema.StringAttribute)
	if !ok || certAttr.Sensitive {
		t.Error("certificate must not be marked Sensitive: public certificate content is not a secret")
	}

	aliasAttr, ok := s.Attributes["alias"].(schema.StringAttribute)
	if !ok || len(aliasAttr.PlanModifiers) == 0 {
		t.Error("alias must carry a RequiresReplace plan modifier")
	}
}

func newCertTestState(t *testing.T, s schema.Schema) tfsdk.State {
	t.Helper()
	return tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
}

func TestCertificateResource_Create(t *testing.T) {
	pemContent := generateTestCertPEM(t, "cert.example.invalid")

	var gotPath, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	r := &certificateResource{client: securitycontent.New(http.DefaultClient, server.URL)}
	s := certificateSchema(t).Schema
	ctx := context.Background()
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	planRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                  tftypes.NewValue(tftypes.String, nil),
		"alias":               tftypes.NewValue(tftypes.String, "my-cert"),
		"runtime_location_id": tftypes.NewValue(tftypes.String, nil),
		"certificate":         tftypes.NewValue(tftypes.String, pemContent),
		"certificate_sha256":  tftypes.NewValue(tftypes.String, nil),
		"subject_dn":          tftypes.NewValue(tftypes.String, nil),
		"issuer_dn":           tftypes.NewValue(tftypes.String, nil),
		"serial_number":       tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: planRaw}}
	resp := &resource.CreateResponse{State: newCertTestState(t, s)}

	r.Create(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create() produced diagnostics: %v", resp.Diagnostics)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	wantPath := "/api/v1/CertificateResources('6d792d63657274')/$value"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	var got certificateModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.ID.ValueString() != "my-cert" {
		t.Errorf("id = %q, want my-cert", got.ID.ValueString())
	}
	if got.CertificateSHA256.ValueString() == "" {
		t.Error("certificate_sha256 must be populated after Create")
	}
	if !strings.Contains(got.SubjectDN.ValueString(), "cert.example.invalid") {
		t.Errorf("subject_dn = %q, want it to contain the common name", got.SubjectDN.ValueString())
	}
}

func TestCertificateResource_Create_RejectsInvalidCertificate(t *testing.T) {
	r := &certificateResource{client: securitycontent.New(http.DefaultClient, "https://example.invalid")}
	s := certificateSchema(t).Schema
	ctx := context.Background()
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	planRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                  tftypes.NewValue(tftypes.String, nil),
		"alias":               tftypes.NewValue(tftypes.String, "bad-cert"),
		"runtime_location_id": tftypes.NewValue(tftypes.String, nil),
		"certificate":         tftypes.NewValue(tftypes.String, "not a certificate"),
		"certificate_sha256":  tftypes.NewValue(tftypes.String, nil),
		"subject_dn":          tftypes.NewValue(tftypes.String, nil),
		"issuer_dn":           tftypes.NewValue(tftypes.String, nil),
		"serial_number":       tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: planRaw}}
	resp := &resource.CreateResponse{State: newCertTestState(t, s)}

	r.Create(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Create() with invalid certificate content did not produce a diagnostic")
	}
}

// TestCertificateResource_Read_PreservesFormattingWhenUnchanged proves the
// core drift-detection design: if the remote certificate is the same
// certificate (same fingerprint) but re-serialized with different line
// endings, Read must not overwrite the practitioner's own PEM text.
func TestCertificateResource_Read_PreservesFormattingWhenUnchanged(t *testing.T) {
	pemContent := generateTestCertPEM(t, "read.example.invalid")
	reformatted := strings.ReplaceAll(pemContent, "\n", "\r\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(reformatted))
	}))
	defer server.Close()

	r := &certificateResource{client: securitycontent.New(http.DefaultClient, server.URL)}
	s := certificateSchema(t).Schema
	ctx := context.Background()
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	stateRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                  tftypes.NewValue(tftypes.String, "my-cert"),
		"alias":               tftypes.NewValue(tftypes.String, "my-cert"),
		"runtime_location_id": tftypes.NewValue(tftypes.String, nil),
		"certificate":         tftypes.NewValue(tftypes.String, pemContent),
		"certificate_sha256":  tftypes.NewValue(tftypes.String, "placeholder"),
		"subject_dn":          tftypes.NewValue(tftypes.String, "placeholder"),
		"issuer_dn":           tftypes.NewValue(tftypes.String, "placeholder"),
		"serial_number":       tftypes.NewValue(tftypes.String, "placeholder"),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: stateRaw}}
	resp := &resource.ReadResponse{State: newCertTestState(t, s)}

	r.Read(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() produced diagnostics: %v", resp.Diagnostics)
	}

	var got certificateModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.Certificate.ValueString() != pemContent {
		t.Error("Read() overwrote the practitioner's PEM text despite the certificate being byte-identical in substance")
	}
}

// TestCertificateResource_Read_DetectsGenuineDrift proves the opposite
// case: a genuinely different certificate on the tenant must be reflected.
func TestCertificateResource_Read_DetectsGenuineDrift(t *testing.T) {
	original := generateTestCertPEM(t, "original.example.invalid")
	changed := generateTestCertPEM(t, "changed.example.invalid")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(changed))
	}))
	defer server.Close()

	r := &certificateResource{client: securitycontent.New(http.DefaultClient, server.URL)}
	s := certificateSchema(t).Schema
	ctx := context.Background()
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	stateRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                  tftypes.NewValue(tftypes.String, "my-cert"),
		"alias":               tftypes.NewValue(tftypes.String, "my-cert"),
		"runtime_location_id": tftypes.NewValue(tftypes.String, nil),
		"certificate":         tftypes.NewValue(tftypes.String, original),
		"certificate_sha256":  tftypes.NewValue(tftypes.String, "placeholder"),
		"subject_dn":          tftypes.NewValue(tftypes.String, "placeholder"),
		"issuer_dn":           tftypes.NewValue(tftypes.String, "placeholder"),
		"serial_number":       tftypes.NewValue(tftypes.String, "placeholder"),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: stateRaw}}
	resp := &resource.ReadResponse{State: newCertTestState(t, s)}

	r.Read(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() produced diagnostics: %v", resp.Diagnostics)
	}

	var got certificateModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.Certificate.ValueString() != changed {
		t.Error("Read() should surface a genuinely different remote certificate as drift")
	}
}

// TestCertificateResource_Delete_SubmitsOnlyItsOwnAlias is the
// destructive-safety regression test at the resource layer.
func TestCertificateResource_Delete_SubmitsOnlyItsOwnAlias(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = body
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	r := &certificateResource{client: securitycontent.New(http.DefaultClient, server.URL)}
	s := certificateSchema(t).Schema
	ctx := context.Background()
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	stateRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                  tftypes.NewValue(tftypes.String, "tf-acc-cert-only"),
		"alias":               tftypes.NewValue(tftypes.String, "tf-acc-cert-only"),
		"runtime_location_id": tftypes.NewValue(tftypes.String, nil),
		"certificate":         tftypes.NewValue(tftypes.String, "placeholder"),
		"certificate_sha256":  tftypes.NewValue(tftypes.String, "placeholder"),
		"subject_dn":          tftypes.NewValue(tftypes.String, "placeholder"),
		"issuer_dn":           tftypes.NewValue(tftypes.String, "placeholder"),
		"serial_number":       tftypes.NewValue(tftypes.String, "placeholder"),
	})

	req := resource.DeleteRequest{State: tfsdk.State{Schema: s, Raw: stateRaw}}
	resp := &resource.DeleteResponse{}

	r.Delete(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete() produced diagnostics: %v", resp.Diagnostics)
	}

	want := `{"Aliases":"tf-acc-cert-only"}`
	if string(gotBody) != want {
		t.Errorf("delete request body = %s, want %s", gotBody, want)
	}
}

func TestCertificateResource_Delete_SAPOwnedEntrySurfacesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": {"code": "403", "message": {"value": "Cannot delete SAP-owned entry"}}}`))
	}))
	defer server.Close()

	r := &certificateResource{client: securitycontent.New(http.DefaultClient, server.URL)}
	s := certificateSchema(t).Schema
	ctx := context.Background()
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	stateRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                  tftypes.NewValue(tftypes.String, "sap_owned"),
		"alias":               tftypes.NewValue(tftypes.String, "sap_owned"),
		"runtime_location_id": tftypes.NewValue(tftypes.String, nil),
		"certificate":         tftypes.NewValue(tftypes.String, "placeholder"),
		"certificate_sha256":  tftypes.NewValue(tftypes.String, "placeholder"),
		"subject_dn":          tftypes.NewValue(tftypes.String, "placeholder"),
		"issuer_dn":           tftypes.NewValue(tftypes.String, "placeholder"),
		"serial_number":       tftypes.NewValue(tftypes.String, "placeholder"),
	})

	req := resource.DeleteRequest{State: tfsdk.State{Schema: s, Raw: stateRaw}}
	resp := &resource.DeleteResponse{}

	r.Delete(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Delete() against an SAP-owned entry should surface SAP's own rejection as an error, not silently succeed")
	}
}

func TestCertificateResource_ImportState_UsesAlias(t *testing.T) {
	r := NewCertificateResource().(resource.ResourceWithImportState)
	s := certificateSchema(t).Schema

	resp := &resource.ImportStateResponse{State: newCertTestState(t, s)}
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

// Replacing the certificate plans the new fingerprint, subject, issuer and
// serial number, computed from the new PEM, instead of keeping the old ones
// from state (a tenant run in September 2026 failed with "inconsistent result
// after apply" on exactly these attributes).
func TestCertificateResource_ModifyPlanComputesNewMetadata(t *testing.T) {
	s := certificateSchema(t).Schema
	objType := s.Type().TerraformType(context.Background())
	oldPEM, newPEM := generateTestCertPEM(t, "tfacc-first"), generateTestCertPEM(t, "tfacc-second")
	row := func(pem, sha, subject string) tftypes.Value {
		return tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":                  tftypes.NewValue(tftypes.String, "my-cert"),
			"alias":               tftypes.NewValue(tftypes.String, "my-cert"),
			"runtime_location_id": tftypes.NewValue(tftypes.String, nil),
			"certificate":         tftypes.NewValue(tftypes.String, pem),
			"certificate_sha256":  tftypes.NewValue(tftypes.String, sha),
			"subject_dn":          tftypes.NewValue(tftypes.String, subject),
			"issuer_dn":           tftypes.NewValue(tftypes.String, subject),
			"serial_number":       tftypes.NewValue(tftypes.String, "999"),
		})
	}
	// The plan still carries the old computed values, as UseStateForUnknown leaves them.
	req := resource.ModifyPlanRequest{
		State: tfsdk.State{Schema: s, Raw: row(oldPEM, "old", "CN=tfacc-first")},
		Plan:  tfsdk.Plan{Schema: s, Raw: row(newPEM, "old", "CN=tfacc-first")},
	}
	resp := resource.ModifyPlanResponse{Plan: req.Plan}
	(&certificateResource{}).ModifyPlan(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var got certificateModel
	resp.Diagnostics.Append(resp.Plan.Get(context.Background(), &got)...)
	if got.SubjectDN.ValueString() != "CN=tfacc-second" || got.CertificateSHA256.ValueString() == "old" || got.SerialNumber.ValueString() == "999" {
		t.Errorf("planned metadata not recomputed: %+v", got)
	}
}
