package apimeta

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func mustParse(t *testing.T, file string) *Service {
	t.Helper()
	data, err := os.ReadFile("testdata/" + file)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ParseEDMX(strings.TrimSuffix(file, ".xml"), data)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func typeNamed(s *Service, name string) *EntityType {
	for i := range s.EntityTypes {
		if s.EntityTypes[i].Name == name {
			return &s.EntityTypes[i]
		}
	}
	for i := range s.ComplexTypes {
		if s.ComplexTypes[i].Name == name {
			return &s.ComplexTypes[i]
		}
	}
	return nil
}

func propertyNamed(t *EntityType, name string) *Property {
	for i := range t.Properties {
		if t.Properties[i].Name == name {
			return &t.Properties[i]
		}
	}
	return nil
}

func TestParseEDMX_V2(t *testing.T) {
	s := mustParse(t, "v2.xml")
	if s.Protocol != ProtocolODataV2 {
		t.Errorf("protocol = %s", s.Protocol)
	}

	policy := typeNamed(s, "com.example.api.Policy")
	if policy == nil {
		t.Fatal("Policy missing")
	}
	// The alias Ex is resolved to the namespace.
	if p := propertyNamed(policy, "Settings"); p == nil || p.Type != "com.example.api.Settings" {
		t.Errorf("Settings property = %+v, want the complex type with its namespace", p)
	}
	role := propertyNamed(policy, "RoleName")
	if role == nil || role.Nullable || role.MaxLength != "100" {
		t.Errorf("RoleName = %+v", role)
	}
	if d := propertyNamed(policy, "Description"); d == nil || !d.Nullable {
		t.Error("an absent Nullable facet must mean nullable")
	}
	// Contract annotations are kept, UI labels are not.
	if policy.Annotations["sap:creatable"] != "true" || policy.Annotations["sap:label"] != "" {
		t.Errorf("annotations = %v", policy.Annotations)
	}
	if role.Annotations != nil {
		t.Errorf("sap:label on a property must be dropped, got %v", role.Annotations)
	}

	// V2 navigation targets come from the association ends.
	if len(policy.Navigation) != 1 || policy.Navigation[0].Target != "com.example.api.Reference" || !policy.Navigation[0].Collection {
		t.Errorf("Policy.References = %+v", policy.Navigation)
	}
	ref := typeNamed(s, "com.example.api.Reference")
	if ref.Navigation[0].Target != "com.example.api.Policy" || ref.Navigation[0].Collection {
		t.Errorf("Reference.Policy = %+v", ref.Navigation[0])
	}
	if len(s.Associations) != 1 {
		t.Fatalf("associations = %+v", s.Associations)
	}
	a := s.Associations[0]
	if a.Principal != "Policy" || a.Dependent != "Reference" ||
		!reflect.DeepEqual(a.Constraints, []ReferentialConstraint{{Property: "PolicyId", ReferencedProperty: "Id"}}) {
		t.Errorf("association = %+v", a)
	}

	resource := typeNamed(s, "com.example.api.Resource")
	if !resource.HasStream || resource.BaseType != "com.example.api.Base" {
		t.Errorf("Resource = %+v, want a media entity derived from Base", resource)
	}
	if !typeNamed(s, "com.example.api.Base").Abstract {
		t.Error("Base must be abstract")
	}

	if len(s.Operations) != 1 {
		t.Fatalf("operations = %+v", s.Operations)
	}
	op := s.Operations[0]
	if op.Kind != KindFunctionImport || op.HTTPMethod != "POST" || op.ReturnType != "Edm.String" ||
		len(op.Parameters) != 2 || op.Parameters[0].Name != "Id" || op.Parameters[1].Mode != "In" {
		t.Errorf("Deploy = %+v", op)
	}
}

func TestParseEDMX_V4(t *testing.T) {
	s := mustParse(t, "v4.xml")
	if s.Protocol != ProtocolODataV4 {
		t.Errorf("protocol = %s", s.Protocol)
	}
	graph := typeNamed(s, "com.example.graph.Graph")
	if graph == nil {
		t.Fatal("Graph missing")
	}
	nav := graph.Navigation[0]
	if nav.Target != "com.example.graph.Version" || !nav.Collection || nav.Partner != "graph" || !nav.ContainsTarget {
		t.Errorf("versions = %+v", nav)
	}
	back := typeNamed(s, "com.example.graph.Version").Navigation[0]
	if !reflect.DeepEqual(back.Constraints, []ReferentialConstraint{{Property: "graphId", ReferencedProperty: "id"}}) {
		t.Errorf("graph constraints = %+v", back.Constraints)
	}
	// Inline and targeted annotations, URLs redacted, the alias Core resolved.
	if got := graph.Annotations["Org.OData.Core.V1.Description"]; got != "See <redacted-url>" {
		t.Errorf("description annotation = %q", got)
	}
	if got := graph.Annotations["Org.OData.Capabilities.V1.InsertRestrictions/Insertable"]; got != "false" {
		t.Errorf("insert restriction = %q", got)
	}
	if p := propertyNamed(graph, "dataSources"); p.Type != "Collection(com.example.graph.DataSource)" {
		t.Errorf("dataSources type = %s", p.Type)
	}

	if len(s.Singletons) != 1 || s.Singletons[0].Name != "settings" {
		t.Errorf("singletons = %+v", s.Singletons)
	}
	if s.EntitySets[0].NavigationBindings["versions"] != "versions" {
		t.Errorf("bindings = %+v", s.EntitySets[0].NavigationBindings)
	}
	kinds := map[string]bool{}
	for _, op := range s.Operations {
		kinds[op.Kind+" "+op.Name] = true
		if op.Name == "com.example.graph.activate" && (!op.Bound || op.ReturnType != "com.example.graph.Graph") {
			t.Errorf("activate = %+v", op)
		}
	}
	for _, want := range []string{"Action com.example.graph.activate", "Function com.example.graph.latest", "FunctionImport latest"} {
		if !kinds[want] {
			t.Errorf("missing operation %s in %v", want, kinds)
		}
	}
}

func TestParseEDMX_RejectsNonEDMX(t *testing.T) {
	if _, err := ParseEDMX("x", []byte(`{"not": "xml"}`)); err == nil {
		t.Error("want an error for JSON")
	}
	if _, err := ParseEDMX("x", []byte(`<edmx:Edmx xmlns:edmx="e"/>`)); err == nil {
		t.Error("want an error for a document without schema")
	}
}

// The traversal follows properties, complex types, base types, navigation
// (including the cycle Policy -> Reference -> Policy) and operation
// signatures, and terminates.
func TestTraverse_V2(t *testing.T) {
	g := mustParse(t, "v2.xml").Graph
	want := []string{"com.example.api.Base", "com.example.api.Policy", "com.example.api.Reference",
		"com.example.api.Resource", "com.example.api.Settings"}
	if !reflect.DeepEqual(g.Reachable, want) {
		t.Errorf("reachable = %v, want %v", g.Reachable, want)
	}
	if !reflect.DeepEqual(g.Unreachable, []string{"com.example.api.Orphan"}) {
		t.Errorf("unreachable = %v", g.Unreachable)
	}
	if len(g.Unresolved) != 0 {
		t.Errorf("unresolved = %v", g.Unresolved)
	}
}

// V4: nested complex types, enums, the singleton, the unbound function's
// return type, a bound action reached through its binding type, and a bound
// action on a type reached only by navigation.
func TestTraverse_V4(t *testing.T) {
	g := mustParse(t, "v4.xml").Graph
	want := []string{"com.example.graph.Audit", "com.example.graph.DataSource", "com.example.graph.Graph",
		"com.example.graph.Service", "com.example.graph.Settings", "com.example.graph.Status", "com.example.graph.Version"}
	if !reflect.DeepEqual(g.Reachable, want) {
		t.Errorf("reachable = %v, want %v", g.Reachable, want)
	}
	if !reflect.DeepEqual(g.Unreachable, []string{"com.example.graph.Unused"}) {
		t.Errorf("unreachable = %v", g.Unreachable)
	}
}

// A reference no schema declares is reported, not followed.
func TestTraverse_Unresolved(t *testing.T) {
	s := &Service{
		EntitySets:  []EntitySet{{Name: "As", EntityType: "ns.A"}},
		EntityTypes: []EntityType{{Name: "ns.A", Navigation: []NavigationProperty{{Name: "b", Target: "ns.Missing"}}}},
	}
	g := Traverse(s)
	if !reflect.DeepEqual(g.Unresolved, []string{"ns.Missing"}) || !reflect.DeepEqual(g.Reachable, []string{"ns.A"}) {
		t.Errorf("graph = %+v", g)
	}
}
