---
page_title: "sapintegrationsuite_access_policy_reference Resource - sapintegrationsuite"
subcategory: "Security"
description: |-
  One artifact reference of an access policy: which artifacts, by type and by name or ID, the
  policy protects. Replace-only.
---

# sapintegrationsuite_access_policy_reference (Resource)

A reference is one rule of an access policy: artifacts of a type whose name or ID matches a
value, exactly or as a regular expression. A policy usually has several. Each reference is its
own entity with an ID that SAP assigns, which is why it is a separate resource and not a block of
[`sapintegrationsuite_access_policy`](access_policy.md).

**Status:** supported. The wire contract (`Name`, `Type`, `ConditionAttribute`, `ConditionType`,
`ConditionValue`) comes from SAP's own access policy automation; the accepted values are listed
below with their evidence.

## Prerequisites

- The access policy, usually from [`sapintegrationsuite_access_policy`](access_policy.md).
- An OAuth client with `AccessPoliciesEdit`.

## Lifecycle

| Operation | What happens |
|---|---|
| Create | Creates the reference in the policy. SAP assigns its ID; the provider finds it by content when SAP answers without a body. |
| Read | Reads the reference with SAP's constants. |
| Update | None. |
| Replacement | Any change deletes the reference and creates it again. In between, the matched artifacts are briefly not protected by it. |
| Delete | Deletes the reference. |

## Values: SAP wire values, not UI labels

`artifact_type`, `attribute` and `operator` take the values SAP's API stores, which differ from
the labels in the UI. The provider checks them while Terraform plans and sends them unchanged; it
never converts a label into a wire value.

| UI | Attribute | Wire value |
|---|---|---|
| Operator *Equals* | `operator` | `exactString` |
| Operator *Matches* | `operator` | `regularExpression` |
| Attribute *Name* | `attribute` | `Name` |
| Attribute *ID* | `attribute` | `ID` |

Do not use `EQUALS`, `MATCHES`, `IntegrationFlow` or other spellings. The plan fails for any value
outside the lists below and names the right value where it is clear, for example
`regularExpression` for `MATCHES`. Earlier releases accepted any non-empty string, so a wrong value
only failed when SAP rejected it during apply, after the policy had been created.

### Supported combinations

| `artifact_type` | UI artifact type | `attribute` | `operator` | Status |
|---|---|---|---|---|
| `INTEGRATION_FLOW` | Integration Flow | `Name`, `ID` | `exactString`, `regularExpression` | supported |
| `INTEGRATION_PACKAGE` | Integration Package | `Name`, `ID` | `exactString` only | supported |
| `API_ARTIFACT` | API | `Name`, `ID` | `exactString`, `regularExpression` | supported |
| `ODATA_SERVICE` | OData API | `Name`, `ID` | `exactString`, `regularExpression` | supported |
| `REST_API_PROVIDER` | REST API | `Name`, `ID` | `exactString`, `regularExpression` | supported |
| `SOAP_API_PROVIDER` | SOAP API | `Name`, `ID` | `exactString`, `regularExpression` | supported |
| `SCRIPT_COLLECTION` | Script Collection | `Name`, `ID` | `exactString`, `regularExpression` | supported |
| `VALUE_MAPPING` | Value Mapping | `Name`, `ID` | `exactString`, `regularExpression` | supported |
| `MESSAGE_MAPPING` | Message Mapping | `Name`, `ID` | `exactString`, `regularExpression` | supported |
| `DATA_TYPE` | Data Type | `Name`, `ID` | `exactString`, `regularExpression` | supported |
| `MESSAGE_TYPE` | Message Type | `Name`, `ID` | `exactString`, `regularExpression` | supported |
| `MESSAGE_QUEUE` | Message Queue | `Name` only | `exactString`, `regularExpression` | supported |
| `GLOBAL_DATA_STORE` | Global Data Store | `Name` only | `exactString`, `regularExpression` | supported |
| `GLOBAL_VARIABLE` | Global Variable | `Name` only | `exactString`, `regularExpression` | supported |
| `AUTH2_AUTHORIZATION_CODE` | not offered | `Name`, `ID` | `exactString`, `regularExpression` | unofficial |
| `AUTH2_SAML_BEARER_ASSERTION` | not offered | `Name`, `ID` | `exactString`, `regularExpression` | unofficial |
| `FAULT_MESSAGE_TYPE` | not offered | `Name`, `ID` | `exactString`, `regularExpression` | unofficial |
| `INTEGRATION_ADAPTER` | not offered | `Name`, `ID` | `exactString`, `regularExpression` | unofficial |
| `OAUTH2_CLIENT_CRED` | not offered | `Name`, `ID` | `exactString`, `regularExpression` | unofficial |
| `SECURE_PARAMETER` | not offered | `Name`, `ID` | `exactString`, `regularExpression` | unofficial |
| `SERVICE_INTERFACE` | not offered | `Name`, `ID` | `exactString`, `regularExpression` | unofficial |
| `USER_CREDENTIAL` | not offered | `Name`, `ID` | `exactString`, `regularExpression` | unofficial |

Where the values come from:

- **SAP Help** lists the artifact types, the attributes *Name* and *ID*, and the operators
  *Equals* and *Matches*. It states both restrictions in the table: *Matches* is not available
  for Integration Package, and message queues, global variables and global data stores support
  only names.
- **SAP's audit log documentation** shows the stored values `INTEGRATION_FLOW`, `Name`,
  `exactString` and `regularExpression`. SAP's access policy CI/CD actions use the same values.
- **The tenant** names the rest. A reference with an unknown value is answered with 400 and the
  list of allowed artifact types, "Only ID and Name allowed", or "Only 'exactString' and
  'regularExpression' are permitted". A tenant test (October 2026) created every combination
  in the table, read each back unchanged, and was refused exactly the combinations the table
  excludes. The UI artifact type column pairs the constants with SAP Help's labels by name;
  only the constant is sent.

The *unofficial* rows are types that SAP's list of artifact types names and the tenant accepted
with every attribute and operator, but that SAP Help does not offer for access policies. A plan
that creates a reference to one of them fails unless the provider block sets
`enable_unofficial = true`; refreshing, importing or destroying such a reference works without
it. SAP may change these types without notice.

### Values

With `exactString`, `value` is the exact name or ID, taken literally; characters such as `*` or
`.` have no special meaning.

With `regularExpression`, `value` is a Java regular expression (`java.util.regex.Pattern`). SAP
Help describes `myName.*` as matching every value that begins with `myName`. A `*` repeats only
the character before it, so the glob `SALES_ORDERS_*` matches `SALES_ORDERS` followed by
underscores, not every name that starts with `SALES_ORDERS_`; write `SALES_ORDERS_.*` or
`^SALES_ORDERS_.*$`. The plan shows a warning for such glob-like patterns and an error for
patterns that Java rejects as well: unbalanced parentheses or brackets, a leading `*`, a reversed
character range. The check uses Go's regular expression parser, which is not Java's, so it reports
only these errors; a pattern that passes can still be rejected by SAP.

`value` may contain at most 150 characters, `name` at most 50 and `description` at most 200.
SAP accepts longer values but keeps only their first characters, so the provider rejects them
while planning (since 0.3.2). A long list of exact names is better written as one
`regularExpression` or split into several references.

## Example Usage

```terraform
resource "sapintegrationsuite_access_policy_reference" "metering_flow" {
  access_policy_id = sapintegrationsuite_access_policy.utilities.id

  name        = "Metering flow"
  description = "The metering integration flow of the utilities package"

  artifact_type = "INTEGRATION_FLOW"
  attribute     = "Name"
  operator      = "exactString"
  value         = "Metering"
}

# "Matches" in the UI is the wire value "regularExpression". The value is a
# Java regular expression: ".*" stands for any characters, so this matches
# every integration flow whose name starts with SALES_ORDERS_.
resource "sapintegrationsuite_access_policy_reference" "core_its_flows" {
  access_policy_id = sapintegrationsuite_access_policy.utilities.id

  name          = "Sales order integration flows"
  artifact_type = "INTEGRATION_FLOW"
  attribute     = "Name"
  operator      = "regularExpression"
  value         = "^SALES_ORDERS_.*$"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `access_policy_id` (String) Numeric ID of the access policy this reference belongs to.
- `artifact_type` (String) Artifact type constant as SAP's API stores it in the Type property, for example "INTEGRATION_FLOW" or "INTEGRATION_PACKAGE", not the UI label. Only the types listed on this page are accepted; the plan fails for any other value.
- `attribute` (String) Artifact attribute the condition is evaluated against, as stored in ConditionAttribute: "Name" or "ID". Message queues, global variables and global data stores can only be matched by "Name".
- `name` (String) Name of the reference as shown in the policy's References table. Mandatory in SAP. At most 50 characters: SAP stores only the first 50, so the provider rejects a longer name during planning.
- `operator` (String) Condition type as stored in ConditionType: "exactString" (Equals in the UI) or "regularExpression" (Matches in the UI). Integration packages only allow "exactString". UI labels such as EQUALS or MATCHES are rejected.
- `value` (String) Stored in ConditionValue. With "exactString" the exact name or ID, taken literally. With "regularExpression" a Java regular expression, for example "SALES_.*" for every name that starts with SALES_ (not the glob "SALES_*"). At most 150 characters: SAP stores only the first 150, so the provider rejects a longer value during planning.

### Optional

- `description` (String) Optional description, for example what a regular expression is meant to match. At most 200 characters: SAP stores only the first 200, so the provider rejects a longer description during planning.

### Read-Only

- `id` (String) Composite identifier "<access_policy_id>/<reference_id>", both numeric SAP IDs.

## Import

```shell
# "<access_policy_id>/<reference_id>", both numeric SAP IDs.
terraform import sapintegrationsuite_access_policy_reference.metering_flow 1901/55
```

The ID is `<access_policy_id>/<reference_id>`, both numeric. Everything is recovered.

Import and refresh keep whatever SAP returns, including values the provider does not create, for
example a reference to a credential created in the UI. Only configured values are checked: to
manage such a reference with Terraform, its values have to be in the tables above, otherwise the
plan explains that this provider version does not create them.

## Limitations

- **SAP:** SAP documents the artifact types only by their UI labels; the constants come from the
  tenant. The UI can edit a reference, but the public contract only creates and deletes.
- **Provider:** the regular expression check is not a Java parser (see above). Creating references
  to the unofficial artifact types needs `enable_unofficial = true`.

## Related

- [`sapintegrationsuite_access_policy`](access_policy.md).
- [Access Policies guide](../guides/access-policies.md).
