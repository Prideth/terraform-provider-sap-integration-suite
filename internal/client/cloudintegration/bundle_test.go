package cloudintegration

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func zipWithManifest(t *testing.T, manifest string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("META-INF/MANIFEST.MF")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(manifest)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// SAP wraps manifest lines at 72 bytes, even inside Bundle-SymbolicName
// (seen in SAP-samples/connecting-systems-services-integration-suite-codejam).
func TestParseBundleManifest_JoinsWrappedLines(t *testing.T) {
	manifest := "Manifest-Version: 1.0\r\n" +
		"Bundle-SymbolicName: Request_Employee_Dependants_-_Cloud_Connector_Expor\r\n" +
		" t; singleton:=true\r\n" +
		"Bundle-Version: 1.0.0\r\n" +
		"SAP-BundleType: IntegrationFlow\r\n" +
		"SAP-RuntimeProfile: iflmap\r\n" +
		"Import-Package: com.sap.esb.application.services.cxf.interceptor,com.sap\r\n" +
		" .esb.security\r\n\r\n"
	m, err := ParseBundleManifest(zipWithManifest(t, manifest))
	if err != nil {
		t.Fatal(err)
	}
	if m.SymbolicName != "Request_Employee_Dependants_-_Cloud_Connector_Export" {
		t.Errorf("SymbolicName = %q, want the joined name without directives", m.SymbolicName)
	}
	if m.BundleType != "IntegrationFlow" || m.Version != "1.0.0" || m.RuntimeProfile != "iflmap" {
		t.Errorf("manifest = %+v", m)
	}
	if got := m.Headers["Import-Package"]; got != "com.sap.esb.application.services.cxf.interceptor,com.sap.esb.security" {
		t.Errorf("Import-Package = %q, want the joined value", got)
	}
}

func TestParseBundleManifest_Errors(t *testing.T) {
	if _, err := ParseBundleManifest([]byte("not a zip")); err == nil {
		t.Error("want an error for content that is not a ZIP")
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	_, _ = w.Create("src/main/resources/parameters.prop")
	_ = w.Close()
	if _, err := ParseBundleManifest(buf.Bytes()); err == nil {
		t.Error("want an error for a ZIP without META-INF/MANIFEST.MF")
	}
}

func zipEntries(t *testing.T, content []byte) map[string][]byte {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]byte{}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(rc)
		_ = rc.Close()
		out[f.Name] = buf.Bytes()
	}
	return out
}

func TestAlignBundleID(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	manifest := "Manifest-Version: 1.0\r\n" +
		"Bundle-SymbolicName: Request_Employee_Dependants_-_Cloud_Connector_Expor\r\n" +
		" t; singleton:=true\r\n" +
		"Provide-Capability: messagemapping.Other_Name;version:Version=\"1.0.0\"\r\n" +
		"SAP-BundleType: MessageMapping\r\n\r\n"
	for name, body := range map[string]string{
		"META-INF/MANIFEST.MF":              manifest,
		"src/main/resources/mapping/m.mmap": "<mapping/>",
	} {
		f, _ := w.Create(name)
		_, _ = f.Write([]byte(body))
	}
	_ = w.Close()
	original := buf.Bytes()

	id := "tfacc_" + strings.Repeat("y", 70)
	aligned, previous, err := AlignBundleID(original, id)
	if err != nil {
		t.Fatal(err)
	}
	if previous != "Request_Employee_Dependants_-_Cloud_Connector_Export" {
		t.Errorf("previous = %q", previous)
	}
	m, err := ParseBundleManifest(aligned)
	if err != nil {
		t.Fatal(err)
	}
	if m.SymbolicName != id || !strings.HasSuffix(m.Headers["Bundle-SymbolicName"], "; singleton:=true") {
		t.Errorf("Bundle-SymbolicName = %q, want %q with its directive", m.Headers["Bundle-SymbolicName"], id)
	}
	if !strings.HasPrefix(m.Headers["Provide-Capability"], "messagemapping."+id+";version") {
		t.Errorf("Provide-Capability = %q", m.Headers["Provide-Capability"])
	}
	for _, line := range strings.Split(string(zipEntries(t, aligned)["META-INF/MANIFEST.MF"]), "\r\n") {
		if len(line) > 72 {
			t.Errorf("manifest line longer than 72 bytes: %q", line)
		}
	}
	if got := string(zipEntries(t, aligned)["src/main/resources/mapping/m.mmap"]); got != "<mapping/>" {
		t.Errorf("other entry changed: %q", got)
	}

	again, previous, err := AlignBundleID(aligned, id)
	if err != nil || !bytes.Equal(again, aligned) || previous != id {
		t.Errorf("aligning an aligned ZIP must return it unchanged (previous %q, err %v)", previous, err)
	}
	if out, _, err := AlignBundleID([]byte("not a zip"), id); err != nil || string(out) != "not a zip" {
		t.Errorf("content without a manifest must pass through unchanged, got err %v", err)
	}
}
