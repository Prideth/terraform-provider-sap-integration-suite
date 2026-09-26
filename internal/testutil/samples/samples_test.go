package samples_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/samples"
)

// Every pinned sample downloads, matches its hash and has the structure its
// kind promises. The artifact ZIPs go through the provider's own manifest
// parser, so SAP's real files check it, including the wrapped lines.
func TestCatalog(t *testing.T) {
	for _, s := range samples.Catalog {
		t.Run(s.Name, func(t *testing.T) {
			data := samples.Get(t, s.Name)
			switch s.Kind {
			case samples.KindIntegrationFlow, samples.KindMessageMapping:
				m, err := cloudintegration.ParseBundleManifest(data)
				if err != nil {
					t.Fatal(err)
				}
				if m.BundleType != string(s.Kind) || m.SymbolicName != s.BundleID {
					t.Errorf("manifest: type %q, symbolic name %q; catalog says %s %q", m.BundleType, m.SymbolicName, s.Kind, s.BundleID)
				}
			case samples.KindPackageExport:
				checkExport(t, data)
			case samples.KindAPIProxy:
				requireEntry(t, data, "APIProxy/")
			case samples.KindPolicyTemplate:
				requireEntry(t, data, "PolicyTemplateContainer/")
			default:
				t.Fatalf("unknown kind %q", s.Kind)
			}
		})
	}
}

func checkExport(t *testing.T, data []byte) {
	t.Helper()
	artifacts, err := samples.ExportArtifacts(data)
	if err != nil {
		t.Fatal(err)
	}
	var flows int
	for _, a := range artifacts {
		if a.Type != "IFlow" {
			continue
		}
		flows++
		m, err := cloudintegration.ParseBundleManifest(a.Content)
		if err != nil {
			t.Fatalf("%s: %v", a.DisplayName, err)
		}
		if m.BundleType != "IntegrationFlow" || m.SymbolicName == "" || strings.ContainsAny(m.SymbolicName, " ;") {
			t.Errorf("%s: manifest type %q, symbolic name %q", a.DisplayName, m.BundleType, m.SymbolicName)
		}
	}
	if flows == 0 {
		t.Error("export contains no integration flow")
	}
}

func requireEntry(t *testing.T, data []byte, prefix string) {
	t.Helper()
	names, err := samples.FileNames(data)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(names, func(n string) bool { return strings.HasPrefix(n, prefix) }) {
		t.Errorf("no entry under %s in %v", prefix, names)
	}
}

// One of SAP's exports wraps Bundle-SymbolicName over two manifest lines; the
// parser must return the whole name.
func TestCodejamExport_WrappedSymbolicName(t *testing.T) {
	artifacts, err := samples.ExportArtifacts(samples.Get(t, "codejam-package-export"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, a := range artifacts {
		if a.Type == "IFlow" {
			m, err := cloudintegration.ParseBundleManifest(a.Content)
			if err != nil {
				t.Fatal(err)
			}
			names = append(names, m.SymbolicName)
		}
	}
	if !slices.Contains(names, "Request_Employee_Dependants_-_Cloud_Connector_Export") {
		t.Errorf("symbolic names = %v, want the wrapped one joined", names)
	}
}

// WithBundleID gives live tests unique artifact IDs; the result must parse
// back to the new ID, keep everything else, and wrap long lines.
func TestWithBundleID_RoundTrip(t *testing.T) {
	longID := "tfacc_" + strings.Repeat("x", 80)
	for _, name := range []string{"spend-account-dim-map", "spend-step-0-flow"} {
		t.Run(name, func(t *testing.T) {
			s, _ := samples.Lookup(name)
			original := samples.Get(t, name)
			rewritten, err := samples.WithBundleID(original, longID)
			if err != nil {
				t.Fatal(err)
			}
			m, err := cloudintegration.ParseBundleManifest(rewritten)
			if err != nil {
				t.Fatal(err)
			}
			if m.SymbolicName != longID || m.Name != longID || m.BundleType != string(s.Kind) {
				t.Errorf("rewritten manifest = %+v", m)
			}
			if s.Kind == samples.KindMessageMapping && !strings.Contains(m.Headers["Provide-Capability"], "messagemapping."+longID+";") {
				t.Errorf("Provide-Capability = %q, want the new ID", m.Headers["Provide-Capability"])
			}
			before, _ := samples.FileNames(original)
			after, _ := samples.FileNames(rewritten)
			slices.Sort(before)
			slices.Sort(after)
			if !slices.Equal(before, after) {
				t.Errorf("entries changed: %v -> %v", before, after)
			}
		})
	}
}

// The provider's AlignBundleID works on SAP's real artifacts: a flow with a
// wrapped symbolic name and a message mapping.
func TestAlignBundleID_Samples(t *testing.T) {
	flows, err := samples.ExportArtifacts(samples.Get(t, "codejam-package-export"))
	if err != nil {
		t.Fatal(err)
	}
	inputs := [][]byte{samples.Get(t, "spend-account-dim-map")}
	for _, a := range flows {
		if a.Type == "IFlow" {
			inputs = append(inputs, a.Content)
		}
	}
	for i, content := range inputs {
		aligned, previous, err := cloudintegration.AlignBundleID(content, "tfacc_aligned")
		if err != nil {
			t.Fatalf("input %d: %v", i, err)
		}
		m, err := cloudintegration.ParseBundleManifest(aligned)
		if err != nil || m.SymbolicName != "tfacc_aligned" || previous == "tfacc_aligned" {
			t.Errorf("input %d: symbolic name %q (was %q), err %v", i, m.SymbolicName, previous, err)
		}
		before, _ := samples.FileNames(content)
		after, _ := samples.FileNames(aligned)
		if len(before) != len(after) {
			t.Errorf("input %d: %d entries became %d", i, len(before), len(after))
		}
	}
}
