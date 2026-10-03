package cloudintegration

import (
	"archive/zip"
	"bytes"
	"encoding/json"
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

func TestManifestHeaderWrapping(t *testing.T) {
	header := ServiceInterfaceRequireCapability([]ServiceInterfaceOperationSpec{{Request: &ServiceInterfaceMessageRef{ID: "OrderMessage"}, Faults: []ServiceInterfaceMessageRef{{ID: "OrderFault"}}}})
	lines := manifestLines(header)
	for i, l := range lines {
		if len(l) > 72 {
			t.Errorf("line %d has %d bytes", i, len(l))
		}
		if i > 0 && !strings.HasPrefix(l, " ") {
			t.Errorf("continuation line %d does not start with a space", i)
		}
	}
	joined := lines[0]
	for _, l := range lines[1:] {
		joined += l[1:]
	}
	if joined != header {
		t.Error("unwrapping does not give the header back")
	}

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
