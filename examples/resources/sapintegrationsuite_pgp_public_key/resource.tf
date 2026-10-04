# Needs enable_unofficial = true in the provider block.

# A partner's public key, for the PGP encryptor of an integration flow.
# Export it with: gpg --armor --export <key ID>
resource "sapintegrationsuite_pgp_public_key" "partner" {
  public_key = file("${path.module}/keys/partner-public.asc")
}

# Integration flows reference the key by its key ID or user ID.
output "partner_key_id" {
  value = sapintegrationsuite_pgp_public_key.partner.key_id
}
