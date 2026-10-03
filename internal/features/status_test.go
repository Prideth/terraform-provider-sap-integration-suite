package features

import "testing"

// The machine-readable values are part of the provider's interface: the
// feature data sources expose them as support_status, so they must not
// change when a label or an icon does.
func TestSupportStatus_ValuesAreStable(t *testing.T) {
	want := map[SupportStatus]string{
		StatusSupported:        "supported",
		StatusPartial:          "partial",
		StatusReadOnly:         "read_only",
		StatusExperimental:     "experimental",
		StatusUnofficial:       "unofficial",
		StatusResearchRequired: "research_required",
		StatusUnsupported:      "unsupported",
		StatusSeparateProvider: "separate_provider",
	}
	for status, value := range want {
		if string(status) != value {
			t.Errorf("status %q changed its value, want %q", status, value)
		}
		if !status.Valid() {
			t.Errorf("status %q is not Valid()", status)
		}
	}
}

// The legend lists every status exactly once, in the documented order, and
// no two statuses share an icon or a label.
func TestStatusLegend_CoversEveryStatusOnceInOrder(t *testing.T) {
	order := []SupportStatus{
		StatusSupported, StatusPartial, StatusReadOnly, StatusExperimental,
		StatusUnofficial, StatusResearchRequired, StatusUnsupported, StatusSeparateProvider,
	}
	if len(StatusLegend) != len(order) {
		t.Fatalf("StatusLegend has %d entries, want %d", len(StatusLegend), len(order))
	}
	icons, labels := map[string]bool{}, map[string]bool{}
	for i, info := range StatusLegend {
		if info.Status != order[i] {
			t.Errorf("StatusLegend[%d] = %q, want %q", i, info.Status, order[i])
		}
		if info.Icon == "" || info.Label == "" || info.Definition == "" {
			t.Errorf("%q needs an icon, a label and a definition", info.Status)
		}
		if icons[info.Icon] {
			t.Errorf("icon %s is used twice", info.Icon)
		}
		if labels[info.Label] {
			t.Errorf("label %q is used twice", info.Label)
		}
		icons[info.Icon], labels[info.Label] = true, true
	}
}

// Research required, experimental and unofficial are three different
// states and keep their own icons and definitions.
func TestStatusLegend_KeepsResearchExperimentalAndUnofficialApart(t *testing.T) {
	cases := []struct {
		status            SupportStatus
		icon, label, text string
	}{
		{StatusExperimental, "🧪", "Experimental",
			"Implementation exists, but its lifecycle has not yet been sufficiently validated against a real SAP tenant."},
		{StatusUnofficial, "🧭", "Unofficial",
			"Implementation has been validated, but relies on an API contract that SAP does not fully publish or officially document."},
		{StatusResearchRequired, "🔬", "Research required",
			"The capability has been identified, but its public API coverage, lifecycle semantics, or suitability for Terraform still requires further investigation."},
	}
	for _, c := range cases {
		info := c.status.Info()
		if info.Icon != c.icon || info.Label != c.label || info.Definition != c.text {
			t.Errorf("%q = %s %q %q, want %s %q %q", c.status, info.Icon, info.Label, info.Definition, c.icon, c.label, c.text)
		}
	}
}

// The catalog keeps the three statuses apart as well: research required
// means no implementation yet, experimental means an implementation that
// has not passed on a tenant, and an unsupported feature records a finished
// investigation, never an open one.
func TestCatalog_ResearchExperimentalAndUnsupportedStayDistinct(t *testing.T) {
	for _, f := range Catalog {
		implemented := len(f.ResourceTypes)+len(f.DataSourceTypes) > 0
		switch f.SupportStatus {
		case StatusResearchRequired:
			if implemented {
				t.Errorf("%s is research_required but registers Terraform types; an implementation makes it experimental", f.Key)
			}
		case StatusExperimental:
			if !implemented {
				t.Errorf("%s is experimental but registers no Terraform type; without an implementation it is research_required", f.Key)
			}
		}
		if f.SupportReason == ReasonResearchRequired && f.SupportStatus != StatusResearchRequired {
			t.Errorf("%s has the reason research_required but the status %q; open research means the status research_required", f.Key, f.SupportStatus)
		}
	}
}

func TestSupportStatus_InfoOfAnUnknownValue(t *testing.T) {
	if info := SupportStatus("nonsense").Info(); info.Icon != "?" || info.Label != "nonsense" {
		t.Errorf("Info() of an unknown status = %+v", info)
	}
}
