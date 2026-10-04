# SAP generates the private key internally; it is never downloadable
# through this API and this resource has no field for it. Every attribute
# that defines the generated key material is RequiresReplace, since SAP
# documents no operation to update a generated key pair in place.
resource "sapintegrationsuite_key_pair" "client_auth" {
  alias        = "client-auth-key"
  key_type     = "RSA"
  key_size     = 2048
  common_name  = "client.example.com"
  country      = "DE"
  organization = "Example GmbH"
}

# An RSA or DSA key pair's public key can be exported in OpenSSH format
# without a separate resource - SAP documents no independent "SSH Key"
# object.
output "client_auth_ssh_public_key" {
  value = sapintegrationsuite_key_pair.client_auth.public_key_openssh
}

# An Elliptic Curve key pair, selected by named curve instead of key_size.
resource "sapintegrationsuite_key_pair" "ec_signing" {
  alias                   = "ec-signing-key"
  key_type                = "EC"
  key_algorithm_parameter = "secp256r1"
  signature_algorithm     = "SHA-256/ECDSA"
  common_name             = "signing.example.com"
  country                 = "DE"
}

# With enable_unofficial = true the key pair also offers a certificate
# signing request; see sapintegrationsuite_key_pair_certificate_chain.
output "client_auth_csr" {
  value = sapintegrationsuite_key_pair.client_auth.certificate_signing_request
}
