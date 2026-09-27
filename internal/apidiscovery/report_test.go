package apidiscovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"
)

// TestReportIsUpToDate requires the committed discovery report to match what
// the snapshots, contracts, classification and catalog produce, so the
// report can never describe a state the code does not have.
func TestReportIsUpToDate(t *testing.T) {
	dir, err := apimeta.SnapshotDir()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(filepath.Dir(dir)) // testdata/api-metadata -> repository root
	committed, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ReportPath)))
	if err != nil {
		t.Fatalf("reading %s: %v (generate it with make api-discovery-report)", ReportPath, err)
	}
	got := strings.ReplaceAll(string(committed), "\r\n", "\n")
	if got != RenderReport() {
		t.Errorf("%s is out of date; regenerate it with: go run ./cmd/apidiscovery -offline -report %s", ReportPath, ReportPath)
	}
}

func TestReportHasNoURLs(t *testing.T) {
	if r := RenderReport(); apimeta.RedactURLs(r) != r {
		t.Error("the report contains a URL")
	}
}

func TestNotInServiceDocument(t *testing.T) {
	s := &apimeta.Service{EntitySets: []apimeta.EntitySet{{Name: "A"}, {Name: "B"}, {Name: "C"}}}
	if got := notInServiceDocument(s); got != nil {
		t.Errorf("without a service document: %v", got)
	}
	s.ServiceDocument = []string{"A", "C"}
	if got := notInServiceDocument(s); len(got) != 1 || got[0] != "B" {
		t.Errorf("got %v, want [B]", got)
	}
}
