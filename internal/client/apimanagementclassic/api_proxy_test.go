package apimanagementclassic

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// proxyBundle builds a minimal API proxy bundle with the given descriptor
// files (path -> content).
func proxyBundle(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const sampleDescriptor = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<APIProxy xmlns="http://www.sap.com/apimgmt">
    <name>SampleAPI</name>
    <title>Sample API</title>
</APIProxy>`

func TestClient_ImportAPIProxy(t *testing.T) {
	bundle := proxyBundle(t, map[string]string{"APIProxy/SampleAPI.xml": sampleDescriptor})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/apiportal/api/1.0/Transport.svc/APIProxies" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		// The Client SDK's query, character for character.
		if r.URL.RawQuery != "name=?virtualhost=default" {
			t.Errorf("query = %q", r.URL.RawQuery)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/octet-stream" {
			t.Errorf("Content-Type = %q", ct)
		}
		body, _ := io.ReadAll(r.Body)
		if !bytes.Equal(body, bundle) {
			t.Error("the body is not the bundle as given")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := New(http.DefaultClient, server.URL).ImportAPIProxy(context.Background(), bundle); err != nil {
		t.Fatalf("ImportAPIProxy() error: %v", err)
	}
}

func TestClient_ImportAPIProxyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"Bad Request","message":{"lang":"en","value":"Base path already in use"}}}`))
	}))
	defer server.Close()

	err := New(http.DefaultClient, server.URL).ImportAPIProxy(context.Background(), []byte("zip"))
	if err == nil || !strings.Contains(err.Error(), "Base path already in use") {
		t.Fatalf("want SAP's message in the error, got %v", err)
	}
}

func TestClient_GetDeleteAPIProxy(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if want := "/apiportal/api/1.0/Management.svc/APIProxies('SampleAPI')"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"d": {"name": "SampleAPI", "title": "Sample API", "version": "1", "service_code": "REST", "provider_name": "NONE", "state": "DEPLOYED", "status_code": "REGISTERED", "isPublished": false, "isVersioned": false, "life_cycle": {"created_at": "/Date(1790423910729)/"}, "policies": {"__deferred": {"uri": "x"}}}}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)
	proxy, err := client.GetAPIProxy(context.Background(), "SampleAPI")
	if err != nil {
		t.Fatalf("GetAPIProxy() error: %v", err)
	}
	if proxy.Title != "Sample API" || proxy.State != "DEPLOYED" || proxy.ServiceCode != "REST" {
		t.Errorf("proxy = %+v", proxy)
	}
	if err := client.DeleteAPIProxy(context.Background(), "SampleAPI"); err != nil {
		t.Fatalf("DeleteAPIProxy() error: %v", err)
	}
	if strings.Join(methods, ",") != "GET,DELETE" {
		t.Errorf("methods = %v", methods)
	}
}

func TestClient_WaitForAPIProxy(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls < 3 {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"Not Found","message":{"lang":"en","value":"not found"}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"d": {"name": "SampleAPI", "state": "DEPLOYED"}}`))
	}))
	defer server.Close()

	proxy, err := New(http.DefaultClient, server.URL).WaitForAPIProxy(context.Background(), "SampleAPI")
	if err != nil {
		t.Fatalf("WaitForAPIProxy() error: %v", err)
	}
	if proxy.Name != "SampleAPI" || calls != 3 {
		t.Errorf("proxy %+v after %d calls", proxy, calls)
	}
}

func TestAPIProxyBundleName(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		want    string
		wantErr string
	}{
		{"descriptor", map[string]string{
			"APIProxy/SampleAPI.xml":                sampleDescriptor,
			"APIProxy/APIProxyEndPoint/default.xml": "<ProxyEndPoint><name>default</name></ProxyEndPoint>",
			"APIProxy/Policy/quota.xml":             "<Quota/>",
		}, "SampleAPI", ""},
		{"no descriptor", map[string]string{"APIProxy/Policy/quota.xml": "<Quota/>"}, "", "found 0"},
		{"two descriptors", map[string]string{"APIProxy/A.xml": sampleDescriptor, "APIProxy/B.xml": sampleDescriptor}, "", "found 2"},
		{"not a proxy", map[string]string{"APIProxy/A.xml": "<Other><name>x</name></Other>"}, "", "not an API proxy descriptor"},
		{"no name", map[string]string{"APIProxy/A.xml": "<APIProxy><title>x</title></APIProxy>"}, "", "not an API proxy descriptor"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := APIProxyBundleName(proxyBundle(t, tc.files))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
	if _, err := APIProxyBundleName([]byte("not a zip")); err == nil {
		t.Error("want an error for bytes that are not a ZIP")
	}
}
