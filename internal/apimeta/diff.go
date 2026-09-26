package apimeta

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Change kinds.
const (
	ChangeNew     = "NEW"
	ChangeRemoved = "REMOVED"
	ChangeChanged = "CHANGED"
)

// Change is one semantic difference between two contracts.
type Change struct {
	Kind    string // NEW, REMOVED, CHANGED
	Subject string // ENTITY SET, PROPERTY, FUNCTION IMPORT, ...
	Path    string // for example "com.sap.hci.api.AccessPolicy.Description"
	Before  string
	After   string
	// Breaking is true when a client written against the old contract can
	// fail against the new one: something removed, a type or key changed,
	// a property that no longer accepts null, a new required parameter.
	Breaking bool
}

// String renders the change for a report, for example
//
//	PROPERTY CHANGED: com.sap.hci.api.AccessPolicy.Description
//	  Edm.String nullable=true
//	  ->
//	  Edm.String nullable=false
func (c Change) String() string {
	var b strings.Builder
	switch c.Kind {
	case ChangeChanged:
		fmt.Fprintf(&b, "%s CHANGED: %s", c.Subject, c.Path)
		fmt.Fprintf(&b, "\n  %s\n  ->\n  %s", c.Before, c.After)
	default:
		fmt.Fprintf(&b, "%s %s: %s", c.Kind, c.Subject, c.Path)
		if detail := firstNonEmpty(c.After, c.Before); detail != "" {
			fmt.Fprintf(&b, "\n  %s", detail)
		}
	}
	if c.Breaking {
		b.WriteString("\n  (breaking)")
	}
	return b.String()
}

// Diff compares two contracts of the same service and returns the
// semantic changes from old to new, sorted by path.
func Diff(old, new *Service) []Change {
	d := &differ{}
	d.sets("ENTITY SET", old.EntitySets, new.EntitySets)
	d.sets("SINGLETON", old.Singletons, new.Singletons)
	d.types("ENTITY TYPE", old.EntityTypes, new.EntityTypes)
	d.types("COMPLEX TYPE", old.ComplexTypes, new.ComplexTypes)
	d.enums(old.EnumTypes, new.EnumTypes)
	d.operations(old.Operations, new.Operations)
	d.strings("SERVICE DOCUMENT COLLECTION", old.ServiceDocument, new.ServiceDocument)
	if old.REST != nil || new.REST != nil {
		d.rest(orEmpty(old.REST), orEmpty(new.REST))
	}
	sort.SliceStable(d.changes, func(i, j int) bool {
		if d.changes[i].Path != d.changes[j].Path {
			return d.changes[i].Path < d.changes[j].Path
		}
		return d.changes[i].Subject < d.changes[j].Subject
	})
	return d.changes
}

// Breaking returns the breaking changes of a list.
func Breaking(changes []Change) []Change {
	var out []Change
	for _, c := range changes {
		if c.Breaking {
			out = append(out, c)
		}
	}
	return out
}

type differ struct{ changes []Change }

func (d *differ) add(kind, subject, path, before, after string, breaking bool) {
	d.changes = append(d.changes, Change{Kind: kind, Subject: subject, Path: path, Before: before, After: after, Breaking: breaking})
}

func (d *differ) sets(subject string, old, new []EntitySet) {
	o, n := index(old, func(s EntitySet) string { return s.Name }), index(new, func(s EntitySet) string { return s.Name })
	for name, ns := range n {
		os, ok := o[name]
		switch {
		case !ok:
			d.add(ChangeNew, subject, name, "", ns.EntityType, false)
		case os.EntityType != ns.EntityType:
			d.add(ChangeChanged, subject, name, os.EntityType, ns.EntityType, true)
		}
	}
	for name, os := range o {
		if _, ok := n[name]; !ok {
			d.add(ChangeRemoved, subject, name, os.EntityType, "", true)
		}
	}
}

func (d *differ) types(subject string, old, new []EntityType) {
	o, n := index(old, func(t EntityType) string { return t.Name }), index(new, func(t EntityType) string { return t.Name })
	for name, nt := range n {
		ot, ok := o[name]
		if !ok {
			d.add(ChangeNew, subject, name, "", fmt.Sprintf("%d properties, %d navigation properties", len(nt.Properties), len(nt.Navigation)), false)
			continue
		}
		d.typeChanges(subject, ot, nt)
	}
	for name := range o {
		if _, ok := n[name]; !ok {
			d.add(ChangeRemoved, subject, name, "", "", true)
		}
	}
}

func (d *differ) typeChanges(subject string, ot, nt EntityType) {
	if strings.Join(ot.Key, ",") != strings.Join(nt.Key, ",") {
		d.add(ChangeChanged, "KEY", nt.Name, strings.Join(ot.Key, ", "), strings.Join(nt.Key, ", "), true)
	}
	if ot.BaseType != nt.BaseType {
		d.add(ChangeChanged, subject+" BASE TYPE", nt.Name, ot.BaseType, nt.BaseType, true)
	}
	if ot.HasStream != nt.HasStream {
		d.add(ChangeChanged, "MEDIA ENTITY", nt.Name, "hasStream="+strconv.FormatBool(ot.HasStream), "hasStream="+strconv.FormatBool(nt.HasStream), true)
	}
	d.annotations(nt.Name, ot.Annotations, nt.Annotations)

	op, np := index(ot.Properties, func(p Property) string { return p.Name }), index(nt.Properties, func(p Property) string { return p.Name })
	for name, p := range np {
		path := nt.Name + "." + name
		q, ok := op[name]
		if !ok {
			d.add(ChangeNew, "PROPERTY", path, "", describeProperty(p), false)
			continue
		}
		if describeProperty(q) != describeProperty(p) {
			d.add(ChangeChanged, "PROPERTY", path, describeProperty(q), describeProperty(p), propertyBreaking(q, p))
		}
		d.annotations(path, q.Annotations, p.Annotations)
	}
	for name, q := range op {
		if _, ok := np[name]; !ok {
			d.add(ChangeRemoved, "PROPERTY", nt.Name+"."+name, describeProperty(q), "", true)
		}
	}

	on, nn := index(ot.Navigation, func(n NavigationProperty) string { return n.Name }), index(nt.Navigation, func(n NavigationProperty) string { return n.Name })
	for name, n := range nn {
		path := nt.Name + "." + name
		o, ok := on[name]
		switch {
		case !ok:
			d.add(ChangeNew, "NAVIGATION PROPERTY", path, "", describeNav(n), false)
		case describeNav(o) != describeNav(n):
			d.add(ChangeChanged, "NAVIGATION PROPERTY", path, describeNav(o), describeNav(n), true)
		}
	}
	for name, o := range on {
		if _, ok := nn[name]; !ok {
			d.add(ChangeRemoved, "NAVIGATION PROPERTY", nt.Name+"."+name, describeNav(o), "", true)
		}
	}
}

// writeAnnotations are the annotations that say whether a client may
// write; losing "true" breaks writes.
var writeAnnotations = map[string]bool{"sap:creatable": true, "sap:updatable": true, "sap:deletable": true}

func (d *differ) annotations(path string, old, new map[string]string) {
	keys := map[string]bool{}
	for k := range old {
		keys[k] = true
	}
	for k := range new {
		keys[k] = true
	}
	for k := range keys {
		ov, oOK := old[k]
		nv, nOK := new[k]
		if oOK && nOK && ov == nv {
			continue
		}
		// An absent sap:creatable etc. means true.
		was, is := ov, nv
		if !oOK && writeAnnotations[k] {
			was = "true"
		}
		if !nOK && writeAnnotations[k] {
			is = "true"
		}
		if was == is {
			continue
		}
		breaking := writeAnnotations[k] && was == "true" && is == "false"
		d.add(ChangeChanged, "ANNOTATION", path+" @"+k, orNone(ov), orNone(nv), breaking)
	}
}

func orNone(v string) string {
	if v == "" {
		return "(none)"
	}
	return v
}

func describeProperty(p Property) string {
	s := fmt.Sprintf("%s nullable=%t", p.Type, p.Nullable)
	if p.MaxLength != "" {
		s += " maxLength=" + p.MaxLength
	}
	if p.Precision != "" {
		s += " precision=" + p.Precision
	}
	if p.Scale != "" {
		s += " scale=" + p.Scale
	}
	return s
}

// propertyBreaking: a different type, a property that no longer accepts
// null, a shorter maximum length or a different precision or scale can
// break a client; accepting null or longer values cannot.
func propertyBreaking(old, new Property) bool {
	if old.Type != new.Type || old.Precision != new.Precision || old.Scale != new.Scale {
		return true
	}
	if old.Nullable && !new.Nullable {
		return true
	}
	if old.MaxLength != new.MaxLength {
		ol, oerr := strconv.Atoi(old.MaxLength)
		nl, nerr := strconv.Atoi(new.MaxLength)
		switch {
		case new.MaxLength == "" || strings.EqualFold(new.MaxLength, "max"):
			return false // no limit any more
		case old.MaxLength == "" || strings.EqualFold(old.MaxLength, "max"):
			return true // a limit where there was none
		case oerr == nil && nerr == nil:
			return nl < ol
		}
		return true
	}
	return false
}

func describeNav(n NavigationProperty) string {
	target := n.Target
	if n.Collection {
		target = "Collection(" + target + ")"
	}
	return target
}

func (d *differ) enums(old, new []EnumType) {
	o, n := index(old, func(e EnumType) string { return e.Name }), index(new, func(e EnumType) string { return e.Name })
	for name, ne := range n {
		oe, ok := o[name]
		if !ok {
			d.add(ChangeNew, "ENUM TYPE", name, "", strings.Join(ne.Members, ", "), false)
			continue
		}
		om, nm := toSet(oe.Members), toSet(ne.Members)
		for m := range nm {
			if !om[m] {
				d.add(ChangeNew, "ENUM MEMBER", name+"."+m, "", "", false)
			}
		}
		for m := range om {
			if !nm[m] {
				d.add(ChangeRemoved, "ENUM MEMBER", name+"."+m, "", "", true)
			}
		}
	}
	for name := range o {
		if _, ok := n[name]; !ok {
			d.add(ChangeRemoved, "ENUM TYPE", name, "", "", true)
		}
	}
}

func operationSubject(kind string) string {
	switch kind {
	case KindFunctionImport:
		return "FUNCTION IMPORT"
	case KindActionImport:
		return "ACTION IMPORT"
	case KindFunction:
		return "FUNCTION"
	default:
		return "ACTION"
	}
}

func describeOperation(op Operation) string {
	var params []string
	for _, p := range op.Parameters {
		params = append(params, p.Name+" "+p.Type)
	}
	s := op.Name + "(" + strings.Join(params, ", ") + ")"
	if op.HTTPMethod != "" {
		s = op.HTTPMethod + " " + s
	}
	if op.ReturnType != "" {
		s += " -> " + op.ReturnType
	}
	if op.Target != "" {
		s += " exposes " + op.Target
	}
	return s
}

func (d *differ) operations(old, new []Operation) {
	o, n := index(old, OperationKey), index(new, OperationKey)
	for key, nop := range n {
		subject := operationSubject(nop.Kind)
		oop, ok := o[key]
		if !ok {
			d.add(ChangeNew, subject, nop.Name, "", describeOperation(nop), false)
			continue
		}
		if describeOperation(oop) == describeOperation(nop) && paramsNullability(oop) == paramsNullability(nop) {
			continue
		}
		d.add(ChangeChanged, subject, nop.Name, describeOperation(oop), describeOperation(nop), operationBreaking(oop, nop))
	}
	for key, oop := range o {
		if _, ok := n[key]; !ok {
			d.add(ChangeRemoved, operationSubject(oop.Kind), oop.Name, describeOperation(oop), "", true)
		}
	}
}

func paramsNullability(op Operation) string {
	var parts []string
	for _, p := range op.Parameters {
		parts = append(parts, p.Name+"="+strconv.FormatBool(p.Nullable))
	}
	return strings.Join(parts, ",")
}

// operationBreaking: a different method, return type or target, a removed
// or retyped parameter, or a new parameter that must be given.
func operationBreaking(old, new Operation) bool {
	if old.HTTPMethod != new.HTTPMethod || old.ReturnType != new.ReturnType || old.Target != new.Target || old.EntitySet != new.EntitySet {
		return true
	}
	op := index(old.Parameters, func(p Parameter) string { return p.Name })
	np := index(new.Parameters, func(p Parameter) string { return p.Name })
	for name, p := range op {
		q, ok := np[name]
		if !ok || q.Type != p.Type || (p.Nullable && !q.Nullable) {
			return true
		}
	}
	for name, q := range np {
		if _, ok := op[name]; !ok && !q.Nullable {
			return true
		}
	}
	return false
}

func (d *differ) strings(subject string, old, new []string) {
	o, n := toSet(old), toSet(new)
	for v := range n {
		if !o[v] {
			d.add(ChangeNew, subject, v, "", "", false)
		}
	}
	for v := range o {
		if !n[v] {
			d.add(ChangeRemoved, subject, v, "", "", false)
		}
	}
}

func orEmpty(r *RESTContract) *RESTContract {
	if r == nil {
		return &RESTContract{}
	}
	return r
}

func restKey(op RESTOperation) string { return op.Method + " " + op.Path }

func (d *differ) rest(old, new *RESTContract) {
	o, n := index(old.Operations, restKey), index(new.Operations, restKey)
	for key, nop := range n {
		oop, ok := o[key]
		if !ok {
			d.add(ChangeNew, "REST OPERATION", key, "", nop.OperationID, false)
			continue
		}
		d.restOperation(key, oop, nop)
	}
	for key, oop := range o {
		if _, ok := n[key]; !ok {
			d.add(ChangeRemoved, "REST OPERATION", key, oop.OperationID, "", true)
		}
	}
	os, ns := index(old.Schemas, func(s RESTSchema) string { return s.Name }), index(new.Schemas, func(s RESTSchema) string { return s.Name })
	for name, nsch := range ns {
		osch, ok := os[name]
		if !ok {
			d.add(ChangeNew, "REST SCHEMA", name, "", fmt.Sprintf("%d properties", len(nsch.Properties)), false)
			continue
		}
		op, np := index(osch.Properties, func(p RESTProperty) string { return p.Name }), index(nsch.Properties, func(p RESTProperty) string { return p.Name })
		for pname, p := range np {
			q, ok := op[pname]
			switch {
			case !ok:
				d.add(ChangeNew, "REST SCHEMA PROPERTY", name+"."+pname, "", restProp(p), p.Required)
			case restProp(q) != restProp(p):
				d.add(ChangeChanged, "REST SCHEMA PROPERTY", name+"."+pname, restProp(q), restProp(p), q.Type != p.Type || (!q.Required && p.Required))
			}
		}
		for pname, q := range op {
			if _, ok := np[pname]; !ok {
				d.add(ChangeRemoved, "REST SCHEMA PROPERTY", name+"."+pname, restProp(q), "", true)
			}
		}
	}
	for name := range os {
		if _, ok := ns[name]; !ok {
			d.add(ChangeRemoved, "REST SCHEMA", name, "", "", true)
		}
	}
	d.strings("REST SECURITY SCHEME", old.SecuritySchemes, new.SecuritySchemes)
}

func restProp(p RESTProperty) string {
	return fmt.Sprintf("%s required=%t", p.Type, p.Required)
}

func (d *differ) restOperation(key string, old, new RESTOperation) {
	if old.RequestBody != new.RequestBody {
		d.add(ChangeChanged, "REST REQUEST BODY", key, orNone(old.RequestBody), orNone(new.RequestBody), true)
	}
	for status, schema := range new.Responses {
		if prev, ok := old.Responses[status]; ok && prev != schema {
			d.add(ChangeChanged, "REST RESPONSE", key+" "+status, orNone(prev), orNone(schema), true)
		} else if !ok {
			d.add(ChangeNew, "REST RESPONSE", key+" "+status, "", schema, false)
		}
	}
	for status := range old.Responses {
		if _, ok := new.Responses[status]; !ok {
			d.add(ChangeRemoved, "REST RESPONSE", key+" "+status, "", "", false)
		}
	}
	pkey := func(p RESTParameter) string { return p.In + ":" + p.Name }
	op, np := index(old.Parameters, pkey), index(new.Parameters, pkey)
	for k, p := range np {
		q, ok := op[k]
		switch {
		case !ok:
			d.add(ChangeNew, "REST PARAMETER", key+" "+k, "", fmt.Sprintf("%s required=%t", p.Type, p.Required), p.Required)
		case q.Type != p.Type || q.Required != p.Required:
			d.add(ChangeChanged, "REST PARAMETER", key+" "+k, fmt.Sprintf("%s required=%t", q.Type, q.Required),
				fmt.Sprintf("%s required=%t", p.Type, p.Required), q.Type != p.Type || (!q.Required && p.Required))
		}
	}
	for k := range op {
		if _, ok := np[k]; !ok {
			d.add(ChangeRemoved, "REST PARAMETER", key+" "+k, "", "", true)
		}
	}
}

func index[T any](list []T, key func(T) string) map[string]T {
	m := make(map[string]T, len(list))
	for _, v := range list {
		m[key(v)] = v
	}
	return m
}

func toSet(list []string) map[string]bool {
	m := make(map[string]bool, len(list))
	for _, v := range list {
		m[v] = true
	}
	return m
}
