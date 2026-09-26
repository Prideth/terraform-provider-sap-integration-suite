package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/features"
)

// utf8BOM is the three-byte UTF-8 byte-order mark. Markdown renderers,
// "git diff", and this project's own Docs CI check all treat a file that
// starts with it as different from the byte-identical file without one,
// so a generator that reintroduces it — even just on one platform's shell
// — breaks the "regeneration produces zero diff" guarantee CI depends on.
// See writeGeneratedFile's doc comment for how this file avoids that: it
// always writes bytes directly with os.WriteFile, never through a shell
// redirection whose behavior can vary by platform.
const utf8BOM = "\xEF\xBB\xBF"

func TestFeatureSupportDoc_HasNoByteOrderMark(t *testing.T) {
	doc := featureSupportDoc()
	if strings.HasPrefix(doc, utf8BOM) {
		t.Fatal("featureSupportDoc() output starts with a UTF-8 byte-order mark")
	}
}

func TestReadmeFeatureOverview_HasNoByteOrderMark(t *testing.T) {
	overview := readmeFeatureOverview()
	if strings.HasPrefix(overview, utf8BOM) {
		t.Fatal("readmeFeatureOverview() output starts with a UTF-8 byte-order mark")
	}
}

// TestWriteGeneratedFile_ProducesNoByteOrderMark exercises the actual file
// write path (not just the in-memory string), since a byte-order mark
// could in principle be introduced at the encoding step rather than in the
// generated string itself.
func TestWriteGeneratedFile_ProducesNoByteOrderMark(t *testing.T) {
	path := filepath.Join(t.TempDir(), "feature-support.md")

	if err := writeGeneratedFile(path, featureSupportDoc()); err != nil {
		t.Fatalf("writeGeneratedFile() error: %v", err)
	}

	written, err := os.ReadFile(path) //nolint:gosec // G304: path is built from t.TempDir() above, not external input
	if err != nil {
		t.Fatalf("reading back %s: %v", path, err)
	}
	if strings.HasPrefix(string(written), utf8BOM) {
		t.Fatal("the written file starts with a UTF-8 byte-order mark")
	}
}

// TestRegenerateReadme_IsIdempotent proves running the README regeneration
// twice in a row produces byte-identical output both times: the generated
// block must not depend on anything that changes between runs (map
// iteration order, timestamps, and so on), the same determinism guarantee
// docs/feature-support.md depends on.
func TestRegenerateReadme_IsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "README.md")
	seed := "# Example\n\n" + readmeMarkerBegin + "\n" + readmeMarkerEnd + "\n\nTrailing content.\n"
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seeding %s: %v", path, err)
	}

	if err := regenerateReadme(path); err != nil {
		t.Fatalf("first regenerateReadme() error: %v", err)
	}
	first, err := os.ReadFile(path) //nolint:gosec // G304: path is built from t.TempDir() above, not external input
	if err != nil {
		t.Fatalf("reading back %s: %v", path, err)
	}

	if err := regenerateReadme(path); err != nil {
		t.Fatalf("second regenerateReadme() error: %v", err)
	}
	second, err := os.ReadFile(path) //nolint:gosec // G304: path is built from t.TempDir() above, not external input
	if err != nil {
		t.Fatalf("reading back %s: %v", path, err)
	}

	if string(first) != string(second) {
		t.Fatal("regenerateReadme() is not idempotent: two consecutive runs produced different bytes")
	}
	if !strings.Contains(string(first), "Trailing content.") {
		t.Fatal("regenerateReadme() must not touch content outside the marker block")
	}
}

// Every feature that is not fully supported appears in the evidence
// document under a known classification.
func TestEvidenceDoc_ListsEveryIncompleteFeature(t *testing.T) {
	doc := evidenceDoc()
	known := map[string]bool{}
	for _, c := range classificationOrder {
		known[c.label] = true
	}
	for _, f := range features.Catalog {
		if f.SupportStatus == features.StatusSupported {
			continue
		}
		if !known[features.Classification(f)] {
			t.Errorf("%s: classification %q has no section", f.Key, features.Classification(f))
		}
		if !strings.Contains(doc, "`"+f.Key+"`") {
			t.Errorf("%s is missing from the evidence document", f.Key)
		}
	}
	if strings.HasPrefix(doc, utf8BOM) {
		t.Error("the evidence document starts with a byte-order mark")
	}
}
