package apidiscovery

import (
	"context"
	"testing"
	"time"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/accgate"
)

// TestAccMetadata reads the live contract of every configured service and
// checks it in three layers:
//
//  1. reachability: the $metadata (or specification) document is readable
//     and parses;
//  2. contract: every client contract of the provider still holds;
//  3. discovery delta: the live contract is compared with the committed
//     snapshot. A breaking change to something the provider uses fails;
//     other changes are logged as warnings, or fail too with
//     SAP_INTEGRATION_SUITE_DISCOVERY_STRICT=1, which is how new SAP
//     entities are found (then run make api-metadata-refresh).
//
// It only reads. Services without credentials skip with the missing
// variable names.
func TestAccMetadata(t *testing.T) {
	for _, svc := range Services {
		t.Run(svc.ID, func(t *testing.T) {
			if svc.SpecificationOnly() {
				t.Skip("SKIPPED — contract comes only from the official specification: " + svc.Specification)
			}
			accgate.RequireService(t, accgate.Metadata, svc.Env...)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			res, err := Fetch(ctx, svc)
			if err != nil {
				t.Fatalf("layer 1, reachability: %v", err)
			}
			live := res.Service
			t.Logf("layer 1: %d entity sets, %d entity types, %d operations", len(live.EntitySets), len(live.EntityTypes), len(live.Operations))
			for _, issue := range res.ServiceDocumentIssues {
				t.Logf("warning: %s", issue)
			}

			for _, contract := range ContractsFor(svc.ID) {
				for _, p := range apimeta.Verify(live, contract) {
					t.Errorf("layer 2, contract of %s: %s", contract.Package, p)
				}
			}

			snapshot, err := apimeta.LoadSnapshot(svc.ID)
			if err != nil {
				msg := "layer 3: no committed snapshot for " + svc.ID + "; create one with make api-metadata-refresh"
				if accgate.Strict() {
					t.Fatal(msg)
				}
				t.Log("warning: " + msg)
				return
			}
			cmp := Compare(snapshot, live)
			for _, c := range cmp.ProviderBreaking {
				t.Errorf("layer 3, breaking change to something the provider uses:\n%s", c)
			}
			for _, c := range cmp.Changes {
				if contains(cmp.ProviderBreaking, c) {
					continue
				}
				if accgate.Strict() {
					t.Errorf("layer 3, difference from the snapshot:\n%s", c)
				} else {
					t.Logf("warning, difference from the snapshot:\n%s", c)
				}
			}
			if len(cmp.Changes) > 0 {
				t.Logf("%d differences (%d additive, %d breaking); investigate them, classify new entity sets, "+
					"then accept them with make api-metadata-refresh", len(cmp.Changes), len(cmp.Additive), len(cmp.Breaking))
			}
		})
	}
}

func contains(list []apimeta.Change, c apimeta.Change) bool {
	for _, x := range list {
		if x.Kind == c.Kind && x.Subject == c.Subject && x.Path == c.Path {
			return true
		}
	}
	return false
}

// TestServiceCapabilitiesAreGates keeps the capabilities named by the
// services in step with the acceptance gates.
func TestServiceCapabilitiesAreGates(t *testing.T) {
	for _, svc := range Services {
		for _, c := range svc.Capabilities {
			if _, ok := accgate.Lookup(accgate.Capability(c)); !ok {
				t.Errorf("%s names capability %s, which has no acceptance gate", svc.ID, c)
			}
		}
	}
}
