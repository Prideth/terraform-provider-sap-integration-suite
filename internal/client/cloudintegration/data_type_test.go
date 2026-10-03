package cloudintegration

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testDataTypeXSD = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns="urn:example:orders" targetNamespace="urn:example:orders">
   <xsd:complexType name="Order">
      <xsd:sequence>
         <xsd:element name="OrderID" type="xsd:string"/>
      </xsd:sequence>
   </xsd:complexType>
</xsd:schema>`

func readBundle(t *testing.T, bundle []byte) map[string]string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(bundle), int64(len(bundle)))
	if err != nil {
		t.Fatalf("reading bundle: %v", err)
	}
	files := map[string]string{}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("opening %s: %v", f.Name, err)
		}
		content, _ := io.ReadAll(rc)
		_ = rc.Close()
		files[f.Name] = string(content)
	}
	return files
}

// The bundle has the five files of SAP's own $value of a data type
// (2026-10-03); without additionalAttributes.json and metainfo.prop SAP
// rejects the create with "map is null".
func TestDataTypeBundle_LayoutMatchesSAP(t *testing.T) {
	bundle, err := DataTypeBundle{
		ID: "Order", Name: "Order", Namespace: "urn:example:orders", Description: "Sales order", XSD: testDataTypeXSD,
	}.Build(time.Date(2026, 10, 3, 14, 55, 53, 0, time.UTC))
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	files := readBundle(t, bundle)

	for _, name := range []string{"src/main/resources/xsd/Order.xsd", ".project", "META-INF/MANIFEST.MF",
		"src/main/resources/additionalAttributes.json", "metainfo.prop"} {
		if _, ok := files[name]; !ok {
			t.Errorf("bundle lacks %s; it has %v", name, keys(files))
		}
	}
	if files["src/main/resources/xsd/Order.xsd"] != testDataTypeXSD {
		t.Errorf("XSD changed in the bundle")
	}

	var attrs map[string]string
	if err := json.Unmarshal([]byte(files["src/main/resources/additionalAttributes.json"]), &attrs); err != nil {
		t.Fatalf("additionalAttributes.json: %v", err)
	}
	want := map[string]string{
		"Description": "Sales order", "qualifySchema": "0", "BundleVersion": "1.0.0", "contentmodel": "S",
		"namespace": "urn:example:orders", "BundleSymbolicName": "Order", "category": "Complex Type",
		"classification": "Free-Style Data Type",
	}
	for k, v := range want {
		if attrs[k] != v {
			t.Errorf("additionalAttributes %s = %q, want %q", k, attrs[k], v)
		}
	}

	manifest := files["META-INF/MANIFEST.MF"]
	for _, line := range []string{"Bundle-SymbolicName: Order", "Bundle-Name: Order", "SAP-BundleType: DataType",
		`Provide-Capability: datatype.Order;version:Version="1.0.0"`} {
		if !strings.Contains(manifest, line+"\n") {
			t.Errorf("manifest lacks %q:\n%s", line, manifest)
		}
	}
	if !strings.Contains(files["metainfo.prop"], "description=Sales order\n") ||
		!strings.Contains(files["metainfo.prop"], "#Sat Oct 03 14:55:53 UTC 2026") {
		t.Errorf("metainfo.prop = %q", files["metainfo.prop"])
	}
	if !strings.Contains(files[".project"], "<name>Order</name>") {
		t.Errorf(".project = %q", files[".project"])
	}
}

func TestWrapManifestLine_FollowsTheJarSpecification(t *testing.T) {
	long := "Provide-Capability: datatype." + strings.Repeat("A", 100) + `;version:Version="1.0.0"`
	wrapped := wrapManifestLine(long)
	for i, line := range strings.Split(strings.TrimSuffix(wrapped, "\n"), "\n") {
		if len(line) > 72 {
			t.Errorf("line %d has %d bytes", i, len(line))
		}
		if i > 0 && !strings.HasPrefix(line, " ") {
			t.Errorf("continuation line %d does not start with a space", i)
		}
	}
	if joined := strings.ReplaceAll(strings.TrimSuffix(wrapped, "\n"), "\n ", ""); joined != long {
		t.Errorf("unwrapped = %q, want %q", joined, long)
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestClient_DataTypeLifecycleRequests(t *testing.T) {
	var requests []string
	var created map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/DataTypeDesigntimeArtifacts":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &created)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"d":{"Id":"Order","Name":"Order","PackageId":"Sales","Version":"1.0.0","Namespace":"urn:example:orders","IsSimpleType":false}}`))
		case r.Method == http.MethodPut:
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		default:
			_, _ = w.Write([]byte(`{"d":{"Id":"Order","Name":"Order","PackageId":"Sales","Version":"1.0.1"}}`))
		}
	}))
	defer server.Close()

	c := New(http.DefaultClient, server.URL)
	ctx := context.Background()
	if _, err := c.CreateDataType(ctx, "Sales", "Order", "Order", []byte("bundle")); err != nil {
		t.Fatalf("CreateDataType: %v", err)
	}
	if created["Id"] != "Order" || created["PackageId"] != "Sales" || created["ArtifactContent"] != base64.StdEncoding.EncodeToString([]byte("bundle")) {
		t.Errorf("create body = %v", created)
	}
	if dt, err := c.UpdateDataType(ctx, "Order", "Order", []byte("bundle2")); err != nil || dt.Version != "1.0.1" {
		t.Fatalf("UpdateDataType = %+v, %v", dt, err)
	}
	if _, err := c.SaveDataTypeAsVersion(ctx, "Order", "1.0.2"); err != nil {
		t.Fatalf("SaveDataTypeAsVersion: %v", err)
	}
	if err := c.DeleteDataType(ctx, "Order"); err != nil {
		t.Fatalf("DeleteDataType: %v", err)
	}

	want := []string{
		"POST /api/v1/DataTypeDesigntimeArtifacts?",
		"PUT /api/v1/DataTypeDesigntimeArtifacts(Id='Order',Version='active')?",
		"GET /api/v1/DataTypeDesigntimeArtifacts(Id='Order',Version='active')?",
		"POST /api/v1/DataTypeDesigntimeArtifactSaveAsVersion?Id='Order'&SaveAsVersion='1.0.2'",
		"GET /api/v1/DataTypeDesigntimeArtifacts(Id='Order',Version='active')?",
		"DELETE /api/v1/DataTypeDesigntimeArtifacts(Id='Order',Version='active')?",
	}
	got := strings.Join(requests, "\n")
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("missing request %q in\n%s", w, got)
		}
	}
}
