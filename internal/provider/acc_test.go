package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/samples"
)

// Acceptance tests run the provider through Terraform against a real tenant.
// They only run with TF_ACC=1 and the SAP_INTEGRATION_SUITE_* variables set
// (.specs/run-acceptance.ps1 fills them from keys.local.json). Every object
// they create is named tfacc<random> and is destroyed at the end of its test.

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"sapintegrationsuite": providerserver.NewProtocol6WithError(New("acctest")()),
}

// testAccPreCheck skips an acceptance test unless the Cloud Integration
// credentials are present in the environment.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"SAP_INTEGRATION_SUITE_HOST", "SAP_INTEGRATION_SUITE_TOKEN_URL",
		"SAP_INTEGRATION_SUITE_CLIENT_ID", "SAP_INTEGRATION_SUITE_CLIENT_SECRET",
	} {
		if os.Getenv(name) == "" {
			t.Skipf("%s is not set", name)
		}
	}
}

// testAccName returns a unique object name that SAP accepts as a package,
// flow or mapping ID.
func testAccName() string {
	return "tfacc" + acctest.RandStringFromCharSet(10, "abcdefghijklmnopqrstuvwxyz0123456789")
}

// testAccArtifactFile writes a sample artifact under a new bundle ID into a
// temporary file and returns its path in the forward-slash form HCL needs.
// SAP rejects a content update whose Bundle-SymbolicName differs from the
// artifact ID, so the ID in the file always equals the ID in Terraform.
func testAccArtifactFile(t *testing.T, content []byte, id string) string {
	t.Helper()
	rewritten, err := samples.WithBundleID(content, id)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), id+".zip")
	if err := os.WriteFile(path, rewritten, 0o600); err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(path)
}

// testAccExportFlow returns the integration flow ZIP whose display name
// starts with prefix from a package export sample.
func testAccExportFlow(t *testing.T, sample, prefix string) []byte {
	t.Helper()
	artifacts, err := samples.ExportArtifacts(samples.Get(t, sample))
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range artifacts {
		if a.Type == "IFlow" && strings.HasPrefix(a.DisplayName, prefix) {
			return a.Content
		}
	}
	t.Fatalf("sample %s has no integration flow %q", sample, prefix)
	return nil
}
