package apimeta

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var httpMethods = []string{"get", "put", "post", "delete", "patch", "head", "options"}

// ParseOpenAPI reads an OpenAPI 3.x or Swagger 2.0 specification (JSON or
// YAML) into a normalized Service with Protocol openapi.
func ParseOpenAPI(id string, data []byte) (*Service, error) {
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("apimeta: %s: not a JSON or YAML document: %w", id, err)
	}
	doc, _ := stringKeys(raw).(map[string]any)
	_, v3 := doc["openapi"]
	_, v2 := doc["swagger"]
	if !v3 && !v2 {
		return nil, fmt.Errorf("apimeta: %s: neither an OpenAPI 3 nor a Swagger 2 document", id)
	}

	rest := &RESTContract{}
	if info, ok := doc["info"].(map[string]any); ok {
		rest.Title = RedactURLs(str(info["title"]))
		rest.Version = str(info["version"])
	}

	var schemas map[string]any
	var schemes map[string]any
	if v3 {
		components, _ := doc["components"].(map[string]any)
		schemas, _ = components["schemas"].(map[string]any)
		schemes, _ = components["securitySchemes"].(map[string]any)
	} else {
		schemas, _ = doc["definitions"].(map[string]any)
		schemes, _ = doc["securityDefinitions"].(map[string]any)
	}
	for name := range schemes {
		rest.SecuritySchemes = append(rest.SecuritySchemes, name)
	}
	for name, raw := range schemas {
		node, _ := raw.(map[string]any)
		rest.Schemas = append(rest.Schemas, restSchema(name, node))
	}

	globalSecurity := securityNames(doc["security"])
	paths, _ := doc["paths"].(map[string]any)
	for path, rawItem := range paths {
		item, _ := rawItem.(map[string]any)
		shared := parameters(item["parameters"])
		for _, method := range httpMethods {
			rawOp, ok := item[method]
			if !ok {
				continue
			}
			node, _ := rawOp.(map[string]any)
			op := RESTOperation{Method: strings.ToUpper(method), Path: path, OperationID: str(node["operationId"])}
			op.Parameters = mergeParameters(shared, parameters(node["parameters"]))
			if v3 {
				op.RequestBody = contentSchema(node["requestBody"])
			} else {
				for _, p := range parameterList(node["parameters"]) {
					if str(p["in"]) == "body" {
						op.RequestBody = schemaType(p["schema"])
					}
				}
			}
			if responses, ok := node["responses"].(map[string]any); ok {
				op.Responses = map[string]string{}
				for status, raw := range responses {
					r, _ := raw.(map[string]any)
					if v3 {
						op.Responses[status] = contentSchema(r)
					} else {
						op.Responses[status] = schemaType(r["schema"])
					}
				}
			}
			op.Security = globalSecurity
			if s, ok := node["security"]; ok {
				op.Security = securityNames(s)
			}
			rest.Operations = append(rest.Operations, op)
		}
	}

	s := &Service{ID: id, Protocol: ProtocolOpenAPI, REST: rest}
	Normalize(s)
	return s, nil
}

func normalizeREST(r *RESTContract) {
	sort.Strings(r.SecuritySchemes)
	sort.Slice(r.Operations, func(i, j int) bool { return restKey(r.Operations[i]) < restKey(r.Operations[j]) })
	for i := range r.Operations {
		params := r.Operations[i].Parameters
		sort.Slice(params, func(a, b int) bool { return params[a].In+params[a].Name < params[b].In+params[b].Name })
		sort.Strings(r.Operations[i].Security)
	}
	sort.Slice(r.Schemas, func(i, j int) bool { return r.Schemas[i].Name < r.Schemas[j].Name })
	for i := range r.Schemas {
		props := r.Schemas[i].Properties
		sort.Slice(props, func(a, b int) bool { return props[a].Name < props[b].Name })
	}
}

func restSchema(name string, node map[string]any) RESTSchema {
	s := RESTSchema{Name: name, Type: schemaType(node)}
	required := map[string]bool{}
	if list, ok := node["required"].([]any); ok {
		for _, r := range list {
			required[str(r)] = true
		}
	}
	if props, ok := node["properties"].(map[string]any); ok {
		for pname, raw := range props {
			s.Properties = append(s.Properties, RESTProperty{Name: pname, Type: schemaType(raw), Required: required[pname]})
		}
	}
	return s
}

// schemaType describes a schema node compactly: "#Name" for a reference,
// "array<...>" for arrays, the type and format otherwise.
func schemaType(raw any) string {
	node, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	if ref := str(node["$ref"]); ref != "" {
		return "#" + ref[strings.LastIndex(ref, "/")+1:]
	}
	for _, combinator := range []string{"allOf", "oneOf", "anyOf"} {
		if list, ok := node[combinator].([]any); ok {
			var parts []string
			for _, item := range list {
				parts = append(parts, schemaType(item))
			}
			sort.Strings(parts)
			return combinator + "(" + strings.Join(parts, ",") + ")"
		}
	}
	t := str(node["type"])
	if t == "array" {
		return "array<" + schemaType(node["items"]) + ">"
	}
	if f := str(node["format"]); f != "" {
		return t + "/" + f
	}
	return t
}

func contentSchema(raw any) string {
	node, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	content, ok := node["content"].(map[string]any)
	if !ok {
		return ""
	}
	var types []string
	for mediaType, rawMedia := range content {
		media, _ := rawMedia.(map[string]any)
		types = append(types, mediaType+" "+schemaType(media["schema"]))
	}
	sort.Strings(types)
	return strings.Join(types, "; ")
}

func parameterList(raw any) []map[string]any {
	list, _ := raw.([]any)
	var out []map[string]any
	for _, item := range list {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func parameters(raw any) []RESTParameter {
	var out []RESTParameter
	for _, p := range parameterList(raw) {
		in := str(p["in"])
		if in == "body" || in == "" {
			continue // Swagger 2 body parameters are the request body
		}
		typ := schemaType(p["schema"])
		if typ == "" {
			typ = schemaType(p) // Swagger 2 keeps type and format on the parameter
		}
		required, _ := p["required"].(bool)
		out = append(out, RESTParameter{Name: str(p["name"]), In: in, Type: typ, Required: required})
	}
	return out
}

// mergeParameters applies operation parameters over path-level ones with
// the same name and location.
func mergeParameters(shared, own []RESTParameter) []RESTParameter {
	byKey := map[string]RESTParameter{}
	for _, p := range shared {
		byKey[p.In+":"+p.Name] = p
	}
	for _, p := range own {
		byKey[p.In+":"+p.Name] = p
	}
	out := make([]RESTParameter, 0, len(byKey))
	for _, p := range byKey {
		out = append(out, p)
	}
	return out
}

func securityNames(raw any) []string {
	list, _ := raw.([]any)
	var out []string
	for _, item := range list {
		if m, ok := item.(map[string]any); ok {
			for name := range m {
				out = append(out, name)
			}
		}
	}
	sort.Strings(out)
	return out
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// stringKeys turns every map key into a string. YAML reads unquoted
// response codes (200:) as numbers, which yaml.v3 puts into maps with
// non-string keys; without this those responses would be lost.
func stringKeys(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, item := range t {
			t[k] = stringKeys(item)
		}
		return t
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, item := range t {
			out[fmt.Sprint(k)] = stringKeys(item)
		}
		return out
	case []any:
		for i, item := range t {
			t[i] = stringKeys(item)
		}
		return t
	default:
		return v
	}
}
