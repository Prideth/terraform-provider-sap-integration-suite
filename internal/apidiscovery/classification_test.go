package apidiscovery

import (
	"context"
	"testing"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/features"
)

// TestEverySnapshotEntitySetIsClassified fails as soon as a snapshot holds
// an entity set or operation that the provider neither uses nor lists as a
// candidate or a deliberate exclusion. A refreshed snapshot with new SAP
// entities therefore cannot pass CI until someone has looked at them.
func TestEverySnapshotEntitySetIsClassified(t *testing.T) {
	for _, svc := range Services {
		s, err := apimeta.LoadSnapshot(svc.ID)
		if err != nil {
			continue // no snapshot for this service yet
		}
		for _, row := range Classify(s) {
			if row.Status == StatusUnclassified {
				t.Errorf("%s: %s %s is not classified; add it to the classification of %s", svc.ID, row.Kind, row.Name, svc.ID)
			}
		}
	}
}

// TestClassificationHasNoStaleEntries keeps the classification honest: an
// entry that no longer matches anything in its snapshot, or that names an
// entity set the provider already uses, is an error.
func TestClassificationHasNoStaleEntries(t *testing.T) {
	for id, rules := range classification {
		s, err := apimeta.LoadSnapshot(id)
		if err != nil {
			t.Errorf("classification for %s, but no snapshot", id)
			continue
		}
		used := UsageOf(id)
		for _, r := range rules {
			matched := false
			for _, es := range s.EntitySets {
				if r.matches(es.Name) {
					matched = true
					if _, ok := used.EntitySets[es.Name]; ok && r.Prefix == "" {
						t.Errorf("%s: %s is used by the provider but classified as %s", id, es.Name, r.Status)
					}
				}
			}
			for _, op := range s.Operations {
				if r.matches(op.Name) {
					matched = true
				}
			}
			if !matched {
				t.Errorf("%s: classification entry %q matches nothing in the snapshot", id, r.Name+r.Prefix)
			}
		}
	}
}

// TestCandidatesNameCatalogEntries checks that every candidate points at a
// feature key of the catalog, so the report can link the entity to the
// support status users see.
func TestCandidatesNameCatalogEntries(t *testing.T) {
	keys := map[string]bool{}
	for _, f := range features.Catalog {
		keys[f.Key] = true
	}
	for id, rules := range classification {
		for _, r := range rules {
			if r.Status == StatusCandidate && !keys[r.Feature] {
				t.Errorf("%s: candidate %s names feature %q, which the catalog does not have", id, r.Name+r.Prefix, r.Feature)
			}
			if r.Status == StatusExcluded && r.Reason == "" {
				t.Errorf("%s: exclusion %s has no reason", id, r.Name+r.Prefix)
			}
		}
	}
}

// A REST contract's operations are classified like entity sets, so an
// OpenAPI snapshot with an unclassified operation fails the test above.
func TestClassifyRESTOperations(t *testing.T) {
	s := &apimeta.Service{ID: "rest-example", Protocol: apimeta.ProtocolOpenAPI, REST: &apimeta.RESTContract{
		Operations: []apimeta.RESTOperation{{Method: "POST", Path: "/assets", OperationID: "createAsset"}},
	}}
	rows := Classify(s)
	if len(rows) != 1 || rows[0].Kind != "rest operation" || rows[0].Name != "POST /assets" || rows[0].Status != StatusUnclassified {
		t.Fatalf("rows = %+v", rows)
	}
}

// Services whose contract is only published as a download never call a
// service and report the document to fetch.
func TestSpecificationOnlyServices(t *testing.T) {
	found := 0
	for _, svc := range Services {
		if !svc.SpecificationOnly() {
			if svc.Specification != "" {
				t.Errorf("%s has both a live root and a specification", svc.ID)
			}
			continue
		}
		found++
		if svc.Specification == "" {
			t.Errorf("%s: no specification named", svc.ID)
		}
		if ok, missing := svc.Configured(); ok || len(missing) != 1 {
			t.Errorf("%s: Configured() = %v, %v", svc.ID, ok, missing)
		}
		if _, err := Fetch(context.Background(), svc); err == nil {
			t.Errorf("%s: Fetch must refuse a specification-only service", svc.ID)
		}
	}
	if found == 0 {
		t.Fatal("no specification-only service; expected the REST APIs of the API portal and DSIAPI")
	}
}
