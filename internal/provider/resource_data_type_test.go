package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const validDataTypeXSD = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns="urn:example:sales" targetNamespace="urn:example:sales">
  <xsd:complexType name="Order"><xsd:sequence><xsd:element name="OrderID" type="xsd:string"/></xsd:sequence></xsd:complexType>
</xsd:schema>`

func validateDataType(t *testing.T, name, namespace, xsd string) []*tfprotov6.Diagnostic {
	t.Helper()
	return validateStringResourceConfig(t, "sapintegrationsuite_data_type", map[string]*string{
		"id": nil, "package_id": ptr("Sales"), "data_type_id": ptr("Order"), "name": ptr(name),
		"namespace": ptr(namespace), "description": nil, "xsd": ptr(xsd), "version": nil, "save_as_version": nil,
	})
}

func TestDataTypeResource_ValidSchemaPasses(t *testing.T) {
	for _, d := range validateDataType(t, "Order", "urn:example:sales", validDataTypeXSD) {
		t.Errorf("unexpected diagnostic: %s: %s", d.Summary, d.Detail)
	}
}

func TestDataTypeResource_SchemaChecks(t *testing.T) {
	cases := []struct {
		name, xsdName, namespace, xsd, wantSummary string
		severity                                   tfprotov6.DiagnosticSeverity
	}{
		{"not XML", "Order", "urn:example:sales", "<xsd:schema", "Invalid data type schema", tfprotov6.DiagnosticSeverityError},
		{"wrong root", "Order", "urn:example:sales", `<schema xmlns="urn:other"/>`, "Invalid data type schema", tfprotov6.DiagnosticSeverityError},
		{"two complex types", "Order", "urn:example:sales",
			strings.Replace(validDataTypeXSD, "</xsd:schema>", `<xsd:complexType name="Item"/></xsd:schema>`, 1),
			"Invalid data type schema", tfprotov6.DiagnosticSeverityError},
		{"namespace differs", "Order", "urn:example:other", validDataTypeXSD, "Schema namespace does not match", tfprotov6.DiagnosticSeverityError},
		{"type renamed by SAP", "SalesOrder", "urn:example:sales", validDataTypeXSD, "SAP renames the complex type", tfprotov6.DiagnosticSeverityWarning},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			diags := validateDataType(t, c.xsdName, c.namespace, c.xsd)
			found := false
			for _, d := range diags {
				if d.Summary == c.wantSummary && d.Severity == c.severity {
					found = true
				}
			}
			if !found {
				t.Errorf("diagnostics = %v, want %q", diags, c.wantSummary)
			}
		})
	}
}
