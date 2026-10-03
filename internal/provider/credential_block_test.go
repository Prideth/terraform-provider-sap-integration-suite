package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var testAPICompositionBlock = credentialBlock{
	title: "API Composition", block: "api_composition", urlAttr: "host",
	envPrefix: "SAP_INTEGRATION_SUITE_API_COMPOSITION_", usedBy: "sapintegrationsuite_business_data_graph",
}

// clearAPICompositionEnv keeps a developer's own environment out of the
// test: an empty variable counts as unset.
func clearAPICompositionEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"HOST", "TOKEN_URL", "CLIENT_ID", "CLIENT_SECRET", "USERNAME", "PASSWORD", "ORIGIN"} {
		t.Setenv("SAP_INTEGRATION_SUITE_API_COMPOSITION_"+name, "")
	}
}

func resolveTestBlock(user *userLogin) (string, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	host, client, ok := testAPICompositionBlock.resolve(context.Background(), "test",
		types.StringValue("https://eu10.graph.sap"), types.StringValue("https://auth.example.com/oauth/token"),
		types.StringValue("client-id"), types.StringValue("client-secret"), user, &diags)
	if ok && client == nil {
		diags.AddError("test", "resolve reported success without a client")
	}
	return host, ok, diags
}

func TestCredentialBlock_ResolveAcceptsAUserLogin(t *testing.T) {
	clearAPICompositionEnv(t)
	cases := map[string]*userLogin{
		"no user block":       nil,
		"empty user":          {},
		"user":                {username: types.StringValue("key.user@example.com"), password: types.StringValue("secret")},
		"user with an origin": {username: types.StringValue("key.user@example.com"), password: types.StringValue("secret"), origin: types.StringValue("custom-idp")},
	}
	for name, user := range cases {
		t.Run(name, func(t *testing.T) {
			host, ok, diags := resolveTestBlock(user)
			if !ok || diags.HasError() {
				t.Fatalf("resolve() failed: %v", diags)
			}
			if host != "https://eu10.graph.sap" {
				t.Errorf("host = %q", host)
			}
		})
	}
}

func TestCredentialBlock_ResolveRejectsAnIncompleteUserLogin(t *testing.T) {
	clearAPICompositionEnv(t)
	cases := map[string]*userLogin{
		"username only":        {username: types.StringValue("key.user@example.com")},
		"password only":        {password: types.StringValue("secret")},
		"origin without login": {origin: types.StringValue("custom-idp")},
	}
	for name, user := range cases {
		t.Run(name, func(t *testing.T) {
			_, ok, diags := resolveTestBlock(user)
			if ok || !diags.HasError() {
				t.Fatal("resolve() accepted an incomplete user login")
			}
			if got := diags.Errors()[0].Summary(); got != "Incomplete API Composition user login" {
				t.Errorf("summary = %q", got)
			}
		})
	}
}

func TestCredentialBlock_ResolveReadsTheUserFromTheEnvironment(t *testing.T) {
	clearAPICompositionEnv(t)
	t.Setenv("SAP_INTEGRATION_SUITE_API_COMPOSITION_USERNAME", "key.user@example.com")
	_, ok, diags := resolveTestBlock(&userLogin{})
	if ok || !diags.HasError() {
		t.Fatal("a username from the environment without a password was accepted")
	}

	t.Setenv("SAP_INTEGRATION_SUITE_API_COMPOSITION_PASSWORD", "secret")
	if _, ok, diags := resolveTestBlock(&userLogin{}); !ok || diags.HasError() {
		t.Fatalf("resolve() failed with username and password from the environment: %v", diags)
	}
}
