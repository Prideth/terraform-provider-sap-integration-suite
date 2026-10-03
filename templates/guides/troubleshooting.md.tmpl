---
page_title: "Troubleshooting"
subcategory: "Getting Started"
description: |-
  Common errors when running the provider against SAP Integration Suite, what causes them and
  how to resolve them.
---

# Troubleshooting

Errors from SAP carry the HTTP status, SAP's error code and SAP's message. The message usually
names the field or rule that failed, so read it before anything else. The provider logs no
requests, and no error contains a token or secret.

## Authentication and authorization

**`401 Unauthorized` on every request.** The token URL, client ID or secret is wrong, or the
client belongs to another service. The `oauth` block needs a key of Process Integration Runtime
plan `api`; the API portal key of plan `apiportal-apiaccess` only works in `api_management`.
Check which environment variables are set, because they fill in every attribute that the
provider block leaves empty.

**`403 Forbidden`, often with an empty body.** The token is valid but lacks a role. Decode the
token's payload and look at `scope`; the
[Authorization and Roles guide](authorization-and-roles.md) lists the role each resource
family needs. Roles added to a service instance only appear in tokens issued afterwards, which
the next Terraform run fetches.

**An artifact is missing although it exists.** An access policy can hide artifacts even from
administrators. Artifacts it protects are only visible with the policy's role or
`AccessAllAccessPoliciesArtifacts`; see the [Access Policies guide](access-policies.md).

**`context canceled` on the token URL.** Provider 0.1.0. Upgrade to 0.2.0 or later.

## Provider configuration

**`... is experimental` or `... is unofficial`.** The resource, or one operation of it, needs
`enable_experimental = true` or `enable_unofficial = true`. The error names what it refused;
the resource page explains why.

**`Incomplete Classic API Management configuration`.** The `api_management` block, or its
`SAP_INTEGRATION_SUITE_API_MANAGEMENT_*` environment variables, has some but not all of host,
token URL, client ID and secret. Set all four or none. The same applies to `api_composition`.

**`Incomplete API Composition user login`.** `api_composition` has `username` without
`password`, or the reverse, or `origin` without both. Set username and password together.

**Business data graph: HTTP 403, code 2707.** The token of `api_composition` lacks the
Configuration API's scope. Add the login of a user with the role collection `Graph.KeyUser`; see
"Logging in as a key user" in the [API Composition guide](api-composition.md). A token error
`invalid_grant` "User authentication failed." means the user login itself failed: check the
password, the origin, and whether the identity provider enforces a second factor.

**Write-only attributes are rejected.** Terraform older than 1.11 cannot handle `*_wo`
attributes. Upgrade Terraform; the provider itself does not check the version.

## Content

**`Could not update artifact of the package; due to change in the Bundle-symbolicName`** or
**`BUNDLE_SYMBOLIC_NAME_CANNOT_BE_UPDATED`.** SAP writes the artifact ID into
`Bundle-SymbolicName` in `META-INF/MANIFEST.MF` when the artifact is created and rejects later
uploads with another value. The provider aligns integration flows and message mappings itself
and warns you. For script collections and value mappings, set `Bundle-SymbolicName` in the ZIP
to the ID you use in Terraform.

**A new ZIP was uploaded, but the runtime still runs the old content.** Uploading does not
change an artifact's version, and deployments only redeploy when their version changes. Raise
`save_as_version` together with the content, or pass the artifact's `content_hash` to the
deployment's `redeploy_triggers`. See the [Integration Content guide](integration-content.md).

**`Property 'ShortText' cannot be empty`.** Integration packages need `short_text`.

**A create fails because the object already exists.** Several resources refuse to create over
an existing object, because SAP does not document what a create over an existing name does.
Import the object instead; see [Importing Existing Content](importing-existing-content.md).

## Deployments

**The deployment times out and the resource is tainted.** SAP did not report `STARTED` within
`timeouts.create`. Deploy times on a tenant ranged from seconds to several minutes, and value
and message mappings sometimes stayed `STARTING` for minutes. Raise the timeout. The tainted
resource keeps SAP's last status; the next apply undeploys and deploys again.

**The deploy task succeeded, but the flow never appears.** The flow was deployed to another
runtime. A flow whose externalized parameter `SAP_ProfileId` is `integrationcell` goes to the
Integration Cell. Set it to `iflmap` with
[`sapintegrationsuite_integration_flow_configuration`](../resources/integration_flow_configuration.md)
and redeploy.

**The deployment ends in `ERROR`.** The error contains SAP's text from the runtime, for example
a missing credential alias. Deploy the referenced security material, value mappings and script
collections first; SAP does not deploy referenced artifacts automatically, and Terraform only
orders them if the configuration references them or uses `depends_on`.

## Security material

**`409` or `notImported` when adding a certificate.** Fixed in 0.2.0: the provider confirms
the fingerprint (`fingerprintVerified=true`). Compare `certificate_sha256` with the fingerprint
you expect before applying; listing a certificate is the decision to trust it.

**A new secret was configured but not sent.** Terraform cannot see changes to write-only
values. Change the matching `*_wo_version` to send the new secret.

## Classic API Management

**`405` when changing an API product.** SAP does not allow updating API products. The provider
replaces the product instead, which drops the subscriptions of applications using it.

**An API provider is not found right after it was created.** SAP caches reads for about 20
seconds. The provider waits after a create; a read shortly after an import can still miss it,
and the next run finds it.
