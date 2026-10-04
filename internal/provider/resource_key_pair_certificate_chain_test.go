package provider

import (
	"context"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/securitycontent"
)

// The fixtures are a throw-away root, intermediate and leaf (public
// certificates only), as PEM and as the PKCS#7 bundle SAP exports.
func chainFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("../client/securitycontent/testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func chainSubjects(t *testing.T, s tfsdk.State) string {
	t.Helper()
	var m keyPairCertificateChainModel
	if d := s.Get(context.Background(), &m); d.HasError() {
		t.Fatal(d)
	}
	var names []string
	for _, v := range m.Certificates.Elements() {
		names = append(names, v.(types.Object).Attributes()["subject_dn"].(types.String).ValueString())
	}
	return strings.Join(names, ",")
}

func TestKeyPairCertificateChain_ModifyPlanDerivesTheCertificates(t *testing.T) {
	r := &keyPairCertificateChainResource{}
	plan := map[string]tftypes.Value{"key_pair_alias": str("client"), "certificate_chain": str(chainFixture(t, "chain-leaf-intermediate-root.pem"))}
	raw, empty := resourceObject(t, r, plan)
	req := resource.ModifyPlanRequest{
		State: empty,
		Plan:  tfsdk.Plan{Schema: empty.Schema, Raw: raw},
	}
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var m keyPairCertificateChainModel
	resp.Plan.Get(context.Background(), &m)
	if len(m.CertificateSHA256.ValueString()) != 64 {
		t.Errorf("certificate_sha256 = %q", m.CertificateSHA256.ValueString())
	}
	if got := chainSubjects(t, tfsdk.State(resp.Plan)); got != "CN=tfacc-test-leaf,CN=tfacc-test-intermediate,CN=tfacc-test-root" {
		t.Errorf("certificates = %s", got)
	}
}

func TestKeyPairCertificateChain_ValidateConfigRefusesPrivateKeys(t *testing.T) {
	r := &keyPairCertificateChainResource{}
	chain := chainFixture(t, "chain-leaf-intermediate-root.pem") + "-----BEGIN RSA PRIVATE KEY-----\nMIIB\n-----END RSA PRIVATE KEY-----\n"
	raw, state := resourceObject(t, r, map[string]tftypes.Value{"key_pair_alias": str("client"), "certificate_chain": str(chain)})
	var resp resource.ValidateConfigResponse
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: state.Schema, Raw: raw}}, &resp)
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics[0].Detail(), "RSA PRIVATE KEY") {
		t.Errorf("diagnostics = %v, want the private key refused", resp.Diagnostics)
	}
}

func chainServer(t *testing.T, status int) *httptest.Server {
	t.Helper()
	p7 := chainFixture(t, "chain-leaf-intermediate-root.p7c")
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/KeystoreEntries('636c69656e74')/ChainResource/$value" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		_, _ = w.Write([]byte(p7))
	}))
}

func TestKeyPairCertificateChain_ReadKeepsTheConfiguredText(t *testing.T) {
	server := chainServer(t, http.StatusOK)
	defer server.Close()
	r := &keyPairCertificateChainResource{client: securitycontent.New(http.DefaultClient, server.URL)}
	pemChain := chainFixture(t, "chain-leaf-intermediate-root.pem")
	certs, _ := securitycontent.ParseCertificateChainPEM([]byte(pemChain))
	rootFirst := string(securitycontent.EncodeCertificateChainPEM([]*x509.Certificate{certs[2], certs[1], certs[0]}))

	for _, c := range []struct {
		name, configured string
		kept             bool
	}{
		{"same certificates, other order", rootFirst, true},
		{"leaf only on the configuration side", string(securitycontent.EncodeCertificateChainPEM(certs[:1])), false},
	} {
		t.Run(c.name, func(t *testing.T) {
			raw, empty := resourceObject(t, r, map[string]tftypes.Value{"id": str("client"), "key_pair_alias": str("client"), "certificate_chain": str(c.configured)})
			prior := tfsdk.State{Schema: empty.Schema, Raw: raw}
			resp := &resource.ReadResponse{State: prior}
			r.Read(context.Background(), resource.ReadRequest{State: prior}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			var m keyPairCertificateChainModel
			resp.State.Get(context.Background(), &m)
			if kept := m.CertificateChain.ValueString() == c.configured; kept != c.kept {
				t.Errorf("configured text kept = %v, want %v", kept, c.kept)
			}
			if got := chainSubjects(t, resp.State); got != "CN=tfacc-test-leaf,CN=tfacc-test-intermediate,CN=tfacc-test-root" {
				t.Errorf("certificates = %s", got)
			}
		})
	}
}

func TestKeyPairCertificateChain_ReadRemovesAMissingKeyPair(t *testing.T) {
	server := chainServer(t, http.StatusNotFound)
	defer server.Close()
	r := &keyPairCertificateChainResource{client: securitycontent.New(http.DefaultClient, server.URL)}
	raw, empty := resourceObject(t, r, map[string]tftypes.Value{"id": str("client"), "key_pair_alias": str("client"), "certificate_chain": str("x")})
	prior := tfsdk.State{Schema: empty.Schema, Raw: raw}
	resp := &resource.ReadResponse{State: prior}
	r.Read(context.Background(), resource.ReadRequest{State: prior}, resp)
	if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
		t.Errorf("diagnostics %v, state null = %v; want the resource removed", resp.Diagnostics, resp.State.Raw.IsNull())
	}
}

func TestKeyPairCertificateChain_DeleteWarnsAndSendsNothing(t *testing.T) {
	r := &keyPairCertificateChainResource{}
	raw, empty := resourceObject(t, r, map[string]tftypes.Value{"key_pair_alias": str("client")})
	resp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: tfsdk.State{Schema: empty.Schema, Raw: raw}}, resp)
	if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 {
		t.Errorf("diagnostics = %v, want one warning", resp.Diagnostics)
	}
}
