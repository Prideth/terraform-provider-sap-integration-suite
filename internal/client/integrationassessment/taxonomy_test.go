package integrationassessment

import (
	"context"
	"testing"
)

func TestListIntegrationPatterns_ExpandsDomainAndStyle(t *testing.T) {
	c, got := fakeService(t, `{"d":{"results":[{"Id":"ip-1","Name":"Cloud process integration",`+
		`"Domain":{"Id":"dom-1","Name":"Cloud2Cloud"},"Style":{"Id":"sty-1","Name":"Process"}}]}}`)
	patterns, err := c.ListIntegrationPatterns(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(patterns) != 1 || patterns[0].DomainID() != "dom-1" || patterns[0].StyleID() != "sty-1" {
		t.Errorf("patterns = %+v", patterns)
	}
	if r := (*got)[0]; r.path != "/intas/entities/v1/IntegrationPattern" || r.query != "$expand=Domain,Style" {
		t.Errorf("request = %s?%s", r.path, r.query)
	}
}

func TestListUseCasePatterns_ExpandsTheStyle(t *testing.T) {
	c, got := fakeService(t, `{"d":{"results":[{"Id":"uc-1","Name":"Event streaming","Description":null,"Example":"IoT",`+
		`"Style":{"Id":"sty-2","Name":"Event"}}]}}`)
	patterns, err := c.ListUseCasePatterns(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(patterns) != 1 || patterns[0].StyleID() != "sty-2" || patterns[0].Example == nil || *patterns[0].Example != "IoT" {
		t.Errorf("patterns = %+v", patterns)
	}
	if r := (*got)[0]; r.query != "$expand=Style" {
		t.Errorf("query = %q", r.query)
	}
}

func TestListDomainDeterminations_ExpandsAllThreeLinks(t *testing.T) {
	c, got := fakeService(t, `{"d":{"results":[{"Id":"dd-1","Domain":{"Id":"dom-1"},`+
		`"SourceDeploymentModel":{"Id":"dm-cloud"},"TargetDeploymentModel":{"Id":"dm-onprem"}}]}}`)
	all, err := c.ListDomainDeterminations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	d := all[0]
	if d.DomainID() != "dom-1" || d.SourceDeploymentModelID() != "dm-cloud" || d.TargetDeploymentModelID() != "dm-onprem" {
		t.Errorf("determination = %+v", d)
	}
	if r := (*got)[0]; r.path != "/intas/entities/v1/DomainDetermination" ||
		r.query != "$expand=Domain,SourceDeploymentModel,TargetDeploymentModel" {
		t.Errorf("request = %s?%s", r.path, r.query)
	}
}

func TestListKeyCharacteristicGroups_IsAPlainList(t *testing.T) {
	c, got := fakeService(t, `{"d":{"results":[{"Id":"grp-1","Name":"Operations","Description":"Run"}]}}`)
	groups, err := c.ListKeyCharacteristicGroups(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Name != "Operations" {
		t.Errorf("groups = %+v", groups)
	}
	if r := (*got)[0]; r.path != "/intas/entities/v1/KeyCharacteristicGroup" || r.query != "" {
		t.Errorf("request = %s?%s", r.path, r.query)
	}
}
