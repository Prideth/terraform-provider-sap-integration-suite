package features

import (
	"strings"
	"testing"
	"time"
)

func TestCatalog_KeysAreNonEmptyAndUnique(t *testing.T) {
	seen := make(map[string]bool, len(Catalog))
	for _, f := range Catalog {
		if f.Key == "" {
			t.Errorf("feature %q (%s) has an empty Key", f.Name, f.Domain)
			continue
		}
		if seen[f.Key] {
			t.Errorf("duplicate feature key %q", f.Key)
		}
		seen[f.Key] = true
	}
}

func TestCatalog_RequiredFieldsAreNonEmpty(t *testing.T) {
	for _, f := range Catalog {
		if f.Domain == "" {
			t.Errorf("feature %q has an empty Domain", f.Key)
		}
		if f.Name == "" {
			t.Errorf("feature %q has an empty Name", f.Key)
		}
		if f.Description == "" {
			t.Errorf("feature %q has an empty Description", f.Key)
		}
	}
}

func TestCatalog_SupportStatusAndReasonAreValidEnumValues(t *testing.T) {
	for _, f := range Catalog {
		if !f.SupportStatus.Valid() {
			t.Errorf("feature %q has an invalid SupportStatus %q", f.Key, f.SupportStatus)
		}
		if !f.SupportReason.Valid() {
			t.Errorf("feature %q has an invalid SupportReason %q", f.Key, f.SupportReason)
		}
	}
}

// TestCatalog_ReasonPresenceMatchesStatus locks in the rule described on
// Feature.SupportReason: a fully supported feature has no reason to give
// (there is nothing to explain), and anything else must explain itself.
// This is the check that would catch a copy-paste feature entry left with
// a stale or missing reason.
func TestCatalog_ReasonPresenceMatchesStatus(t *testing.T) {
	for _, f := range Catalog {
		switch f.SupportStatus {
		case StatusSupported:
			if f.SupportReason != "" {
				t.Errorf("feature %q is %q but has a SupportReason %q; supported features should not need one", f.Key, f.SupportStatus, f.SupportReason)
			}
		default:
			if f.SupportReason == "" {
				t.Errorf("feature %q is %q but has no SupportReason explaining why", f.Key, f.SupportStatus)
			}
		}
	}
}

// TestCatalog_TypesAreProviderPrefixed catches an entry that names a
// resource or data source type with a typo or a missing provider prefix,
// which would otherwise silently fail to match anything real.
func TestCatalog_TypesAreProviderPrefixed(t *testing.T) {
	const prefix = "sapintegrationsuite_"
	for _, f := range Catalog {
		for _, rt := range f.ResourceTypes {
			if !strings.HasPrefix(rt, prefix) {
				t.Errorf("feature %q lists resource type %q without the %q prefix", f.Key, rt, prefix)
			}
		}
		for _, dt := range f.DataSourceTypes {
			if !strings.HasPrefix(dt, prefix) {
				t.Errorf("feature %q lists data source type %q without the %q prefix", f.Key, dt, prefix)
			}
		}
	}
}

// TestCatalog_UnimplementedFeaturesClaimNoTerraformTypes catches the
// inverse of the usual mistake: a feature marked unsupported (no Terraform
// resource exists yet) should not simultaneously claim resource or data
// source types, since nothing would actually back them.
func TestCatalog_UnimplementedFeaturesClaimNoTerraformTypes(t *testing.T) {
	for _, f := range Catalog {
		if f.SupportStatus != StatusUnsupported {
			continue
		}
		if len(f.ResourceTypes) > 0 {
			t.Errorf("feature %q is unsupported but lists resource types %v", f.Key, f.ResourceTypes)
		}
		if len(f.DataSourceTypes) > 0 {
			t.Errorf("feature %q is unsupported but lists data source types %v", f.Key, f.DataSourceTypes)
		}
	}
}

// TestCatalog_PublicAPIFalseImpliesNoAPIProtocol catches an entry that
// claims a specific wire protocol for an API this catalog otherwise says
// does not exist.
func TestCatalog_PublicAPIFalseImpliesNoAPIProtocol(t *testing.T) {
	for _, f := range Catalog {
		if !f.PublicAPI && f.APIProtocol != "" {
			t.Errorf("feature %q has PublicAPI=false but APIProtocol=%q", f.Key, f.APIProtocol)
		}
	}
}

func TestLookup(t *testing.T) {
	f, ok := Lookup("cloud_integration.value_mapping")
	if !ok {
		t.Fatal("Lookup() did not find cloud_integration.value_mapping")
	}
	if f.SupportStatus != StatusPartial {
		t.Errorf("SupportStatus = %q, want %q", f.SupportStatus, StatusPartial)
	}

	if _, ok := Lookup("cloud_integration.does_not_exist"); ok {
		t.Error("Lookup() unexpectedly found a feature for an unknown key")
	}
}

// TestEvidence_EveryIncompleteFeatureHasARecord keeps the evidence document
// complete: every feature that is not fully supported needs a dated record
// with sources, a finding and the next step, and no record may outlive its
// feature or describe a supported one.
func TestEvidence_EveryIncompleteFeatureHasARecord(t *testing.T) {
	keys := map[string]Feature{}
	for _, f := range Catalog {
		keys[f.Key] = f
		if f.SupportStatus == StatusSupported {
			continue
		}
		rec, ok := Evidence[f.Key]
		if !ok {
			t.Errorf("%s (%s) has no evidence record", f.Key, Classification(f))
			continue
		}
		if _, err := time.Parse("2006-01-02", rec.CheckedOn); err != nil {
			t.Errorf("%s: CheckedOn %q is not a date", f.Key, rec.CheckedOn)
		}
		if len(rec.Sources) == 0 || rec.Finding == "" || rec.NextStep == "" {
			t.Errorf("%s: evidence record needs sources, a finding and a next step", f.Key)
		}
	}
	for key := range Evidence {
		f, ok := keys[key]
		if !ok {
			t.Errorf("evidence record %s has no catalog entry", key)
		} else if f.SupportStatus == StatusSupported {
			t.Errorf("evidence record %s describes a supported feature; remove it", key)
		}
	}
}
