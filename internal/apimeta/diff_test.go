package apimeta

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// clone returns a deep copy through the snapshot format, which also proves
// that a snapshot round-trips.
func clone(t *testing.T, s *Service) *Service {
	t.Helper()
	data, err := Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var out Service
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return &out
}

func findChange(changes []Change, kind, subject, path string) *Change {
	for i := range changes {
		c := changes[i]
		if c.Kind == kind && c.Subject == subject && c.Path == path {
			return &changes[i]
		}
	}
	return nil
}

func TestDiff_IdenticalIsEmpty(t *testing.T) {
	s := mustParse(t, "v2.xml")
	if changes := Diff(s, clone(t, s)); len(changes) != 0 {
		t.Errorf("changes = %v", changes)
	}
}

func TestDiff_AdditiveChanges(t *testing.T) {
	old := mustParse(t, "v2.xml")
	new := clone(t, old)
	new.EntitySets = append(new.EntitySets, EntitySet{Name: "OAuth2PasswordCredentials", EntityType: "com.example.api.Credential"})
	new.EntityTypes = append(new.EntityTypes, EntityType{Name: "com.example.api.Credential", Key: []string{"Name"}})
	policy := typeNamed(new, "com.example.api.Policy")
	policy.Properties = append(policy.Properties, Property{Name: "Owner", Type: "Edm.String", Nullable: true})
	policy.Navigation = append(policy.Navigation, NavigationProperty{Name: "Runtimes", Target: "com.example.api.Reference", Collection: true})
	new.Operations = append(new.Operations, Operation{Name: "DeployAPIArtifact", Kind: KindFunctionImport, HTTPMethod: "POST",
		Parameters: []Parameter{{Name: "Id", Type: "Edm.String"}}})
	// Accepting null is not breaking.
	propertyNamed(policy, "RoleName").Nullable = true
	Normalize(new)

	changes := Diff(old, new)
	for _, want := range []struct{ kind, subject, path string }{
		{ChangeNew, "ENTITY SET", "OAuth2PasswordCredentials"},
		{ChangeNew, "ENTITY TYPE", "com.example.api.Credential"},
		{ChangeNew, "PROPERTY", "com.example.api.Policy.Owner"},
		{ChangeNew, "NAVIGATION PROPERTY", "com.example.api.Policy.Runtimes"},
		{ChangeNew, "FUNCTION IMPORT", "DeployAPIArtifact"},
		{ChangeChanged, "PROPERTY", "com.example.api.Policy.RoleName"},
	} {
		if findChange(changes, want.kind, want.subject, want.path) == nil {
			t.Errorf("missing %s %s %s in\n%v", want.kind, want.subject, want.path, changes)
		}
	}
	if b := Breaking(changes); len(b) != 0 {
		t.Errorf("additive changes reported as breaking: %v", b)
	}
	c := findChange(changes, ChangeNew, "ENTITY SET", "OAuth2PasswordCredentials")
	if got := c.String(); !strings.HasPrefix(got, "NEW ENTITY SET: OAuth2PasswordCredentials") {
		t.Errorf("rendering = %q", got)
	}
}

func TestDiff_BreakingChanges(t *testing.T) {
	old := mustParse(t, "v2.xml")
	new := clone(t, old)
	// Removed entity set and property, changed key, stricter nullability,
	// shorter MaxLength, retyped property, a navigation to another target,
	// a lost write annotation and a changed operation signature.
	new.EntitySets = new.EntitySets[:len(new.EntitySets)-1] // Resources
	policy := typeNamed(new, "com.example.api.Policy")
	policy.Properties = policy.Properties[1:] // Description (sorted first)
	propertyNamed(policy, "RoleName").MaxLength = "50"
	propertyNamed(policy, "Id").Type = "Edm.String"
	policy.Annotations["sap:creatable"] = "false"
	ref := typeNamed(new, "com.example.api.Reference")
	ref.Key = []string{"Id", "PolicyId"}
	ref.Navigation[0].Target = "com.example.api.Resource"
	settings := typeNamed(new, "com.example.api.Settings")
	propertyNamed(settings, "Level").Nullable = false
	new.Operations[0].Parameters = append(new.Operations[0].Parameters, Parameter{Name: "Target", Type: "Edm.String", Nullable: false})

	changes := Diff(old, new)
	for _, want := range []struct{ kind, subject, path string }{
		{ChangeRemoved, "ENTITY SET", "Resources"},
		{ChangeRemoved, "PROPERTY", "com.example.api.Policy.Description"},
		{ChangeChanged, "PROPERTY", "com.example.api.Policy.RoleName"},
		{ChangeChanged, "PROPERTY", "com.example.api.Policy.Id"},
		{ChangeChanged, "ANNOTATION", "com.example.api.Policy @sap:creatable"},
		{ChangeChanged, "KEY", "com.example.api.Reference"},
		{ChangeChanged, "NAVIGATION PROPERTY", "com.example.api.Reference.Policy"},
		{ChangeChanged, "PROPERTY", "com.example.api.Settings.Level"},
		{ChangeChanged, "FUNCTION IMPORT", "Deploy"},
	} {
		c := findChange(changes, want.kind, want.subject, want.path)
		if c == nil {
			t.Errorf("missing %s %s %s in\n%v", want.kind, want.subject, want.path, changes)
			continue
		}
		if !c.Breaking {
			t.Errorf("%s %s %s must be breaking", want.kind, want.subject, want.path)
		}
	}
	c := findChange(changes, ChangeChanged, "PROPERTY", "com.example.api.Settings.Level")
	want := "PROPERTY CHANGED: com.example.api.Settings.Level\n  Edm.Int32 nullable=true\n  ->\n  Edm.Int32 nullable=false\n  (breaking)"
	if c != nil && c.String() != want {
		t.Errorf("rendering =\n%s\nwant\n%s", c.String(), want)
	}
}

func TestDiff_RemovedOperationAndLongerLimit(t *testing.T) {
	old := mustParse(t, "v2.xml")
	new := clone(t, old)
	new.Operations = nil
	propertyNamed(typeNamed(new, "com.example.api.Policy"), "RoleName").MaxLength = "200"
	changes := Diff(old, new)
	if c := findChange(changes, ChangeRemoved, "FUNCTION IMPORT", "Deploy"); c == nil || !c.Breaking {
		t.Errorf("removed operation must be reported as breaking: %v", changes)
	}
	if c := findChange(changes, ChangeChanged, "PROPERTY", "com.example.api.Policy.RoleName"); c == nil || c.Breaking {
		t.Errorf("a longer MaxLength must be a non-breaking change: %v", c)
	}
}

// The snapshot of a document does not depend on the order of its elements.
func TestNormalize_Deterministic(t *testing.T) {
	data, err := os.ReadFile("testdata/v2.xml")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := ParseEDMX("x", data)
	// Move the Orphan type and the References set to other positions.
	text := string(data)
	orphan := text[strings.Index(text, `      <EntityType Name="Orphan">`):strings.Index(text, `      <ComplexType Name="Settings">`)]
	text = strings.Replace(text, orphan, "", 1)
	text = strings.Replace(text, `      <EntityType Name="Policy"`, orphan+`      <EntityType Name="Policy"`, 1)
	set := `        <EntitySet Name="References" EntityType="Ex.Reference"/>` + "\n"
	text = strings.Replace(text, set, "", 1)
	text = strings.Replace(text, `        <EntitySet Name="Policies"`, set+`        <EntitySet Name="Policies"`, 1)
	b, err := ParseEDMX("x", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	ja, _ := Marshal(a)
	jb, _ := Marshal(b)
	if !bytes.Equal(ja, jb) {
		t.Errorf("snapshots differ after reordering:\n%s\n---\n%s", ja, jb)
	}
	if !bytes.HasSuffix(ja, []byte("\n")) {
		t.Error("snapshot must end with a newline")
	}
}

func TestRedactURLs(t *testing.T) {
	in := `see https://tenant.example.invalid/api/v1 and http://x.y/z?a=1`
	got := RedactURLs(in)
	if strings.Contains(got, "http") || strings.Contains(got, "ondemand") {
		t.Errorf("RedactURLs = %q", got)
	}
}

// A snapshot must not carry a URL, even one hidden in an annotation.
func TestSnapshot_ContainsNoURL(t *testing.T) {
	for _, f := range []string{"v2.xml", "v4.xml"} {
		data, _ := Marshal(mustParse(t, f))
		if strings.Contains(string(data), "https://") || strings.Contains(string(data), "http://") {
			t.Errorf("%s snapshot contains a URL:\n%s", f, data)
		}
	}
}
