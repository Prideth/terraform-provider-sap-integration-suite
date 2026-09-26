---
page_title: "Integration Content and Deployments"
subcategory: "Cloud Integration"
description: |-
  Managing integration flows, message mappings, script collections and value mappings from
  ZIP files, getting them to the runtime, and the behaviour of SAP's API that shapes how.
---

# Integration Content and Deployments

Integration flows, message mappings, script collections and value mappings are designed in the
Integration Suite UI and stored as ZIP archives. This provider uploads those archives into a
package and deploys them. It does not model a flow's steps as Terraform attributes: the ZIP is
the source of truth, kept in version control next to the configuration.

```terraform
resource "sapintegrationsuite_integration_package" "orders" {
  id         = "Orders"
  name       = "Orders"
  short_text = "Order processing"
}

resource "sapintegrationsuite_integration_flow" "order_intake" {
  package_id      = sapintegrationsuite_integration_package.orders.id
  flow_id         = "Order_Intake"
  name            = "Order Intake"
  content         = "${path.module}/content/Order_Intake.zip"
  content_hash    = filesha256("${path.module}/content/Order_Intake.zip")
  save_as_version = "1.0.4"
}

resource "sapintegrationsuite_integration_flow_deployment" "order_intake" {
  package_id   = sapintegrationsuite_integration_package.orders.id
  flow_id      = sapintegrationsuite_integration_flow.order_intake.flow_id
  flow_version = sapintegrationsuite_integration_flow.order_intake.version
}
```

The same pattern applies to `sapintegrationsuite_message_mapping`,
`sapintegrationsuite_script_collection` and `sapintegrationsuite_value_mapping`, each with its own
`*_deployment` resource.

## Getting the ZIP

The UI offers two ways to get content out of a tenant, and they produce different files:

- **Download** of a single artifact (or of a package) gives one folder per artifact, with
  `META-INF/MANIFEST.MF`, `src/main/resources/...` and `metainfo.prop`. Zip the *contents* of
  such a folder, not the folder itself, so that `META-INF/` sits at the top of the archive. This
  is the form to keep in version control: the files are readable and diff well.
- **Export** of a package gives a single archive with `resources.cnt` and one
  `<id>_content` file per artifact. Each `<id>_content` is already an artifact ZIP, but the
  export is meant for importing whole packages in the UI, not for this provider.

`content_hash` is the SHA-256 of the local file, normally `filesha256()` of the same path.
Terraform uploads the file again only when the hash changes. Files up to 32 MiB are accepted.

## The bundle ID

Every artifact ZIP names itself in `Bundle-SymbolicName` in `META-INF/MANIFEST.MF`. When an
artifact is created, SAP writes the artifact ID there, whatever the ZIP said. It then rejects
every content update whose `Bundle-SymbolicName` differs: an integration flow with
`400 Could not update artifact of the package; due to change in the Bundle-symbolicName`, a
message mapping with `500 BUNDLE_SYMBOLIC_NAME_CANNOT_BE_UPDATED` (tenant tests, September 2026).
A ZIP downloaded from one artifact and managed under another ID could therefore be created, but
never changed afterwards.

For integration flows and message mappings the provider handles this for you: it uploads a copy
whose `Bundle-SymbolicName` (and, for mappings, the name in `Provide-Capability`) equals
`flow_id` or `mapping_id`, and shows a warning when it had to change anything. Your file is not
touched, and `content_hash` still refers to it. To silence the warning, set the ID in the
manifest yourself:

```text
Bundle-SymbolicName: Order_Intake; singleton:=true
```

Manifest lines are at most 72 bytes; a longer value continues on the next line, which starts
with a single space. SAP's own exports wrap even `Bundle-SymbolicName` this way.

Script collections and value mappings are uploaded as they are. Keep their
`Bundle-SymbolicName` equal to the ID you give them in Terraform.

## Versions, and getting new content to the runtime

Uploading new content does **not** change an artifact's version: on a tenant, a flow created
with version 1.0.1 still reported 1.0.1 after its content was replaced. Deployments, however,
redeploy only when the version they point at changes. A new ZIP therefore reaches the runtime
only if something moves the version or triggers a redeploy:

- **`save_as_version`** (integration flows, message mappings, script collections) saves the
  current content under the version you name, for example `"1.0.4"`. Raise it together with
  the content; a version is saved only when this value changes. This is the clearest option,
  since the version in the UI then matches your release.
- **`redeploy_triggers`** on `sapintegrationsuite_integration_flow_deployment` redeploys in
  place whenever the map changes. Passing the content hash redeploys on every content change:

  ```terraform
  redeploy_triggers = {
    content = sapintegrationsuite_integration_flow.order_intake.content_hash
  }
  ```

- **Value mappings** have no `save_as_version`, and changing their content replaces the
  artifact. Their version comes from `Bundle-Version` in the ZIP (a value mapping created from a
  ZIP with `Bundle-Version: 1.0.1` reported version 1.0.1), so raise it with each change.

## Deploying

A deployment resource asks SAP to deploy one version and then polls the runtime status until it
is `STARTED` or `ERROR`, or the create timeout (10 minutes by default) runs out. Destroying it
undeploys the artifact.

**How long it takes varies.** On the same tenant an integration flow was running within about
ten seconds, while message mappings and value mappings sometimes stayed `STARTING` for several
minutes before starting. Raise `timeouts.create` if that is common on yours:

```terraform
  timeouts {
    create = "20m"
  }
```

**When the wait fails**, for example on a timeout or an `ERROR` status, the deployment is still
recorded in state with the last status SAP reported, and Terraform marks it *tainted*. The next
apply redeploys it, and `terraform destroy` undeploys it. Earlier versions dropped such a
deployment from state, which left it running on the tenant without Terraform knowing.

**Integration flows follow SAP's deploy task.** A flow deploy returns a task ID, and while the
flow does not yet appear in the runtime the resource reads the task's status. A failed task ends
the wait at once. A task that succeeded while the flow still does not appear ends it after a few
polls, with an error that names `SAP_ProfileId`: see the next section.

## The runtime profile

Tenants that also have Integration Cell give flows an externalized parameter `SAP_ProfileId`,
which picks the runtime the flow is deployed to. A flow uploaded from a ZIP started with
`integrationcell` on the tested tenant. SAP then reports the deployment as successful, but the
flow runs on Integration Cell, which this provider does not support, and never appears in the
Cloud Integration runtime. Set the parameter to `iflmap`, the value in the flow's own
`SAP-RuntimeProfile` manifest header, before deploying:

```terraform
resource "sapintegrationsuite_integration_flow_configuration" "order_intake" {
  flow_id      = sapintegrationsuite_integration_flow.order_intake.flow_id
  flow_version = sapintegrationsuite_integration_flow.order_intake.version
  parameters = {
    SAP_ProfileId = "iflmap"
  }
}

resource "sapintegrationsuite_integration_flow_deployment" "order_intake" {
  package_id        = sapintegrationsuite_integration_package.orders.id
  flow_id           = sapintegrationsuite_integration_flow.order_intake.flow_id
  flow_version      = sapintegrationsuite_integration_flow.order_intake.version
  redeploy_triggers = sapintegrationsuite_integration_flow_configuration.order_intake.parameters
}
```

See the [Integration Flow Configuration guide](integration-flow-configuration.md) for how
parameters relate to versions and redeployments.

## What is safe to deploy in a shared tenant

Deploying starts whatever the artifact does on its own. A flow with a timer start event, or
with a polling sender such as SFTP, a mail adapter or a data store consumer, runs as soon as it
is deployed and calls its receivers. A flow that only has an HTTPS sender runs only when someone
calls its endpoint. Mappings, script collections and value mappings do nothing by themselves.
Keep that in mind before deploying content copied from another landscape into a shared tenant.

## Import

- `sapintegrationsuite_integration_flow`: `terraform import sapintegrationsuite_integration_flow.order_intake Orders/Order_Intake`
  (`<package_id>/<flow_id>`); the same form applies to message mappings, script collections
  and value mappings.
- The deployment resources import by `<package_id>/<artifact_id>`.

SAP does not return a local file path, so `content` and `content_hash` stay empty after an
import until the next apply with a matching configuration.
