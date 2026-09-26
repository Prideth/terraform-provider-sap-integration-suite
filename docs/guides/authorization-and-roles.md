---
page_title: "Authorization and Roles"
subcategory: ""
description: |-
  Which SAP role templates the OAuth clients behind this provider need for each resource
  family, how to assign them, and how to diagnose a 403 Forbidden.
---

# Authorization and Roles

The provider calls SAP's APIs with the OAuth clients you configure. What it may do is decided
entirely by the roles attached to those clients on SAP's side. A client without the right role
gets `403 Forbidden`, usually with an empty body, which is easy to mistake for a broken
provider. This guide lists the roles each resource family needs and shows how to check what a
client actually has.

## Where the roles come from

**Cloud Integration, Security Content, Partner Directory, access policies.** These APIs are
served by the `/api/v1` host of the Integration Suite tenant, and the provider's main `oauth`
block authenticates against them. The client comes from a service instance of *Process
Integration Runtime* with plan **`api`**. The roles are chosen when you create or update that
instance, in the *Roles* field of the instance parameters in the BTP cockpit, which SAP
recommends over entering JSON. They are role templates of the Cloud Integration application and
apply to every service key of the instance.

**Classic API Management.** The `api_management` block authenticates against the API Portal. Its
client comes from a service instance of *API Management, API portal* with plan
**`apiportal-apiaccess`**, and there the role is passed as an instance parameter.

A role change only shows up in tokens issued afterwards. For plan `api`, access tokens are valid
for 12 hours by default, so a token fetched before the change keeps the old scopes. Each
Terraform run fetches a fresh token, so a new run is enough.

## Creating the client with Terraform

The service instance and its key can themselves be managed with Terraform, using SAP's BTP and
Cloud Foundry providers. SAP's sample
[`trial_integration_suite`](https://github.com/SAP-samples/btp-terraform-samples/tree/main/released/usecases/trial_integration_suite)
(step `04_setup_oauth_clients`) entitles plan `api` of service `it-rt`, creates the instance with
the roles as instance parameters, and creates a service key for it. Adapted to what this
provider needs, the instance looks like this:

```hcl
resource "cloudfoundry_service_instance" "integration_suite_api" {
  name         = "terraform-integration-suite-api"
  type         = "managed"
  space        = var.space_id
  service_plan = data.cloudfoundry_service_plans.it_api.service_plans[0].id
  parameters = jsonencode({
    "roles"          = ["WorkspacePackagesEdit", "WorkspaceArtifactsDeploy", "MonitoringDataRead"]
    "grant-types"    = ["client_credentials"]
    "redirect-uris"  = []
    "token-validity" = 43200
  })
}

resource "cloudfoundry_service_credential_binding" "integration_suite_api" {
  type             = "key"
  name             = "terraform-integration-suite-api"
  service_instance = cloudfoundry_service_instance.integration_suite_api.id
}
```

Pick the roles from the table below. Two points from SAP's sample:

- Plan `api` only appears in the Cloud Foundry marketplace after the Integration Suite
  capabilities have been activated in the Integration Suite application, which is a manual
  step. The sample therefore splits the setup into several Terraform configurations.
- SAP notes that the key's details cannot be read back through the Cloud Foundry provider's data
  source ("This service does not support fetching service binding parameters"). Take the URL,
  token URL, client ID and secret from the key in the BTP cockpit, and pass the secret to this
  provider through a variable or your secret store, not through the configuration file.

## Roles per resource family

Cloud Integration client (plan `api`), based on SAP's *Tasks and Permissions for Cloud
Integration*:

| Resources and data sources | Role templates |
|---|---|
| Integration packages and the design-time artifacts in them (integration flows, message mappings, value mappings, script collections, integration adapters) | `WorkspacePackagesRead` to read, `WorkspacePackagesEdit` to change |
| `sapintegrationsuite_integration_flow_configuration` | `WorkspacePackagesConfigure` |
| All `*_deployment` resources | `WorkspaceArtifactsDeploy`, plus `MonitoringDataRead` to read the runtime status |
| `sapintegrationsuite_service_endpoints` | `MonitoringDataRead` (SAP's table has no row for service endpoints; this is the role for reading deployed artifacts) |
| User and OAuth2 credentials | `MonitoringDataRead` to read, `CredentialsEdit` to change |
| Certificates, key pairs, keystore entry data sources | `MonitoringDataRead` to read, `SecurityMaterialEdit` to change |
| Access policies, references, runtime assignments | `AccessPoliciesRead` to read, `AccessPoliciesEdit` to change |
| `sapintegrationsuite_number_range` | `MonitoringArtifactsDeploy` |
| Partner Directory resources and data sources | `AuthGroup_TenantPartnerDirectoryConfigurator` |
| `sapintegrationsuite_custom_tag_configuration` | `WebToolingSettingsProductProfiles.savetenantconfiguration`, which SAP's Integration Content page places in the `PI_Administrator` role collection |

Edge Integration Cell targeting (`runtime_location_id`) is not supported; see the Edge
Integration Cell guide.

Classic API Management client (plan `apiportal-apiaccess`):

| Purpose | Role |
|---|---|
| API providers, products, certificate store references, key value maps | `APIPortal.Administrator` (`APIPortal.Guest` is read-only) |
| Virtual host configuration (not implemented by this provider yet) | `APIManagement.SelfService.Administrator` |

API Composition client (`api_composition` block): the credentials come from a service key of an
*API Composition* service instance with plan **`configuration`**, which SAP names as the plan for
the Configuration API. SAP documents no role parameters for this plan. For people, the role
collection `Graph.KeyUser` (role `Graph_Key_User`) allows creating and changing business data
graphs, and `Graph.Guest` gives read-only access. See the [API Composition guide](api-composition.md).

`sapintegrationsuite_provider_features` and `sapintegrationsuite_provider_feature` need no
credentials at all.

## Least privilege

Give each client only what its configuration uses. A pipeline that only deploys content needs
`WorkspacePackagesRead`, `WorkspaceArtifactsDeploy` and `MonitoringDataRead`, not
`CredentialsEdit`. If one configuration manages content and another manages security material,
two clients with separate role sets keep the blast radius of a leaked secret small. Read-only
data sources work with the read roles alone.

## Diagnosing 403 Forbidden

A 403 means the token was valid but lacked a scope. To see which scopes a client has, decode
the payload (the second, base64url-encoded part) of an access token and look at `scope`. The
`client_id` claim confirms which service key the token came from. Typical causes:

- The role was added to a different service instance than the one the service key belongs to.
- A new service key was created but the configuration still uses the old client ID or secret.
- The token was issued before the role change. Fetch a new one.

For access policies, note that a policy restricts even administrators: artifacts protected by a
policy are only accessible with the policy's role or with `AccessAllAccessPoliciesArtifacts`.
See the [Access Policies guide](access-policies.md).
