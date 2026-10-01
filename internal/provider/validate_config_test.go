package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func ptr(s string) *string { return &s }

// validateStringResourceConfig runs a configuration of a resource whose
// attributes are all strings through the provider's ValidateResourceConfig
// RPC, the call Terraform makes during validate and plan, before the provider
// is configured or SAP is called. A nil value means the attribute is omitted.
func validateStringResourceConfig(t *testing.T, typeName string, values map[string]*string) []*tfprotov6.Diagnostic {
	t.Helper()
	server, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatal(err)
	}

	attributeTypes := map[string]tftypes.Type{}
	attributeValues := map[string]tftypes.Value{}
	for name, value := range values {
		attributeTypes[name] = tftypes.String
		if value == nil {
			attributeValues[name] = tftypes.NewValue(tftypes.String, nil)
		} else {
			attributeValues[name] = tftypes.NewValue(tftypes.String, *value)
		}
	}
	objectType := tftypes.Object{AttributeTypes: attributeTypes}
	config, err := tfprotov6.NewDynamicValue(objectType, tftypes.NewValue(objectType, attributeValues))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := server.ValidateResourceConfig(context.Background(), &tfprotov6.ValidateResourceConfigRequest{
		TypeName: typeName,
		Config:   &config,
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp.Diagnostics
}
