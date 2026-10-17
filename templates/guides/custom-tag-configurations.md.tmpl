---
page_title: "Custom Tag Configurations"
subcategory: "Cloud Integration"
description: |-
  What SAP's tenant-wide Custom Tag Configuration is, how this provider owns and manages it, and
  why destroying it is deliberately not supported.
---

# Custom Tag Configurations

SAP Cloud Integration lets a tenant administrator define a set of **custom tags** — attributes
every integration package owner classifies their package with, some mandatory, each optionally
restricted to a closed list of permitted values. This guide covers what
[`sapintegrationsuite_custom_tag_configuration`](../resources/custom_tag_configuration.md) and
[`data.sapintegrationsuite_custom_tag_configuration`](../data-sources/custom_tag_configuration.md)
do, and is explicit throughout about which claims are confirmed by SAP's own documentation,
which are this provider's own design decisions, and which remain genuinely unverified.

## Tenant-wide scope, not a per-package object

Unlike an integration package, an integration flow, or every other design-time artifact this
provider manages, a custom tag configuration is not an independent object you create many of.
SAP documents exactly **one** configuration per tenant, addressed by a fixed, confirmed key:
`CustomTags`. `sapintegrationsuite_custom_tag_configuration` is therefore a **singleton
resource** — declare it once per tenant (or import the tenant's existing configuration into it),
and manage the tenant's complete tag list from that one resource block, the same way you would
manage any other tenant-wide setting.

## Tag values on packages

The configuration only defines which tags exist. The values a package has for them are a
separate, per-package setting:
[`sapintegrationsuite_integration_package_custom_tag`](../resources/integration_package_custom_tag.md)
sets the values of one tag on one package, and
[`data.sapintegrationsuite_integration_package_custom_tags`](../data-sources/integration_package_custom_tags.md)
lists every tag of a package with its values. A typical split is that the tenant administrator
owns the configuration, and each team sets the tags of its own packages:

```terraform
resource "sapintegrationsuite_integration_package_custom_tag" "owner" {
  package_id = sapintegrationsuite_integration_package.orders.id
  name       = "Owner"
  values     = ["Finance"]

  depends_on = [sapintegrationsuite_custom_tag_configuration.governance]
}
```

A tenant probe (October 2026) showed how the two interact: every package lists every configured
tag, a tag without a value with an empty one, and a value for a tag the configuration does not
have is refused with `400`, which is why the `depends_on` above makes Terraform define a new tag
before it sets values for it. Removing a tag from the configuration removes its values from all
packages; a package tag resource for it then disappears from the state on the next refresh.

## Confirmed API contract

Everything below is confirmed directly from SAP's own "Create New Custom Tags Configuration" and
"Get Custom Tags Defined on the Tenant" Help Portal pages — including the exact example payloads,
which this provider's client-layer tests assert against byte-for-byte.

- **Create/Update**: `POST /api/v1/CustomTagConfigurations`, body
  `{"CustomTagsConfigurationContent": "<base64-encoded JSON>"}`, where the decoded JSON is
  `{"customTagsConfiguration": [{"tagName": "...", "isMandatory": true|false,
  "permittedValues": [...]}]}`. Add the query parameter `Overwrite=true` "if a custom tags
  configuration is already available on the tenant."
- **Read**: `GET /api/v1/CustomTagConfigurations('CustomTags')/$value`, returning the same
  decoded JSON shape directly (this is OData's `$value` raw-media-stream convention — the
  response is **not** wrapped in the standard `{"d": {...}}` envelope every other entity in this
  API uses).

This provider's Create and Update both always send `Overwrite=true`, on the very first write and
every one after: SAP's documentation never shows what a plain `POST` does once a configuration
already exists (a conflict is plausible, mirroring the confirmed duplicate-ID-is-an-error
behavior for custom Integration Adapters — see `docs/guides/integration-adapters.md` — but this
is not confirmed either way for this entity), while `Overwrite=true` is documented to work once
something exists and nothing suggests it behaves differently against an empty tenant. Always
using it avoids depending on an unconfirmed distinction.

## What "Overwrite" is understood to mean

The request body you send is not a delta — it is always documented as "the following JSON
content" representing the complete configuration, and `Overwrite=true` is the flag that lets you
resend it once something already exists. This provider therefore treats every write as a
**full replace**: tags present in a previous configuration but absent from your new one are
understood to be removed. SAP's documentation never uses the word "replace" explicitly, so this
is this provider's own reasonable reading of "Overwrite," not an independently confirmed fact —
flagged here rather than asserted with more confidence than the evidence supports.

## Required role

SAP's documentation states that maintaining custom tags requires the role template
`WebToolingSettingsProductProfiles.savetenantconfiguration`, which happens to be part of the
`PI_Administrator` role collection in the Cloud Foundry environment (and part of
`AuthGroup.Administrator` in Neo). The role **template** is what the API actually checks; a
custom, narrower role collection that includes only that template should work equally well; you
do not need the entire `PI_Administrator` role collection specifically, only whichever role
collection your tenant assigns that template through.

## Ownership model: semantic fields, not raw payloads

Terraform configuration works with plain, typed fields — `name`, `mandatory`,
`permitted_values` — never the raw base64-encoded payload SAP's API actually expects. This
provider serializes and base64-encodes the configuration internally; you never construct or read
that encoding yourself.

```hcl
resource "sapintegrationsuite_custom_tag_configuration" "governance" {
  tags = [
    {
      name      = "Owner"
      mandatory = true
    },
    {
      name             = "BusinessUnit"
      mandatory        = true
      permitted_values = ["Network", "Sales", "Water", "Transport"]
    },
  ]
}
```

## Ordering is not significant — `tags` and `permitted_values` are sets

SAP's documentation never states that the order tags or permitted values are submitted or
returned in carries any meaning. To guarantee `terraform plan` never reports a spurious change
purely because SAP happened to return the same tags in a different order than you wrote them,
both `tags` and each tag's `permitted_values` are modeled as **Terraform sets**, not lists: two
configurations with the same tags in a different order are the same configuration as far as
Terraform is concerned. The request body this provider actually sends to SAP is separately
canonicalized (sorted by tag name, and each tag's permitted values sorted alphabetically) before
encoding, so the exact bytes sent are also deterministic and independent of input order — useful
for auditing exactly what a `terraform apply` will transmit.

One consequence of using a set for `permitted_values`: an exact duplicate value is not
representable. SAP's documentation does not confirm whether duplicate permitted values are
meaningful or even accepted, so this provider does not allow configuring one in the first place.

`tags` also cannot contain two entries with the same `name` — enforced by this resource's own
validation, independent of the Terraform set mechanics (two tags with the same name but
different `mandatory`/`permitted_values` would otherwise be two distinct, and contradictory, set
elements). SAP's documentation does not state whether tag names must be unique, but a
configuration with two conflicting definitions for the same name is inherently ambiguous, and
this provider would have no principled way to decide which one SAP should end up storing.

## A documentation quirk worth knowing about

One of SAP's own documented example responses shows a tag with permitted values `Mr. Bean` and
`Ms. Bean` rendered as a *single* array element containing a comma-separated string
(`"permittedValues":["Mr. Bean, Ms. Bean"]`) rather than two separate array elements. Every other
array-typed field in SAP's Security/Integration Content APIs — and everywhere else in this
provider — uses one array element per value, and the surrounding prose introduces "two examples,"
so this is treated as a documentation authoring artifact, not a confirmed wire format. If your
tenant's actual behavior differs, please report it.

## Destroying this resource is not supported

This is the most important limitation to understand before adopting this resource. SAP documents
**no delete or clear operation** for `CustomTagConfigurations` anywhere — unlike essentially
every other resource this provider manages, there is no confirmed `DELETE`, and no documented
"send an empty configuration to clear everything" behavior either.

Given that, `terraform destroy` (or removing the resource block and applying) returns an
explicit, actionable error rather than one of the two unsafe alternatives this provider
categorically avoids:

- **Silently making Delete a no-op** — Terraform would believe the tag configuration is gone
  while the tenant still enforces it, a dangerous mismatch between state and reality.
- **Guessing that posting an empty configuration deletes everything** — plausible, but
  unconfirmed; if wrong, it could silently do something SAP interprets differently than
  intended, against tenant-wide governance configuration that affects every integration package
  owner.

If you need to stop managing this configuration with Terraform without changing anything on the
tenant, run `terraform state rm sapintegrationsuite_custom_tag_configuration.<name>` instead of
`terraform destroy`. If you actually want to clear all tags, do so through the SAP Integration
Suite UI, or — only if you have independently confirmed the behavior against your own tenant —
apply a configuration with `tags = []` (an empty set), which this provider will send through the
same confirmed `Overwrite=true` write path used for every other update, exercising the same
"full replace" semantics described above, without this provider itself claiming to know what SAP
does with an empty list.

## Import

Import uses the same fixed key SAP documents for reading the configuration:

```shell
terraform import sapintegrationsuite_custom_tag_configuration.governance CustomTags
```

Any other import ID is rejected outright, rather than silently accepted and then failing later
against an API that does not recognize it.

## Drift detection

`Read` re-fetches the complete configuration on every plan and refresh, so any tag added,
removed, or changed outside Terraform (through the UI, or by another automation) shows up as
drift on the next `terraform plan` — with one exception: this provider cannot distinguish "SAP
returned the tags in a different order" from "someone changed something," which is exactly why
ordering is treated as insignificant (see above) rather than as a potential source of drift.

## Brownfield adoption

If a tenant already has a custom tag configuration (created through the UI, or by a previous
process), import it:

```shell
terraform import sapintegrationsuite_custom_tag_configuration.governance CustomTags
```

then write a `sapintegrationsuite_custom_tag_configuration` resource block matching what was
imported (`terraform plan` will show the difference if your configuration does not match) before
running `terraform apply` — the same brownfield workflow as every other resource in this
provider, with the added note from the section above that there is no way to walk this back via
`terraform destroy` once adopted.
