---
page_title: "Security Content"
subcategory: "Security"
description: |-
  What SAP Integration Suite Security Content (security material) is, which artifact types this
  provider manages — credentials, keystore entries, certificates, and SAP-generated key pairs —
  its write-only secret model, alias encoding, SAP-owned entry handling, and why several
  documented artifact types are deliberately not implemented.
---

# Security Content

SAP Cloud Integration's **Security Content** area (also called **Security Material** in the
tenant UI, under *Monitor* > *Manage Security* > *Security Material*) holds the credentials,
certificates, and keys integration flow adapters use for outbound and inbound authentication. It
is backed by a public OData V2 API, documented on SAP Business Accelerator Hub as the "Security
Content" API package, sharing the same `/api/v1` host as the Cloud Integration content APIs this
provider already manages.

This guide covers what this provider implements today, the security model behind every secret
attribute, and — just as importantly — which documented Security Content artifact types this
provider deliberately does not implement yet, and why. See `internal/features/catalog.go` (the
`security.*` entries) for the single source of truth this guide is generated from, and
`docs/feature-support.md` for the full, auto-generated support matrix.

## Implemented: User Credentials and OAuth2 Client Credentials

- [`sapintegrationsuite_user_credential`](../resources/user_credential.md) /
  [`data.sapintegrationsuite_user_credential`](../data-sources/user_credential.md) — a
  username/password credential (SAP's "User Credentials" artifact type), for outbound basic or
  username-token authentication, with optional SuccessFactors/OpenConnectors system binding.
- [`sapintegrationsuite_oauth2_client_credential`](../resources/oauth2_client_credential.md) /
  [`data.sapintegrationsuite_oauth2_client_credential`](../data-sources/oauth2_client_credential.md)
  — a client ID/client secret/token service URL credential (SAP's "OAuth2 Client Credentials"
  artifact type), for the OAuth2 client credentials grant (RFC 6749) on outbound requests.

Both follow the same shape: an identity (`id`, the artifact's name/alias), readable metadata
(`description`, `user`/`client_id`, and so on), and a **write-only** secret plus a plain version
marker that drives rotation.

## The write-only secret model

Sensitive != not stored. Terraform's ordinary `Sensitive: true` attribute flag only masks a
value in CLI output and logs — the value is still written to the Terraform state file and to
plan files, in the clear, for anyone with file access to read. For a credential's password or an
OAuth2 client secret, that is not an acceptable place for the value to live.

Instead, both resources use Terraform's **write-only attributes**
(`password_wo`/`password_wo_version` and `client_secret_wo`/`client_secret_wo_version`):

- The `_wo` attribute is never persisted to plan or state. Terraform sends it to the provider
  only at apply time, from configuration, and the provider must not — and in this provider's
  case, structurally cannot, since the Go client types have no field to decode one into — copy it
  into anything that gets written back to state.
- The paired `_wo_version` attribute *is* stored in state, as an ordinary string. It carries no
  secret material; it exists purely so Terraform has something to compare between plans. Changing
  it is how you tell Terraform "the secret changed, redeploy this credential" — Terraform cannot
  infer that from the write-only value itself, since state never remembers what it was.

This is why every example in this guide bumps `password_wo_version`/`client_secret_wo_version`
whenever `password_wo`/`client_secret_wo` changes, and why a config that keeps referencing the
same `var.something` with the version left unchanged does **not** re-send the secret on every
apply beyond what SAP's own "re-enter the secret on every edit" requirement already forces.

**Requires Terraform CLI 1.11 or later.** Write-only attributes are a Terraform Core / provider
protocol feature that only ships from Terraform CLI 1.11 onward. This provider does not enforce a
`required_version` constraint itself (Terraform module authors set that in their own
configuration), but both write-only resources will fail to plan on an older Terraform CLI. If
your organization pins an older Terraform version, either upgrade it or do not use
`sapintegrationsuite_user_credential`/`sapintegrationsuite_oauth2_client_credential` yet.

## Rotation is an in-place update, not a replacement

SAP's Manage Security Material UI documents an explicit **Edit** action for Credentials
artifacts ("You can also edit and redeploy an existing artifact"), and states that a secret must
be re-entered on every edit. This provider follows that lifecycle: changing
`password_wo_version`/`client_secret_wo_version` (or any other mutable attribute) triggers
Terraform's Update, which this provider implements as a full `PUT` redeploy — not a delete/create
replacement. Only `id` (the artifact's name/alias) and, for user credentials, `kind` (which
system-specific sub-type the credential is) force replacement, since SAP does not document
changing either of those via Edit.

## OAuth2 token request settings

Many token services need more than a client ID and secret. SAP's OAuth2 Client Credentials
artifact covers that with four settings, which the tenant `$metadata` names
`ClientAuthentication`, `ScopeContentType`, `Resource` and `Audience`. The resource exposes
them as `client_authentication`, `scope_content_type`, `resource` and `audience`:

```terraform
resource "sapintegrationsuite_oauth2_client_credential" "graph" {
  id                = "MS_GRAPH_OAUTH"
  token_service_url = "https://login.example.invalid/tenant-id/oauth2/v2.0/token"
  client_id         = "00000000-0000-0000-0000-000000000000"
  scope             = "https://graph.example.invalid/.default"
  resource          = "https://graph.example.invalid"

  client_secret_wo         = var.graph_client_secret
  client_secret_wo_version = "1"
}
```

Two details shape how they behave:

- SAP does not document the constants the API expects. The UI offers *Send as Body
  Parameter* (the default) and *Send as Request Header* for client authentication, but the
  stored values may differ from those labels. The provider passes whatever you write through
  unchanged. The easiest way to learn the right value is to set it once in the UI and read it
  back with the data source.
- An update is a `PUT`, and in OData V2 a `PUT` replaces the whole entity. If the provider only
  sent the attributes in your configuration, every rotation would wipe settings someone made
  in the UI. The four attributes are therefore *optional and computed*: when you leave one out,
  Terraform keeps the value SAP currently holds and sends it back on every update. The flip
  side is that you cannot clear a value from Terraform by deleting the attribute. Clear it in
  the UI instead.

Two parts of the UI dialog are not covered. The grant-type placement (URL or body) has no
property in the API at all. The **custom parameters** table (up to 20 key/value pairs sent in
the body, header or URL) is modeled in `$metadata` as a `CustomParameters` navigation property.
A tenant check of 2026-10-04 settled how SAP treats it: the parameters can only be created
together with the credential, every `PUT` that does not send them deletes them, and a `PUT` that
does is refused. Every rotation through Terraform therefore removes custom parameters maintained
in the UI. Since 0.5.1 the plan warns when a credential that is about to be updated or replaced
has custom parameters. Do not combine custom parameters with Terraform-managed rotation for the
same credential, or set them again in the UI after every apply.

## Drift detection is limited for secrets

Terraform can never detect that a password or client secret changed outside Terraform: SAP's
Security Content API does not return secret values on `GET`, by design, and this provider's Go
client types have no field to receive one even if a future API version did. `Read` only compares
the metadata SAP does return (username, description, token service URL, scope, and so on).
Treat a stored credential's secret as **owned by whoever last set `_wo_version`** — Terraform,
your CI pipeline, or a human editing it directly in the SAP UI — and rotate deliberately by
bumping the version, rather than expecting `terraform plan` to notice an externally-rotated
secret.

## Importing an existing credential

```shell
terraform import sapintegrationsuite_user_credential.backend BACKEND_BASIC
terraform import sapintegrationsuite_oauth2_client_credential.backend BACKEND_OAUTH
```

Import recovers `id` and every readable metadata field. It cannot recover the secret — SAP never
returns one, and this provider's state model has no field for one regardless. Until you add
`password_wo`/`client_secret_wo` and a `_wo_version` to your configuration, Terraform has no
opinion on the secret at all: the imported resource plans cleanly with no password/secret
drift. The first time you add both to your configuration (to take ownership of rotation), that
plans as an ordinary in-place Update — not a replacement — and redeploys the credential with the
password/secret you supplied. This is a deliberate design, not an accident: it means importing a
credential never silently rotates its secret, and taking ownership of rotation is always an
explicit, visible step in a plan.

## Implemented, unofficially: Secure Parameters

**Status: unofficial.** The resource works and passed its acceptance test on a tenant, but SAP
does not document the API it uses: the `SecureParameters` entity set is known only from the
service's `$metadata`. SAP may change it without notice, so the provider marks it `unofficial`
rather than `supported` until SAP documents it.

[`sapintegrationsuite_secure_parameter`](../resources/secure_parameter.md) manages SAP's
"Secure Parameter" artifact: a confidential value stored under an alias, which custom adapters
and scripts read at run time (in a Groovy script through the `SecureStoreService`).

```hcl
resource "sapintegrationsuite_secure_parameter" "custom_adapter_api_key" {
  id          = "CUSTOM_ADAPTER_API_KEY"
  description = "API key of the custom adapter's backend"

  secure_param_wo         = var.custom_adapter_api_key
  secure_param_wo_version = "1"
}
```

SAP Help documents this artifact only in the Monitor UI (*Security Material > Add > Secure
Parameter*); its Security Content API page does not list it. The provider relies on two other
sources:

- The tenant `$metadata` defines `SecureParameters` with the key `Name` (up to 150 characters),
  `Description` (up to 1024), `SecureParam` (up to 4096, matching the UI's Cloud Foundry limit),
  `DeployedBy`, `DeployedOn` and `Status`.
- A tenant test in September 2026 created a secure parameter (`POST`), read it by name, changed
  it with `PUT` and deleted it. Every write answered `202 Accepted` without a body; reads return
  `SecureParam` as `null` and `Status` as `DEPLOYED`.

The value follows the same write-only model as the credentials above: `secure_param_wo` is never
stored, and changing `secure_param_wo_version` redeploys the artifact with the new value in place.
The value is sent on every update, as the UI also asks for it on every edit. Create stops if a
secure parameter with that name already exists, because SAP does not document what a create on
an existing name does; import it instead. After an import, the first apply sends the configured
value, since the stored one cannot be read.

## Implemented: Keystore Entries, Certificates, and SAP-generated Key Pairs

The tenant keystore (*Monitor* > *Manage Security* > *Keystore*) holds certificates and key
pairs, distinct from the credentials above. This provider manages it through three types:

- [`data.sapintegrationsuite_keystore_entry`](../data-sources/keystore_entry.md) /
  [`data.sapintegrationsuite_keystore_entries`](../data-sources/keystore_entries.md) — read-only
  discovery of any keystore entry by alias, or of every entry in the tenant keystore. Besides
  alias, key type, size and validity, they return what SAP stores about the certificate: subject
  and issuer DN, serial number, signature algorithm, SHA-1/256/512 fingerprints, owner, status,
  and who created and last changed the entry. Validity and timestamps are converted to RFC 3339,
  which makes expiry checks straightforward:

  ```terraform
  data "sapintegrationsuite_keystore_entries" "all" {}

  locals {
    expiring_within_30_days = [
      for e in data.sapintegrationsuite_keystore_entries.all.entries : e.alias
      if e.valid_not_after != null &&
      timecmp(e.valid_not_after, timeadd(plantimestamp(), "720h")) < 0
    ]
  }
  ```
- [`sapintegrationsuite_certificate`](../resources/certificate.md) — manages a standalone X.509
  certificate (for example a partner's or CA's public certificate you trust).
- [`sapintegrationsuite_key_pair`](../resources/key_pair.md) — generates an SAP-managed key pair;
  the private key is created and retained by SAP and never enters this provider at all.

There is deliberately **no generic `sapintegrationsuite_keystore_entry` resource**: that entity
represents fundamentally different object types with different lifecycles, so a mutable resource
covering all of them would either be too vague to be safe or would have to reimplement
Certificate's and Key Pair's logic behind one confusing interface. Use the specific resource for
what you are actually managing.

### Alias encoding — you never touch it

Every keystore entry's real OData identity is a lowercase hex encoding of its alias's UTF-8
bytes (SAP's own documentation states this explicitly, and explains why: "the server doesn't
allow slashes or backslashes in a URI, even if they're percent encoded"). You never compute or
supply this yourself — every resource and data source here takes a plain `alias` and encodes it
internally (`internal/client/odata/v2/hexkey.go`), the same shared helper Partner Directory's
`AlternativePartners` key encoding was refactored to use once this provider confirmed both
follow the identical rule. It is asserted against every alias SAP's own documentation uses as a
worked example, plus Unicode, punctuation, semicolons, slashes, and backslashes.

### `sapintegrationsuite_certificate`

```hcl
resource "sapintegrationsuite_certificate" "backend_ca" {
  alias       = "backend-root-ca"
  certificate = file("${path.module}/backend-root-ca.pem")
}
```

`certificate` is PEM content and is **never marked `Sensitive`** — a public X.509 certificate is
not a secret, and conflating it with private key material (which this resource never handles at
all) would be a mistake. Create and Update both use `PUT
CertificateResources('<hexalias>')/$value` with the PEM as body, which SAP's own documentation
notes creates a new entity despite the PUT verb.

**Configuring a certificate means trusting it.** When you import a self-signed or otherwise
untrusted certificate in the UI, SAP shows its fingerprint and asks you to confirm it. The API
does the same: on a tenant in September 2026, the plain request answered a self-signed
certificate with `409` and the status `notImported`. The provider therefore always sends
`fingerprintVerified=true`, so the certificate in your configuration is imported as it is.
Check `certificate_sha256` against the fingerprint your partner gave you, ideally before the
first apply (for example with `openssl x509 -noout -fingerprint -sha256 -in cert.pem`).

**Replacing a certificate keeps its alias.** Changing `certificate` updates the entry in place
with `update=true`; without it SAP refuses with `400 Entry with alias ... already exists`. A
new alias, in contrast, replaces the resource. Creating a certificate under an alias that
already exists on the tenant fails for the same reason; import that entry instead
(`terraform import sapintegrationsuite_certificate.backend_ca backend-root-ca`).

**Drift detection compares a canonical fingerprint, not raw PEM text.** Two PEM encodings of the
identical certificate can differ in line endings, wrapping, or a trailing newline without
representing any real change. On every `Read`, this provider parses both the certificate
currently on the tenant and whatever is already in your Terraform state using Go's own
`crypto/x509`, and only replaces your state's PEM text with SAP's own re-serialization when the
SHA-256 fingerprint of the certificate's actual DER bytes has genuinely changed. `subject_dn`,
`issuer_dn`, `serial_number`, and `certificate_sha256` are all derived this same way — locally,
from certificate bytes this provider can confirmedly retrieve — never from a guessed SAP field
name (SAP's own documented example response for a keystore entry is truncated before these
properties are shown).

Delete uses SAP's documented keystore mass-deletion operation
(`KeystoreResources('system')?deleteEntries=true`) with **exactly the one alias this resource
owns** — never a caller-assembled list, so a `terraform destroy` here can never accidentally
reach an unrelated alias.

### `sapintegrationsuite_key_pair`

```hcl
resource "sapintegrationsuite_key_pair" "client_auth" {
  alias        = "client-auth-key"
  key_type     = "RSA"
  key_size     = 2048
  common_name  = "client.example.com"
  country      = "DE"
  organization = "Example GmbH"
}
```

SAP generates the private key internally; **it is never downloadable through this API, and this
resource has no field for it anywhere** — not write-only, not sensitive, simply absent, because
no code path in this provider ever asks SAP for one.

`key_type` is one of `RSA` (default), `DSA`, `EC` — SAP's complete, fixed enum. `key_size` is
required for `RSA`/`DSA`; for `EC`, either `key_size` (112–571) or `key_algorithm_parameter` (a
named curve, for example `secp256r1`) is required — enforced at plan time, since the rule is
conditional on `key_type`. `signature_algorithm`, if set, is validated against the exact
SAP-documented enum for your chosen `key_type` (RSA/DSA/EC each have their own list).

**No update operation is documented** for a generated key pair's material — every attribute that
defines it (`key_type`, `key_size`, the subject DN fields, validity dates, and so on) is
`RequiresReplace`. Changing any of them generates an entirely new key pair under the same alias
lifecycle (replace), never an in-place mutation of existing key material.

**Read covers only part of the configuration.** `KeystoreEntries` returns `key_type`,
`key_size` and the validity period, and according to the tenant `$metadata` also the signature
algorithm and the subject DN as one combined string. It does not return the individual subject
fields (common name, organization and so on) or `key_algorithm_parameter` that the resource is
configured with. The provider refreshes what it can read on every plan and trusts the rest from
the last successful write, which is why this resource's support status is `partial`. It is not
a statement that anything about it is unsafe to use.

**Importing a key pair.** `terraform import sapintegrationsuite_key_pair.signing <alias>` reads
the entry and fills the subject fields (`common_name`, `organization_unit`, `organization`,
`locality`, `state`, `country`, `email`) from the combined subject DN SAP returns. This matters
because those fields are `RequiresReplace`: without them, the first apply after an import would
see every configured subject field as new and generate a fresh key pair, throwing away the one
you imported. `signature_algorithm` and `key_algorithm_parameter` cannot be read back, so leave
them out of the configuration of an imported key pair, or accept that setting them plans a
replacement.

**There is no separate "SSH Key" resource.** SAP's own Security Content API overview lists no
independent SSH Key entity, and the tenant keystore UI's own "Creating a Key Pair/SSH Key Pair"
documentation uses the identical field set for both — "Create > Key Pair" and "Create > SSH Key"
are the same underlying mechanism with a different label. An RSA or DSA
`sapintegrationsuite_key_pair`'s public key is available in OpenSSH format directly:

```hcl
output "client_auth_ssh_public_key" {
  value = sapintegrationsuite_key_pair.client_auth.public_key_openssh
}
```

`public_key_openssh` is populated via SAP's confirmed
`KeystoreEntries('<hexalias>')/Sshkey/$value` export whenever `key_type` is `RSA` or `DSA` (SAP
documents EC as unsupported for this export); it stays `null` for an EC key pair.

Delete uses the same single-alias mass-deletion mechanism as `sapintegrationsuite_certificate`.

### SAP-owned keystore entries

A tenant keystore typically contains entries the tenant administrator owns and entries SAP owns
(for example SAP's own root certificates). The keystore data sources show this in the `owner`
attribute, which comes from the entry's `Owner` property in the API. SAP does not document the
values `owner` can take, so the resources do not use it to refuse operations in advance.
`sapintegrationsuite_certificate` and `sapintegrationsuite_key_pair` attempt the operation you
asked for, and SAP's server-side protection rejects an Update or Delete against a protected
entry with an ordinary API error, which the provider passes on to you. Before importing an alias
into either resource, check its `owner` with the data source. That is the practical way to avoid
adopting an SAP-owned entry by mistake.

### Whole-keystore management stays out of scope

SAP's `KeystoreResources` entity also supports importing an entire keystore (`POST
KeystoreResources`, a base64-encoded JKS/JCEKS file plus password) and backing up/restoring sets
of entries. This provider deliberately does not implement a `sapintegrationsuite_keystore`
resource around that operation, confirmed contract or not: a single import call can create,
update, leave unchanged, or remove many entries at once, based entirely on the uploaded file's
contents, with no way for this provider to know whether any given affected entry belongs to a
different Terraform module, a different administrator, or SAP itself. This is the same
blast-radius concern this provider already avoids for the confirmed mass-delete operation, which
is why Certificate and Key Pair only ever call it with exactly the one alias they own, never an
open-ended list. If you need whole-keystore import/export/backup, use the SAP Integration Suite
UI.

## Deliberately not implemented

SAP documents more Security Content artifact types than this provider manages. The reasons
differ, and the difference matters when you plan around them. Each item is recorded in
`internal/features/catalog.go`.

**SAP's API exists, but its contract is incomplete.** For these, a tenant's `$metadata` shows
the entities, but SAP documents neither the requests nor which operations are allowed, and
each one involves secret or key material where a wrong guess is costly:

- **Certificate Chain** (`security.certificate_chain`). `CertificateChainResources` is a media
  entity per key pair alias, and `ChainCertificates` lists the chain's certificates. The media
  type and request for uploading a chain are undocumented. The intended shape is a resource
  scoped to one key pair.
- **PGP keyrings** (`security.pgp_keyring`). Public and secret keyrings, keys, subkeys and user
  IDs all have entity sets, with no documented requests. A secret keyring is private key
  material, so this needs a confirmed upload format and write-only handling first.
- **OAuth2 custom parameters**. See [OAuth2 token request settings](#oauth2-token-request-settings).

**SAP offers no API.** The tenant `$metadata` of `/api/v1` has no entity for these; they
exist only in the Security Material UI:

- **OAuth2 Password Credentials** (`security.oauth2_password_credential`), new in 2026.
- **OAuth2 SAML Bearer Assertion** (`security.oauth2_saml_bearer`).
- **Known Hosts (SSH)** (`security.known_hosts`).
- **Where-used** for security material (`security.where_used`). If an API appears, this would
  become a read-only data source, never managed state.
- **Certificate-to-User Mapping** (`security.certificate_user_mapping`), which only exists in
  the Neo environment.

**The API exists, but the object does not fit Terraform.** An **OAuth2 Authorization Code**
artifact has a full entity in the API, including refresh token handling. Using it requires a
person to complete an authorization in the browser, and the refresh token is a secret SAP obtains
and keeps. Terraform's non-interactive plan/apply model has no place for that step, so this
stays out of scope on purpose.

For any of these, use the SAP Integration Suite UI. The provider does not guess at an
unconfirmed contract for security-sensitive artifacts.

## Security notes

- Passwords and client secrets never appear in this provider's logs, error diagnostics, or debug
  output: the Go client types for both credential resources structurally have no field to decode
  a secret into, even from a response body that happened to contain one.
- `sapintegrationsuite_key_pair` never requests, stores, or exposes a private key: no code path
  in this provider calls an operation that could return one, and no field exists anywhere in the
  schema or the Go client types to receive one.
- Error diagnostics for certificate and key pair operations identify the alias and SAP's own
  status/error, never a full certificate dump or the request/response body verbatim — a public
  certificate is not confidential, but there is no reason to put its full content in an error
  message either.
- Every acceptance test for these resources uses unmistakably synthetic secrets/certificates and
  a `tf-acc-`-prefixed alias, never a real credential or a customer certificate, and never
  touches a pre-existing tenant artifact (`sap_*`, `hcicertificate*`, or any alias not created by
  the test itself).
