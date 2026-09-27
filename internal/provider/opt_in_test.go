package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/features"
)

// optInError reports whether the diagnostics contain the opt-in error for a
// status.
func optInError(diags diag.Diagnostics, status features.SupportStatus) bool {
	for _, d := range diags.Errors() {
		if strings.HasSuffix(d.Summary(), " is "+string(status)) {
			return true
		}
	}
	return false
}

// checkGate verifies one type: an experimental or unofficial type refuses to
// configure without its switch and configures past the gate with it; every
// other type never asks for a switch. The provider data carries no clients,
// so a type may still fail later for missing credentials; only the opt-in
// error is checked.
func checkGate(t *testing.T, typeName string, configure func(*Data) diag.Diagnostics) {
	t.Helper()
	status := catalogStatus(typeName)
	gated := status == features.StatusExperimental || status == features.StatusUnofficial

	off := configure(&Data{})
	if gated && !optInError(off, status) {
		t.Errorf("%s is %s in the catalog but configures without enable_%s", typeName, status, status)
	}
	if !gated && (optInError(off, features.StatusExperimental) || optInError(off, features.StatusUnofficial)) {
		t.Errorf("%s is %q in the catalog but asks for an opt-in", typeName, status)
	}
	on := configure(&Data{EnableExperimental: true, EnableUnofficial: true})
	if optInError(on, features.StatusExperimental) || optInError(on, features.StatusUnofficial) {
		t.Errorf("%s still refuses with both switches on", typeName)
	}
}

// TestOptIn_MatchesCatalog keeps the gates in step with the catalog: when a
// status changes (an experimental resource passes its acceptance test, an
// unofficial one gets documented, a new resource starts out experimental),
// this test fails until the resource's Configure calls requireOptIn or stops
// calling it.
func TestOptIn_MatchesCatalog(t *testing.T) {
	ctx := context.Background()
	p := &sapIntegrationSuiteProvider{version: "test"}
	for _, newResource := range p.Resources(ctx) {
		r := newResource()
		var meta resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "sapintegrationsuite"}, &meta)
		rc, ok := r.(resource.ResourceWithConfigure)
		if !ok {
			continue
		}
		checkGate(t, meta.TypeName, func(d *Data) diag.Diagnostics {
			var resp resource.ConfigureResponse
			rc.Configure(ctx, resource.ConfigureRequest{ProviderData: d}, &resp)
			return resp.Diagnostics
		})
	}
	for _, newDataSource := range p.DataSources(ctx) {
		ds := newDataSource()
		var meta datasource.MetadataResponse
		ds.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "sapintegrationsuite"}, &meta)
		dc, ok := ds.(datasource.DataSourceWithConfigure)
		if !ok {
			continue
		}
		checkGate(t, meta.TypeName, func(d *Data) diag.Diagnostics {
			var resp datasource.ConfigureResponse
			dc.Configure(ctx, datasource.ConfigureRequest{ProviderData: d}, &resp)
			return resp.Diagnostics
		})
	}
}

func TestBoolOrEnv(t *testing.T) {
	t.Setenv("SAP_INTEGRATION_SUITE_TEST_FLAG", "true")
	if !boolOrEnv(boolNull(), "SAP_INTEGRATION_SUITE_TEST_FLAG") {
		t.Error("true from the environment")
	}
	t.Setenv("SAP_INTEGRATION_SUITE_TEST_FLAG", "no")
	if boolOrEnv(boolNull(), "SAP_INTEGRATION_SUITE_TEST_FLAG") {
		t.Error("unparsable must be false")
	}
	if !boolOrEnv(boolValue(true), "SAP_INTEGRATION_SUITE_TEST_FLAG") {
		t.Error("the configured value wins")
	}
}

func boolNull() types.Bool        { return types.BoolNull() }
func boolValue(v bool) types.Bool { return types.BoolValue(v) }
