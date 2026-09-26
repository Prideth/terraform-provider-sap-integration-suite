---
page_title: "Integration Flow Configuration"
subcategory: "Cloud Integration"
description: |-
  Setting externalized parameters of integration flows per environment, how the values relate
  to flow versions and deployments, and what happens on import and destroy.
---

# Integration Flow Configuration

An integration flow usually contains values that differ between landscapes: the receiver's
host name, the name of the credential used for it, a batch size, a polling interval. Flow
designers *externalize* these values, which turns them into named parameters with a default.
The flow content then stays identical from development to production, and each environment
only sets its own parameter values. In the UI this is the *Configure* action of an integration
flow.

`sapintegrationsuite_integration_flow_configuration` sets those values from Terraform:

```terraform
resource "sapintegrationsuite_integration_flow_configuration" "metering" {
  flow_id      = sapintegrationsuite_integration_flow.metering.flow_id
  flow_version = sapintegrationsuite_integration_flow.metering.version

  parameters = {
    receiver_host       = "meter-gateway.prod.example.invalid"
    receiver_credential = "METER_GATEWAY_OAUTH"
    batch_size          = "250"
  }
}
```

## What the resource owns

The resource owns exactly the keys listed in `parameters`. Other externalized parameters of the
flow keep whatever value they have, so you can manage the environment-specific parameters in
Terraform and leave defaults alone. Values are always strings. The provider reads each
parameter's current data type (for example `xsd:integer`) and sends it back unchanged, so
`"250"` stays an integer parameter.

Before writing anything, the provider checks that every key exists in the flow version. A typo
does not produce a half-applied configuration. The plan fails with an error that names the
unknown key and lists the parameters the flow actually has.

The API behind this is documented by SAP: parameters are listed through the `Configurations`
of `IntegrationDesigntimeArtifacts(Id,Version)` and changed with
`PUT …/$links/Configurations('<key>')`, the same call SAP's own Piper library uses.

## Versions

Parameters belong to a design-time version of the flow, so `flow_version` is required. Use the
`version` attribute of the `sapintegrationsuite_integration_flow` resource. This also gives
Terraform the right order: the content is uploaded first, then the parameters are set. When the
flow gets a new version, Terraform updates this resource in place and writes every managed
parameter into the new version.

SAP does not document whether uploading new content for the *same* version keeps parameter
values. If it resets them, the next `terraform plan` shows the managed keys as drift, and
applying writes them again.

## Redeploying

A running flow only picks up new parameter values when it is deployed again. The deployment
resource redeploys when `flow_version` changes, but a parameter change does not change the
version. Pass the parameters to the deployment's `redeploy_triggers`:

```terraform
resource "sapintegrationsuite_integration_flow_deployment" "metering" {
  package_id   = sapintegrationsuite_integration_package.utilities.id
  flow_id      = sapintegrationsuite_integration_flow.metering.flow_id
  flow_version = sapintegrationsuite_integration_flow.metering.version

  redeploy_triggers = sapintegrationsuite_integration_flow_configuration.metering.parameters
}
```

Any change to the map redeploys the flow in place. There is no undeploy, so the flow keeps
running the old configuration until the new deployment is started.

## The runtime profile (`SAP_ProfileId`)

Tenants that also have Integration Cell give flows an externalized parameter `SAP_ProfileId`,
which picks the runtime the flow is deployed to. On a tenant in September 2026, a flow
created from a ZIP had `SAP_ProfileId = "integrationcell"`: SAP accepted the deployment and
reported the task as `SUCCESS`, but the flow never appeared in the Cloud Integration runtime,
because it went to Integration Cell, which this provider does not support. With the value
`iflmap` (the `SAP-RuntimeProfile` in the flow's own MANIFEST) the same flow was running at once.

Set the parameter before deploying:

```terraform
resource "sapintegrationsuite_integration_flow_configuration" "metering" {
  flow_id      = sapintegrationsuite_integration_flow.metering.flow_id
  flow_version = sapintegrationsuite_integration_flow.metering.version
  parameters = {
    SAP_ProfileId = "iflmap"
  }
}
```

and make the deployment depend on it (for example through `redeploy_triggers` as above). If
a flow still lands on another runtime, `sapintegrationsuite_integration_flow_deployment` stops
with an error naming `SAP_ProfileId` as soon as SAP reports the deployment task as done,
instead of waiting until its timeout.

## Drift, import and destroy

On refresh, the provider reads the current values of the managed keys. A value changed in the
UI shows up as a diff and is set back on apply. A key that has disappeared from the flow (for
example because the new content no longer externalizes it) drops out of state, and the next
apply fails with the unknown-key error until you remove it from the configuration.

Import takes `<flow_id>/<flow_version>` and adopts every parameter the version currently has:

```shell
terraform import sapintegrationsuite_integration_flow_configuration.metering metering/1.0.3
```

If your configuration lists fewer keys, the first plan shows the others being removed from
`parameters`. Applying that makes no API call. Terraform just stops managing those keys.

Destroying the resource does not change the flow. SAP has no operation to delete or reset
an externalized parameter, since parameters come from the flow model. The provider therefore
removes the resource from state and prints a warning that the values stay as they are.

## Related

- [`sapintegrationsuite_integration_flow`](../resources/integration_flow.md) for the content.
- [`sapintegrationsuite_integration_flow_deployment`](../resources/integration_flow_deployment.md)
  for the runtime deployment.
- Parameters that name security material (credentials, key aliases) only store the name. Manage
  the artifact itself with the resources described in the [Security Content guide](security-content.md).
