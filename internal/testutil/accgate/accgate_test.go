package accgate

import (
	"strings"
	"testing"
)

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

var ciCredentials = map[string]string{
	"SAP_INTEGRATION_SUITE_HOST": "h", "SAP_INTEGRATION_SUITE_TOKEN_URL": "t",
	"SAP_INTEGRATION_SUITE_CLIENT_ID": "i", "SAP_INTEGRATION_SUITE_CLIENT_SECRET": "s",
}

func with(base map[string]string, extra ...string) map[string]string {
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for i := 0; i+1 < len(extra); i += 2 {
		out[extra[i]] = extra[i+1]
	}
	return out
}

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name        string
		vars        map[string]string
		capability  Capability
		destructive bool
		extra       []string
		wantRun     bool
		wantReason  string
	}{
		{"no TF_ACC", with(ciCredentials, EnvAll, "1"), CloudIntegration, false, nil, false, ReasonNoTFAcc},
		{"TF_ACC alone", with(ciCredentials, "TF_ACC", "1"), CloudIntegration, false, nil, false, ReasonGateOff},
		{"ALL", with(ciCredentials, "TF_ACC", "1", EnvAll, "1"), CloudIntegration, false, nil, true, ""},
		{"capability gate", with(ciCredentials, "TF_ACC", "1", CloudIntegration.Gate(), "true"), CloudIntegration, false, nil, true, ""},
		{"other capability gate", with(ciCredentials, "TF_ACC", "1", SecurityContent.Gate(), "1"), CloudIntegration, false, nil, false, ReasonGateOff},
		{"gate set to 0", with(ciCredentials, "TF_ACC", "1", CloudIntegration.Gate(), "0"), CloudIntegration, false, nil, false, ReasonGateOff},
		{"no credentials", map[string]string{"TF_ACC": "1", EnvAll: "1"}, CloudIntegration, false, nil, false, ReasonCredentials},
		{"missing input", with(ciCredentials, "TF_ACC", "1", EnvAll, "1"), CloudIntegration, false, []string{"SAP_LOCAL_CONTENT_DIR"}, false, ReasonMissingInput},
		{"input set", with(ciCredentials, "TF_ACC", "1", EnvAll, "1", "SAP_LOCAL_CONTENT_DIR", "d"), CloudIntegration, false, []string{"SAP_LOCAL_CONTENT_DIR"}, true, ""},
		{"destructive with ALL and DESTRUCTIVE", with(ciCredentials, "TF_ACC", "1", EnvAll, "1", EnvDestructive, "1"), CloudIntegration, true, nil, false, ReasonDestructive},
		{"destructive with gate only", with(ciCredentials, "TF_ACC", "1", CloudIntegration.Gate(), "1"), CloudIntegration, true, nil, false, ReasonDestructive},
		{"destructive with gate and DESTRUCTIVE", with(ciCredentials, "TF_ACC", "1", CloudIntegration.Gate(), "1", EnvDestructive, "1"), CloudIntegration, true, nil, true, ""},
		{"capability without service", with(ciCredentials, "TF_ACC", "1", EnvAll, "1"), DataSpaceIntegration, false, nil, false, "no Data Space Integration service"},
		{"unknown capability", with(ciCredentials, "TF_ACC", "1", EnvAll, "1"), Capability("NOPE"), false, nil, false, "unknown capability"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := Evaluate(env(tc.vars), tc.capability, tc.destructive, tc.extra...)
			if d.Run != tc.wantRun {
				t.Fatalf("Run = %v, want %v (%s)", d.Run, tc.wantRun, d.Status)
			}
			if tc.wantRun && d.Status != "RUN" {
				t.Errorf("status %q", d.Status)
			}
			if !tc.wantRun && !strings.Contains(d.Status, tc.wantReason) {
				t.Errorf("status %q does not contain %q", d.Status, tc.wantReason)
			}
		})
	}
}

func TestStatusNeverContainsValues(t *testing.T) {
	vars := with(ciCredentials, "TF_ACC", "1", EnvAll, "1")
	delete(vars, "SAP_INTEGRATION_SUITE_CLIENT_SECRET")
	vars["SAP_INTEGRATION_SUITE_CLIENT_ID"] = "very-secret-client-id"
	d := Evaluate(env(vars), CloudIntegration, false)
	if strings.Contains(d.Status, "very-secret-client-id") {
		t.Errorf("status leaks a value: %q", d.Status)
	}
	if !strings.Contains(d.Status, "SAP_INTEGRATION_SUITE_CLIENT_SECRET") {
		t.Errorf("status should name the missing variable: %q", d.Status)
	}
}

func TestServiceCredentialsAreCredentials(t *testing.T) {
	vars := map[string]string{"TF_ACC": "1", Metadata.Gate(): "1"}
	d := Check{Capability: Metadata, Credentials: []string{"SAP_INTEGRATION_SUITE_API_MANAGEMENT_HOST"}}.Evaluate(env(vars))
	if d.Run || !strings.Contains(d.Status, ReasonCredentials) {
		t.Fatalf("status %q", d.Status)
	}
	vars["SAP_INTEGRATION_SUITE_API_MANAGEMENT_HOST"] = "h"
	if d := (Check{Capability: Metadata, Credentials: []string{"SAP_INTEGRATION_SUITE_API_MANAGEMENT_HOST"}}).Evaluate(env(vars)); !d.Run {
		t.Fatalf("status %q", d.Status)
	}
}

func TestEveryCapabilityIsListed(t *testing.T) {
	for _, c := range []Capability{Metadata, CloudIntegration, SecurityContent, PartnerDirectory, APIManagementClassic,
		APIManagementCurrent, APIComposition, IntegrationAssessment, EdgeIntegrationCell, DataSpaceIntegration} {
		if _, ok := Lookup(c); !ok {
			t.Errorf("%s is not in Capabilities", c)
		}
	}
}
