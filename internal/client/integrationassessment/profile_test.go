package integrationassessment

import (
	"context"
	"net/http"
	"testing"
)

func TestCreateTechnologyDomain_LinksBothByID(t *testing.T) {
	c, got := fakeService(t, `{"d":{"Id":"td-1","Technology":{"__deferred":{"uri":"x"}},"Domain":{"__deferred":{"uri":"x"}}}}`)
	d, err := c.CreateTechnologyDomain(context.Background(), "tech-1", "dom-1")
	if err != nil {
		t.Fatal(err)
	}
	r := (*got)[0]
	if r.method != http.MethodPost || r.path != "/intas/entities/v1/TechnologyDomain" || d.ID != "td-1" {
		t.Errorf("request = %s %s, created = %+v", r.method, r.path, d)
	}
	if linkID(r.body, "Technology") != "tech-1" || linkID(r.body, "Domain") != "dom-1" || len(r.body) != 2 {
		t.Errorf("body = %v, want exactly the two links", r.body)
	}
}

func TestGetTechnologyStyle_ExpandsItsLinks(t *testing.T) {
	c, got := fakeService(t, `{"d":{"Id":"ts-1","Technology":{"Id":"tech-1","Name":"ESB"},"Style":{"Id":"sty-1","Name":"Process"}}}`)
	s, err := c.GetTechnologyStyle(context.Background(), "ts-1")
	if err != nil {
		t.Fatal(err)
	}
	if s.TechnologyID() != "tech-1" || s.StyleID() != "sty-1" {
		t.Errorf("style = %+v", s)
	}
	if r := (*got)[0]; r.path != "/intas/entities/v1/TechnologyStyle('ts-1')" || r.query != "$expand=Technology,Style" {
		t.Errorf("request = %s?%s", r.path, r.query)
	}
}

func TestCreateTechnologyKeyCharacteristic_OmitsAnUnsetDescription(t *testing.T) {
	c, got := fakeService(t, `{"d":{"Id":"tk-1"}}`)
	if _, err := c.CreateTechnologyKeyCharacteristic(context.Background(), "tech-1", "val-1", "deg-1", nil); err != nil {
		t.Fatal(err)
	}
	r := (*got)[0]
	if linkID(r.body, "Technology") != "tech-1" || linkID(r.body, "KeyCharacteristicValue") != "val-1" ||
		linkID(r.body, "RecommendationDegree") != "deg-1" {
		t.Errorf("body = %v", r.body)
	}
	if _, present := r.body["Description"]; present {
		t.Errorf("body = %v, want no Description", r.body)
	}
}

func TestCreateTechnologyKeyCharacteristic_SendsTheDescription(t *testing.T) {
	c, got := fakeService(t, `{"d":{"Id":"tk-1"}}`)
	desc := "rated by the board"
	if _, err := c.CreateTechnologyKeyCharacteristic(context.Background(), "tech-1", "val-1", "deg-1", &desc); err != nil {
		t.Fatal(err)
	}
	if d := (*got)[0].body["Description"]; d != desc {
		t.Errorf("Description = %v", d)
	}
}

func TestListKeyCharacteristicValues_ExpandsTheKeyCharacteristic(t *testing.T) {
	c, got := fakeService(t, `{"d":{"results":[{"Id":"val-1","Name":"High","Description":null,"KeyCharacteristic":{"Id":"kc-1","Name":"Throughput"}}]}}`)
	values, err := c.ListKeyCharacteristicValues(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].KeyCharacteristicID() != "kc-1" {
		t.Errorf("values = %+v", values)
	}
	if r := (*got)[0]; r.path != "/intas/entities/v1/KeyCharacteristicValue" || r.query != "$expand=KeyCharacteristic" {
		t.Errorf("request = %s?%s", r.path, r.query)
	}
}

func TestDeleteTechnologyKeyCharacteristic(t *testing.T) {
	c, got := fakeService(t, "")
	if err := c.DeleteTechnologyKeyCharacteristic(context.Background(), "tk-1"); err != nil {
		t.Fatal(err)
	}
	if r := (*got)[0]; r.method != http.MethodDelete || r.path != "/intas/entities/v1/TechnologyKeyCharacteristic('tk-1')" {
		t.Errorf("request = %s %s", r.method, r.path)
	}
}
