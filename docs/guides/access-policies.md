---
page_title: "Access Policies"
subcategory: "Security"
description: |-
  How SAP Integration Suite access policies work, how they connect to BTP roles, which parts
  this provider manages, and what happens to them across the Terraform lifecycle.
---

# Access Policies

Out of the box, anyone with the right Integration Suite role collection can work with every
integration artifact on a tenant. An **access policy** narrows that down. It names a role and
lists *artifact references*, which are rules like "the integration flow called `Metering`" or
"every artifact in package `UTILITIES`". From then on, only users who hold that role can
change the matched artifacts, operate on them once deployed, or read the data they process at
runtime: message processing log attachments, trace payloads, data store entries, variables
and queue content.

A few properties of the mechanism are worth knowing before you automate it:

- Protection covers the UI and the public APIs alike. A technical user calling the
  Integration Content API is subject to the same policies as a person in the browser.
- An unauthorized user can still *see* that a protected artifact exists. The policy restricts
  operations and data, not visibility of the artifact list.
- Custom header properties in message processing logs are never protected. SAP expects them
  not to carry sensitive data.
- When several policies match the same artifact, holding the role of *any one* of them is
  enough. A package-level policy therefore overrides a stricter artifact-level policy for
  anyone who holds the package role.

## How a policy grants access

The policy itself does not know which users exist. The link is made on the BTP side. You
create a role in the subaccount from the `CustomRoleTemplate` role template of application
`it`, set its `custom_role` attribute to *Static*, and put the policy's `role_name` into
**Values**. You then add that role to a role collection and assign the collection to users or
groups. SAP Help describes this in *Creating Custom Roles for Access Policies*. Newly assigned
roles only take effect after the user logs in again, so a freshly granted user may still see
"access denied" for a while.

That BTP half is deliberately out of this provider's reach. Roles, role collections and their
assignments are subaccount objects, and the [SAP/btp](https://registry.terraform.io/providers/SAP/btp/latest)
provider manages them. This provider treats `role_name` as an opaque string. SAP accepts a
policy whose role does not exist yet, and nothing breaks until someone expects to get in, so
keep the two configurations side by side and pass the same string to both.

## The two resources

A policy and its references are separate entities in SAP's API. Each reference has its own
numeric ID and is created and deleted on its own, so the provider models them as two
resources rather than a nested block:

```terraform
resource "sapintegrationsuite_access_policy" "utilities" {
  role_name   = "UTILITIES_ARCHITECT"
  description = "Integration flows owned by the utilities architecture team"
}

resource "sapintegrationsuite_access_policy_reference" "metering_flow" {
  access_policy_id = sapintegrationsuite_access_policy.utilities.id

  name          = "Metering flow"
  artifact_type = "INTEGRATION_FLOW"
  attribute     = "Name"
  operator      = "exactString"
  value         = "Metering"
}
```

The separation matters in brownfield setups. You can attach your own references to a policy
another team manages, which you look up with the `sapintegrationsuite_access_policy` data
source, without ever claiming the policy itself. Terraform only ever deletes the reference
IDs it created or imported. References that someone added through the UI stay untouched.

## Where the reference values come from

Each attribute of `sapintegrationsuite_access_policy_reference` maps directly onto a property
of SAP's `ArtifactReferences` entity:

| Terraform attribute | SAP property | Label in the UI | Values |
|---|---|---|---|
| `name` | `Name` | Name (mandatory) | free text |
| `description` | `Description` | Description | free text |
| `artifact_type` | `Type` | Artifact Type | SAP's constant, for example `INTEGRATION_FLOW` or `INTEGRATION_PACKAGE` |
| `attribute` | `ConditionAttribute` | Attribute | `Name` or `ID` |
| `operator` | `ConditionType` | Operator | `exactString` (*Equals*) or `regularExpression` (*Matches*) |
| `value` | `ConditionValue` | Value / Expression | exact name or ID, or a Java regular expression |

The values are SAP's wire values, not the UI labels: the operator *Matches* is stored as
`regularExpression`, not `MATCHES`. (With the provider's `convert_ui_labels` and
`enable_experimental`, the proven labels such as *Matches* and *Integration Flow* are converted;
see the resource page.) The provider accepts only values that SAP is known to
store and combinations SAP allows, and fails the plan for anything else, so a wrong value can
no longer leave a policy without its references halfway through an apply. The complete table
of artifact types, with the evidence for each, is on the
[`sapintegrationsuite_access_policy_reference`](../resources/access_policy_reference.md) page.
Two rules from SAP Help are part of it: *Matches* is not available for Integration Package
references, and message queues, global variables and global data stores can only be matched
by name.

References created in the UI with other values can still be read with the data source and
imported:

```terraform
data "sapintegrationsuite_access_policy_reference" "created_in_ui" {
  access_policy_id = "1901"
  reference_id     = "56"
}
```

The `artifact_type`, `attribute` and `operator` attributes return what SAP actually stored.

For regular expressions, `value` must be a valid `java.util.regex.Pattern`, not a glob.
`UTIL_.*` matches everything starting with `UTIL_`. `UTIL_*` is also valid Java syntax, but it
means "`UTIL` followed by any number of underscores"; the plan warns about such patterns.

## Lifecycle

**Create.** The policy is created with a POST carrying `RoleName` and `Description`. SAP
returns a numeric ID, which becomes the resource `id`. Each reference is then created with its
own POST that points back at the policy. Terraform orders these correctly as long as the
reference uses `sapintegrationsuite_access_policy.<name>.id`.

A policy `description` may contain at most 200 characters. SAP keeps only the first 200 of a
longer one, which used to fail the apply after the policy had been created and left it without
references; since 0.3.1 the provider rejects such a description while planning. Use the
description for a one-line summary and leave the artifact scope to the references.

SAP shortens the other strings of a policy the same way, without an error. A tenant probe
(October 2026) found these limits, and since 0.3.2 the provider checks all of them while
planning:

| Attribute | SAP keeps |
|---|---|
| policy `role_name` | 200 characters |
| policy `description` | 200 characters |
| reference `name` | 50 characters |
| reference `description` | 200 characters |
| reference `value` | 150 characters |

SAP counts characters, not bytes: 200 umlauts are kept. A character outside the Basic
Multilingual Plane, such as an emoji, counts as two, because how SAP counts it was not tested.

**Update.** Changing a policy's `description` is an in-place update: the provider sends a
`PATCH` with only `Description`. A tenant test (September 2026) settled the method. `PUT`
answers `501 Not Implemented`, with or without `Id` in the body, even though SAP's CI/CD
upload action contains a `PUT` branch; `PATCH` and `MERGE` answer `204` and the new
description is read back. SAP's own update action sidesteps the question by deleting and
recreating the policy. Changing `role_name` replaces the policy, because the role name is the
policy's identity and renaming it in place was not tested. Replacing a policy
has a knock-on effect: SAP deletes a policy's references along with it, and because the new
policy has a new ID, Terraform also replaces every reference that points at it. Expect the
plan to show all of them.

References have no in-place update at all. The UI can edit a reference, but no public API
contract for doing so has been confirmed. Every change to a reference, including a new
description, deletes the old reference and creates a new one. In the moment between the two
calls the artifacts that reference matched are not protected by it. If that gap matters to
you, add `lifecycle { create_before_destroy = true }` to the reference, and give the
replacement a different `name` in case SAP enforces unique reference names within a policy.
SAP does not document whether it does.

**Delete.** Destroying a policy deletes it and, on SAP's side, all its references. When a
reference resource is destroyed afterwards, SAP answers *not found* and the provider treats
the reference as already gone. This makes `terraform destroy` order-independent.

**Drift.** On every refresh the provider reads the policy by ID and the reference through its
policy's reference list. Reading through the list also confirms that the reference still
belongs to that policy. A policy or reference deleted outside Terraform drops out of state,
and the next plan recreates it. A reference edited in the UI shows up as a diff on the edited
attributes, which Terraform resolves by replacing it.

**Import.** Both resources import by numeric SAP ID:

```shell
terraform import sapintegrationsuite_access_policy.utilities 1901
terraform import sapintegrationsuite_access_policy_reference.metering_flow 1901/55
```

IDs differ between tenants, so do not copy them from one landscape to another. The
`sapintegrationsuite_access_policy` data source can look a policy up by `role_name`, which is
unique within a tenant and the same everywhere. That is the lookup SAP's own transport
tooling relies on too.

## Runtimes and replication

Since Edge Integration Cell arrived, a policy can be replicated to several runtimes: the
Cloud Integration runtime, Integration Cell, and individual Edge Integration Cell
deployments. In the UI you pick the runtimes when you create a policy and can change them
later. A per-runtime reconciliation status then reports *Pending* until an offline runtime
has picked the policy up, and *Success* or *Fail* after that.

On the API side, each policy has a navigation property `AccessPolicyRuntimeAssignments`. The
service's `$metadata` describes one assignment as an `Id`, a `RuntimeLocationId` naming the
runtime, a `TransferStatus`, `TransferErrors` and a `StatusUpdatedAt` timestamp. The
`sapintegrationsuite_access_policy_runtime_assignments` data source reads exactly that, so
you can see from Terraform where a policy has arrived:

```terraform
data "sapintegrationsuite_access_policy_runtime_assignments" "utilities" {
  access_policy_id = sapintegrationsuite_access_policy.utilities.id
}
```

It is read-only on purpose. Nothing documents whether assignments can be created or removed
through the API, and `$metadata` does not say either, so choosing the runtimes stays a UI step.
That also means SAP decides which runtimes a policy created through the API lands on. After
the first apply, check the data source or the *Runtimes* column in the Access Policies screen,
and adjust the selection in the UI if needed. `transfer_status` is passed through unchanged:
the UI shows Fail, Success and Pending, but SAP does not document the values the API uses.

Because the status changes on SAP's side without any Terraform action, do not use the data
source as a gate inside the same apply that creates the policy. Read it in a separate run, or
in monitoring that polls it.

Earlier releases exposed a `reconciliation_status` attribute on the policy. It was removed
because the policy entity has no such property; the status lives with the runtime
assignments. Removing it does not disturb existing state.

## Upgrading from releases before the contract correction

Releases up to and including v0.1.0 sent invented property names (`ArtifactType`,
`Attribute`, `Operator`, `Value`) and quoted string keys. SAP's API expects the properties
above and numeric `Edm.Int64` keys, so creating a reference could not have succeeded against a
real tenant. To move to the corrected schema:

1. Add a `name` to every `sapintegrationsuite_access_policy_reference`.
2. Replace `artifact_type = "IntegrationFlow"` with `"INTEGRATION_FLOW"`, and
   `operator = "EQUALS"` with `"exactString"`. The provider rejects the old values at plan
   time with a message that points to the new ones.
3. For other old values (`"MATCHES"`, `"ODataAPI"` and so on), look up the real constants
   with the data source as described above.
4. If your state somehow holds a reference resource from the old release, remove it with
   `terraform state rm` and let Terraform create it again.

Policies themselves were created with the right properties before as well. Existing policy
resources keep working without changes.

## Further reading

- SAP Help: *Access Policies*, *Defining Access Policies*, *Access Policies Examples*,
  *Creating Custom Roles for Access Policies* and *Manage Access Policies for Edge Integration
  Cell*.
- SAP Business Accelerator Hub: *Security Content* API, resource *Access Policies*.
- `docs/sap-api-references.md` in this repository records the evidence behind each statement
  in this guide, including where SAP's documentation is silent.
