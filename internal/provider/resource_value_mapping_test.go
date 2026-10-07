package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

// fakeValueMappings answers like the tenant of the probe of 2026-10-06: one
// stored version per value mapping, which SaveAsVersion relabels (200,
// empty body) without touching anything else.
type fakeValueMappings struct {
	mu       sync.Mutex
	version  string
	requests []string
}

func (f *fakeValueMappings) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	switch {
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/ValueMappingDesigntimeArtifactSaveAsVersion"):
		f.version = strings.Trim(r.URL.Query().Get("SaveAsVersion"), "'")
		w.WriteHeader(http.StatusOK)
	case r.Method == http.MethodPost:
		f.version = "1.0.0"
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, `{"d":{"Id":"CompanyCodes","Name":"Company Codes","PackageId":"P","Version":%q}}`, f.version)
	default:
		_, _ = fmt.Fprintf(w, `{"d":{"Id":"CompanyCodes","Name":"Company Codes","PackageId":"P","Version":%q}}`, f.version)
	}
}

// testValueMappingHash is the SHA-256 of the test content, "PK".
const testValueMappingHash = "fcab7fcc2b4cffd9bb45003bfc2e468a04ef6f77ca8200a7341f027631584d25"

func valueMappingSchema(r *valueMappingResource) schema.Schema {
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp.Schema
}

// valueMappingPlan plans the CompanyCodes value mapping from contentFile;
// an empty saveAs leaves save_as_version out.
func valueMappingPlan(r *valueMappingResource, contentFile, saveAs string) tfsdk.Plan {
	s := valueMappingSchema(r)
	var saveAsValue any
	if saveAs != "" {
		saveAsValue = saveAs
	}
	return tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"package_id":      tftypes.NewValue(tftypes.String, "P"),
		"mapping_id":      tftypes.NewValue(tftypes.String, "CompanyCodes"),
		"name":            tftypes.NewValue(tftypes.String, "Company Codes"),
		"content":         tftypes.NewValue(tftypes.String, contentFile),
		"content_hash":    tftypes.NewValue(tftypes.String, testValueMappingHash),
		"version":         tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"save_as_version": tftypes.NewValue(tftypes.String, saveAsValue),
	})}
}

func TestValueMappingResource_SaveAsVersion(t *testing.T) {
	fake := &fakeValueMappings{}
	server := httptest.NewServer(fake)
	defer server.Close()
	r := &valueMappingResource{client: cloudintegration.New(http.DefaultClient, server.URL), allowUnofficial: true}
	ctx := context.Background()
	content := filepath.Join(t.TempDir(), "company-codes.zip")
	if err := os.WriteFile(content, []byte("PK"), 0o600); err != nil {
		t.Fatal(err)
	}
	version := func(st tfsdk.State) (string, string) {
		var m valueMappingModel
		st.Get(ctx, &m)
		return m.Version.ValueString(), m.SaveAsVersion.ValueString()
	}

	createResp := &resource.CreateResponse{State: newTestState(t, valueMappingSchema(r))}
	r.Create(ctx, resource.CreateRequest{Plan: valueMappingPlan(r, content, "2.0.0")}, createResp)
	if v, s := version(createResp.State); createResp.Diagnostics.HasError() || v != "2.0.0" || s != "2.0.0" {
		t.Fatalf("Create() = %v, version %q, save_as_version %q; want both 2.0.0", createResp.Diagnostics, v, s)
	}

	// A lower number relabels in place.
	relabel := &resource.UpdateResponse{State: createResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: valueMappingPlan(r, content, "1.9.0"), State: createResp.State}, relabel)
	if v, _ := version(relabel.State); relabel.Diagnostics.HasError() || v != "1.9.0" {
		t.Fatalf("Update(relabel) = %v, version %q; want 1.9.0", relabel.Diagnostics, v)
	}

	// Without save_as_version nothing is sent; the label stays on SAP's side.
	saves := strings.Count(strings.Join(fake.requests, "\n"), "SaveAsVersion")
	removed := &resource.UpdateResponse{State: relabel.State}
	r.Update(ctx, resource.UpdateRequest{Plan: valueMappingPlan(r, content, ""), State: relabel.State}, removed)
	v, s := version(removed.State)
	if removed.Diagnostics.HasError() || v != "1.9.0" || s != "" || strings.Count(strings.Join(fake.requests, "\n"), "SaveAsVersion") != saves {
		t.Errorf("Update(removed) = %v, version %q, save_as_version %q, requests %v", removed.Diagnostics, v, s, fake.requests)
	}

	// Without enable_unofficial the relabel is refused before any request.
	r.allowUnofficial = false
	before := len(fake.requests)
	refused := &resource.UpdateResponse{State: relabel.State}
	r.Update(ctx, resource.UpdateRequest{Plan: valueMappingPlan(r, content, "3.0.0"), State: relabel.State}, refused)
	if !unofficialOperationError(refused.Diagnostics) || len(fake.requests) != before {
		t.Errorf("Update() without enable_unofficial = %v after %d requests; want the unofficial error and none", refused.Diagnostics, len(fake.requests)-before)
	}
}
