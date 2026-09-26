package apimeta

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Contract is what one client package relies on in one service: the wire
// structs it decodes or sends, entity keys, navigation properties,
// operations and length limits. Each client package declares its Contract
// in code, so the same declaration drives the offline contract test
// (against the committed snapshot), the live acceptance test (against the
// tenant) and the discovery report (which entity sets the provider uses).
type Contract struct {
	// Service is the snapshot ID, for example "cloud-integration".
	Service string
	// Package names the client package, for messages.
	Package string

	Reads       []StructUse
	Writes      []StructUse
	Keys        []KeyUse
	Navigations []NavigationUse
	Operations  []OperationUse
	MaxLengths  []MaxLengthUse
}

// StructUse is a wire struct used with an entity set.
type StructUse struct {
	EntitySet string
	Value     any
}

// KeyUse is the key the client builds requests with, as name/type pairs.
type KeyUse struct {
	EntitySet string
	Pairs     []string // "Id", "Edm.Int64", ...
}

// NavigationUse is a navigation property the client follows.
type NavigationUse struct {
	EntitySet string
	Property  string
}

// OperationUse is a function import the client calls, with its method and
// exact parameter names.
type OperationUse struct {
	Name       string
	HTTPMethod string
	Parameters []string
}

// MaxLengthUse pins a limit the client enforces to the service's MaxLength.
type MaxLengthUse struct {
	EntitySet string
	Property  string
	Value     int
}

// Key is a helper for KeyUse.
func Key(entitySet string, pairs ...string) KeyUse { return KeyUse{EntitySet: entitySet, Pairs: pairs} }

// EntitySets lists every entity set a contract touches, sorted.
func (c Contract) EntitySets() []string {
	seen := map[string]bool{}
	for _, u := range c.Reads {
		seen[u.EntitySet] = true
	}
	for _, u := range c.Writes {
		seen[u.EntitySet] = true
	}
	for _, k := range c.Keys {
		seen[k.EntitySet] = true
	}
	for _, n := range c.Navigations {
		seen[n.EntitySet] = true
	}
	for _, m := range c.MaxLengths {
		seen[m.EntitySet] = true
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// OperationNames lists the operations a contract calls, sorted.
func (c Contract) OperationNames() []string {
	var out []string
	for _, o := range c.Operations {
		out = append(out, o.Name)
	}
	sort.Strings(out)
	return out
}

// Verify checks a contract against a service and returns one message per
// problem; an empty result means the service still offers everything the
// client relies on.
func Verify(s *Service, c Contract) []string {
	v := verifier{s: s}
	for _, u := range c.Reads {
		v.structUse(u, true)
	}
	for _, u := range c.Writes {
		v.structUse(u, false)
	}
	for _, k := range c.Keys {
		v.key(k)
	}
	for _, n := range c.Navigations {
		if t := v.typeOf(n.EntitySet); t != nil {
			if _, ok := v.navigations(t)[n.Property]; !ok {
				v.problem("%s has no navigation property %s", t.Name, n.Property)
			}
		}
	}
	for _, o := range c.Operations {
		v.operation(o)
	}
	for _, m := range c.MaxLengths {
		if t := v.typeOf(m.EntitySet); t != nil {
			p, ok := v.properties(t)[m.Property]
			switch {
			case !ok:
				v.problem("%s has no property %s", t.Name, m.Property)
			case p.MaxLength != strconv.Itoa(m.Value):
				v.problem("%s.%s MaxLength = %q, the client enforces %d", t.Name, m.Property, p.MaxLength, m.Value)
			}
		}
	}
	return v.problems
}

type verifier struct {
	s        *Service
	problems []string
}

func (v *verifier) problem(format string, args ...any) {
	v.problems = append(v.problems, fmt.Sprintf(format, args...))
}

func (v *verifier) typeOf(entitySet string) *EntityType {
	for _, es := range v.s.EntitySets {
		if es.Name == entitySet {
			for i := range v.s.EntityTypes {
				if v.s.EntityTypes[i].Name == es.EntityType {
					return &v.s.EntityTypes[i]
				}
			}
			v.problem("entity type %s of entity set %s is not declared", es.EntityType, entitySet)
			return nil
		}
	}
	v.problem("entity set %s does not exist in %s", entitySet, v.s.ID)
	return nil
}

// chain returns a type and its base types, most derived first.
func (v *verifier) chain(t *EntityType) []*EntityType {
	out := []*EntityType{t}
	seen := map[string]bool{t.Name: true}
	for base := t.BaseType; base != "" && !seen[base]; {
		seen[base] = true
		var next *EntityType
		for i := range v.s.EntityTypes {
			if v.s.EntityTypes[i].Name == base {
				next = &v.s.EntityTypes[i]
			}
		}
		if next == nil {
			break
		}
		out = append(out, next)
		base = next.BaseType
	}
	return out
}

func (v *verifier) properties(t *EntityType) map[string]Property {
	out := map[string]Property{}
	for _, c := range v.chain(t) {
		for _, p := range c.Properties {
			if _, own := out[p.Name]; !own {
				out[p.Name] = p
			}
		}
	}
	return out
}

func (v *verifier) navigations(t *EntityType) map[string]NavigationProperty {
	out := map[string]NavigationProperty{}
	for _, c := range v.chain(t) {
		for _, n := range c.Navigation {
			if _, own := out[n.Name]; !own {
				out[n.Name] = n
			}
		}
	}
	return out
}

func (v *verifier) keyOf(t *EntityType) []string {
	for _, c := range v.chain(t) {
		if len(c.Key) > 0 {
			return c.Key
		}
	}
	return nil
}

func (v *verifier) structUse(u StructUse, decoded bool) {
	t := v.typeOf(u.EntitySet)
	if t == nil {
		return
	}
	props, navs := v.properties(t), v.navigations(t)
	for _, field := range jsonFields(reflect.TypeOf(u.Value)) {
		p, isProp := props[field.name]
		_, isNav := navs[field.name]
		switch {
		case !isProp && !isNav:
			v.problem("%T: JSON field %q is not a property of %s (entity set %s)", u.Value, field.name, t.Name, u.EntitySet)
		case !decoded:
		case isNav:
			if !decodesNavigation(field.typ) {
				v.problem("%T: JSON field %q maps the navigation property %s.%s as %s, which cannot decode SAP's "+
					"{\"__deferred\": ...} or {\"results\": [...]} object; drop it from the read struct or use "+
					"v2.ExpandedCollection with $expand", u.Value, field.name, t.Name, field.name, field.typ)
			}
		case v.s.Protocol == ProtocolODataV2:
			if ok, want := decodesEdmV2(p.Type, field); !ok {
				v.problem("%T: JSON field %q is %s, but %s.%s is %s, which OData V2 JSON sends as %s",
					u.Value, field.name, field.typ, t.Name, field.name, p.Type, want)
			}
		}
	}
}

func (v *verifier) key(k KeyUse) {
	t := v.typeOf(k.EntitySet)
	if t == nil {
		return
	}
	props := v.properties(t)
	var want []string
	for i := 0; i+1 < len(k.Pairs); i += 2 {
		want = append(want, k.Pairs[i])
		if got := props[k.Pairs[i]].Type; got != k.Pairs[i+1] {
			v.problem("%s key %s has type %q, the client uses %q", t.Name, k.Pairs[i], got, k.Pairs[i+1])
		}
	}
	if got := v.keyOf(t); strings.Join(got, ",") != strings.Join(want, ",") {
		v.problem("%s key = %v, the client uses %v", t.Name, got, want)
	}
}

func (v *verifier) operation(o OperationUse) {
	for _, op := range v.s.Operations {
		if op.Name != o.Name || (op.Kind != KindFunctionImport && op.Kind != KindActionImport) {
			continue
		}
		if o.HTTPMethod != "" && op.HTTPMethod != "" && !strings.EqualFold(op.HTTPMethod, o.HTTPMethod) {
			v.problem("operation %s uses %s, the client sends %s", o.Name, op.HTTPMethod, o.HTTPMethod)
		}
		have := map[string]bool{}
		for _, p := range op.Parameters {
			have[p.Name] = true
		}
		for _, p := range o.Parameters {
			if !have[p] {
				v.problem("operation %s has no parameter %s", o.Name, p)
			}
		}
		if len(op.Parameters) != len(o.Parameters) {
			v.problem("operation %s has %d parameters, the client sends %d (%v)", o.Name, len(op.Parameters), len(o.Parameters), o.Parameters)
		}
		return
	}
	v.problem("operation %s does not exist in %s", o.Name, v.s.ID)
}

var (
	unmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
	timeType        = reflect.TypeOf(time.Time{})
	numberType      = reflect.TypeOf(json.Number(""))
)

// decodesEdmV2 reports whether a Go field can decode a property's value as
// OData V2 JSON represents it: Edm.Int64 and Edm.Decimal as strings ("51"),
// Edm.DateTime as "/Date(ms)/", smaller integers, doubles and booleans as
// JSON numbers and booleans. The second result describes the JSON shape.
func decodesEdmV2(edmType string, f jsonField) (bool, string) {
	typ := f.typ
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	kind := typ.Kind()
	isInt := kind >= reflect.Int && kind <= reflect.Uint64
	isFloat := kind == reflect.Float32 || kind == reflect.Float64

	switch edmType {
	case "Edm.DateTime", "Edm.DateTimeOffset", "Edm.Time":
		// time.Time implements json.Unmarshaler but only reads RFC 3339.
		if typ == timeType {
			return false, `a string such as "/Date(1790424405369)/"`
		}
	}
	if typ == reflect.TypeOf(json.RawMessage(nil)) || kind == reflect.Interface ||
		reflect.PointerTo(typ).Implements(unmarshalerType) {
		return true, ""
	}

	switch edmType {
	case "Edm.String", "Edm.Guid", "Edm.DateTime", "Edm.DateTimeOffset", "Edm.Time":
		return kind == reflect.String, "a JSON string"
	case "Edm.Int64", "Edm.Decimal":
		return kind == reflect.String || typ == numberType || ((isInt || isFloat) && f.quoted),
			`a JSON string such as "51" (use a string, json.Number or the ",string" tag option)`
	case "Edm.Int32", "Edm.Int16", "Edm.Byte", "Edm.SByte", "Edm.Double", "Edm.Single":
		return isInt || isFloat || typ == numberType, "a JSON number"
	case "Edm.Boolean":
		return kind == reflect.Bool, "a JSON boolean"
	case "Edm.Binary":
		return kind == reflect.String || (kind == reflect.Slice && typ.Elem().Kind() == reflect.Uint8), "a base64 string"
	default:
		// A complex type arrives as a JSON object.
		return kind == reflect.Struct || kind == reflect.Map, "a JSON object"
	}
}

// decodesNavigation reports whether a Go type can hold a navigation
// property's JSON object: {"__deferred": ...} or {"results": [...]}.
func decodesNavigation(typ reflect.Type) bool {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ == reflect.TypeOf(json.RawMessage(nil)) {
		return true
	}
	if typ.Kind() != reflect.Struct {
		return false
	}
	for _, f := range jsonFields(typ) {
		if f.name == "results" || f.name == "__deferred" {
			return true
		}
	}
	return false
}

type jsonField struct {
	name   string
	typ    reflect.Type
	quoted bool // the ",string" tag option
}

// jsonFields lists a struct's JSON fields the way encoding/json sees them:
// embedded structs flattened, "-" skipped, unexported fields ignored.
func jsonFields(typ reflect.Type) []jsonField {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	var fields []jsonField
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		tag := f.Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name == "-" {
			continue
		}
		if f.Anonymous && name == "" {
			fields = append(fields, jsonFields(f.Type)...)
			continue
		}
		if !f.IsExported() {
			continue
		}
		if name == "" {
			name = f.Name
		}
		fields = append(fields, jsonField{name: name, typ: f.Type, quoted: strings.Contains(tag, ",string")})
	}
	return fields
}
