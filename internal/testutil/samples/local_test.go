package samples_test

import (
	"strings"
	"testing"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/samples"
)

// exportTypeToBundleType maps an export's resourceType to the SAP-BundleType
// the artifact's MANIFEST must carry.
var exportTypeToBundleType = map[string]string{
	"IFlow":            "IntegrationFlow",
	"MessageMapping":   "MessageMapping",
	"ScriptCollection": "ScriptCollection",
	"ValueMapping":     "ValueMapping",
}

// Every artifact in the local content (SAP_LOCAL_CONTENT_DIR) goes through
// the provider's manifest parser: the type must match what the export says,
// and the symbolic name must be a usable artifact ID. Skipped without the
// directory; the content itself never enters the repository.
func TestLocalContent_Manifests(t *testing.T) {
	artifacts := samples.LocalArtifacts(t)
	counts := map[string]int{}
	for _, a := range artifacts {
		if strings.HasPrefix(a.Type, "UnreadableExport") {
			t.Errorf("%s: %s", a.Source, a.Type)
			continue
		}
		m, err := cloudintegration.ParseBundleManifest(a.Content)
		if err != nil {
			t.Errorf("%s (%s): %v", a.Source, a.Name, err)
			continue
		}
		counts[m.BundleType]++
		if want, ok := exportTypeToBundleType[a.Type]; ok && m.BundleType != want {
			t.Errorf("%s (%s): export says %s, MANIFEST says %q", a.Source, a.Name, a.Type, m.BundleType)
		}
		if m.SymbolicName == "" || strings.ContainsAny(m.SymbolicName, " ;\t") {
			t.Errorf("%s (%s): symbolic name %q", a.Source, a.Name, m.SymbolicName)
		}
	}
	t.Logf("local artifacts by bundle type: %v", counts)
	if len(artifacts) == 0 {
		t.Error("no artifacts found")
	}
}

// Renaming works for every local script collection, message mapping and
// value mapping, including their Provide-Capability header.
func TestLocalContent_WithBundleID(t *testing.T) {
	for _, a := range samples.LocalArtifacts(t) {
		if a.Type != "ScriptCollection" && a.Type != "MessageMapping" && a.Type != "ValueMapping" {
			continue
		}
		rewritten, err := samples.WithBundleID(a.Content, "tfacc_renamed")
		if err != nil {
			t.Fatalf("%s: %v", a.Name, err)
		}
		m, err := cloudintegration.ParseBundleManifest(rewritten)
		if err != nil {
			t.Fatalf("%s: %v", a.Name, err)
		}
		if m.SymbolicName != "tfacc_renamed" {
			t.Errorf("%s: symbolic name %q after rename", a.Name, m.SymbolicName)
		}
		if capability := m.Headers["Provide-Capability"]; capability != "" && !strings.Contains(capability, ".tfacc_renamed;") {
			t.Errorf("%s: Provide-Capability %q still names the old ID", a.Name, capability)
		}
	}
}

// Live tests upload made-up value mapping entries instead of real ones.
func TestLocalContent_SyntheticValueMapping(t *testing.T) {
	vm := samples.LocalArtifactOfType(t, "ValueMapping")
	synthetic, err := samples.SyntheticValueMapping(vm.Content, "x")
	if err != nil {
		t.Fatal(err)
	}
	m, err := cloudintegration.ParseBundleManifest(synthetic)
	if err != nil || m.BundleType != "ValueMapping" {
		t.Fatalf("synthetic value mapping: %+v, %v", m, err)
	}
	names, _ := samples.FileNames(synthetic)
	for _, n := range names {
		if n == "value_mapping.xml" {
			return
		}
	}
	t.Errorf("synthetic value mapping lost value_mapping.xml: %v", names)
}
