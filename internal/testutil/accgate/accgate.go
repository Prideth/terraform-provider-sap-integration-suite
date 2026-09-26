// Package accgate decides whether an acceptance test runs. A test runs only
// with TF_ACC=1, its capability enabled, and the credentials of that
// capability present:
//
//	SAP_INTEGRATION_SUITE_ACC_ALL=1           every capability, never destructive tests
//	SAP_INTEGRATION_SUITE_ACC_<CAPABILITY>=1  one capability
//	SAP_INTEGRATION_SUITE_ACC_DESTRUCTIVE=1   additionally allows destructive tests of the
//	                                          capabilities enabled one by one
//
// A destructive test is one that changes tenant-wide singleton
// configuration or could otherwise disturb content it did not create. It
// runs only when its capability is enabled by name and the destructive gate
// is set; SAP_INTEGRATION_SUITE_ACC_ALL is never enough.
//
// Every decision has a status line (RUN, or SKIPPED with the reason) that
// names environment variables but never prints their values; cmd/accplan
// prints these lines for every acceptance test without running any.
package accgate

import (
	"os"
	"sort"
	"strings"
	"testing"
)

// Capability is an acceptance suite; its gate is SAP_INTEGRATION_SUITE_ACC_<Capability>.
type Capability string

// The capabilities. The constant names are read by cmd/accplan.
const (
	Metadata              Capability = "METADATA"
	CloudIntegration      Capability = "CLOUD_INTEGRATION"
	SecurityContent       Capability = "SECURITY_CONTENT"
	PartnerDirectory      Capability = "PARTNER_DIRECTORY"
	APIManagementClassic  Capability = "API_MANAGEMENT_CLASSIC"
	APIManagementCurrent  Capability = "API_MANAGEMENT_CURRENT"
	APIComposition        Capability = "API_COMPOSITION"
	IntegrationAssessment Capability = "INTEGRATION_ASSESSMENT"
	EdgeIntegrationCell   Capability = "EDGE_INTEGRATION_CELL"
	DataSpaceIntegration  Capability = "DATA_SPACE_INTEGRATION"
)

// Environment variables of the gates.
const (
	EnvTFAcc       = "TF_ACC"
	EnvPrefix      = "SAP_INTEGRATION_SUITE_ACC_"
	EnvAll         = EnvPrefix + "ALL"
	EnvDestructive = EnvPrefix + "DESTRUCTIVE"
	// EnvDiscoveryStrict makes the metadata acceptance tests fail on any
	// difference from the snapshots, additive ones included.
	EnvDiscoveryStrict = "SAP_INTEGRATION_SUITE_DISCOVERY_STRICT"
)

// Gate returns the environment variable that enables a capability.
func (c Capability) Gate() string { return EnvPrefix + string(c) }

var (
	mainCredentials = []string{"SAP_INTEGRATION_SUITE_HOST", "SAP_INTEGRATION_SUITE_TOKEN_URL",
		"SAP_INTEGRATION_SUITE_CLIENT_ID", "SAP_INTEGRATION_SUITE_CLIENT_SECRET"}
	apiManagementCredentials = []string{"SAP_INTEGRATION_SUITE_API_MANAGEMENT_HOST", "SAP_INTEGRATION_SUITE_API_MANAGEMENT_TOKEN_URL",
		"SAP_INTEGRATION_SUITE_API_MANAGEMENT_CLIENT_ID", "SAP_INTEGRATION_SUITE_API_MANAGEMENT_CLIENT_SECRET"}
)

// Info describes a capability.
type Info struct {
	Capability Capability
	// Credentials are the variables every test of the capability needs.
	Credentials []string
	// NoService is set while no service root of the capability is
	// confirmed; its tests always skip with this reason.
	NoService string
}

// Capabilities lists every capability in a fixed order.
var Capabilities = []Info{
	// The metadata tests take the credentials of each service they read.
	{Capability: Metadata},
	{Capability: CloudIntegration, Credentials: mainCredentials},
	{Capability: SecurityContent, Credentials: mainCredentials},
	{Capability: PartnerDirectory, Credentials: mainCredentials},
	{Capability: APIManagementClassic, Credentials: apiManagementCredentials},
	{Capability: APIManagementCurrent, NoService: "no public service root of the current API Management is confirmed yet"},
	{Capability: APIComposition, Credentials: []string{"SAP_INTEGRATION_SUITE_API_COMPOSITION_HOST",
		"SAP_INTEGRATION_SUITE_API_COMPOSITION_TOKEN_URL", "SAP_INTEGRATION_SUITE_API_COMPOSITION_CLIENT_ID",
		"SAP_INTEGRATION_SUITE_API_COMPOSITION_CLIENT_SECRET"}},
	{Capability: IntegrationAssessment, Credentials: []string{"SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_TOKEN_URL",
		"SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_CLIENT_ID", "SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_CLIENT_SECRET"}},
	{Capability: EdgeIntegrationCell, Credentials: append(append([]string{}, mainCredentials...), "SAP_INTEGRATION_SUITE_RUNTIME_LOCATION_ID")},
	{Capability: DataSpaceIntegration, NoService: "no Data Space Integration service is configured in the provider yet"},
}

// Lookup returns a capability's description.
func Lookup(c Capability) (Info, bool) {
	for _, i := range Capabilities {
		if i.Capability == c {
			return i, true
		}
	}
	return Info{}, false
}

// Decision is whether a test runs, and why not.
type Decision struct {
	Run bool
	// Status is "RUN" or "SKIPPED — <reason>".
	Status string
}

// Skip reasons, as the plan prints them.
const (
	ReasonNoTFAcc      = "TF_ACC is not set"
	ReasonGateOff      = "capability gate not enabled"
	ReasonDestructive  = "destructive gate not enabled"
	ReasonCredentials  = "credentials unavailable"
	ReasonMissingInput = "test input not set"
)

// Check is what one acceptance test needs.
type Check struct {
	Capability  Capability
	Destructive bool
	// Credentials are needed on top of the capability's own, for tests
	// that address one particular service.
	Credentials []string
	// Inputs are further variables the test needs, such as a local
	// content directory; they are reported separately from credentials.
	Inputs []string
}

// Evaluate decides whether a test of capability c runs; extraEnv lists the
// test's inputs.
func Evaluate(getenv func(string) string, c Capability, destructive bool, extraEnv ...string) Decision {
	return Check{Capability: c, Destructive: destructive, Inputs: extraEnv}.Evaluate(getenv)
}

// Evaluate decides whether the test runs.
func (ch Check) Evaluate(getenv func(string) string) Decision {
	c := ch.Capability
	info, ok := Lookup(c)
	if !ok {
		return skipped("unknown capability " + string(c))
	}
	if getenv(EnvTFAcc) == "" {
		return skipped(ReasonNoTFAcc)
	}
	named := enabled(getenv(c.Gate()))
	if !named && !enabled(getenv(EnvAll)) {
		return skipped(ReasonGateOff + " (set " + EnvAll + "=1 or " + c.Gate() + "=1)")
	}
	if ch.Destructive && (!named || !enabled(getenv(EnvDestructive))) {
		return skipped(ReasonDestructive + " (needs " + c.Gate() + "=1 and " + EnvDestructive + "=1; " + EnvAll + " never runs destructive tests)")
	}
	if info.NoService != "" {
		return skipped(info.NoService)
	}
	creds := append(append([]string{}, info.Credentials...), ch.Credentials...)
	if missing := missingVars(getenv, creds); len(missing) > 0 {
		return skipped(ReasonCredentials + " (missing " + strings.Join(missing, ", ") + ")")
	}
	if missing := missingVars(getenv, ch.Inputs); len(missing) > 0 {
		return skipped(ReasonMissingInput + " (missing " + strings.Join(missing, ", ") + ")")
	}
	return Decision{Run: true, Status: "RUN"}
}

func skipped(reason string) Decision {
	return Decision{Status: "SKIPPED — " + reason}
}

func enabled(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func missingVars(getenv func(string) string, names []string) []string {
	var missing []string
	for _, n := range names {
		if getenv(n) == "" {
			missing = append(missing, n)
		}
	}
	sort.Strings(missing)
	return missing
}

// Require skips the test unless a non-destructive test of capability c may
// run. Call it first in every acceptance test, before any setup.
func Require(t testing.TB, c Capability, extraEnv ...string) {
	t.Helper()
	if d := Evaluate(os.Getenv, c, false, extraEnv...); !d.Run {
		t.Skip(d.Status)
	}
}

// RequireService skips the test unless a non-destructive test of capability
// c may run and the credentials of one particular service are set.
func RequireService(t testing.TB, c Capability, credentials ...string) {
	t.Helper()
	if d := (Check{Capability: c, Credentials: credentials}).Evaluate(os.Getenv); !d.Run {
		t.Skip(d.Status)
	}
}

// RequireDestructive skips the test unless a destructive test of capability
// c may run.
func RequireDestructive(t testing.TB, c Capability, extraEnv ...string) {
	t.Helper()
	if d := Evaluate(os.Getenv, c, true, extraEnv...); !d.Run {
		t.Skip(d.Status)
	}
}

// Strict reports whether the metadata acceptance tests fail on additive
// differences too.
func Strict() bool { return enabled(os.Getenv(EnvDiscoveryStrict)) }
