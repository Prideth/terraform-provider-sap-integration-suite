package apimeta

import (
	"sort"
	"strings"
)

// Traverse walks the metadata graph from the service's entry points (entity
// sets, singletons and operations) through property types, base types,
// navigation targets and operation signatures, until no unseen type is left.
// A visited set keeps cyclic navigation (A -> B -> A) from recursing forever.
//
// The result says which declared types an entry point reaches, which are
// declared but unreachable, and which references no schema declares. It
// covers exactly one service: $metadata never describes other service roots.
func Traverse(s *Service) *GraphSummary {
	declared := map[string]*EntityType{}
	for i := range s.EntityTypes {
		declared[s.EntityTypes[i].Name] = &s.EntityTypes[i]
	}
	for i := range s.ComplexTypes {
		declared[s.ComplexTypes[i].Name] = &s.ComplexTypes[i]
	}
	enums := map[string]bool{}
	for _, e := range s.EnumTypes {
		enums[e.Name] = true
	}
	// Bound V4 operations become reachable once their binding type is.
	boundTo := map[string][]Operation{}
	unbound := map[string]Operation{}
	for _, op := range s.Operations {
		if op.Kind != KindAction && op.Kind != KindFunction {
			continue
		}
		if op.Bound && len(op.Parameters) > 0 {
			binding := elementType(op.Parameters[0].Type)
			boundTo[binding] = append(boundTo[binding], op)
		} else {
			unbound[op.Name] = op
		}
	}

	visited := map[string]bool{}
	unresolved := map[string]bool{}
	var queue []string
	push := func(ref string) {
		t := elementType(ref)
		if t == "" || strings.HasPrefix(t, "Edm.") || visited[t] {
			return
		}
		visited[t] = true
		if declared[t] == nil && !enums[t] {
			unresolved[t] = true
			return
		}
		queue = append(queue, t)
	}
	pushOperation := func(op Operation) {
		for _, p := range op.Parameters {
			push(p.Type)
		}
		push(op.ReturnType)
	}

	for _, es := range s.EntitySets {
		push(es.EntityType)
	}
	for _, sg := range s.Singletons {
		push(sg.EntityType)
	}
	for _, op := range s.Operations {
		switch op.Kind {
		case KindFunctionImport, KindActionImport:
			pushOperation(op)
			if target, ok := unbound[op.Target]; ok {
				pushOperation(target)
			}
		}
	}

	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		t := declared[name]
		if t == nil {
			continue // enum type: no further references
		}
		push(t.BaseType)
		for _, p := range t.Properties {
			push(p.Type)
		}
		for _, n := range t.Navigation {
			push(n.Target)
		}
		for _, op := range boundTo[name] {
			pushOperation(op)
		}
	}

	g := &GraphSummary{}
	for name := range visited {
		if !unresolved[name] {
			g.Reachable = append(g.Reachable, name)
		}
	}
	for name := range declared {
		if !visited[name] {
			g.Unreachable = append(g.Unreachable, name)
		}
	}
	for name := range enums {
		if !visited[name] {
			g.Unreachable = append(g.Unreachable, name)
		}
	}
	for name := range unresolved {
		g.Unresolved = append(g.Unresolved, name)
	}
	sort.Strings(g.Reachable)
	sort.Strings(g.Unreachable)
	sort.Strings(g.Unresolved)
	return g
}

// elementType strips Collection(...) from a type reference.
func elementType(ref string) string {
	if inner, ok := strings.CutPrefix(ref, "Collection("); ok {
		return strings.TrimSuffix(inner, ")")
	}
	return ref
}
