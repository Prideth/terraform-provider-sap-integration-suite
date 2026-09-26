package provider

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/samples"
)

// A flow exported under another ID (here SAP's codejam flow) is uploaded with
// the flow ID as Bundle-SymbolicName, the local file stays as it is, and the
// plan shows a warning. Without this SAP rejects every later content update.
func TestIntegrationFlowCreate_AlignsBundleID(t *testing.T) {
	ctx := context.Background()
	content := testAccExportFlow(t, "codejam-package-export", "Request Employee Dependants - Exercise 05")
	path := filepath.Join(t.TempDir(), "flow.zip")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)

	var uploaded []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var sent struct {
			ArtifactContent string `json:"ArtifactContent"`
		}
		_ = json.Unmarshal(body, &sent)
		uploaded, _ = base64.StdEncoding.DecodeString(sent.ArtifactContent)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"d": {"Id": "Order_Flow", "Version": "1.0.1", "Name": "Order Flow", "PackageId": "ORDERS"}}`))
	}))
	defer server.Close()

	r := &integrationFlowResource{client: cloudintegration.New(http.DefaultClient, server.URL)}
	var sresp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sresp)
	plan := tfsdk.Plan{Schema: sresp.Schema, Raw: tftypes.NewValue(sresp.Schema.Type().TerraformType(ctx), nil)}
	plan.Set(ctx, integrationFlowModel{
		ID: types.StringUnknown(), PackageID: types.StringValue("ORDERS"), FlowID: types.StringValue("Order_Flow"),
		Name: types.StringValue("Order Flow"), Content: types.StringValue(path),
		ContentHash: types.StringValue(hex.EncodeToString(sum[:])), Version: types.StringUnknown(),
		SaveAsVersion: types.StringNull(),
	})
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sresp.Schema, Raw: tftypes.NewValue(sresp.Schema.Type().TerraformType(ctx), nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create() diagnostics: %v", resp.Diagnostics)
	}

	m, err := cloudintegration.ParseBundleManifest(uploaded)
	if err != nil || m.SymbolicName != "Order_Flow" {
		t.Errorf("uploaded Bundle-SymbolicName = %q (err %v), want the flow ID", m.SymbolicName, err)
	}
	if local, _ := os.ReadFile(path); string(local) != string(content) {
		t.Error("the local file was changed")
	}
	warned := false
	for _, d := range resp.Diagnostics.Warnings() {
		warned = warned || strings.Contains(d.Summary(), "Bundle-SymbolicName adjusted")
	}
	if !warned {
		t.Errorf("want a warning about the adjusted bundle ID, got %v", resp.Diagnostics)
	}
}

// A ZIP whose bundle ID already equals the flow ID is uploaded byte for byte.
func TestAlignUploadBundleID_UnchangedWhenMatching(t *testing.T) {
	content, err := samples.WithBundleID(samples.Get(t, "spend-account-dim-map"), "Order_Map")
	if err != nil {
		t.Fatal(err)
	}
	var diags = resource.CreateResponse{}.Diagnostics
	if got := alignUploadBundleID(content, "Order_Map", "map.zip", &diags); string(got) != string(content) || diags.WarningsCount() != 0 {
		t.Errorf("content changed or warned (%v)", diags)
	}
}
