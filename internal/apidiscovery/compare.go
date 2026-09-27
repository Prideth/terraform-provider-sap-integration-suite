package apidiscovery

import (
	"sort"
	"strings"
	"time"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apimanagementclassic"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/integrationassessment"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/partnerdirectory"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/securitycontent"
)

// Contracts returns every client contract the provider declares.
func Contracts() []apimeta.Contract {
	return []apimeta.Contract{
		cloudintegration.Contract,
		securitycontent.Contract,
		partnerdirectory.Contract,
		apimanagementclassic.Contract,
		integrationassessment.Contract,
	}
}

// ContractsFor returns the contracts that target one service.
func ContractsFor(serviceID string) []apimeta.Contract {
	var out []apimeta.Contract
	for _, c := range Contracts() {
		if c.Service == serviceID {
			out = append(out, c)
		}
	}
	return out
}

// Usage is what the provider uses of one service.
type Usage struct {
	// EntitySets maps an entity set to the client packages using it.
	EntitySets map[string][]string
	// Operations maps an operation to the client packages calling it.
	Operations map[string][]string
}

// UsageOf collects the entity sets and operations the provider's client
// contracts use in a service.
func UsageOf(serviceID string) Usage {
	u := Usage{EntitySets: map[string][]string{}, Operations: map[string][]string{}}
	for _, c := range ContractsFor(serviceID) {
		for _, es := range c.EntitySets() {
			u.EntitySets[es] = appendUnique(u.EntitySets[es], c.Package)
		}
		for _, op := range c.OperationNames() {
			u.Operations[op] = appendUnique(u.Operations[op], c.Package)
		}
	}
	return u
}

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

// Comparison is the outcome of comparing a live contract with its snapshot.
type Comparison struct {
	ServiceID string
	Changes   []apimeta.Change
	// Additive changes add something; the provider is unaffected.
	Additive []apimeta.Change
	// Breaking changes can break any client; ProviderBreaking are those
	// that touch what this provider uses and fail the contract suite.
	Breaking         []apimeta.Change
	ProviderBreaking []apimeta.Change
	// ContractProblems are the provider contracts checked against the live
	// contract directly.
	ContractProblems []string
}

// Compare diffs a live contract against its snapshot and checks the
// provider's contracts against the live contract.
func Compare(snapshot, live *apimeta.Service) Comparison {
	c := Comparison{ServiceID: live.ID, Changes: apimeta.Diff(snapshot, live)}
	used := usedNames(live.ID, snapshot, live)
	for _, ch := range c.Changes {
		switch {
		case ch.Breaking:
			c.Breaking = append(c.Breaking, ch)
			if used.touches(ch) {
				c.ProviderBreaking = append(c.ProviderBreaking, ch)
			}
		case ch.Kind == apimeta.ChangeNew:
			c.Additive = append(c.Additive, ch)
		}
	}
	for _, contract := range ContractsFor(live.ID) {
		for _, p := range apimeta.Verify(live, contract) {
			c.ContractProblems = append(c.ContractProblems, contract.Package+": "+p)
		}
	}
	sort.Strings(c.ContractProblems)
	return c
}

type usedSet struct {
	sets, types, operations map[string]bool
}

// usedNames resolves the entity sets the provider uses to their entity
// types in both contracts, so that a change to a type is recognised as
// touching the provider.
func usedNames(serviceID string, services ...*apimeta.Service) usedSet {
	u := UsageOf(serviceID)
	out := usedSet{sets: map[string]bool{}, types: map[string]bool{}, operations: map[string]bool{}}
	for es := range u.EntitySets {
		out.sets[es] = true
	}
	for op := range u.Operations {
		out.operations[op] = true
	}
	for _, s := range services {
		for _, es := range s.EntitySets {
			if out.sets[es.EntityType] || out.sets[es.Name] {
				out.types[es.EntityType] = true
			}
		}
	}
	return out
}

func (u usedSet) touches(c apimeta.Change) bool {
	switch {
	case strings.Contains(c.Subject, "ENTITY SET"):
		return u.sets[c.Path]
	case strings.Contains(c.Subject, "FUNCTION IMPORT"), strings.Contains(c.Subject, "ACTION IMPORT"):
		return u.operations[c.Path]
	}
	for t := range u.types {
		if c.Path == t || strings.HasPrefix(c.Path, t+".") || strings.HasPrefix(c.Path, t+" @") {
			return true
		}
	}
	return false
}

// CarryCapturedAt keeps the snapshot's capture date when nothing changed,
// so that a refresh without changes leaves the file byte-identical, and
// sets today's date otherwise.
func CarryCapturedAt(snapshot, live *apimeta.Service, changed bool) {
	if !changed && snapshot != nil && snapshot.CapturedAt != "" {
		live.CapturedAt = snapshot.CapturedAt
		return
	}
	live.CapturedAt = time.Now().UTC().Format("2006-01-02")
}
