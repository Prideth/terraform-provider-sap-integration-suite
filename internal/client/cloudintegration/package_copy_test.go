package cloudintegration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCopyPackage_SendsModeAndSuffixAsSAPDocumentsThem(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mode   CopyImportMode
		suffix string
		want   string
	}{
		{"new", CopyAsNew, "ignored", "/api/v1/CopyIntegrationPackage?Id=%27Orders%27"},
		{"merge", CopyOverwriteMerge, "ignored", "/api/v1/CopyIntegrationPackage?Id=%27Orders%27&ImportMode=%27OVERWRITE_MERGE%27"},
		{"copy", CopyCreateCopy, "dev", "/api/v1/CopyIntegrationPackage?Id=%27Orders%27&ImportMode=%27CREATE_COPY%27&Suffix=%27dev%27"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					got = r.RequestURI
				}
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"d":{"Id":"Orders","Name":"Orders","Version":"2.0.0","Vendor":"SAP","Mode":"READ_ONLY"}}`))
			}))
			defer server.Close()
			pkg, err := New(http.DefaultClient, server.URL).CopyPackage(context.Background(), "Orders", tc.mode, tc.suffix)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("request = %s, want %s", got, tc.want)
			}
			if pkg.Version != "2.0.0" || pkg.Mode != "READ_ONLY" {
				t.Errorf("package = %+v", pkg)
			}
		})
	}
}

func TestCopiedPackageID_AppendsTheSuffix(t *testing.T) {
	if got := CopiedPackageID("Orders", ""); got != "Orders" {
		t.Errorf("without suffix: %q", got)
	}
	if got := CopiedPackageID("Orders", "dev"); got != "Orders.dev" {
		t.Errorf("with suffix: %q", got)
	}
}
