package integrationassessment

import (
	"testing"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"
)

// TestWireContractAgainstMetadata checks Contract against the committed
// snapshot of the Entities API $metadata in testdata/api-metadata.
func TestWireContractAgainstMetadata(t *testing.T) {
	s, err := apimeta.LoadSnapshot(Contract.Service)
	if err != nil {
		t.Fatalf("loading the %s snapshot: %v", Contract.Service, err)
	}
	for _, problem := range apimeta.Verify(s, Contract) {
		t.Error(problem)
	}
}
