# Needs enable_unofficial = true in the provider block: the CSR and the
# chain upload are known only from the tenant $metadata.

variable "ca_cert_pem" {
  type        = string
  description = "PEM certificate of the issuing certificate authority."
}

variable "ca_private_key_pem" {
  type      = string
  sensitive = true
}

resource "sapintegrationsuite_key_pair" "client_auth" {
  alias       = "client-auth-key"
  key_type    = "RSA"
  key_size    = 2048
  common_name = "client.example.com"
  country     = "DE"
}

# Sign the key pair's certificate signing request. The hashicorp/tls
# provider stands in for a certificate authority here; in practice the
# signed certificate usually comes from your PKI or a partner's CA.
resource "tls_locally_signed_cert" "client_auth" {
  cert_request_pem      = sapintegrationsuite_key_pair.client_auth.certificate_signing_request
  ca_cert_pem           = var.ca_cert_pem
  ca_private_key_pem    = var.ca_private_key_pem
  validity_period_hours = 8760
  allowed_uses          = ["digital_signature", "key_encipherment", "client_auth"]

  # A regenerated key pair has a new CSR. tls updates a changed
  # cert_request_pem in place without signing it again, so sign anew
  # whenever the key pair is replaced.
  lifecycle {
    replace_triggered_by = [sapintegrationsuite_key_pair.client_auth]
  }
}

# The signed certificate with the certificate of its issuer. When the key
# pair is replaced, the new CSR is signed and the chain uploaded again in
# the same apply.
resource "sapintegrationsuite_key_pair_certificate_chain" "client_auth" {
  key_pair_alias    = sapintegrationsuite_key_pair.client_auth.alias
  certificate_chain = join("", [tls_locally_signed_cert.client_auth.cert_pem, var.ca_cert_pem])
}
