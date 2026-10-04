# Needs enable_unofficial = true in the provider block, and Terraform 1.11
# or later for the write-only attributes.

variable "pgp_secret_key" {
  type        = string
  sensitive   = true
  description = "Output of gpg --armor --export-secret-keys <key ID>."
}

variable "pgp_passphrase" {
  type      = string
  sensitive = true
}

# The tenant's own key, for the PGP decryptor and the PGP signer. Neither
# the key nor the passphrase is stored in plan or state.
resource "sapintegrationsuite_pgp_secret_key" "tenant" {
  secret_key_wo         = var.pgp_secret_key
  passphrase_wo         = var.pgp_passphrase
  secret_key_wo_version = "1" # change it to send a new key
}
