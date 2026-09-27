---
page_title: "Getting Started"
subcategory: "Getting Started"
description: |-
  From a service key to a deployed integration flow: provider setup, an integration package,
  an integration flow with externalized parameters, its deployment, and the endpoint URL.
---

# Getting Started

This guide takes one integration flow from a ZIP file on disk to a running deployment and
prints the URL it is reachable at. It uses only resources whose lifecycle SAP documents, so it
needs no opt-in switch.

## 1. Prerequisites outside Terraform (or in the SAP/btp provider)

This provider manages content inside a tenant. Before the first `terraform apply`, you need:

1. A subaccount with an SAP Integration Suite subscription and the **Cloud Integration**
   capability activated in the Integration Suite application. Activation is a manual step.
2. A service instance of **Process Integration Runtime** (`it-rt`) with plan **`api`**,
   grant type `client_credentials`, and these roles:

   | Role | Needed for |
   |---|---|
   | `WorkspacePackagesEdit` | packages and integration flows |
   | `WorkspacePackagesConfigure` | externalized parameters |
   | `WorkspaceArtifactsDeploy` | deployments |
   | `MonitoringDataRead` | deployment status and service endpoints |

3. A service key of that instance. Its `oauth` section holds the four values the provider
   needs.

Steps 2 and 3 can be done with the SAP/btp and Cloud Foundry providers; the
[Authorization and Roles guide](authorization-and-roles.md) shows the instance as Terraform
code and lists the roles for every other resource family.

## 2. Configure the provider

Put the service key values into environment variables, so they stay out of your files and out
of saved plans:

```shell
export SAP_INTEGRATION_SUITE_HOST="<oauth.url>"
export SAP_INTEGRATION_SUITE_TOKEN_URL="<oauth.tokenurl>"
export SAP_INTEGRATION_SUITE_CLIENT_ID="<oauth.clientid>"
export SAP_INTEGRATION_SUITE_CLIENT_SECRET="<oauth.clientsecret>"
```

```terraform
terraform {
  required_version = ">= 1.11"

  required_providers {
    sapintegrationsuite = {
      source  = "Prideth/sap-integration-suite"
      version = "~> 0.3.0"
    }
  }
}

provider "sapintegrationsuite" {}
```

## 3. Create an integration package

A package groups artifacts. `id` is the technical ID and cannot change without replacing the
package; `short_text` is required by SAP.

```terraform
resource "sapintegrationsuite_integration_package" "order_processing" {
  id          = "ORDER_PROCESSING"
  name        = "Order Processing"
  short_text  = "Order intake and confirmation"
  description = "Receives sales orders from the web shop and confirms them to the customer."
}
```

## 4. Upload the integration flow

The provider uploads an integration flow from a ZIP file: the format the Integration Suite UI
produces when you download an integration flow, or a flow from one of SAP's sample
repositories such as `SAP-samples/btp-spend-analysis`. Keep the ZIP in version control next to
the configuration.

```terraform
resource "sapintegrationsuite_integration_flow" "order_intake" {
  package_id = sapintegrationsuite_integration_package.order_processing.id
  flow_id    = "ORDER_INTAKE"
  name       = "Order Intake"

  content      = "${path.module}/iflows/order-intake.zip"
  content_hash = filesha256("${path.module}/iflows/order-intake.zip")

  save_as_version = "1.0.0"
}
```

`content_hash` is what Terraform compares: the file is uploaded again only when the hash
changes. SAP writes the flow ID into the ZIP's `Bundle-SymbolicName` and rejects later uploads
with a different one; the provider aligns it in the uploaded copy and warns you, your file is
not changed. `save_as_version` saves the upload under a version you name, which matters in the
next steps.

## 5. Set externalized parameters

Integration flows expose settings such as receiver addresses or credential aliases as
externalized parameters. Their keys are defined in the flow model; the UI shows them under
*Configure*. Set only the ones that differ per environment:

```terraform
resource "sapintegrationsuite_integration_flow_configuration" "order_intake" {
  flow_id      = sapintegrationsuite_integration_flow.order_intake.flow_id
  flow_version = sapintegrationsuite_integration_flow.order_intake.version

  parameters = {
    # Deploy to the Cloud Integration runtime, not to Integration Cell.
    SAP_ProfileId     = "iflmap"
    ERP_Receiver_Host = "s4hana.example.com"
  }
}
```

Keys that are not externalized by the flow are rejected before anything is written. Other
parameters keep their values. Destroying this resource stops managing the values but leaves
them in place, because SAP offers no way to delete a parameter.

Include `SAP_ProfileId` only if your flow has it; flows created for the Integration Cell do.

## 6. Deploy

Deployment is a separate resource, because design-time content and the runtime have separate
lifecycles in SAP: an upload changes the design-time artifact, and nothing runs until it is
deployed.

```terraform
resource "sapintegrationsuite_integration_flow_deployment" "order_intake" {
  package_id   = sapintegrationsuite_integration_package.order_processing.id
  flow_id      = sapintegrationsuite_integration_flow.order_intake.flow_id
  flow_version = sapintegrationsuite_integration_flow.order_intake.version

  # Parameters reach the runtime only through a redeploy, and uploading new
  # content keeps the version, so redeploy whenever either changes.
  redeploy_triggers = merge(
    sapintegrationsuite_integration_flow_configuration.order_intake.parameters,
    { content = sapintegrationsuite_integration_flow.order_intake.content_hash },
  )

  timeouts {
    create = "15m"
    update = "15m"
  }
}
```

The resource waits until SAP reports `STARTED`. If the flow ends in `ERROR` or does not start
before the timeout, it stays in state as tainted with SAP's error text, and the next apply
deploys it again.

## 7. Find the endpoint

Flows with an HTTP-based sender adapter get a service endpoint once they are deployed. List
the endpoints with the name SAP reports for each:

```terraform
data "sapintegrationsuite_service_endpoints" "all" {
  depends_on = [sapintegrationsuite_integration_flow_deployment.order_intake]
}

output "service_endpoint_urls" {
  value = {
    for endpoint in data.sapintegrationsuite_service_endpoints.all.endpoints :
    endpoint.name => [for entry_point in endpoint.entry_points : entry_point.url]
  }
}
```

Once you know the name SAP lists for your flow's endpoint, set it as `name` on the data source
to narrow the lookup to that endpoint. The `depends_on` is needed: the data source has no
reference to the deployment and would otherwise read before the flow is deployed.

## 8. Apply, change, destroy

Run `terraform init` and `terraform apply`. For a new release of the flow, replace the ZIP and
raise `save_as_version`; the next apply uploads the content, saves the version and redeploys.

`terraform destroy` undeploys the flow, deletes the flow and the package, and leaves the
externalized parameter values alone. To take over content that already exists on the tenant
instead of creating it, see [Importing Existing Content](importing-existing-content.md).

## Next steps

- [Integration Content and Deployments](integration-content.md): bundle IDs, versions, deploy
  times and which content is safe to deploy.
- [Integration Flow Configuration](integration-flow-configuration.md): externalized parameters
  in detail.
- [Security Content](security-content.md): credentials and certificates that flows reference.
- [Partner Directory](partner-directory.md): partner-specific parameters for B2B scenarios.
