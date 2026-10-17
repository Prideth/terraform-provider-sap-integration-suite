package cloudintegration

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSplitCustomTagValue_FollowsSAPsSeparator(t *testing.T) {
	for value, want := range map[string][]string{
		"":                  nil,
		"Finance":           {"Finance"},
		"Finance | Sales":   {"Finance", "Sales"},
		" Finance|Sales ":   {"Finance", "Sales"},
		"Finance, Controls": {"Finance, Controls"},
		" | ":               nil,
	} {
		if got := SplitCustomTagValue(value); !reflect.DeepEqual(got, want) {
			t.Errorf("SplitCustomTagValue(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestListPackageCustomTags_ReadsTheNavigation(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"d":{"results":[` +
			`{"Name":"Owner","Value":"Finance | Sales"},{"Name":"Region","Value":""},{"Name":"LoB","Value":null}]}}`))
	}))
	defer server.Close()
	tags, err := New(http.DefaultClient, server.URL).ListPackageCustomTags(context.Background(), "Orders")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/v1/IntegrationPackages('Orders')/CustomTags" {
		t.Errorf("path = %q", path)
	}
	want := []PackageCustomTag{{Name: "Owner", Values: []string{"Finance", "Sales"}}, {Name: "Region"}, {Name: "LoB"}}
	if !reflect.DeepEqual(tags, want) {
		t.Errorf("tags = %+v, want %+v", tags, want)
	}
}

func TestSetPackageCustomTag_PutsTheJoinedValue(t *testing.T) {
	var got []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, r.Method+" "+r.URL.Path+" "+string(body))
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	c := New(http.DefaultClient, server.URL)
	if err := c.SetPackageCustomTag(context.Background(), "Orders", "Owner", []string{"Finance", "Sales"}); err != nil {
		t.Fatal(err)
	}
	if err := c.SetPackageCustomTag(context.Background(), "Orders", "Owner", nil); err != nil {
		t.Fatal(err)
	}
	want := []string{
		`PUT /api/v1/IntegrationPackages('Orders')/$links/CustomTags('Owner') {"Value":"Finance|Sales"}`,
		`PUT /api/v1/IntegrationPackages('Orders')/$links/CustomTags('Owner') {"Value":""}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("requests = %q\nwant %q", got, want)
	}
}
