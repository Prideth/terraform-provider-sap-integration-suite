package cloudintegration

import (
	"archive/zip"
	"bytes"
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
