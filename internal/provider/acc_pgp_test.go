package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/auth"
	sapthttp "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/http"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/securitycontent"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

// testAccSecurityContentClient builds the Security Content client from the
// same environment variables the provider reads.
func testAccSecurityContentClient(t *testing.T) *securitycontent.Client {
	t.Helper()
	httpClient, invalidate, err := auth.Config{
		TokenURL:     os.Getenv("SAP_INTEGRATION_SUITE_TOKEN_URL"),
		ClientID:     os.Getenv("SAP_INTEGRATION_SUITE_CLIENT_ID"),
		ClientSecret: os.Getenv("SAP_INTEGRATION_SUITE_CLIENT_SECRET"),
	}.HTTPClient(context.Background(), http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	return securitycontent.New(sapthttp.New(sapthttp.Config{Transport: httpClient, InvalidateToken: invalidate}),
		os.Getenv("SAP_INTEGRATION_SUITE_HOST"))
}

// testAccPGPKey is a throw-away PGP key gpg generates for one test.
type testAccPGPKey struct {
	keyID, public, secret, passphrase string
}

// newTestAccPGPKey generates an RSA key with an encryption sub key, valid
// for two days, in a temporary gpg home. The test is skipped without gpg.
func newTestAccPGPKey(t *testing.T, name string) testAccPGPKey {
	t.Helper()
	gpg, err := exec.LookPath("gpg")
	if err != nil {
		t.Skip("gpg not found; the PGP acceptance test generates its keys with it")
	}
	home := t.TempDir()
	t.Cleanup(func() {
		if gpgconf, err := exec.LookPath("gpgconf"); err == nil {
			_ = exec.Command(gpgconf, "--homedir", home, "--kill", "all").Run() // #nosec G204 -- a fixed tool with a temporary directory
		}
	})
	key := testAccPGPKey{passphrase: "tfacc-" + testAccName()}
	run := func(args ...string) string {
		t.Helper()
		all := append([]string{"--homedir", home, "--batch", "--yes", "--pinentry-mode", "loopback", "--passphrase", key.passphrase}, args...)
		out, err := exec.Command(gpg, all...).Output() // #nosec G204 -- gpg with arguments built here
		if err != nil {
			t.Fatalf("gpg %s: %v", args[0], err)
		}
		return string(out)
	}
	uid := fmt.Sprintf("%s <%s@example.invalid>", name, name)
	run("--quick-gen-key", uid, "rsa2048", "sign,cert", "2d")
	var fpr string
	for _, line := range strings.Split(run("--with-colons", "--list-keys", uid), "\n") {
		if strings.HasPrefix(line, "fpr:") {
			fpr = strings.Split(line, ":")[9]
			break
		}
	}
	run("--quick-add-key", fpr, "rsa2048", "encr", "2d")
	key.keyID = fpr[len(fpr)-16:]
	key.public = run("--armor", "--export", fpr)
	key.secret = run("--armor", "--export-secret-keys", fpr)
	return key
}

// The tenant's keyrings are shared: the test adds only keys it generated
// and deletes exactly those key IDs.
func TestAccPGPKeys_publicAndSecret(t *testing.T) {
	accgate.Require(t, accgate.SecurityContent)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "true")
	public := newTestAccPGPKey(t, testAccName())
	secret := newTestAccPGPKey(t, testAccName())
	config := fmt.Sprintf(`
resource "sapintegrationsuite_pgp_public_key" "test" {
  public_key = %q
}

variable "secret_key" {
  type      = string
  sensitive = true
}

variable "passphrase" {
  type      = string
  sensitive = true
}

resource "sapintegrationsuite_pgp_secret_key" "test" {
  secret_key_wo         = var.secret_key
  passphrase_wo         = var.passphrase
  secret_key_wo_version = "1"
}
`, public.public)
	vars := map[string]string{"TF_VAR_secret_key": secret.secret, "TF_VAR_passphrase": secret.passphrase}
	for k, v := range vars {
		t.Setenv(k, v)
	}
	client := testAccSecurityContentClient(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			for _, id := range []string{public.keyID, secret.keyID} {
				if _, err := client.GetPGPKey(context.Background(), id); !isNotFound(err) {
					return fmt.Errorf("PGP key %s still exists (%v)", id, err)
				}
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sapintegrationsuite_pgp_public_key.test", "key_id", public.keyID),
					resource.TestCheckResourceAttr("sapintegrationsuite_pgp_public_key.test", "type", "Public"),
					resource.TestCheckResourceAttr("sapintegrationsuite_pgp_public_key.test", "key_length", "2048"),
					resource.TestCheckResourceAttrSet("sapintegrationsuite_pgp_public_key.test", "valid_until"),
					resource.TestCheckResourceAttr("sapintegrationsuite_pgp_secret_key.test", "key_id", secret.keyID),
					resource.TestCheckResourceAttr("sapintegrationsuite_pgp_secret_key.test", "type", "Secret"),
				),
			},
			{
				ResourceName:            "sapintegrationsuite_pgp_public_key.test",
				ImportState:             true,
				ImportStateId:           public.keyID,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"public_key"},
			},
			{
				ResourceName:            "sapintegrationsuite_pgp_secret_key.test",
				ImportState:             true,
				ImportStateId:           secret.keyID,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret_key_wo_version"},
			},
		},
	})
}
