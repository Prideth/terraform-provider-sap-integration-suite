package provider

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

// Security Content acceptance tests. Every object is named tfacc<random>
// and destroyed at the end. Secrets are made up; certificates are generated
// here and are self-signed, the case that needs the fingerprint confirmation
// SAP asks for.

func TestAccUserCredential_basic(t *testing.T) {
	accgate.Require(t, accgate.SecurityContent)
	name := testAccName()
	config := func(description, version string) string {
		return fmt.Sprintf(`
resource "sapintegrationsuite_user_credential" "test" {
  id                  = %[1]q
  description         = %[2]q
  user                = "tfacc-user"
  password_wo         = "tfacc-%[3]s-not-a-real-password"
  password_wo_version = %[3]q
}
`, name, description, version)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("created", "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sapintegrationsuite_user_credential.test", "id", name),
					resource.TestCheckResourceAttr("sapintegrationsuite_user_credential.test", "kind", "default"),
				),
			},
			{
				// Rotating the password and changing the description are one
				// in-place update.
				Config: config("rotated", "2"),
				Check:  resource.TestCheckResourceAttr("sapintegrationsuite_user_credential.test", "description", "rotated"),
			},
			{
				ResourceName:            "sapintegrationsuite_user_credential.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password_wo_version"},
			},
		},
	})
}

// SAP accepts the kinds only in lower case (tenant check of 2026-10-04);
// releases before 0.5.1 documented "SuccessFactors", which failed at apply.
func TestAccUserCredential_successFactorsKind(t *testing.T) {
	accgate.Require(t, accgate.SecurityContent)
	name := testAccName()
	config := func(kind string) string {
		return fmt.Sprintf(`
resource "sapintegrationsuite_user_credential" "test" {
  id                  = %[1]q
  kind                = %[2]q
  user                = "tfacc-user"
  company_id          = "TFACCCOMPANY"
  password_wo         = "tfacc-not-a-real-password"
  password_wo_version = "1"
}
`, name, kind)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("SuccessFactors"), PlanOnly: true, ExpectError: regexp.MustCompile(`Use "successfactors"`)},
			{
				Config: config("successfactors"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sapintegrationsuite_user_credential.test", "kind", "successfactors"),
					resource.TestCheckResourceAttr("sapintegrationsuite_user_credential.test", "company_id", "TFACCCOMPANY"),
				),
			},
			{
				ResourceName:            "sapintegrationsuite_user_credential.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password_wo_version"},
			},
		},
	})
}

func TestAccOAuth2ClientCredential_basic(t *testing.T) {
	accgate.Require(t, accgate.SecurityContent)
	name := testAccName()
	config := func(description string) string {
		return fmt.Sprintf(`
resource "sapintegrationsuite_oauth2_client_credential" "test" {
  id                       = %[1]q
  description              = %[2]q
  token_service_url        = "https://tfacc.invalid/oauth/token"
  client_id                = "tfacc-client"
  client_secret_wo         = "tfacc-not-a-real-secret"
  client_secret_wo_version = "1"
}
`, name, description)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("created"), Check: resource.TestCheckResourceAttr("sapintegrationsuite_oauth2_client_credential.test", "id", name)},
			{Config: config("updated"), Check: resource.TestCheckResourceAttr("sapintegrationsuite_oauth2_client_credential.test", "description", "updated")},
			{
				ResourceName:            "sapintegrationsuite_oauth2_client_credential.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"client_secret_wo_version"},
			},
		},
	})
}

func TestAccSecureParameter_basic(t *testing.T) {
	accgate.Require(t, accgate.SecurityContent)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "true")
	name := testAccName()
	config := func(description, version string) string {
		return fmt.Sprintf(`
resource "sapintegrationsuite_secure_parameter" "test" {
  id                      = %[1]q
  description             = %[2]q
  secure_param_wo         = "tfacc-%[3]s-not-a-real-value"
  secure_param_wo_version = %[3]q
}
`, name, description, version)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("created", "1"), Check: resource.TestCheckResourceAttr("sapintegrationsuite_secure_parameter.test", "id", name)},
			{Config: config("rotated", "2"), Check: resource.TestCheckResourceAttr("sapintegrationsuite_secure_parameter.test", "description", "rotated")},
			{
				ResourceName:            "sapintegrationsuite_secure_parameter.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secure_param_wo_version"},
			},
		},
	})
}

// testAccCertificateFile writes a self-signed certificate to a temporary
// PEM file; the private key is thrown away.
func testAccCertificateFile(t *testing.T, commonName string) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(30 * 24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), commonName+".pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(path)
}

// Import needs fingerprintVerified=true for a self-signed certificate, and
// replacing it under the same alias needs update=true (tenant test,
// September 2026).
func TestAccCertificate_selfSignedAndReplace(t *testing.T) {
	accgate.Require(t, accgate.SecurityContent)
	alias := testAccName()
	first := testAccCertificateFile(t, "tfacc-first")
	second := testAccCertificateFile(t, "tfacc-second")
	config := func(path string) string {
		return fmt.Sprintf(`
resource "sapintegrationsuite_certificate" "test" {
  alias       = %[1]q
  certificate = file(%[2]q)
}
`, alias, path)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config(first), Check: resource.TestCheckResourceAttr("sapintegrationsuite_certificate.test", "subject_dn", "CN=tfacc-first")},
			{Config: config(second), Check: resource.TestCheckResourceAttr("sapintegrationsuite_certificate.test", "subject_dn", "CN=tfacc-second")},
			{
				ResourceName:            "sapintegrationsuite_certificate.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"certificate"}, // SAP re-serializes the PEM
			},
		},
	})
}

// SAP generates the key pair; destroy removes exactly this alias with the
// keystore mass-delete operation.
func TestAccKeyPair_basic(t *testing.T) {
	accgate.Require(t, accgate.SecurityContent)
	alias := testAccName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "sapintegrationsuite_key_pair" "test" {
  alias       = %q
  common_name = "tfacc-key-pair"
  country     = "DE"
  key_size    = 2048
}
`, alias),
				Check: resource.TestCheckResourceAttrSet("sapintegrationsuite_key_pair.test", "public_key_openssh"),
			},
			{
				ResourceName:      "sapintegrationsuite_key_pair.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// testAccNumberRangeConfig is a number range whose counter is set to value
// whenever version changes. Names may not contain hyphens; tfacc<random> has
// none.
func testAccNumberRangeConfig(name, description, value, version string) string {
	return fmt.Sprintf(`
resource "sapintegrationsuite_number_range" "test" {
  name                     = %[1]q
  description              = %[2]q
  min_value                = "1"
  max_value                = "999999"
  rotate                   = true
  current_value_wo         = %[3]q
  current_value_wo_version = %[4]q
}
`, name, description, value, version)
}

// The full lifecycle, including the unofficial operations: drift detection
// (GET by name), an update that keeps the live counter, import and delete.
func TestAccNumberRange_basic(t *testing.T) {
	accgate.Require(t, accgate.CloudIntegration)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "true")
	name := testAccName()
	config := func(description string) string {
		return testAccNumberRangeConfig(name, description, "1", "1")
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("created"), Check: resource.TestCheckResourceAttr("sapintegrationsuite_number_range.test", "name", name)},
			{
				// An update without a new counter version keeps the live
				// counter, which SAP still requires in the PUT.
				Config: config("updated"),
				Check:  resource.TestCheckResourceAttr("sapintegrationsuite_number_range.test", "description", "updated"),
			},
			{
				ResourceName:            "sapintegrationsuite_number_range.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"current_value_wo_version"},
			},
		},
	})
}

// Without enable_unofficial a number range uses only what SAP documents:
// create (POST), and an update (PUT) that sends the configured counter. The
// last step turns the switch on, which reads the number range back to check
// the counter SAP stored and lets the test delete it.
func TestAccNumberRange_documentedOnly(t *testing.T) {
	accgate.Require(t, accgate.CloudIntegration)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "false")
	name := testAccName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNumberRangeConfig(name, "created", "1", "1"),
				Check:  resource.TestCheckResourceAttr("sapintegrationsuite_number_range.test", "current_value", "1"),
			},
			{
				Config: testAccNumberRangeConfig(name, "updated", "5", "2"),
				Check:  resource.TestCheckResourceAttr("sapintegrationsuite_number_range.test", "current_value", "5"),
			},
			{
				PreConfig: func() { t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "true") },
				Config:    testAccNumberRangeConfig(name, "updated", "5", "2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sapintegrationsuite_number_range.test", "current_value", "5"),
					resource.TestCheckResourceAttr("sapintegrationsuite_number_range.test", "description", "updated"),
					resource.TestCheckResourceAttrSet("sapintegrationsuite_number_range.test", "deployed_on"),
				),
			},
		},
	})
}
