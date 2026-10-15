package cloudintegration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var testPair = ValueMappingAgencyPair{
	SourceAgency: "ERP", SourceIdentifier: "Country", TargetAgency: "CRM", TargetIdentifier: "CountryCode",
}

// valMapServer answers the entry list with entries and records every other
// request's method and raw request URI.
func valMapServer(t *testing.T, entries string) (*Client, *[]string) {
	t.Helper()
	var got []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(entries))
			return
		}
		got = append(got, r.Method+" "+r.RequestURI)
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)
	return New(http.DefaultClient, server.URL), &got
}

func TestAddValueMappingEntry_QuotesAndEncodesEveryValue(t *testing.T) {
	c, got := valMapServer(t, `{"d":{"results":[{"Id":"e1","Value":{"SrcValue":"O'Neil St","TgtValue":"a&b"}}]}}`)
	e, err := c.AddValueMappingEntry(context.Background(), "Country Codes", testPair, "O'Neil St", "a&b")
	if err != nil {
		t.Fatal(err)
	}
	if e.ID != "e1" {
		t.Errorf("entry = %+v", e)
	}
	want := "POST /api/v1/UpsertValMaps?Id=%27Country%20Codes%27&Version=%27active%27" +
		"&SrcAgency=%27ERP%27&SrcId=%27Country%27&TgtAgency=%27CRM%27&TgtId=%27CountryCode%27" +
		"&SrcValue=%27O%27%27Neil%20St%27&TgtValue=%27a%26b%27&IsConfigured=false"
	if len(*got) != 1 || (*got)[0] != want {
		t.Errorf("requests = %v\nwant [%s]", *got, want)
	}
}

func TestAddValueMappingEntry_FailsWhenTheEntryIsMissingAfterwards(t *testing.T) {
	c, _ := valMapServer(t, `{"d":{"results":[]}}`)
	if _, err := c.AddValueMappingEntry(context.Background(), "VM", testPair, "DE", "276"); err == nil {
		t.Fatal("want an error when SAP accepted the entry but does not list it")
	}
}

func TestChangeValueMappingEntry_SendsValMapID(t *testing.T) {
	c, got := valMapServer(t, "")
	if err := c.ChangeValueMappingEntry(context.Background(), "VM", testPair, "e1", "DE", "276"); err != nil {
		t.Fatal(err)
	}
	if len(*got) != 1 || !strings.Contains((*got)[0], "&ValMapId=%27e1%27&SrcValue=%27DE%27&TgtValue=%27276%27&IsConfigured=false") {
		t.Errorf("requests = %v", *got)
	}
}

func TestListValueMappingEntries_ReadsThePairUnderTheActiveVersion(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"d":{"results":[{"Id":"e1","Value":{"SrcValue":"DE","TgtValue":"276"}}]}}`))
	}))
	defer server.Close()
	entries, err := New(http.DefaultClient, server.URL).ListValueMappingEntries(context.Background(), "VM", testPair)
	if err != nil {
		t.Fatal(err)
	}
	want := "/api/v1/ValueMappingDesigntimeArtifacts(Id='VM',Version='active')" +
		"/ValMapSchema(SrcAgency='ERP',SrcId='Country',TgtAgency='CRM',TgtId='CountryCode')/ValMaps"
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	if len(entries) != 1 || entries[0] != (ValueMappingEntry{ID: "e1", SourceValue: "DE", TargetValue: "276"}) {
		t.Errorf("entries = %+v", entries)
	}
}

func TestHasValueMappingAgencyPair_MatchesOneDirection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"d":{"results":[` +
			`{"SrcAgency":"CRM","SrcId":"CountryCode","TgtAgency":"ERP","TgtId":"Country","State":""}]}}`))
	}))
	defer server.Close()
	c := New(http.DefaultClient, server.URL)
	has, err := c.HasValueMappingAgencyPair(context.Background(), "VM", testPair)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("the reverse direction must not count as the pair")
	}
}
