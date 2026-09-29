package integrationassessment

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// recorded is one request the fake service received.
type recorded struct {
	method, path, query, contentType string
	body                             map[string]any
}

// fakeService answers like the tenant did in September 2026: 201 with the
// entity on create, 204 on update and delete, the entity on read.
func fakeService(t *testing.T, reply string) (*Client, *[]recorded) {
	t.Helper()
	var got []recorded
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := recorded{method: r.Method, path: r.URL.Path, query: r.URL.RawQuery, contentType: r.Header.Get("Content-Type")}
		if b, _ := io.ReadAll(r.Body); len(b) > 0 {
			if err := json.Unmarshal(b, &rec.body); err != nil {
				t.Errorf("request body is not JSON: %s", b)
			}
		}
		got = append(got, rec)
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(reply))
		case http.MethodGet:
			_, _ = w.Write([]byte(reply))
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(srv.Close)
	return New(http.DefaultClient, srv.URL+"/intas/entities/v1/"), &got
}

func TestCreateApplication_LinksTheVendorByID(t *testing.T) {
	c, got := fakeService(t, `{"d":{"Id":"app-1","Name":"ERP","Vendor":{"__deferred":{"uri":"x"}}}}`)
	a, err := c.CreateApplication(context.Background(), "ERP", "vendor-1")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "app-1" || a.VendorID() != "" {
		t.Errorf("created = %+v; a deferred link must not look like a vendor Id", a)
	}
	r := (*got)[0]
	if r.path != "/intas/entities/v1/Application" || r.contentType != "application/json" {
		t.Errorf("request = %s %s (%s)", r.method, r.path, r.contentType)
	}
	if v, _ := r.body["Vendor"].(map[string]any); v["Id"] != "vendor-1" || len(v) != 1 {
		t.Errorf("Vendor = %v, want {\"Id\": \"vendor-1\"}; the service rejects __metadata links", r.body["Vendor"])
	}
}

func TestCreateApplication_WithoutVendorOmitsTheLink(t *testing.T) {
	c, got := fakeService(t, `{"d":{"Id":"app-1","Name":"ERP"}}`)
	if _, err := c.CreateApplication(context.Background(), "ERP", ""); err != nil {
		t.Fatal(err)
	}
	if _, present := (*got)[0].body["Vendor"]; present {
		t.Errorf("body = %v, want no Vendor key", (*got)[0].body)
	}
}

func TestUpdateApplication_RemovesTheVendorWithNull(t *testing.T) {
	c, got := fakeService(t, "")
	if err := c.UpdateApplication(context.Background(), "app-1", "ERP 2", ""); err != nil {
		t.Fatal(err)
	}
	r := (*got)[0]
	if r.method != http.MethodPatch || r.path != "/intas/entities/v1/Application('app-1')" {
		t.Errorf("request = %s %s", r.method, r.path)
	}
	if v, present := r.body["Vendor"]; !present || v != nil {
		t.Errorf("body = %v, want Vendor: null", r.body)
	}
}

// linkID returns the Id of a {"Id": ...} link in a recorded body, or "" if
// the key is missing or not such a link.
func linkID(body map[string]any, key string) string {
	v, _ := body[key].(map[string]any)
	if len(v) != 1 {
		return ""
	}
	id, _ := v["Id"].(string)
	return id
}

func TestUpdateApplicationInstance_SendsBothLinks(t *testing.T) {
	c, got := fakeService(t, "")
	desc := "moved"
	if err := c.UpdateApplicationInstance(context.Background(), "inst-1", "ERP PRD", &desc, "app-2", "dm-2"); err != nil {
		t.Fatal(err)
	}
	r := (*got)[0]
	if r.method != http.MethodPatch || r.path != "/intas/entities/v1/ApplicationInstance('inst-1')" {
		t.Errorf("request = %s %s", r.method, r.path)
	}
	if linkID(r.body, "Application") != "app-2" || linkID(r.body, "DeploymentModel") != "dm-2" || r.body["Description"] != "moved" {
		t.Errorf("body = %v", r.body)
	}
}

func TestUpdateTechnology_SendsTheVendor(t *testing.T) {
	c, got := fakeService(t, "")
	if err := c.UpdateTechnology(context.Background(), "tech-1", "Kafka", "vendor-2"); err != nil {
		t.Fatal(err)
	}
	if r := (*got)[0]; r.path != "/intas/entities/v1/Technology('tech-1')" || linkID(r.body, "Vendor") != "vendor-2" || r.body["Name"] != "Kafka" {
		t.Errorf("request = %s %s %v", r.method, r.path, r.body)
	}
}

func TestUpdateTechnologyInstance_LeavesTheTechnologyAlone(t *testing.T) {
	c, got := fakeService(t, "")
	if err := c.UpdateTechnologyInstance(context.Background(), "ti-1", "Kafka PRD", "dm-2"); err != nil {
		t.Fatal(err)
	}
	r := (*got)[0]
	if r.method != http.MethodPatch || r.path != "/intas/entities/v1/TechnologyInstance('ti-1')" {
		t.Errorf("request = %s %s", r.method, r.path)
	}
	if _, present := r.body["Technology"]; present || linkID(r.body, "DeploymentModel") != "dm-2" || r.body["Name"] != "Kafka PRD" {
		t.Errorf("body = %v, want Name and DeploymentModel only", r.body)
	}
}

func TestGetApplicationInstance_ExpandsItsLinks(t *testing.T) {
	c, got := fakeService(t, `{"d":{"Id":"inst-1","Name":"ERP PRD","Description":null,`+
		`"Application":{"Id":"app-1","Name":"ERP"},"DeploymentModel":{"Id":"dm-1","Name":"Cloud"}}}`)
	i, err := c.GetApplicationInstance(context.Background(), "inst-1")
	if err != nil {
		t.Fatal(err)
	}
	if i.ApplicationID() != "app-1" || i.DeploymentModelID() != "dm-1" || i.Description != nil {
		t.Errorf("instance = %+v", i)
	}
	if q := (*got)[0].query; q != "$expand=Application,DeploymentModel" {
		t.Errorf("query = %q", q)
	}
}

func TestGetApplication_NullVendor(t *testing.T) {
	c, _ := fakeService(t, `{"d":{"Id":"app-1","Name":"ERP","Vendor":null}}`)
	a, err := c.GetApplication(context.Background(), "app-1")
	if err != nil {
		t.Fatal(err)
	}
	if a.VendorID() != "" {
		t.Errorf("vendor = %q, want none", a.VendorID())
	}
}

func TestDeleteTechnologyInstance(t *testing.T) {
	c, got := fakeService(t, "")
	if err := c.DeleteTechnologyInstance(context.Background(), "ti-1"); err != nil {
		t.Fatal(err)
	}
	if r := (*got)[0]; r.method != http.MethodDelete || r.path != "/intas/entities/v1/TechnologyInstance('ti-1')" {
		t.Errorf("request = %s %s", r.method, r.path)
	}
}
