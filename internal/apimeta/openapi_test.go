package apimeta

import (
	"os"
	"reflect"
	"testing"
)

func mustParseOpenAPI(t *testing.T, file string) *Service {
	t.Helper()
	data, err := os.ReadFile("testdata/" + file)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ParseOpenAPI(file, data)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func restOp(s *Service, method, path string) *RESTOperation {
	for i := range s.REST.Operations {
		if s.REST.Operations[i].Method == method && s.REST.Operations[i].Path == path {
			return &s.REST.Operations[i]
		}
	}
	return nil
}

func TestParseOpenAPI_V3(t *testing.T) {
	s := mustParseOpenAPI(t, "openapi3.json")
	if s.Protocol != ProtocolOpenAPI || s.REST.Title != "Example Configuration API" || s.REST.Version != "1.2.0" {
		t.Errorf("service = %+v", s.REST)
	}
	if !reflect.DeepEqual(s.REST.SecuritySchemes, []string{"oauth2"}) {
		t.Errorf("security schemes = %v", s.REST.SecuritySchemes)
	}
	post := restOp(s, "POST", "/graphs")
	if post == nil || post.OperationID != "createGraph" || post.RequestBody != "application/json #Graph" ||
		post.Responses["201"] != "application/json #Graph" || !reflect.DeepEqual(post.Security, []string{"oauth2"}) {
		t.Errorf("POST /graphs = %+v", post)
	}
	get := restOp(s, "GET", "/graphs")
	if get.Responses["200"] != "application/json array<#Graph>" || get.Parameters[0].Type != "integer/int32" {
		t.Errorf("GET /graphs = %+v", get)
	}
	// Path-level parameters apply to every method on the path.
	del := restOp(s, "DELETE", "/graphs/{id}")
	if del == nil || len(del.Parameters) != 1 || del.Parameters[0].Name != "id" || !del.Parameters[0].Required {
		t.Errorf("DELETE /graphs/{id} = %+v", del)
	}
	var graph RESTSchema
	for _, sch := range s.REST.Schemas {
		if sch.Name == "Graph" {
			graph = sch
		}
	}
	want := []RESTProperty{{Name: "dataSources", Type: "array<#DataSource>"}, {Name: "id", Type: "string", Required: true}}
	if !reflect.DeepEqual(graph.Properties, want) {
		t.Errorf("Graph properties = %+v", graph.Properties)
	}
}

// Swagger 2 in YAML: body parameters become the request body, numeric
// response codes are read as strings, types sit on the parameter.
func TestParseOpenAPI_Swagger2YAML(t *testing.T) {
	s := mustParseOpenAPI(t, "swagger2.yaml")
	post := restOp(s, "POST", "/APIProxies")
	if post == nil || post.RequestBody != "#Proxy" || post.Responses["201"] != "#Proxy" || len(post.Parameters) != 0 {
		t.Errorf("POST /APIProxies = %+v", post)
	}
	get := restOp(s, "GET", "/APIProxies")
	if get.Responses["200"] != "string/binary" || get.Parameters[0].Type != "string" || !get.Parameters[0].Required {
		t.Errorf("GET /APIProxies = %+v", get)
	}
	if !reflect.DeepEqual(s.REST.SecuritySchemes, []string{"basic"}) {
		t.Errorf("security schemes = %v", s.REST.SecuritySchemes)
	}
}

func TestParseOpenAPI_Rejects(t *testing.T) {
	if _, err := ParseOpenAPI("x", []byte("<xml/>")); err == nil {
		t.Error("want an error for XML")
	}
	if _, err := ParseOpenAPI("x", []byte(`{"info": {}}`)); err == nil {
		t.Error("want an error for a document without openapi or swagger")
	}
}

func TestDiff_REST(t *testing.T) {
	old := mustParseOpenAPI(t, "openapi3.json")
	new := clone(t, old)
	new.REST.Operations = append(new.REST.Operations, RESTOperation{Method: "PATCH", Path: "/graphs/{id}", OperationID: "updateGraph"})
	restOp(new, "DELETE", "/graphs/{id}").Parameters[0].Type = "integer"
	restOp(new, "GET", "/graphs").Parameters = append(restOp(new, "GET", "/graphs").Parameters,
		RESTParameter{Name: "tenant", In: "header", Type: "string", Required: true})
	for i := range new.REST.Schemas {
		if new.REST.Schemas[i].Name == "DataSource" {
			new.REST.Schemas[i].Properties = nil
		}
	}
	Normalize(new)
	changes := Diff(old, new)
	if c := findChange(changes, ChangeNew, "REST OPERATION", "PATCH /graphs/{id}"); c == nil || c.Breaking {
		t.Errorf("new operation: %v", changes)
	}
	if c := findChange(changes, ChangeChanged, "REST PARAMETER", "DELETE /graphs/{id} path:id"); c == nil || !c.Breaking {
		t.Errorf("retyped parameter must be breaking: %v", changes)
	}
	if c := findChange(changes, ChangeNew, "REST PARAMETER", "GET /graphs header:tenant"); c == nil || !c.Breaking {
		t.Errorf("new required parameter must be breaking: %v", changes)
	}
	if c := findChange(changes, ChangeRemoved, "REST SCHEMA PROPERTY", "DataSource.destination"); c == nil || !c.Breaking {
		t.Errorf("removed schema property must be breaking: %v", changes)
	}
}
