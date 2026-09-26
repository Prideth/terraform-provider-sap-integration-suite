package apimeta

import (
	"encoding/json"
	"strings"
	"testing"
)

type policyRead struct {
	ID          string `json:"Id"` // Edm.Int64 arrives as a string
	RoleName    string `json:"RoleName"`
	Description string `json:"Description,omitempty"`
}

type policyWrongTypes struct {
	ID         int64      `json:"Id"`         // Edm.Int64 is a JSON string in V2
	References []struct{} `json:"References"` // navigation: needs {"results": ...}
	Unknown    string     `json:"Unknown"`
}

type expanded struct {
	Results []json.RawMessage `json:"results"`
}

type policyExpanded struct {
	References expanded `json:"References"`
}

type resourceRead struct {
	Name string `json:"Name"` // inherited from Base
	Size int32  `json:"Size"`
}

type flagged struct {
	Flag bool `json:"Description"` // Edm.String arrives as a string
}

func testContract() Contract {
	return Contract{
		Service: "v2",
		Reads: []StructUse{
			{EntitySet: "Policies", Value: policyRead{}},
			{EntitySet: "Policies", Value: policyExpanded{}},
			{EntitySet: "Resources", Value: resourceRead{}},
		},
		Writes:      []StructUse{{EntitySet: "Policies", Value: policyWrongTypes{References: nil}}},
		Keys:        []KeyUse{Key("Policies", "Id", "Edm.Int64"), Key("Resources", "Name", "Edm.String")},
		Navigations: []NavigationUse{{EntitySet: "Policies", Property: "References"}},
		Operations:  []OperationUse{{Name: "Deploy", HTTPMethod: "POST", Parameters: []string{"Id", "Version"}}},
		MaxLengths:  []MaxLengthUse{{EntitySet: "Policies", Property: "RoleName", Value: 100}},
	}
}

func TestVerify_Passes(t *testing.T) {
	c := testContract()
	c.Writes = nil // the write struct names an unknown field on purpose
	if problems := Verify(mustParse(t, "v2.xml"), c); len(problems) != 0 {
		t.Errorf("problems = %v", problems)
	}
}

func TestVerify_FindsProblems(t *testing.T) {
	s := mustParse(t, "v2.xml")
	c := Contract{
		Reads: []StructUse{{EntitySet: "Policies", Value: policyWrongTypes{}}, {EntitySet: "Policies", Value: flagged{}}},
		Keys:  []KeyUse{Key("References", "Id", "Edm.String")},
		Operations: []OperationUse{
			{Name: "Deploy", HTTPMethod: "GET", Parameters: []string{"Id"}},
			{Name: "Undeploy"},
		},
		MaxLengths: []MaxLengthUse{{EntitySet: "Policies", Property: "RoleName", Value: 255}},
	}
	c.Reads = append(c.Reads, StructUse{EntitySet: "Missing", Value: policyRead{}})
	problems := strings.Join(Verify(s, c), "\n")
	for _, want := range []string{
		`JSON field "Id" is int64`,
		`JSON field "References" maps the navigation property`,
		`JSON field "Unknown" is not a property`,
		`JSON field "Description" is bool`,
		`key Id has type "Edm.Int64", the client uses "Edm.String"`,
		`operation Deploy uses POST, the client sends GET`,
		`operation Deploy has 2 parameters, the client sends 1`,
		`operation Undeploy does not exist`,
		`MaxLength = "100", the client enforces 255`,
		`entity set Missing does not exist`,
	} {
		if !strings.Contains(problems, want) {
			t.Errorf("missing %q in\n%s", want, problems)
		}
	}
}

// A write struct may shape navigation properties freely, but may not name
// unknown fields.
func TestVerify_WriteStruct(t *testing.T) {
	problems := Verify(mustParse(t, "v2.xml"), Contract{Writes: []StructUse{{EntitySet: "Policies", Value: policyWrongTypes{}}}})
	if len(problems) != 1 || !strings.Contains(problems[0], `"Unknown"`) {
		t.Errorf("problems = %v", problems)
	}
}

func TestContract_EntitySets(t *testing.T) {
	got := strings.Join(testContract().EntitySets(), ",")
	if got != "Policies,Resources" {
		t.Errorf("EntitySets = %s", got)
	}
}
