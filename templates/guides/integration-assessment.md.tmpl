---
page_title: "Integration Assessment"
subcategory: "Additional Capabilities"
description: |-
  Integration Assessment's confirmed API surface, its full entity inventory classified by
  Terraform suitability, and exactly why this provider implements none of it yet.
---

# Integration Assessment

Integration Assessment implements SAP's Integration Solution Advisory Methodology (ISA-M): a
guided approach to assessing an organization's integration landscape and strategy, and to
recording decisions about which integration technology should handle which kind of integration.
It is a separate SAP BTP service subscription (entitlement `integration-assessment`), not a
sub-feature of Cloud Integration or either API Management model.

If you came here looking for `sapintegrationsuite_integration_assessment_application` or
similar: **nothing in this capability is implemented yet, and this guide explains exactly why**,
together with the confirmed entity inventory this provider's decision rests on.

## Authentication is confirmed, and genuinely separate again

Integration Assessment authenticates through its own BTP service instance — **Integration
Assessment APIs**, plan `default` — provisioned independently of the Integration Suite
subscription this provider otherwise assumes. Its service key is confirmed to contain:

- `entities` — base URL for the Entities API
- `management` — base URL for the Management API
- `clientid` / `clientsecret` / `url` (token server; SAP says to append `/oauth/token` to it) —
  the same OAuth 2.0 client-credentials shape this provider already uses for Cloud Integration,
  current API Management, and Classic API Management

The SAP Business Accelerator Hub's public catalog (re-audit September 2026) lists exactly two API
artifacts in the package `SAPIntegrationAssessment`, both of type **OData**, version 1.0.0:
**Entities** (`EntitiesAPI`, "Access entities of Integration Assessment") and **Management**
(`ManagementAPI`, "Manage content of Integration Assessment"). The entity inventory below
therefore belongs to the Entities API. The Management API is about content, which matches the
UI's *Content Management* page: updating SAP-delivered content and importing and exporting a
tenant's data. That is an operation, not desired state.

This provider does not add a `provider.integration_assessment` configuration block in this
phase. Adding provider schema with nothing behind it to configure would be dead surface area —
the same discipline this provider already applies elsewhere (see `docs/provider-scope.md`). If
and when a resource or data source is implemented here, the block will be added alongside it,
following the same pattern as `provider.api_management`.

## The confirmed entity inventory

SAP's own "Integration Assessment APIs" documentation page lists every entity in this capability,
each with a one-paragraph description — genuinely the most complete *inventory* this provider has
found for any capability audited without a lucky primary-source PDF (compare
`docs/guides/classic-api-management.md`, where a complete official user guide with worked
examples was found). What is missing, despite substantial effort (the SAP-docs mirror, the
official "SAP Integration Solution Advisory Methodology" PDF user guide, SAP's own TechEd
IN262 hands-on sample repository, and in September 2026 the Business Accelerator Hub catalog,
all checked), was any field-level contract. That changed on 2026-09-27: the live `$metadata` of
both services was read with a service key and is committed as a normalized snapshot
(`testdata/api-metadata/integration-assessment-entities.json` and `-management.json`). The
Entities service has 27 entity sets, each keyed by a string `Id`, with one function import
(`InterfaceRequestReport`); the Management service has `ImportContent` and the function
`ExportContent`. The document says nothing about which entity sets accept writes (no
`sap:creatable` or `sap:updatable` annotations), so the classification below rests on the
confirmed shape plus SAP's descriptions.

### Master data — SAP-maintained ISA-M taxonomy

Domain, Style, Use Case Pattern, Integration Pattern, Key Characteristic (with its own Group,
Value, and Recommendation sub-entities), Deployment Model, Domain Determination. This is the
reference taxonomy the whole methodology is built on — SAP ships it, and a tenant can review and
adjust it (SAP's documentation includes a dedicated "Update Content Maintained by SAP" procedure).
If a public API is ever confirmed for it, this looks like a genuine data-source candidate at
minimum; whether the "adjust" capability rises to full resource-level suitability would need its
own suitability check once the API itself is confirmed.

### Landscape configuration — the strongest resource candidate

Application, Application Instance, Technology, Technology Instance, Vendor, and their
association entities (Technology Domain, Technology Style, Technology Key Characteristic). This
is practitioner-authored configuration, not reference data or workflow state, and SAP documents
concrete per-tenant limits that confirm real, bounded, persisted storage: a maximum of 20,000
Applications, 20,000 Application Instances, 50 Technologies, 150 Technology Instances, and
10,000 Vendors. The `$metadata` now confirms their fields: `Application` (`Name` required,
`ApiHubId`, links to `Vendor` and its instances), `ApplicationInstance` (`Name` required,
`Description`, links to `Application` and a `DeploymentModel`), `Technology` and
`TechnologyInstance` (the same pattern for middleware), and `Vendor` (`Name`). What is still
unconfirmed is write support: whether SAP accepts create, update and delete through the API,
whether it generates the `Id`s, and how a link such as application to vendor is written. A probe
(`.specs/ia-probe.ps1 -LandscapeTests`) checks exactly that with objects named `tfacc-probe-*`
and deletes them again; the resources follow once it passes.

### Requests and assessment workflow — out of scope regardless

Request and Request Line Item, the entry points for a business solution/interface request
workflow, plus the Integration Flow/Message Flow content a request references and Request Line
Item Technology Instance Decision. SAP documents an explicit status machine for Request: `draft`
(automatic on Create) → `new` (Submit) → `in progress` (automatic, once interface requests are
assessed, or via Reopen) → `completed` (Complete). This is workflow/project state, the same
category this provider already excludes for Cloud Integration's Message Processing Logs and
Developer Hub's Subscription object — out of scope by its nature, independent of whether a field
contract is ever confirmed for it.

## What this provider manages today

Nothing. See `internal/features/catalog.go`'s `integration_assessment.*` entries for the
per-group classification recorded above, and `docs/sap-api-references.md` for the full research
trail, including the specific sources checked and found to contain UI procedures only.

## Revisiting this decision

The field-level contract is no longer the gap: it is in the committed snapshots, and the API
discovery (`cmd/apidiscovery`, `TestAccMetadata`) compares them with the live services and
reports any change. The remaining gap is write support for the landscape objects, which the probe
settles. The order stays: landscape configuration first, the ISA-M taxonomy as data sources
next, requests and the content import/export out of scope. An implementation adds a
`provider.integration_assessment` block, since the credentials are a separate service key.
