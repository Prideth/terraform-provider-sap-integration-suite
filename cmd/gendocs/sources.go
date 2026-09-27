package main

import (
	"fmt"
	"strings"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/features"
)

// writeContractSources explains where each implemented feature's contract
// comes from and lists the operations that work but are not documented.
func writeContractSources(b *strings.Builder, sorted []features.Feature) {
	fmt.Fprintln(b, "## Contract sources")
	fmt.Fprintln(b)
	fmt.Fprintln(b, "Every implemented feature says where its contract comes from, because that decides how far "+
		"it can be trusted across SAP releases:")
	fmt.Fprintln(b)
	fmt.Fprintln(b, "| Source | Meaning | Highest status |")
	fmt.Fprintln(b, "|---|---|---|")
	fmt.Fprintln(b, "| `sap_documentation` | SAP Help describes the operations (example requests, the API's list of resources) | supported |")
	fmt.Fprintln(b, "| `api_specification` | The official API specification on the SAP Business Accelerator Hub | supported |")
	fmt.Fprintln(b, "| `sap_tooling` | SAP's own SDK or tooling sends these requests (API Management Client SDK, CI/CD actions, Project Piper) | supported |")
	fmt.Fprintln(b, "| `metadata_only` | Only the service's `$metadata` and tests on a tenant; nothing official | unofficial once verified, experimental before |")
	fmt.Fprintln(b)
	fmt.Fprintln(b, "**Unofficial** means that it works and was verified on a tenant, but SAP has not documented "+
		"it, so SAP may change it without notice. Individual operations of an otherwise documented "+
		"feature can be unofficial too:")
	fmt.Fprintln(b)
	fmt.Fprintln(b, "| Feature | Implemented operation that SAP does not document |")
	fmt.Fprintln(b, "|---|---|")
	for _, f := range sorted {
		for _, op := range f.UndocumentedOperations {
			fmt.Fprintf(b, "| `%s` | %s |\n", f.Key, op)
		}
	}
	fmt.Fprintln(b)
}

// sourceCell renders a feature's contract source for the overview table.
func sourceCell(f features.Feature) string {
	if f.ContractSource == "" {
		return "—"
	}
	if len(f.UndocumentedOperations) > 0 {
		return string(f.ContractSource) + " (some operations unofficial)"
	}
	return string(f.ContractSource)
}
