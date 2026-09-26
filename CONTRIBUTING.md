# Contributing

Thank you for considering a contribution to the Terraform Provider for SAP
Integration Suite.

## Ground rules

1. **Every resource and data source must trace back to an officially
   documented, SAP-supported public API.** See `docs/sap-api-references.md`
   for the pattern each resource follows, and never implement against an
   internal/UI-only endpoint discovered through browser network traffic.
2. **Respect the provider boundary.** BTP control-plane concerns (accounts,
   subaccounts, entitlements, subscriptions, generic destinations, role
   collections) belong in the official `SAP/btp` provider, not here. See
   `docs/provider-scope.md` and `docs/provider-boundaries.md`.
3. **Quality over quantity.** A resource is only added once it passes the
   suitability checklist in `docs/resource-design.md` (desired state,
   identity, read-back, create, update, delete, drift, import).

## Branch model

This repository uses a permanent two-branch model:

```
master
  ^
 dev
  ^
feature/<name>
```

- **`master`** is the stable/release branch, and remains the GitHub default
  branch. It only moves forward as an explicit release/stabilization step
  — never automatically after a feature merges.
- **`dev`** is the permanent integration branch. All feature work lands
  here first. `dev` is not the GitHub default branch.
- **`feature/<name>`** branches are always created from `dev`, and always
  merge back into `dev`, never directly into `master`. Name them after the
  feature (for example `feature/access-policy-completion`), never with a
  tooling/vendor/author prefix such as `claude/`, `ai/`, `bot/`, or
  `anthropic/`.

Workflow for a feature:

```shell
git switch dev
git pull --ff-only origin dev
git switch -c feature/<name>
# ... do the work, commit ...
git switch dev
git pull --ff-only origin dev
git merge --ff-only feature/<name>
git push origin dev
```

Prefer `--ff-only` while development is a single sequential stream. If a
fast-forward merge fails, stop and inspect why rather than forcing a merge
commit, rebasing, or force-pushing a published branch.

Promoting `dev` to `master` is a separate, deliberate release step,
performed only when explicitly requested — never automatically after a
feature merges into `dev`.

## Development setup

Requirements: Go (version pinned in `go.mod`; the `tools/` module tracks
the same version independently, since it is a separate Go module), the
Terraform CLI (see README.md's Requirements section for the minimum
supported version — needed for documentation generation and acceptance
tests), and `golangci-lint` (the exact version this project lints with is
pinned in `.github/workflows/lint.yml`'s `golangci-lint-action` step; keep
your local binary on the same major.minor to avoid lint results that only
reproduce in CI or only reproduce locally).

```shell
go build ./...
go test ./...
make lint
make docs   # regenerate docs/ and README.md's feature table after any schema, description, or catalog change
```

`go.mod`, not this file or the README, is this project's single source of
truth for the required Go version — CI, GoReleaser, and every workflow
under `.github/workflows/` all read it via `go-version-file: go.mod`
rather than a separately hardcoded version, specifically so upgrading Go
in one place upgrades it everywhere.

## Adding or changing a resource

1. Update the relevant matrix in `docs/api-capability-matrix.md` and/or
   `docs/provisioning-capability-matrix.md` with what you found and where.
2. Document the resource in `docs/resource-design.md` and
   `docs/sap-api-references.md`.
3. Implement the API client method(s) in `internal/client/...` — Terraform
   resource code must never perform raw HTTP calls directly.
4. Implement the resource/data source in `internal/provider/...` with
   Create/Read/Update/Delete/ImportState, using `RequiresReplace()` for any
   field SAP does not support updating in place.
5. Add unit tests (`httptest`-based) for the client method(s) and a basic
   schema/import sanity test for the resource. For an OData entity, also
   register every wire struct, key and function import in the package's
   `metadata_contract_test.go` (see "Checking wire contracts against
   `$metadata`" below).
6. Add an example under `examples/resources/<name>/` or
   `examples/data-sources/<name>/`, then run `make docs`.
7. If the change is acceptance-testable, add an acceptance test
   (`TestAcc*`, `terraform-plugin-testing`) next to the ones in
   `internal/provider/acc_*_test.go`. Name every object it creates with
   `testAccName()` (`tfacc` plus random characters; no hyphens, because
   several SAP IDs reject them), and take design-time content from
   `internal/testutil/samples` (see "Testing with SAP's samples" below).
   Acceptance tests only run with `TF_ACC=1` and the
   `SAP_INTEGRATION_SUITE_*` variables set.

   A test that creates, modifies, or deletes **tenant-wide singleton**
   configuration (for example `sapintegrationsuite_custom_tag_configuration`,
   or any future Classic API Management resource whose identity is scoped
   to the whole tenant rather than a named object a test can safely
   namespace with a `tf-acc-` prefix) must be gated behind an *additional*,
   capability-specific opt-in environment variable beyond plain `TF_ACC=1`
   — for example `SAP_INTEGRATION_SUITE_ACC_API_MANAGEMENT=1` for Classic
   API Management acceptance tests. `TF_ACC=1` alone must never be
   sufficient to run a test that could disrupt a shared tenant's existing
   configuration; the extra gate makes that risk an explicit, opt-in
   decision for whoever runs the test suite against a real tenant.
8. **Update the feature support catalog.** This is mandatory, not optional
   — no feature implementation is complete until this step is done:
   1. Add or update the feature's entry in `internal/features/catalog.go`
      (`support_status`, `support_reason`, `resource_types`,
      `data_source_types`, and every `Operations` flag).
   2. Only set an `Operations` flag to `true`, or move `support_status`
      toward `supported`, once the corresponding operation is actually
      implemented and covered by a test proving the claimed SAP API
      behavior — an `Update` method existing in Go is not by itself
      evidence that `operations.update` should be `true`; a resource
      whose `Update` exists to satisfy the Terraform Plugin Framework
      interface but never gets called for a real diff (`RequiresReplace`
      on everything mutable) is not `operations.update = true` either.
   3. Regenerate `docs/feature-support.md` with `go run ./cmd/gendocs > docs/feature-support.md`
      (also done by `make docs`).
   4. Run the catalog consistency tests
      (`go test ./internal/features/... ./internal/provider/... -run 'TestCatalog|TestFeatureCatalog'`)
      — they fail if a registered resource/data source has no catalog
      entry, or if a catalog entry claims a resource/data source type that
      does not exist.

## Testing with SAP's samples

Resources that upload content (integration flows, message mappings and
their deployments) are tested with real artifacts from SAP's public sample
repositories on github.com/SAP-samples (Apache-2.0).
`internal/testutil/samples/catalog.go` lists each file with its repository,
commit and SHA-256 hash; `samples.Get` downloads it into a local cache and
fails the test if the bytes differ from the pinned hash. Nothing from those
repositories is copied into this one.

- **Offline tests** (`go test ./internal/testutil/samples/`) check every
  catalog entry against the provider's own parsers, for example the
  MANIFEST reader, which has to join SAP's wrapped manifest lines. They
  download only with `SAP_SAMPLES_DOWNLOAD=1` (CI sets it) and skip
  otherwise, unless the samples are already cached.
- **Acceptance tests** give every artifact a unique ID with
  `samples.WithBundleID`, because SAP rejects a content update whose
  `Bundle-SymbolicName` differs from the artifact ID. A test deploys only
  samples marked `Deployable`: no timer and no polling sender, so they run
  only when someone calls their endpoint.

To add a sample, pick a file at a fixed commit, compute its SHA-256, add the
entry to the catalog with the right `Kind`, and check that
`TestCatalog` passes with `SAP_SAMPLES_DOWNLOAD=1`.

## Running the acceptance tests against a tenant

```sh
TF_ACC=1 \
SAP_INTEGRATION_SUITE_HOST=https://<tenant>.it-cpi<...>.cfapps.<region>.hana.ondemand.com \
SAP_INTEGRATION_SUITE_TOKEN_URL=https://<subdomain>.authentication.<region>.hana.ondemand.com/oauth/token \
SAP_INTEGRATION_SUITE_CLIENT_ID=... \
SAP_INTEGRATION_SUITE_CLIENT_SECRET=... \
go test ./internal/provider/ -run '^TestAcc' -v -timeout 60m
```

The client needs the roles listed in the "Authorization and Roles" guide
for packages, content, configuration and deployment. Each test destroys what
it created; an interrupted run can leave `tfacc*` packages behind, which you
can delete in the Design UI.

## Checking wire contracts against `$metadata`

Several past mistakes in this provider had the same cause: a property name
was inferred from SAP's UI labels or prose documentation and never checked
against the service itself. The access policy reference fields, the
integration adapter's `Type`/`Application` and the service endpoint API
definition `Type` all looked plausible and did not exist.

The cheapest guard is the OData service's own `$metadata` document. Every
OData client package has a `metadata_contract_test.go` that uses
`internal/testutil/edmx` to check, by reflection, that each wire struct's
JSON fields are properties of the entity type behind its entity set
(including properties inherited through `BaseType`), that key names and EDM
types match what the client sends, and that function imports have the
parameters the client passes.

The document is tenant data and is never committed. To run the checks,
download it once from a tenant you are allowed to use:

```shell
curl -H "Authorization: Bearer <token>" \
  "https://<tenant-api-host>/api/v1/\$metadata" \
  -o .specs/cloudintegration-metadata.xml
```

Add `.specs/` to `.git/info/exclude`, then run `go test ./internal/client/...`.
The Classic API Management client is checked the same way against the API
portal's `/apiportal/api/1.0/Management.svc/$metadata`, stored as
`.specs/apim-management-metadata.xml` (or named by
`SAP_API_PORTAL_METADATA_FILE`); it needs a service key of plan
`apiportal-apiaccess`.
The helper also honors `SAP_INTEGRATION_SUITE_METADATA_FILE`. Without a
document the contract tests skip, which is why CI stays green without tenant
access. A skipped contract test is not evidence, so mention in the PR
whether you ran them.

Do not add query options to reads without checking them on a tenant. Several
Cloud Integration entity sets reject options the `$metadata` does not warn
about: `OAuth2ClientCredentials`, `SecureParameters`, `UserCredentials` and
`NumberRanges` answer `$top` and `$select` with 501, and `KeystoreEntries`
answers any option, even `$format=json`, with 400. The OData client asks for
JSON through the `Accept` header for that reason (see
`docs/sap-api-references.md`, "Tenant probe results").

`$metadata` settles property names, keys, types, length facets such as
`MaxLength`, and navigation. It does not settle enum values (they are plain
`Edm.String`) or whether SAP actually accepts a create or update on an entity
set. Those still need SAP's documentation, SAP's own published tooling, or a
live request.

The same approach works for other SAP OData services whose specification is
only available behind the Business Accelerator Hub login. Integration
Assessment's two APIs and API Composition's Configuration API are OData
services; with a service key of the respective instance, fetch
`<service root>/$metadata` the same way and store it under `.specs/`. For
Integration Assessment the service roots are the key's `entities` and
`management` values, and the token endpoint is its `url` plus `/oauth/token`.
A document like that is what an implementation of those areas needs first.

## Documentation standards

Documentation is part of a change, not a follow-up. Write it for an engineer
who knows Terraform but not necessarily SAP Integration Suite:

- Explain what SAP object a resource represents, where it sits in the
  product, and which public API backs it, with a reference in
  `docs/sap-api-references.md`.
- Describe the lifecycle: what Create, Update and Delete do on SAP's side,
  what `terraform plan` shows after drift, how import works, and which
  attributes force replacement and why.
- Say plainly whether a limitation is SAP's (the API does not offer the
  operation), the provider's (SAP offers it, this provider has not
  implemented it), or a deliberate boundary (the object belongs to another
  provider or is runtime data).
- Be explicit about secrets: which values SAP can return, which cannot be
  read back, and how `*_wo`/`*_wo_version` attributes rotate them.
- Use realistic, synthetic examples that match the schema exactly.
- Prefer clear prose over mechanical lists, and avoid filler.

Generated pages come from schema descriptions, `templates/`, `examples/`
and `internal/features/catalog.go`. Change those sources and regenerate with
`make docs`. Never hand-edit a generated file under `docs/resources/`,
`docs/data-sources/`, `docs/guides/`, `docs/feature-support.md` or
README.md's generated feature table.

## Commit and PR expectations

- Pull requests target `dev`, not `master` — see the branch model above.
- Run `go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .`, and
  `golangci-lint run ./...` before opening a PR; CI enforces all of these
  on pull requests and on pushes to both `dev` and `master`.
- Keep commits logically scoped; a single PR can contain multiple commits.
- Fill in the PR template, including the SAP API reference for any new
  capability.

## Code of Conduct

This project follows the Code of Conduct in `CODE_OF_CONDUCT.md`.
