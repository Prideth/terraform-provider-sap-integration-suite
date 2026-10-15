package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestKeepSymbolicVersion(t *testing.T) {
	for _, tc := range []struct{ configured, deployed, want string }{
		{"Draft", "1.0.1", "Draft"},
		{"active", "1.0.1", "active"},
		{"1.0.2", "1.0.1", "1.0.1"},
	} {
		if got := keepSymbolicVersion(types.StringValue(tc.configured), types.StringValue(tc.deployed)).ValueString(); got != tc.want {
			t.Errorf("keepSymbolicVersion(%q, %q) = %q, want %q", tc.configured, tc.deployed, got, tc.want)
		}
	}
}
