---
page_title: "Number Ranges, Variables, and Data Stores"
subcategory: "Cloud Integration"
description: |-
  What SAP's Message Stores API family (Number Ranges, Variables, Data Stores, Data Store
  Entries) supports, why this provider manages only Number Ranges' static configuration, and
  why the others are intentionally out of scope.
---

# Number Ranges, Variables, and Data Stores

SAP Cloud Integration's "Message Stores" OData V2 API groups four related object types that
integration flows use to persist data during message processing: **Number Ranges**, **Variables**,
**Data Stores**, and **Data Store Entries**. This provider manages only one of them —
[`sapintegrationsuite_number_range`](../resources/number_range.md), and only its static
configuration, never its runtime counter. The other three are deliberately not implemented as
Terraform resources or data sources at all. This guide explains why, in detail, because the
reasoning is not obvious from the API surface alone: every one of these four object types has a
real, documented, working API — the difference is what each API actually lets a practitioner
safely do with Terraform.

If you take away one thing from this guide: **a public API is not the same thing as a Terraform
resource.** See `docs/provider-scope.md` for that principle stated in general terms; this guide
is the concrete case study.

## Number Ranges

A Number Range generates unique interchange numbers for outbound EDI/EDIFACT documents — SAP's
own description: "While sending out a document when using EDI processing, a unique interchange
number must be added to each document." It has two very different parts:

- **Static configuration** a practitioner defines: `name`, `min_value`, `max_value`,
  `description`, `rotate`, `field_length`.
- **A live runtime counter** (SAP's `CurrentValue`, shown as **"Next Value"** in the Integration
  Suite Monitor UI — same value, different name) that advances every time deployed content
  consumes a number.

`sapintegrationsuite_number_range` manages the first part. It sets the counter only when you ask
for it explicitly, and reports its live value in `current_value`.

### Reading, deleting and importing: verified on a tenant, unofficial

SAP documents only two operations for Number Ranges: *Add* (`POST /api/v1/NumberRanges`) and
*Update* (`PUT /api/v1/NumberRanges('<name>')`). Earlier releases of this provider therefore had a
`Read` that never contacted SAP, refused `terraform destroy` and refused imports.

In September 2026 a tenant test settled the rest. `GET NumberRanges('<name>')` returned the
object with every field exactly as it was sent, `DELETE NumberRanges('<name>')` answered `202`
and a read afterwards returned `404`, and the tenant `$metadata` lists the same properties plus
`DeployedBy` and `DeployedOn`. SAP still does not document them, so they are *unofficial*: they
work today, but SAP may change them without notice. With `enable_unofficial = true` in the
provider block, the resource uses both operations:

- `Read` reports what SAP holds, so a description or range changed in the Monitor UI shows up as
  drift, and a number range deleted outside Terraform drops out of state.
- `current_value` shows the live counter from the last read, next to `deployed_by` and
  `deployed_on`.
- `terraform destroy` deletes the number range. Content that still uses it fails at run time
  afterwards, so remove references first.
- Import by name: `terraform import sapintegrationsuite_number_range.invoice_numbers InvoiceNumbers`.

Because SAP does not document what a create does when the name already exists, `Create` checks
first and stops with an error that asks for an import instead.

### Without `enable_unofficial`

Without the switch the resource uses only the two documented operations:

- A refresh keeps the state as it is. Terraform shows no drift, and `deployed_by` and
  `deployed_on` stay empty; `current_value` is the counter Terraform last sent.
- Every update has to change `current_value_wo_version`, which sets the counter to
  `current_value_wo`. SAP rejects an update without a counter, and keeping the live one would
  mean reading it first. A plan that changes only the other attributes fails with an error that
  says so.
- `terraform destroy`, a replacement (a new `name`) and `terraform import` fail with an error
  that names the operation. To stop managing a number range without deleting it, use
  `terraform state rm`.
- `Create` cannot check whether the name already exists.

Switching `enable_unofficial` on later is safe: the next refresh reads the number range and fills
in what the state is missing.

Two request details from the same test matter if you call the API yourself: the entity set
answers `$top` with `501`, so the provider reads without query options; and every write returned
`202 Accepted` with an empty body.

### Why the runtime counter is a separate, write-only attribute

If `current_value` were an ordinary attribute, every `terraform plan` would either try to
"correct" SAP's live counter back to whatever Terraform last wrote (resetting real consumption
that happened through deployed EDI/EDIFACT content — a destructive, silent, and very unwelcome
surprise) or show permanent, unresolvable drift Terraform could never converge. Neither is
acceptable.

Instead, the counter is exposed as two attributes, mirroring the write-only credential-rotation
pattern this provider already uses for `sapintegrationsuite_user_credential`:

```hcl
resource "sapintegrationsuite_number_range" "invoice_numbers" {
  name         = "InvoiceNumbers"
  min_value    = "0"
  max_value    = "999999"
  description  = "Interchange numbers for outbound EDIFACT invoices"
  rotate       = true
  field_length = "6"

  current_value_wo         = "0"
  current_value_wo_version = "initial"
}
```

- `current_value_wo` is `WriteOnly` — Terraform never stores it in plan or state, matching every
  other `_wo` attribute in this provider.
- It is sent to SAP on every `Create` (SAP's documented example always includes `CurrentValue`
  when adding an object).
- On `Update`, it is sent **only when `current_value_wo_version` changes** from what is already
  in state.
- Every other update still has to carry a counter: a tenant rejected a `PUT` without
  `CurrentValue` with `500` and left the number range unchanged (September 2026). The provider
  therefore reads the live counter right before the `PUT` and sends it back unchanged. An apply
  that changes only `description`, `min_value`, `max_value`, `rotate` or `field_length` thus
  never resets a counter that deployed content has advanced. The tests
  `TestNumberRangeResource_Update_SendsLiveCounterWhenVersionUnchanged` and
  `TestClient_UpdateNumberRange_RejectsMissingCurrentValue` keep it that way.
- One narrow window remains: a number handed out by the runtime between that read and the
  `PUT`, a single round trip, would be handed out again. Change number ranges that are in heavy
  use when little traffic is flowing.

To deliberately push a new counter value — for example, correcting the counter after a manual
intervention outside Terraform — bump `current_value_wo_version` to any new string and set
`current_value_wo` to the value you want:

```hcl
resource "sapintegrationsuite_number_range" "invoice_numbers" {
  # ...unchanged static configuration...
  current_value_wo         = "1000"
  current_value_wo_version = "manual-correction-2026-01"
}
```

### Importing a number range that is in use

After an import, `current_value_wo_version` is not yet in state. The first apply records the
value from your configuration and sends back the live counter instead of `current_value_wo`,
so adopting a number range that deployed content is already consuming never resets it. To set
the counter deliberately afterwards, change `current_value_wo_version` once more.

### Names

Names may only contain letters, digits, spaces and underscores. A tenant rejected
`tf-acc-probe-nr` with a `500` and the message "Invalid NRO name input. It can only contain
alphanumeric characters, space and underscore." SAP's own example uses spaces
(`My NRO Object`). The provider checks the rule at plan time.

### A UI capability this provider does not expose: multi-runtime deployment

The Monitor UI's Add/Edit dialog for a Number Range includes a **"Runtimes"** field: "One or more
runtime nodes to deploy the artifact to... including Cloud Integration and any active Edge
Integration Cell nodes." Neither of SAP's two documented API examples (Add, Update) shows any
runtime/location parameter. This provider's client always targets the implicit default runtime
and does not attempt to reconstruct or guess at Edge Integration Cell-specific deployment — if
your tenant has active Edge Integration Cell nodes and you need Number Ranges deployed there
specifically, use the Monitor UI for that.

SAP's *Add a Number Ranges Object* page does name the Edge Integration Cell service root,
`/location/<runtime location id>/api/v1/NumberRanges`. Number ranges do not offer
`runtime_location_id`, because Edge Integration Cell targeting is not supported by this
provider (see the Edge Integration Cell guide).

### Numeric fields are strings, on purpose

`min_value`, `max_value`, `field_length`, and `current_value_wo` are all Terraform **strings**,
not numbers, matching SAP's own wire format exactly (every one of these fields is a JSON string
in SAP's documented example payloads). SAP documents `max_value` as allowing up to 14 digits —
this provider validates the digit-string shape and range client-side, but deliberately never
converts these values to a Go/Terraform numeric type, to avoid any risk of precision loss on
values this large.

### Authorization

No Cloud Foundry role template specific to Number Ranges creation/update was found documented
anywhere in SAP's public documentation — unlike Data Stores and Variables, which have explicit
role templates (see below). If you find the specific role template SAP requires, please
contribute a documentation update; until then, this provider does not claim any specific role
collection is sufficient or necessary beyond what your tenant already requires for other Message
Stores API access.

## Variables — no resource, no data source

A Variable is a value an integration flow's **Write Variables** step writes during message
processing, to be read by a later step in the same flow (a local variable) or by any other flow
deployed on the tenant (a global variable). SAP documents exactly **one** public operation for
this entity:

```
GET /api/v1/Variables(VariableName='{VariableName}',IntegrationFlow='{IntegrationFlowName}')/$value
```

This downloads the raw value — nothing else. There is no collection GET (you must already know
the exact variable name and integration flow), no `POST`, no `PUT`, and no confirmed `DELETE`.
Variables are created and updated **exclusively** by deployed integration flow content, through
an entirely different ownership model than Terraform-managed infrastructure — SAP's own
documentation is explicit that this is a mechanism "to share data across different integration
flows," not a configuration object. Variables also expire automatically after 400 days of
inactivity, extended by every successful processing run.

A read-only `data.sapintegrationsuite_variable` was considered and rejected. The single confirmed
read endpoint returns nothing but the raw runtime value, with no metadata fields (no
`Visibility`/`UpdatedAt`/`RetainUntil`) to fall back to as a safer, non-payload alternative. Every
`terraform plan` against such a data source would either show spurious churn as the underlying
business value changes for reasons entirely outside Terraform's control, or silently capture a
snapshot of runtime business data into `.tfstate`. This provider's principle — a Terraform state
file is not an Integration Suite runtime database — applies here without exception.

## Data Stores and Data Store Entries — no resources, no data sources

A **Data Store** is a runtime container that comes into existence implicitly, the first time an
integration flow's Data Store Write step (or an XI adapter configured with
`Temporary Storage = Data Store`) writes an entry to it. A **Data Store Entry** is one persisted
message — payload, headers, and processing metadata (`Status`, `MessageId`, `DueAt`, `CreatedAt`,
`RetainUntil`) — inside a Data Store.

SAP documents `GET`-only access to both:

- `GET /api/v1/DataStores?overdueonly=true` — an aggregate **monitoring** endpoint, returning
  message counts per store, not configuration. This is the same class of data as this provider's
  already-excluded Message Processing Logs: useful for observability, not something Terraform
  reconciles.
- `GET .../DataStoreEntries(...)` (a single entry, or all entries for a store) — runtime business
  message records.

There is no independent `POST`/`PUT` for either entity anywhere — a Data Store and its entries
come into existence only as side effects of deployed integration flow content executing.
**Delete does exist, but not as a REST operation**: SAP documents a design-time "Data Store
Delete" integration-flow step, explicitly scoped to "single entries only" (it "can't be used to
delete whole data stores"), operating on messages as an integration flow processes them — not
something this provider could reasonably wrap in a `terraform destroy` without misrepresenting
what that verb means for every other resource in this provider.

Neither entity gets a resource or a data source. This is intentional and, for Data Store Entries
specifically, treated as **`out_of_scope`** rather than merely `not_implemented`: exposing
message payloads and processing status in Terraform state has no legitimate infrastructure-as-code
purpose, and "an entry can be deleted somehow" is not a reason to model that deletion as
`terraform destroy` any more than it would be for a Message Processing Log record.

Both `DataStores` and `Variables` confirm, verbatim, that they do not support `$filter`,
`$inlinecount`, `$orderby`, `$skip`, `$top`, `$expand`, or `$select` — two separate statements in
SAP's documentation, one per entity, not a single rule this provider generalized across the whole
API family. Number Ranges makes no such statement, but a tenant answered `$top` on `NumberRanges`
with `501` all the same (September 2026), so the provider reads it without query options.

### Authorization

For Data Stores/Data Store Entries/Variables, SAP documents these Cloud Foundry role templates:
`DataStoresAndQueuesRead` (view data store entries), `DataStorePayloadsRead` (download payloads,
view variables), `DataStoresAndQueuesDelete` (delete data store entries/variables). The existence
of a delete-scoped role template confirms some delete capability is authorized for this data,
consistent with the design-time Delete step described above, even though this provider found no
REST `DELETE` example to call directly.

## Edge Integration Cell

The only hint of an Edge Integration Cell-specific API path in this entire research pass was the
Number Ranges "Runtimes" UI field mentioned above. No `/location/<runtime-location-id>/...`-style
path, or any other Edge-specific endpoint, was found documented for any of the four object types
covered in this guide. This remains genuinely unconfirmed, and nothing Edge-specific is
implemented here as a result — consistent with this provider's rule against guessing at
undocumented endpoints.
