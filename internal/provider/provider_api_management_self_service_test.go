package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	provschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// selfServiceProviderConfig builds a provider config in which only the
// api_management_self_service block has values.
func selfServiceProviderConfig(t *testing.T, s provschema.Schema, values map[string]string) tfsdk.Config {
	t.Helper()
	objType := s.Type().TerraformType(context.Background()).(tftypes.Object)
	attrValues := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, at := range objType.AttributeTypes {
		if name != "api_management_self_service" {
			attrValues[name] = tftypes.NewValue(at, nil)
			continue
		}
		blockType := at.(tftypes.Object)
		blockValues := make(map[string]tftypes.Value, len(blockType.AttributeTypes))
		for attr := range blockType.AttributeTypes {
			blockValues[attr] = tftypes.NewValue(tftypes.String, stringOrNilValue(values[attr]))
		}
		attrValues[name] = tftypes.NewValue(blockType, blockValues)
	}
	return tfsdk.Config{Schema: s, Raw: tftypes.NewValue(objType, attrValues)}
}

func configureSelfService(t *testing.T, values map[string]string) (*Data, provider.ConfigureResponse) {
	t.Helper()
	var resp provider.ConfigureResponse
	New("test")().Configure(context.Background(), provider.ConfigureRequest{
		Config: selfServiceProviderConfig(t, providerSchemaForAPIManagementTests(t), values),
	}, &resp)
	data, _ := resp.ResourceData.(*Data)
	return data, resp
}

func TestProviderConfigure_APIManagementSelfService(t *testing.T) {
	complete := map[string]string{
		"host":          "https://mysubaccount.apiportal.example.com",
		"token_url":     "https://mysubaccount.authentication.eu10.hana.ondemand.com/oauth/token",
		"client_id":     "client-id",
		"client_secret": "client-secret",
	}
	data, resp := configureSelfService(t, complete)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Configure() produced diagnostics: %v", resp.Diagnostics)
	}
	if data.APIManagementSelfServiceHTTPClient == nil || data.APIManagementSelfServiceHost != complete["host"] {
		t.Errorf("self-service client not configured: host %q", data.APIManagementSelfServiceHost)
	}
	if data.APIManagementSelfServiceSubdomain != "mysubaccount" {
		t.Errorf("subdomain = %q, want the token URL's first label", data.APIManagementSelfServiceSubdomain)
	}
	if data.APIManagementClassicHTTPClient != nil {
		t.Error("the api_management client must stay unset when only the self-service block is configured")
	}

	explicit := map[string]string{"subaccount_subdomain": "other"}
	for k, v := range complete {
		explicit[k] = v
	}
	if data, _ := configureSelfService(t, explicit); data.APIManagementSelfServiceSubdomain != "other" {
		t.Errorf("subdomain = %q, want the configured one", data.APIManagementSelfServiceSubdomain)
	}

	if _, resp := configureSelfService(t, map[string]string{"host": complete["host"]}); !resp.Diagnostics.HasError() {
		t.Error("a partial api_management_self_service block produced no error")
	}
}

func TestSubdomainFromTokenURL(t *testing.T) {
	for tokenURL, want := range map[string]string{
		"https://mysubaccount.authentication.eu10.hana.ondemand.com/oauth/token": "mysubaccount",
		"https://mysubaccount.authentication.sap.hana.ondemand.com/oauth/token":  "mysubaccount",
		"https://login.example.com/oauth/token":                                  "",
		"https://authentication.eu10.hana.ondemand.com/oauth/token":              "",
		"not a url\x7f": "",
	} {
		if got := subdomainFromTokenURL(tokenURL); got != want {
			t.Errorf("subdomainFromTokenURL(%q) = %q, want %q", tokenURL, got, want)
		}
	}
}
