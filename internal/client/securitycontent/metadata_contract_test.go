package securitycontent

import (
	"testing"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"
)

// TestWireContractAgainstMetadata checks Contract (the wire structs, keys,
// navigation properties, operations and limits this client relies on)
// against the committed snapshot of the service's $metadata in
// testdata/api-metadata. The metadata acceptance test checks the same
// contract against a live tenant.
func TestWireContractAgainstMetadata(t *testing.T) {
	s, err := apimeta.LoadSnapshot(Contract.Service)
	if err != nil {
		t.Fatalf("loading the %s snapshot: %v", Contract.Service, err)
	}
	for _, problem := range apimeta.Verify(s, Contract) {
		t.Error(problem)
	}
}
