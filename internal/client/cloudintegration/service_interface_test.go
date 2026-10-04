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
)

func zipOf(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// An operation model in the shape of a service interface SAP's editor wrote
// (package export, 2026-10-03), with neutral names.
const testServiceInterfaceModel = `{
  "typeId": "ifmmessif", "name": "OrderService", "namespace": "urn:example:orders",
  "isSensitive": false, "isPtoPDisabled": false, "isdocumentOriented": false,
  "objectState": "NOT_RELEASED", "category": "OUTBOUND", "interfacePattern": "STATELESS",
  "securityProfile": "LOW_BASIC",
  "operations": [{
    "name": "OrderService",
    "request": {"typeId": "ifmmessage", "role": "Request", "name": "OrderMessage", "packageName": "Orders",
      "packageTechnicalName": "Orders", "bundleSymbolicName": "OrderMessage", "namespace": ""},
    "faults": [{"bundleSymbolicName": "OrderFault", "name": "OrderFault", "packageName": "Orders",
      "packageTechnicalName": "Orders", "role": "", "typeId": "ifmfaultm", "xmlns": "", "namespace": ""}],
    "isUnreliable": false, "isSynchronous": false, "operationIdempotent": false, "objectState": "NOT_RELEASED",
    "operationPattern": "NORMAL_OPERATION", "operationMode": "ASYNCHRONOUS", "messageDetailsCount": 1
  }],
  "bundleSymbolicName": "OrderService", "packageTechnicalName": "Orders", "version": "1.0.0"
}`

// The $value layout: an archive with the interface's own bundle and a
// resolved child, both archives, under generated names.
func testServiceInterfaceValue(t *testing.T, model string) []byte {
	inner := zipOf(t, map[string][]byte{
		"META-INF/MANIFEST.MF": []byte("Manifest-Version: 1.0\r\nSAP-BundleType: ServiceInterface\r\n\r\n"),
		".project":             []byte("<projectDescription/>"),
		"src/main/resources/json/OrderService.json": []byte(model),
	})
	child := zipOf(t, map[string][]byte{"META-INF/MANIFEST.MF": []byte("Manifest-Version: 1.0\r\n\r\n")})
	return zipOf(t, map[string][]byte{
		"0123456789abcdef0123456789abcdef":                   inner,
		"fedcba9876543210fedcba9876543210_SI_RESOLVED_CHILD": child,
	})
}

func TestParseServiceInterfaceBundle_nested(t *testing.T) {
	model, err := ParseServiceInterfaceBundle(testServiceInterfaceValue(t, testServiceInterfaceModel))
	if err != nil {
		t.Fatal(err)
	}
	if model.Category != "OUTBOUND" || model.InterfacePattern != "STATELESS" || len(model.Operations) != 1 {
		t.Fatalf("model = %+v", model)
	}
	op := model.Operations[0]
	if op.OperationMode != "ASYNCHRONOUS" || op.IsSynchronous {
		t.Errorf("operation mode = %q, synchronous %v", op.OperationMode, op.IsSynchronous)
	}
	if op.Request == nil || op.Request.BundleSymbolicName != "OrderMessage" || op.Request.Role != "Request" {
		t.Errorf("request = %+v", op.Request)
	}
	if len(op.Faults) != 1 || op.Faults[0].BundleSymbolicName != "OrderFault" || op.Faults[0].TypeID != "ifmfaultm" {
		t.Errorf("faults = %+v", op.Faults)
	}
}

func TestParseServiceInterfaceBundle_noModel(t *testing.T) {
	if _, err := ParseServiceInterfaceBundle(zipOf(t, map[string][]byte{"a.txt": []byte("x")})); err == nil {
		t.Fatal("want an error for a bundle without an operation model")
	}
	if _, err := ParseServiceInterfaceBundle([]byte("not a zip")); err == nil {
		t.Fatal("want an error for content that is not a ZIP archive")
	}
}

func TestWithServiceInterfaceOperations(t *testing.T) {
	ops := []ServiceInterfaceOperationSpec{{
		Name:    "CreateOrder",
		Request: &ServiceInterfaceMessageRef{ID: "OrderMessage", Name: "OrderMessage", PackageName: "Orders", PackageTechnicalName: "Orders"},
		Faults:  []ServiceInterfaceMessageRef{{ID: "OrderFault", Name: "OrderFault", PackageName: "Orders", PackageTechnicalName: "Orders"}},
	}}
	out, err := WithServiceInterfaceOperations([]byte(testServiceInterfaceModel), ops)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	// Fields SAP wrote are kept.
	for _, k := range []string{"typeId", "securityProfile", "isdocumentOriented", "bundleSymbolicName", "version"} {
		if _, ok := doc[k]; !ok {
			t.Errorf("field %q was dropped", k)
		}
	}
	model, err := ParseServiceInterfaceBundle(zipOf(t, map[string][]byte{"src/main/resources/json/x.json": out}))
	if err != nil {
		t.Fatal(err)
	}
	op := model.Operations[0]
	if op.Name != "CreateOrder" || op.OperationMode != "ASYNCHRONOUS" || op.Request.BundleSymbolicName != "OrderMessage" || op.Faults[0].Role != "" {
		t.Errorf("operation = %+v", op)
	}
	raw := doc["operations"].([]any)[0].(map[string]any)
	if raw["messageDetailsCount"].(float64) != 1 || len(raw["messageDetails"].([]any)) != 2 {
		t.Errorf("messageDetails = %v, count %v", raw["messageDetails"], raw["messageDetailsCount"])
	}
	if raw["faults"].([]any)[0].(map[string]any)["xmlns"] != "" {
		t.Errorf("a fault carries xmlns as SAP's editor writes it")
	}
}

func TestServiceInterfaceRequireCapability(t *testing.T) {
	got := ServiceInterfaceRequireCapability([]ServiceInterfaceOperationSpec{
		{Name: "a", Request: &ServiceInterfaceMessageRef{ID: "OrderMessage"}, Faults: []ServiceInterfaceMessageRef{{ID: "OrderFault"}}},
		{Name: "b", Request: &ServiceInterfaceMessageRef{ID: "OrderMessage"}},
	})
	want := `Require-Capability: faultmessagetype.OrderFault;resolution:=optional;bundleType:String="FaultMessageType";source:String="reference",` +
		`messagetype.OrderMessage;resolution:=optional;bundleType:String="MessageType";source:String="reference"`
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if ServiceInterfaceRequireCapability(nil) != "" {
		t.Error("no references, no header")
	}
}

func TestSetManifestHeader(t *testing.T) {
	manifest := "Manifest-Version: 1.0\r\nRequire-Capability: old;x\r\n more\r\nSAP-BundleType: ServiceInterface\r\n\r\n"
	replaced := setManifestHeader(manifest, "Require-Capability", "Require-Capability: new")
	if strings.Contains(replaced, "old") || strings.Contains(replaced, " more") || !strings.Contains(replaced, "Require-Capability: new\r\n") {
		t.Errorf("replaced = %q", replaced)
	}
	if !strings.Contains(replaced, "SAP-BundleType: ServiceInterface") || !strings.HasSuffix(replaced, "\r\n\r\n") {
		t.Errorf("other headers or the trailing blank line lost: %q", replaced)
	}
	removed := setManifestHeader(manifest, "Require-Capability", "")
	if strings.Contains(removed, "Require-Capability") {
		t.Errorf("removed = %q", removed)
	}
}

// A synchronous operation in the shape of the 2026-10-04 export, with
// neutral names.
const testSyncOperation = `{"name": "OrderService", "operations": [{
  "name": "OrderService",
  "request": {"typeId": "ifmmessage", "name": "OrderMessage", "namespace": "", "packageName": "Orders",
    "packageTechnicalName": "Orders", "bundleSymbolicName": "OrderMessage", "version": "1.0.1", "xmlns": "", "role": "Request"},
  "response": {"typeId": "ifmmessage", "name": "OrderReply", "namespace": "urn:example:reply", "packageName": "Orders",
    "packageTechnicalName": "Orders", "bundleSymbolicName": "OrderReply", "version": "1.0.1", "xmlns": "urn:example:reply", "role": "Response"},
  "faults": [{"typeId": "ifmfaultm", "name": "OrderFault", "namespace": "", "packageName": "Orders",
    "packageTechnicalName": "Orders", "bundleSymbolicName": "OrderFault", "version": "1.0.1", "xmlns": ""}],
  "isSynchronous": true, "operationMode": "SYNCHRONOUS", "messageDetailsCount": 3
}]}`

func TestParseServiceInterfaceBundle_synchronous(t *testing.T) {
	model, err := ParseServiceInterfaceBundle(zipOf(t, map[string][]byte{"src/main/resources/json/OrderService.json": []byte(testSyncOperation)}))
	if err != nil {
		t.Fatal(err)
	}
	op := model.Operations[0]
	if !op.IsSynchronous || op.OperationMode != "SYNCHRONOUS" {
		t.Errorf("mode = %q, synchronous %v", op.OperationMode, op.IsSynchronous)
	}
	if op.Response == nil || op.Response.BundleSymbolicName != "OrderReply" || op.Response.Role != "Response" || op.Response.XMLNS != "urn:example:reply" {
		t.Errorf("response = %+v", op.Response)
	}
	if op.Request.Version != "1.0.1" || op.Faults[0].Role != "" {
		t.Errorf("request version %q, fault role %q", op.Request.Version, op.Faults[0].Role)
	}
}

func TestWithServiceInterfaceOperations_synchronous(t *testing.T) {
	reply := &ServiceInterfaceMessageRef{ID: "OrderReply", Name: "OrderReply", Namespace: "urn:example:reply", Version: "1.0.1", PackageName: "Orders", PackageTechnicalName: "Orders"}
	ops := []ServiceInterfaceOperationSpec{{
		Name:     "ProcessOrder",
		Request:  &ServiceInterfaceMessageRef{ID: "OrderMessage", Name: "OrderMessage", Version: "1.0.1", PackageName: "Orders", PackageTechnicalName: "Orders"},
		Response: reply,
		Faults:   []ServiceInterfaceMessageRef{{ID: "OrderFault", Name: "OrderFault", Version: "1.0.1", PackageName: "Orders", PackageTechnicalName: "Orders"}},
	}}
	out, err := WithServiceInterfaceOperations([]byte(testServiceInterfaceModel), ops)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	raw := doc["operations"].([]any)[0].(map[string]any)
	if raw["operationMode"] != "SYNCHRONOUS" || raw["isSynchronous"] != true || raw["messageDetailsCount"].(float64) != 3 {
		t.Errorf("operation = %v", raw)
	}
	resp := raw["response"].(map[string]any)
	if resp["role"] != "Response" || resp["xmlns"] != "urn:example:reply" || resp["version"] != "1.0.1" {
		t.Errorf("response = %v", resp)
	}
	if _, hasRole := raw["faults"].([]any)[0].(map[string]any)["role"]; hasRole {
		t.Error("a fault carries no role in SAP's current form")
	}
	if got := ServiceInterfaceRequireCapability(ops); !strings.Contains(got, "faultmessagetype.OrderFault;") ||
		strings.Index(got, "messagetype.OrderMessage;") > strings.Index(got, "messagetype.OrderReply;") {
		t.Errorf("require capability = %s", got)
	}
}

func TestWithServiceInterfaceBundleOperations(t *testing.T) {
	value := testServiceInterfaceValue(t, testServiceInterfaceModel)
	ops := []ServiceInterfaceOperationSpec{{
		Name:     "ProcessOrder",
		Request:  &ServiceInterfaceMessageRef{ID: "OrderMessage", Name: "OrderMessage"},
		Response: &ServiceInterfaceMessageRef{ID: "OrderReply", Name: "OrderReply", Namespace: "urn:example:reply"},
	}}
	out, err := WithServiceInterfaceBundleOperations(value, ops)
	if err != nil {
		t.Fatal(err)
	}
	model, err := ParseServiceInterfaceBundle(out)
	if err != nil {
		t.Fatal(err)
	}
	if op := model.Operations[0]; op.Name != "ProcessOrder" || op.Response == nil || op.Response.BundleSymbolicName != "OrderReply" {
		t.Errorf("operation = %+v", op)
	}
	outer, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range outer.File {
		data, err := readZipFile(f)
		if err != nil {
			t.Fatal(err)
		}
		inner, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			t.Fatal(err)
		}
		for _, g := range inner.File {
			if g.Name != "META-INF/MANIFEST.MF" {
				continue
			}
			manifest, _ := readZipFile(g)
			has := strings.Contains(strings.ReplaceAll(string(manifest), "\r\n ", ""), "messagetype.OrderReply;")
			if strings.HasSuffix(f.Name, "_SI_RESOLVED_CHILD") == has {
				t.Errorf("%s: Require-Capability present = %v", f.Name, has)
			}
		}
	}
	if _, err := WithServiceInterfaceBundleOperations(zipOf(t, map[string][]byte{"a.txt": []byte("x")}), ops); err == nil {
		t.Error("want an error for content without a service interface bundle")
	}
}

func TestClient_ServiceInterfaceUpdateRequest(t *testing.T) {
	value := testServiceInterfaceValue(t, testServiceInterfaceModel)
	var put map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &put)
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/$value"):
			_, _ = w.Write(value)
		default:
			_, _ = w.Write([]byte(`{"d":{"Id":"OrderService","Name":"OrderService","PackageId":"Orders","Version":"1.0.0"}}`))
		}
	}))
	defer server.Close()
	c := New(http.DefaultClient, server.URL)
	ops := []ServiceInterfaceOperationSpec{{Name: "CreateOrder", Request: &ServiceInterfaceMessageRef{ID: "OrderMessage", Name: "OrderMessage"}}}
	if _, err := c.UpdateServiceInterface(context.Background(), "OrderService", "OrderService", "Orders in", ops); err != nil {
		t.Fatal(err)
	}
	if put["Name"] != "OrderService" || put["Description"] != "Orders in" {
		t.Errorf("PUT body = %v", put)
	}
	content, err := base64.StdEncoding.DecodeString(put["ArtifactContent"])
	if err != nil {
		t.Fatal(err)
	}
	model, err := ParseServiceInterfaceBundle(content)
	if err != nil || model.Operations[0].Name != "CreateOrder" {
		t.Errorf("uploaded model = %+v, %v", model, err)
	}
}
