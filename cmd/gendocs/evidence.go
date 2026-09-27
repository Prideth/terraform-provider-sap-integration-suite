package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/features"
)

const evidencePath = "docs/research/capability-evidence-2026.md"

// classificationOrder fixes the section order of the evidence document and
// explains each classification in one line.
var classificationOrder = []struct {
	label, heading, meaning string
}{
	{"unofficial", "Unofficial", "Works and verified on a tenant, but the contract is known only from the service's $metadata; no SAP documentation, specification or tooling describes it."},
	{"experimental", "Experimental", "Implemented; the lifecycle still has to pass its acceptance test on a tenant."},
	{"partial", "Partial", "Implemented with a deliberate gap in the lifecycle."},
	{"read_only", "Read-only", "Data sources only, by design or because writes are undocumented."},
	{string(features.ReasonNotImplemented), "Public API, not implemented", "A confirmed public API the provider has not implemented yet."},
	{string(features.ReasonPublicAPIIncomplete), "Public API incomplete", "A public API exists, but part of its contract is unconfirmed."},
	{string(features.ReasonUnsafeTerraformLifecycle), "Unsafe Terraform lifecycle", "A public API exists, but a Terraform resource could not implement its lifecycle safely."},
	{string(features.ReasonResearchRequired), "Research required", "Not enough evidence yet to say whether a usable API exists."},
	{string(features.ReasonNoPublicAPI), "No public API", "Only UI procedures or internal endpoints, which the provider does not use."},
	{string(features.ReasonOutOfScope), "Out of scope", "Runtime data, workflows or imperative actions, not desired configuration."},
	{"separate_provider", "Separate provider", "Belongs to a separate, independently versioned provider."},
}

// evidenceDoc renders every feature that is not fully supported with its
// dated evidence record, grouped by classification.
func evidenceDoc() string {
	var b strings.Builder
	b.WriteString(`# Capability evidence, September 2026

<!-- Generated from internal/features (catalog.go and evidence.go) by "go run ./cmd/gendocs". Do not edit by hand. -->

This document records, for every catalog entry that is not fully supported,
what was checked, what it showed, and what would change its classification.
It is the reference behind the statuses in [feature-support.md](../feature-support.md);
the rationale for the method is in
[sap-2026-public-api-gap-closure.md](sap-2026-public-api-gap-closure.md), and the
entity-level view of each service contract is in
[api-discovery-report.md](../api-discovery-report.md).

Two rules hold throughout. An entity set in a service's ` + "`$metadata`" + ` shows that the
entity exists, not that SAP supports writing it: a mutable resource additionally
needs SAP documentation, SAP's own tooling, or a safe verification on a tenant.
And the provider never uses browser endpoints of SAP's UIs.

`)
	byClass := map[string][]features.Feature{}
	for _, f := range features.Catalog {
		if f.SupportStatus == features.StatusSupported {
			continue
		}
		c := features.Classification(f)
		byClass[c] = append(byClass[c], f)
	}

	b.WriteString("| Classification | Features |\n|---|---:|\n")
	for _, c := range classificationOrder {
		if n := len(byClass[c.label]); n > 0 {
			fmt.Fprintf(&b, "| %s (`%s`) | %d |\n", c.heading, c.label, n)
		}
	}
	b.WriteString("\n")

	for _, c := range classificationOrder {
		list := byClass[c.label]
		if len(list) == 0 {
			continue
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Key < list[j].Key })
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", c.heading, c.meaning)
		for _, f := range list {
			rec := features.Evidence[f.Key]
			fmt.Fprintf(&b, "### %s\n\n`%s` · checked %s\n\n", f.Name, f.Key, rec.CheckedOn)
			fmt.Fprintf(&b, "- **Finding:** %s\n- **Next step:** %s\n- **Sources:**\n", rec.Finding, rec.NextStep)
			for _, s := range rec.Sources {
				fmt.Fprintf(&b, "  - %s\n", s)
			}
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}
