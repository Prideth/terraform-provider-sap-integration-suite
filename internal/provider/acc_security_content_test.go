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
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

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

// Custom parameters exist only from the create on (tenant check of
// 2026-10-04): a change of a credential with custom parameters replaces it,
// and the parameters are there again afterwards; an import reads them.
func TestAccOAuth2ClientCredential_customParameters(t *testing.T) {
	accgate.Require(t, accgate.SecurityContent)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "true")
	name := testAccName()
	config := func(description, params string) string {
		return fmt.Sprintf(`
resource "sapintegrationsuite_oauth2_client_credential" "test" {
  id                       = %[1]q
  description              = %[2]q
  token_service_url        = "https://tfacc.invalid/oauth/token"
  client_id                = "tfacc-client"
  client_secret_wo         = "tfacc-not-a-real-secret"
  client_secret_wo_version = "1"
  %[3]s
}
`, name, description, params)
	}
	params := `custom_parameters = [
    { key = "resource", value = "https://tfacc.invalid", send_as_part_of = "body" },
    { key = "x-tfacc", value = "1", send_as_part_of = "header" },
  ]`
	addr := "sapintegrationsuite_oauth2_client_credential.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("created", params), Check: resource.TestCheckResourceAttr(addr, "custom_parameters.#", "2")},
			{
				Config: config("changed", params),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(addr, plancheck.ResourceActionReplace),
				}},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "description", "changed"),
					resource.TestCheckResourceAttr(addr, "custom_parameters.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(addr, "custom_parameters.*", map[string]string{"key": "x-tfacc", "send_as_part_of": "header"}),
				),
			},
			{
				ResourceName:            addr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"client_secret_wo_version"},
			},
			{
				Config: config("changed", ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(addr, plancheck.ResourceActionReplace),
				}},
				Check: resource.TestCheckNoResourceAttr(addr, "custom_parameters.#"),
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

// testAccKeyPairChainConfig is a key pair whose CSR a throw-away CA from
// the hashicorp/tls provider signs, and the chain uploaded to it.
func testAccKeyPairChainConfig(alias, commonName string) string {
	return fmt.Sprintf(`
resource "tls_private_key" "ca" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

resource "tls_self_signed_cert" "ca" {
  private_key_pem       = tls_private_key.ca.private_key_pem
  is_ca_certificate     = true
  validity_period_hours = 48
  allowed_uses          = ["cert_signing", "crl_signing"]
  subject {
    common_name = "tfacc-ca"
  }
}

resource "sapintegrationsuite_key_pair" "test" {
  alias       = %[1]q
  common_name = %[2]q
  country     = "DE"
  key_size    = 2048
}

resource "tls_locally_signed_cert" "test" {
  cert_request_pem      = sapintegrationsuite_key_pair.test.certificate_signing_request
  ca_private_key_pem    = tls_private_key.ca.private_key_pem
  ca_cert_pem           = tls_self_signed_cert.ca.cert_pem
  validity_period_hours = 24
  allowed_uses          = ["digital_signature", "key_encipherment", "client_auth"]

  # tls updates a changed cert_request_pem in place without signing again.
  lifecycle {
    replace_triggered_by = [sapintegrationsuite_key_pair.test]
  }
}

resource "sapintegrationsuite_key_pair_certificate_chain" "test" {
  key_pair_alias    = sapintegrationsuite_key_pair.test.alias
  certificate_chain = join("", [tls_locally_signed_cert.test.cert_pem, tls_self_signed_cert.ca.cert_pem])
}
`, alias, commonName)
}

// The whole CA flow: the key pair's CSR is signed and the chain uploaded,
// read back and imported; regenerating the key pair signs its new CSR and
// uploads the new chain in the same apply.
func TestAccKeyPairCertificateChain_signedByCA(t *testing.T) {
	accgate.Require(t, accgate.SecurityContent)
	t.Setenv("SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL", "true")
	alias := testAccName()
	chain := "sapintegrationsuite_key_pair_certificate_chain.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders:        map[string]resource.ExternalProvider{"tls": {Source: "hashicorp/tls", VersionConstraint: "~> 4.0"}},
		Steps: []resource.TestStep{
			{
				Config: testAccKeyPairChainConfig(alias, "tfacc-chain"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("sapintegrationsuite_key_pair.test", "certificate_signing_request"),
					resource.TestCheckResourceAttr(chain, "certificates.#", "2"),
					resource.TestCheckResourceAttr(chain, "certificates.0.issuer_dn", "CN=tfacc-ca"),
					resource.TestCheckResourceAttr(chain, "certificates.1.subject_dn", "CN=tfacc-ca"),
				),
			},
			{
				ResourceName:            chain,
				ImportState:             true,
				ImportStateId:           alias,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"certificate_chain"},
			},
			{
				// A new subject regenerates the key pair; its new CSR is signed
				// and the chain uploaded again.
				Config: testAccKeyPairChainConfig(alias, "tfacc-chain-renewed"),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("sapintegrationsuite_key_pair.test", plancheck.ResourceActionDestroyBeforeCreate),
					plancheck.ExpectUnknownValue("sapintegrationsuite_key_pair.test", tfjsonpath.New("certificate_signing_request")),
					plancheck.ExpectResourceAction("tls_locally_signed_cert.test", plancheck.ResourceActionReplace),
					plancheck.ExpectResourceAction("sapintegrationsuite_key_pair_certificate_chain.test", plancheck.ResourceActionUpdate),
				}},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sapintegrationsuite_key_pair.test", "common_name", "tfacc-chain-renewed"),
					resource.TestMatchResourceAttr(chain, "certificates.0.subject_dn", regexp.MustCompile(`CN=tfacc-chain-renewed`)),
					resource.TestCheckResourceAttr(chain, "certificates.0.issuer_dn", "CN=tfacc-ca"),
				),
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
