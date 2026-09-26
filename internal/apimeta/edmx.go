package apimeta

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// The XML structures below match elements by local name only, so the same
// structs read OData V2 (edmx 1.0, CSDL 2006-2009) and V4 (edmx 4.0) documents.

type xEdmx struct {
	Version string     `xml:"Version,attr"`
	Schemas []xSchema  `xml:"DataServices>Schema"`
	Refs    []xRefList `xml:"Reference"`
}

type xRefList struct {
	Includes []struct {
		Namespace string `xml:"Namespace,attr"`
		Alias     string `xml:"Alias,attr"`
	} `xml:"Include"`
}

type xSchema struct {
	Namespace    string         `xml:"Namespace,attr"`
	Alias        string         `xml:"Alias,attr"`
	EntityTypes  []xStructured  `xml:"EntityType"`
	ComplexTypes []xStructured  `xml:"ComplexType"`
	EnumTypes    []xEnum        `xml:"EnumType"`
	Associations []xAssociation `xml:"Association"`
	Containers   []xContainer   `xml:"EntityContainer"`
	Actions      []xOperation   `xml:"Action"`
	Functions    []xOperation   `xml:"Function"`
	Annotations  []xAnnotations `xml:"Annotations"`
}

type xStructured struct {
	Name       string        `xml:"Name,attr"`
	BaseType   string        `xml:"BaseType,attr"`
	Abstract   string        `xml:"Abstract,attr"`
	OpenType   string        `xml:"OpenType,attr"`
	HasStream  string        `xml:"HasStream,attr"`
	Key        []xNamed      `xml:"Key>PropertyRef"`
	Properties []xProperty   `xml:"Property"`
	Navigation []xNav        `xml:"NavigationProperty"`
	Attrs      []xml.Attr    `xml:",any,attr"`
	Inline     []xAnnotation `xml:"Annotation"`
}

type xNamed struct {
	Name string `xml:"Name,attr"`
}

type xProperty struct {
	Name      string        `xml:"Name,attr"`
	Type      string        `xml:"Type,attr"`
	Nullable  string        `xml:"Nullable,attr"`
	MaxLength string        `xml:"MaxLength,attr"`
	Precision string        `xml:"Precision,attr"`
	Scale     string        `xml:"Scale,attr"`
	Attrs     []xml.Attr    `xml:",any,attr"`
	Inline    []xAnnotation `xml:"Annotation"`
}

type xNav struct {
	Name           string `xml:"Name,attr"`
	Relationship   string `xml:"Relationship,attr"` // V2
	FromRole       string `xml:"FromRole,attr"`
	ToRole         string `xml:"ToRole,attr"`
	Type           string `xml:"Type,attr"` // V4
	Partner        string `xml:"Partner,attr"`
	ContainsTarget string `xml:"ContainsTarget,attr"`
	Constraints    []struct {
		Property           string `xml:"Property,attr"`
		ReferencedProperty string `xml:"ReferencedProperty,attr"`
	} `xml:"ReferentialConstraint"`
}

type xEnum struct {
	Name    string   `xml:"Name,attr"`
	Members []xNamed `xml:"Member"`
}

type xAssociation struct {
	Name string `xml:"Name,attr"`
	Ends []struct {
		Role         string `xml:"Role,attr"`
		Type         string `xml:"Type,attr"`
		Multiplicity string `xml:"Multiplicity,attr"`
	} `xml:"End"`
	Constraint *struct {
		Principal struct {
			Role string   `xml:"Role,attr"`
			Refs []xNamed `xml:"PropertyRef"`
		} `xml:"Principal"`
		Dependent struct {
			Role string   `xml:"Role,attr"`
			Refs []xNamed `xml:"PropertyRef"`
		} `xml:"Dependent"`
	} `xml:"ReferentialConstraint"`
}

type xContainer struct {
	Name       string `xml:"Name,attr"`
	EntitySets []struct {
		Name       string `xml:"Name,attr"`
		EntityType string `xml:"EntityType,attr"`
		Bindings   []struct {
			Path   string `xml:"Path,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"NavigationPropertyBinding"`
	} `xml:"EntitySet"`
	Singletons []struct {
		Name string `xml:"Name,attr"`
		Type string `xml:"Type,attr"`
	} `xml:"Singleton"`
	FunctionImports []struct {
		Name       string       `xml:"Name,attr"`
		ReturnType string       `xml:"ReturnType,attr"` // V2
		EntitySet  string       `xml:"EntitySet,attr"`
		Function   string       `xml:"Function,attr"` // V4
		Attrs      []xml.Attr   `xml:",any,attr"`
		Parameters []xParameter `xml:"Parameter"`
	} `xml:"FunctionImport"`
	ActionImports []struct {
		Name      string `xml:"Name,attr"`
		Action    string `xml:"Action,attr"`
		EntitySet string `xml:"EntitySet,attr"`
	} `xml:"ActionImport"`
}

type xOperation struct {
	Name       string       `xml:"Name,attr"`
	IsBound    string       `xml:"IsBound,attr"`
	Parameters []xParameter `xml:"Parameter"`
	ReturnType *struct {
		Type string `xml:"Type,attr"`
	} `xml:"ReturnType"`
}

type xParameter struct {
	Name     string `xml:"Name,attr"`
	Type     string `xml:"Type,attr"`
	Nullable string `xml:"Nullable,attr"`
	Mode     string `xml:"Mode,attr"`
}

type xAnnotations struct {
	Target      string        `xml:"Target,attr"`
	Annotations []xAnnotation `xml:"Annotation"`
}

type xAnnotation struct {
	Term       string `xml:"Term,attr"`
	String     string `xml:"String,attr"`
	Bool       string `xml:"Bool,attr"`
	EnumMember string `xml:"EnumMember,attr"`
	Record     *struct {
		Values []struct {
			Property   string `xml:"Property,attr"`
			String     string `xml:"String,attr"`
			Bool       string `xml:"Bool,attr"`
			EnumMember string `xml:"EnumMember,attr"`
		} `xml:"PropertyValue"`
	} `xml:"Record"`
}

// sapDataNamespace is the namespace of SAP's V2 annotation attributes.
const sapDataNamespace = "http://www.sap.com/Protocols/SAPData"

// sapContractAttributes are the sap: attributes that describe the contract
// (what a client may do), as opposed to UI texts such as sap:label, which
// would only add noise to a diff.
var sapContractAttributes = map[string]bool{
	"creatable": true, "updatable": true, "deletable": true, "updatable-path": true,
	"deletable-path": true, "pageable": true, "addressable": true, "searchable": true,
	"filterable": true, "sortable": true, "requires-filter": true, "content-version": true,
	"semantics": true, "action-for": true, "applicable-path": true,
}

// ParseEDMX reads an OData V2 or V4 $metadata document into a normalized
// Service (sorted, aliases resolved, graph summary computed). id names the
// service in the result.
func ParseEDMX(id string, data []byte) (*Service, error) {
	var doc xEdmx
	if err := xml.Unmarshal(data, &doc); err != nil { //nolint:gosec // G709: decodes a service contract into fixed structs
		return nil, fmt.Errorf("apimeta: %s: not an EDMX document: %w", id, err)
	}
	if len(doc.Schemas) == 0 {
		return nil, fmt.Errorf("apimeta: %s: EDMX document has no schema", id)
	}

	s := &Service{ID: id, Protocol: ProtocolODataV2}
	if strings.HasPrefix(doc.Version, "4") {
		s.Protocol = ProtocolODataV4
	}

	aliases := map[string]string{}
	for _, ref := range doc.Refs {
		for _, inc := range ref.Includes {
			if inc.Alias != "" {
				aliases[inc.Alias] = inc.Namespace
			}
		}
	}
	for _, sc := range doc.Schemas {
		if sc.Alias != "" {
			aliases[sc.Alias] = sc.Namespace
		}
	}
	q := func(name string) string { return qualify(name, aliases) }

	targeted := map[string]map[string]string{}
	for _, sc := range doc.Schemas {
		for _, group := range sc.Annotations {
			target := q(group.Target)
			for _, a := range group.Annotations {
				addAnnotation(targeted, target, a, q)
			}
		}
	}

	associations := map[string]xAssociation{}
	for _, sc := range doc.Schemas {
		for _, a := range sc.Associations {
			associations[sc.Namespace+"."+a.Name] = a
		}
	}

	for _, sc := range doc.Schemas {
		s.Namespaces = append(s.Namespaces, sc.Namespace)
		for _, et := range sc.EntityTypes {
			s.EntityTypes = append(s.EntityTypes, structured(sc.Namespace, et, q, associations, targeted))
		}
		for _, ct := range sc.ComplexTypes {
			s.ComplexTypes = append(s.ComplexTypes, structured(sc.Namespace, ct, q, associations, targeted))
		}
		for _, en := range sc.EnumTypes {
			e := EnumType{Name: sc.Namespace + "." + en.Name}
			for _, m := range en.Members {
				e.Members = append(e.Members, m.Name)
			}
			s.EnumTypes = append(s.EnumTypes, e)
		}
		for _, a := range sc.Associations {
			s.Associations = append(s.Associations, association(sc.Namespace, a, q))
		}
		for _, op := range sc.Actions {
			s.Operations = append(s.Operations, operation(sc.Namespace, op, KindAction, q))
		}
		for _, op := range sc.Functions {
			s.Operations = append(s.Operations, operation(sc.Namespace, op, KindFunction, q))
		}
		for _, c := range sc.Containers {
			for _, es := range c.EntitySets {
				set := EntitySet{Name: es.Name, EntityType: q(es.EntityType)}
				for _, b := range es.Bindings {
					if set.NavigationBindings == nil {
						set.NavigationBindings = map[string]string{}
					}
					set.NavigationBindings[b.Path] = b.Target
				}
				s.EntitySets = append(s.EntitySets, set)
			}
			for _, sg := range c.Singletons {
				s.Singletons = append(s.Singletons, EntitySet{Name: sg.Name, EntityType: q(sg.Type)})
			}
			for _, fi := range c.FunctionImports {
				op := Operation{Name: fi.Name, Kind: KindFunctionImport, EntitySet: fi.EntitySet,
					ReturnType: q(fi.ReturnType), Target: q(fi.Function)}
				for _, a := range fi.Attrs {
					if a.Name.Local == "HttpMethod" {
						op.HTTPMethod = strings.ToUpper(a.Value)
					}
				}
				for _, p := range fi.Parameters {
					op.Parameters = append(op.Parameters, parameter(p, q))
				}
				s.Operations = append(s.Operations, op)
			}
			for _, ai := range c.ActionImports {
				s.Operations = append(s.Operations, Operation{Name: ai.Name, Kind: KindActionImport,
					Target: q(ai.Action), EntitySet: ai.EntitySet})
			}
		}
	}

	Normalize(s)
	s.Graph = Traverse(s)
	return s, nil
}

// qualify turns an alias-qualified or collection type reference into its
// namespace-qualified form: "Alias.T" -> "Namespace.T",
// "Collection(Alias.T)" -> "Collection(Namespace.T)".
func qualify(name string, aliases map[string]string) string {
	if name == "" {
		return ""
	}
	if inner, ok := strings.CutPrefix(name, "Collection("); ok {
		return "Collection(" + qualify(strings.TrimSuffix(inner, ")"), aliases) + ")"
	}
	if i := strings.LastIndex(name, "."); i > 0 {
		if ns, ok := aliases[name[:i]]; ok {
			return ns + name[i:]
		}
	}
	return name
}

func boolAttr(v string) bool { return strings.EqualFold(v, "true") }

// nullable reads a Nullable facet: absent means nullable in both versions.
func nullable(v string) bool { return !strings.EqualFold(v, "false") }

func structured(ns string, x xStructured, q func(string) string, assocs map[string]xAssociation, targeted map[string]map[string]string) EntityType {
	name := ns + "." + x.Name
	t := EntityType{
		Name: name, BaseType: q(x.BaseType), Abstract: boolAttr(x.Abstract),
		OpenType: boolAttr(x.OpenType), HasStream: boolAttr(x.HasStream),
		Annotations: mergeAnnotations(sapAttributes(x.Attrs), inline(x.Inline, q), targeted[name]),
	}
	for _, k := range x.Key {
		t.Key = append(t.Key, k.Name)
	}
	for _, p := range x.Properties {
		t.Properties = append(t.Properties, Property{
			Name: p.Name, Type: q(p.Type), Nullable: nullable(p.Nullable),
			MaxLength: p.MaxLength, Precision: p.Precision, Scale: p.Scale,
			Annotations: mergeAnnotations(sapAttributes(p.Attrs), inline(p.Inline, q), targeted[name+"/"+p.Name]),
		})
	}
	for _, n := range x.Navigation {
		nav := NavigationProperty{Name: n.Name, Partner: n.Partner, ContainsTarget: boolAttr(n.ContainsTarget)}
		if n.Relationship != "" {
			nav.Relationship = q(n.Relationship)
			nav.FromRole, nav.ToRole = n.FromRole, n.ToRole
			if a, ok := assocs[nav.Relationship]; ok {
				for _, end := range a.Ends {
					if end.Role == n.ToRole {
						nav.Target = q(end.Type)
						nav.Collection = end.Multiplicity == "*"
					}
				}
			}
		} else {
			typ := q(n.Type)
			if inner, ok := strings.CutPrefix(typ, "Collection("); ok {
				typ = strings.TrimSuffix(inner, ")")
				nav.Collection = true
			}
			nav.Target = typ
		}
		for _, c := range n.Constraints {
			nav.Constraints = append(nav.Constraints, ReferentialConstraint{Property: c.Property, ReferencedProperty: c.ReferencedProperty})
		}
		t.Navigation = append(t.Navigation, nav)
	}
	return t
}

func association(ns string, a xAssociation, q func(string) string) Association {
	out := Association{Name: ns + "." + a.Name}
	for _, e := range a.Ends {
		out.Ends = append(out.Ends, AssociationEnd{Role: e.Role, Type: q(e.Type), Multiplicity: e.Multiplicity})
	}
	if c := a.Constraint; c != nil {
		out.Principal, out.Dependent = c.Principal.Role, c.Dependent.Role
		for i := range c.Dependent.Refs {
			rc := ReferentialConstraint{Property: c.Dependent.Refs[i].Name}
			if i < len(c.Principal.Refs) {
				rc.ReferencedProperty = c.Principal.Refs[i].Name
			}
			out.Constraints = append(out.Constraints, rc)
		}
	}
	return out
}

func operation(ns string, x xOperation, kind string, q func(string) string) Operation {
	op := Operation{Name: ns + "." + x.Name, Kind: kind, Bound: boolAttr(x.IsBound)}
	for _, p := range x.Parameters {
		op.Parameters = append(op.Parameters, parameter(p, q))
	}
	if x.ReturnType != nil {
		op.ReturnType = q(x.ReturnType.Type)
	}
	return op
}

func parameter(p xParameter, q func(string) string) Parameter {
	return Parameter{Name: p.Name, Type: q(p.Type), Nullable: nullable(p.Nullable), Mode: p.Mode}
}

func sapAttributes(attrs []xml.Attr) map[string]string {
	var out map[string]string
	for _, a := range attrs {
		if a.Name.Space == sapDataNamespace && sapContractAttributes[a.Name.Local] {
			if out == nil {
				out = map[string]string{}
			}
			out["sap:"+a.Name.Local] = a.Value
		}
	}
	return out
}

// inline flattens V4 annotations to term -> value; a record's property
// values become term/property -> value.
func inline(list []xAnnotation, q func(string) string) map[string]string {
	out := map[string]map[string]string{"": {}}
	for _, a := range list {
		addAnnotation(out, "", a, q)
	}
	if len(out[""]) == 0 {
		return nil
	}
	return out[""]
}

func addAnnotation(into map[string]map[string]string, target string, a xAnnotation, q func(string) string) {
	if into[target] == nil {
		into[target] = map[string]string{}
	}
	term := q(a.Term)
	if v := firstNonEmpty(a.String, a.Bool, a.EnumMember); v != "" {
		into[target][term] = v
	}
	if a.Record != nil {
		for _, pv := range a.Record.Values {
			if v := firstNonEmpty(pv.String, pv.Bool, pv.EnumMember); v != "" {
				into[target][term+"/"+pv.Property] = v
			}
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func mergeAnnotations(maps ...map[string]string) map[string]string {
	var out map[string]string
	for _, m := range maps {
		for k, v := range m {
			if out == nil {
				out = map[string]string{}
			}
			out[k] = RedactURLs(v)
		}
	}
	return out
}
