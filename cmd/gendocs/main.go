// Command gendocs regenerates generated documentation from the canonical
// feature catalog in internal/features. It is the single place that turns
// features.Catalog into prose and tables, so that catalog and
// documentation cannot drift apart the way independently hand-maintained
// copies would — see CONTRIBUTING.md for the rule that every feature
// change updates the catalog and regenerates this output in the same
// change.
//
// Usage:
//
//	go run ./cmd/gendocs
//	go run ./cmd/gendocs -readme
//
// The first form writes docs/feature-support.md and
// docs/research/capability-evidence-2026.md directly (it no longer
// relies on the caller redirecting stdout with a shell "> docs/feature-support.md":
// that redirection is not portable — a Windows shell's ">" operator can
// prepend a UTF-8 byte-order mark that a POSIX shell's ">" never does,
// which made the committed file's bytes depend on which shell last
// regenerated it, and made "git diff" report a change on every platform
// switch. Writing the file directly with os.WriteFile is deterministic on
// every platform, which is the whole point of a generator whose output is
// meant to be diffed in CI). The second form rewrites the generated block
// of README.md in place (everything between the
// "<!-- BEGIN GENERATED FEATURE SUPPORT -->" and
// "<!-- END GENERATED FEATURE SUPPORT -->" markers); it does not touch any
// other part of README.md.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/features"
)

// reasonOrder fixes the order SupportReason groups appear in the
// "Unsupported and partially supported features" section, and gives each a
// short, human-readable heading.
var reasonOrder = []struct {
	reason  features.SupportReason
	heading string
}{
	{features.ReasonNotImplemented, "Public API exists but provider implementation is pending"},
	{features.ReasonPublicAPIIncomplete, "Public API details are not fully confirmed"},
	{features.ReasonUnsafeTerraformLifecycle, "Public lifecycle insufficient for safe Terraform management"},
	{features.ReasonNoPublicAPI, "No suitable public SAP API"},
	{features.ReasonResearchRequired, "Further research required"},
	{features.ReasonOutOfScope, "Out of provider scope"},
}

// domainOrder fixes the display order and heading of each catalog Domain in
// the README's generated feature overview. A domain present in the catalog
// but missing from this list is never dropped — see domainHeading — it is
// just appended after every named domain, alphabetically, so a newly added
// Domain value always shows up somewhere without requiring this list to be
// updated in lockstep.
var domainOrder = []struct {
	domain  string
	heading string
}{
	{"cloud_integration", "Cloud Integration"},
	{"security", "Security & Access Policies"},
	{"partner_directory", "Partner Directory"},
	{"api_management_classic", "Classic API Management"},
	{"api_gateway", "Current API Management / API Artifacts"},
	{"api_composition", "API Composition"},
	{"integration_cell", "Integration Cell"},
	{"edge_integration_cell", "Edge Integration Cell"},
	{"integration_assessment", "Integration Assessment"},
	{"capability_provisioning", "Capability Provisioning"},
	{"other_capability", "Additional Integration Suite Capabilities"},
}

const (
	readmeMarkerBegin = "<!-- BEGIN GENERATED FEATURE SUPPORT -->"
	readmeMarkerEnd   = "<!-- END GENERATED FEATURE SUPPORT -->"

	featureSupportPath = "docs/feature-support.md"
	readmePath         = "README.md"
)

func main() {
	readme := flag.Bool("readme", false, "rewrite the generated feature table in README.md in place, instead of regenerating docs/feature-support.md")
	flag.Parse()

	if *readme {
		if err := regenerateReadme(readmePath); err != nil {
			fmt.Fprintln(os.Stderr, "gendocs:", err)
			os.Exit(1)
		}
		return
	}

	if err := writeGeneratedFile(featureSupportPath, featureSupportDoc()); err != nil {
		fmt.Fprintln(os.Stderr, "gendocs:", err)
		os.Exit(1)
	}
	if err := writeGeneratedFile(evidencePath, evidenceDoc()); err != nil {
		fmt.Fprintln(os.Stderr, "gendocs:", err)
		os.Exit(1)
	}
}

// writeGeneratedFile writes content to path as plain UTF-8 with no
// byte-order mark, overwriting whatever was there. Every path this
// generator writes is one of the two constants above — never derived from
// user input, an environment variable, or a command-line argument — so
// there is no path-traversal concern despite the variable parameter this
// function takes to stay reusable between the two call sites.
func writeGeneratedFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600) //nolint:gosec // G703: path is always featureSupportPath, evidencePath or readmePath, compile-time constants in this same file, never external input
}

func sortedCatalog() []features.Feature {
	sorted := make([]features.Feature, len(features.Catalog))
	copy(sorted, features.Catalog)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Key < sorted[j].Key })
	return sorted
}

func featureSupportDoc() string {
	var b strings.Builder

	fmt.Fprintln(&b, "# Feature Support")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "Generated from `internal/features/catalog.go` by `go run ./cmd/gendocs`. Do not edit by hand — regenerate it instead, and see `CONTRIBUTING.md` for when this is required.\n\n")
	fmt.Fprintln(&b, "This is what this provider version implements, derived from the same catalog the "+
		"provider binary itself exposes through `sapintegrationsuite_provider_features` (whose "+
		"`provider_version` attribute reports the exact version at apply time). It is queryable "+
		"directly from Terraform, with no SAP tenant connection required:")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "```hcl")
	fmt.Fprintln(&b, `data "sapintegrationsuite_provider_features" "all" {}`)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, `output "supported_features" {`)
	fmt.Fprintln(&b, `  value = [`)
	fmt.Fprintln(&b, `    for f in data.sapintegrationsuite_provider_features.all.features :`)
	fmt.Fprintln(&b, `    f.key`)
	fmt.Fprintln(&b, `    if f.support_status == "supported"`)
	fmt.Fprintln(&b, `  ]`)
	fmt.Fprintln(&b, `}`)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, `data "sapintegrationsuite_provider_feature" "one" {`)
	fmt.Fprintln(&b, `  key = "cloud_integration.value_mapping"`)
	fmt.Fprintln(&b, `}`)
	fmt.Fprintln(&b, "```")
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, "See README.md's \"Feature Support\" section for a compact, high-level dashboard "+
		"generated from this same catalog (`go run ./cmd/gendocs -readme`); this document is the "+
		"detailed per-operation matrix.")
	fmt.Fprintln(&b)

	writeStatusLegend(&b)

	fmt.Fprintln(&b, "## Relationship to the other capability documents")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "This document, `docs/api-capability-matrix.md`, `docs/provisioning-capability-matrix.md`, "+
		"and a possible future tenant-capability data source answer three related but distinct questions:")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "| Document | Answers |")
	fmt.Fprintln(&b, "|---|---|")
	fmt.Fprintln(&b, "| `docs/api-capability-matrix.md` / `docs/provisioning-capability-matrix.md` | What SAP's public APIs expose, independent of this provider |")
	fmt.Fprintln(&b, "| `docs/feature-support.md` (this document) / `sapintegrationsuite_provider_features` | What *this provider version* implements — static provider metadata, no SAP tenant required |")
	fmt.Fprintln(&b, "| A possible future `sapintegrationsuite_tenant_capabilities` data source (not implemented) | Which Integration Suite capabilities are *active in a specific SAP tenant* — would require SAP credentials and a reliable public discovery API, neither of which this provider assumes here |")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Do not conflate these: a feature can be fully supported by this provider and still be "+
		"unusable in a given tenant because the underlying SAP capability was never activated there, and "+
		"vice versa a capability can be active in every tenant while this provider still does not implement "+
		"a resource for it.")
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, "## All features")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "| Feature | Domain | Status | Contract source | Public API | Create | Read | Update | Delete | Import | Deploy | Terraform |")
	fmt.Fprintln(&b, "|---|---|---|---|---|---|---|---|---|---|---|---|")

	sorted := sortedCatalog()
	for _, f := range sorted {
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			f.Key, f.Domain, statusCell(f), sourceCell(f), boolCell(f.PublicAPI),
			opCell(f.Operations.Create), opCell(f.Operations.Read), opCell(f.Operations.Update),
			opCell(f.Operations.Delete), opCell(f.Operations.Import), opCell(f.Operations.Deploy),
			terraformCell(f),
		)
	}
	fmt.Fprintln(&b)

	writeContractSources(&b, sorted)

	fmt.Fprintln(&b, "## Unsupported and partially supported features")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Grouped by why, not just that. A feature can be `partial` and reachable via one of these "+
		"reasons too — see docs/feature-support.md's per-feature `Limitations` (the `limitations` attribute "+
		"in Terraform) for exactly what is and is not covered.")
	fmt.Fprintln(&b)

	for _, group := range reasonOrder {
		var matches []features.Feature
		for _, f := range sorted {
			if f.SupportStatus != features.StatusSupported && f.SupportReason == group.reason {
				matches = append(matches, f)
			}
		}
		if len(matches) == 0 {
			continue
		}

		fmt.Fprintf(&b, "### %s\n\n", group.heading)
		for _, f := range matches {
			statusNote := ""
			switch f.SupportStatus {
			case features.StatusPartial:
				statusNote = " (partial support already implemented — see Limitations below)"
			case features.StatusUnsupported:
			default:
				info := f.SupportStatus.Info()
				statusNote = " (" + info.Icon + " " + info.Label + ")"
			}
			fmt.Fprintf(&b, "- **`%s`** — %s%s\n", f.Key, f.Description, statusNote)
			for _, l := range f.Limitations {
				fmt.Fprintf(&b, "  - %s\n", l)
			}
		}
		fmt.Fprintln(&b)
	}

	return b.String()
}

func boolCell(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func opCell(b bool) string {
	if b {
		return "Yes"
	}
	return "—"
}

func statusCell(f features.Feature) string {
	if f.SupportReason == "" {
		return string(f.SupportStatus)
	}
	return fmt.Sprintf("%s (%s)", f.SupportStatus, f.SupportReason)
}

func terraformCell(f features.Feature) string {
	hasResource := len(f.ResourceTypes) > 0
	hasDataSource := len(f.DataSourceTypes) > 0
	switch {
	case hasResource && hasDataSource:
		return "Resource + Data Source"
	case hasResource:
		return "Resource"
	case hasDataSource:
		return "Data Source"
	default:
		return "—"
	}
}

// statusIcon is the icon of a SupportStatus from features.StatusLegend,
// the one place that defines icons, labels and definitions, so the legends
// this generator writes can never drift from the icons in its tables.
func statusIcon(status features.SupportStatus) string {
	return status.Info().Icon
}

// statusLegendLine is the one-line legend above the README's tables.
func statusLegendLine() string {
	parts := make([]string, 0, len(features.StatusLegend))
	for _, info := range features.StatusLegend {
		parts = append(parts, info.Icon+" "+info.Label)
	}
	return "Legend: " + strings.Join(parts, " · ")
}

// writeStatusLegend writes the status table with each status's icon, label,
// machine-readable value and definition.
func writeStatusLegend(b *strings.Builder) {
	fmt.Fprintln(b, "## Status legend")
	fmt.Fprintln(b)
	fmt.Fprintln(b, "`support_status` in `sapintegrationsuite_provider_features` and "+
		"`sapintegrationsuite_provider_feature` carries the value in the second column.")
	fmt.Fprintln(b)
	fmt.Fprintln(b, "| Status | Value | Meaning |")
	fmt.Fprintln(b, "|---|---|---|")
	for _, info := range features.StatusLegend {
		fmt.Fprintf(b, "| %s %s | `%s` | %s |\n", info.Icon, info.Label, info.Status, info.Definition)
	}
	fmt.Fprintln(b)
	fmt.Fprintln(b, "Three of them are easy to confuse. 🔬 Research required: the capability is known, but "+
		"it has not been investigated far enough to decide whether and how to model it in Terraform, "+
		"and there is no implementation. 🧪 Experimental: an implementation exists, but its lifecycle "+
		"has not yet passed a test on a real SAP tenant; it needs `enable_experimental = true`. "+
		"🧭 Unofficial: the implementation passed on a tenant, but the SAP API contract behind it is not "+
		"fully public or officially documented, so SAP may change it without notice; it needs "+
		"`enable_unofficial = true`.")
	fmt.Fprintln(b)
}

// reasonNote gives a very short, human phrase for why a feature with no
// Terraform resource or data source at all is not supported, used as the
// README table's "Terraform Support" cell for such rows so it never reads
// as a bare, unexplained em dash.
func reasonNote(reason features.SupportReason) string {
	switch reason {
	case features.ReasonNotImplemented:
		return "Planned"
	case features.ReasonPublicAPIIncomplete:
		return "Planned — API details unconfirmed"
	case features.ReasonNoPublicAPI:
		return "No suitable public API"
	case features.ReasonResearchRequired:
		return "Research required"
	case features.ReasonOutOfScope:
		return "Out of scope"
	case features.ReasonUnsafeTerraformLifecycle:
		return "No safe Terraform lifecycle confirmed"
	default:
		return "Not implemented"
	}
}

// readmeTerraformCell renders the README table's compact "Terraform
// Support" cell: the same Resource/Data Source summary as
// docs/feature-support.md when at least one Terraform type exists, a short
// reason phrase when none exists, and — for anything short of full support
// — a trailing pointer to the detailed matrix, satisfying the rule that a
// partial/read-only/unsupported row must never leave a reader guessing why.
func readmeTerraformCell(f features.Feature) string {
	cell := terraformCell(f)
	if cell == "—" {
		cell = reasonNote(f.SupportReason)
	}
	if f.SupportStatus != features.StatusSupported {
		cell += " — see [feature-support.md](docs/feature-support.md#all-features)"
	}
	return cell
}

func domainHeading(domain string) string {
	for _, d := range domainOrder {
		if d.domain == domain {
			return d.heading
		}
	}
	return domain
}

// readmeFeatureOverview renders the README's compact, generated feature
// dashboard: one small table per catalog Domain (in domainOrder's fixed
// sequence, then any unlisted domain alphabetically), one row per catalog
// Feature — every feature, supported or not, since a practitioner needs to
// see gaps as clearly as capabilities. This is intentionally more compact
// than docs/feature-support.md (three columns instead of eleven, no inline
// Limitations prose) and is generated purely from features.Catalog, with
// no independently maintained content of its own.
func readmeFeatureOverview() string {
	var b strings.Builder

	fmt.Fprintln(&b, "Generated from `internal/features/catalog.go` by `go run ./cmd/gendocs -readme` — "+
		"do not hand-edit the table below; regenerate it instead (`make docs` does this "+
		"automatically). See [`docs/feature-support.md`](docs/feature-support.md) for the full "+
		"per-operation matrix and every feature's detailed limitations.")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, statusLegendLine()+". "+
		"The [status legend](docs/feature-support.md#status-legend) defines each one.")
	fmt.Fprintln(&b)

	sorted := sortedCatalog()
	byDomain := make(map[string][]features.Feature)
	var domains []string
	for _, f := range sorted {
		if _, seen := byDomain[f.Domain]; !seen {
			domains = append(domains, f.Domain)
		}
		byDomain[f.Domain] = append(byDomain[f.Domain], f)
	}

	orderedDomains := make([]string, 0, len(domains))
	seen := make(map[string]bool, len(domains))
	for _, d := range domainOrder {
		if _, ok := byDomain[d.domain]; ok {
			orderedDomains = append(orderedDomains, d.domain)
			seen[d.domain] = true
		}
	}
	var remaining []string
	for _, d := range domains {
		if !seen[d] {
			remaining = append(remaining, d)
		}
	}
	sort.Strings(remaining)
	orderedDomains = append(orderedDomains, remaining...)

	var counts = map[features.SupportStatus]int{}

	for _, domain := range orderedDomains {
		fmt.Fprintf(&b, "### %s\n\n", domainHeading(domain))
		fmt.Fprintln(&b, "| Feature | Status | Terraform Support |")
		fmt.Fprintln(&b, "|---|:---:|---|")
		for _, f := range byDomain[domain] {
			counts[f.SupportStatus]++
			fmt.Fprintf(&b, "| %s | %s | %s |\n", f.Name, statusIcon(f.SupportStatus), readmeTerraformCell(f))
		}
		fmt.Fprintln(&b)
	}

	summary := make([]string, 0, len(features.StatusLegend))
	for _, info := range features.StatusLegend {
		summary = append(summary, fmt.Sprintf("%d %s", counts[info.Status], strings.ToLower(info.Label)))
	}
	fmt.Fprintf(&b, "%s, out of %d evaluated Integration Suite features.\n", strings.Join(summary, " · "), len(sorted))

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// regenerateReadme replaces the content strictly between readmeMarkerBegin
// and readmeMarkerEnd in path with a freshly generated feature overview,
// leaving every other line of the file untouched. It fails loudly if
// either marker is missing or out of order, rather than silently doing
// nothing or corrupting the file.
func regenerateReadme(path string) error {
	// path is always the readmePath constant declared above, never
	// external input, despite the parameter — see writeGeneratedFile's
	// doc comment for why that does not make this a path-traversal risk.
	original, err := os.ReadFile(path) //nolint:gosec // G304: path is always readmePath, a compile-time constant
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	begin := strings.Index(string(original), readmeMarkerBegin)
	end := strings.Index(string(original), readmeMarkerEnd)
	if begin == -1 || end == -1 || end < begin {
		return fmt.Errorf("%s is missing matching %q/%q markers", path, readmeMarkerBegin, readmeMarkerEnd)
	}

	before := string(original)[:begin+len(readmeMarkerBegin)]
	after := string(original)[end:]

	updated := before + "\n\n" + readmeFeatureOverview() + "\n" + after

	return writeGeneratedFile(path, updated)
}
