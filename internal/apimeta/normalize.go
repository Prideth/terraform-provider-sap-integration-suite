package apimeta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// Normalize sorts every list of a Service so that its JSON form depends
// only on the contract, not on the order of the source document. Key
// properties and operation parameters keep their declared order, since it
// is significant.
func Normalize(s *Service) {
	sort.Strings(s.Namespaces)
	sort.Strings(s.ServiceDocument)
	sortSets(s.EntitySets)
	sortSets(s.Singletons)
	sortTypes(s.EntityTypes)
	sortTypes(s.ComplexTypes)
	sort.Slice(s.EnumTypes, func(i, j int) bool { return s.EnumTypes[i].Name < s.EnumTypes[j].Name })
	sort.Slice(s.Associations, func(i, j int) bool { return s.Associations[i].Name < s.Associations[j].Name })
	for i := range s.Associations {
		ends := s.Associations[i].Ends
		sort.Slice(ends, func(a, b int) bool { return ends[a].Role < ends[b].Role })
	}
	sort.Slice(s.Operations, func(i, j int) bool { return OperationKey(s.Operations[i]) < OperationKey(s.Operations[j]) })
	if s.REST != nil {
		normalizeREST(s.REST)
	}
}

func sortSets(sets []EntitySet) {
	sort.Slice(sets, func(i, j int) bool { return sets[i].Name < sets[j].Name })
}

func sortTypes(types []EntityType) {
	sort.Slice(types, func(i, j int) bool { return types[i].Name < types[j].Name })
	for i := range types {
		props := types[i].Properties
		sort.Slice(props, func(a, b int) bool { return props[a].Name < props[b].Name })
		navs := types[i].Navigation
		sort.Slice(navs, func(a, b int) bool { return navs[a].Name < navs[b].Name })
	}
}

// OperationKey identifies an operation across snapshots. Bound V4
// operations can be overloaded by their binding type, which is therefore
// part of the key.
func OperationKey(op Operation) string {
	key := op.Kind + " " + op.Name
	if op.Bound && len(op.Parameters) > 0 {
		key += "(" + op.Parameters[0].Type + ")"
	}
	return key
}

var urlPattern = regexp.MustCompile(`https?://[^\s"'<>]+`)

// RedactURLs replaces every URL in s. Snapshots and reports must not carry
// tenant host names, which can appear in annotations or descriptions.
func RedactURLs(s string) string {
	return urlPattern.ReplaceAllString(s, "<redacted-url>")
}

// Marshal renders a Service as indented JSON with a trailing newline: the
// snapshot format.
func Marshal(s *Service) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// SnapshotDir returns testdata/api-metadata of the module the working
// directory belongs to, found by walking up to go.mod.
func SnapshotDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "testdata", "api-metadata"), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("apimeta: no go.mod above the working directory")
		}
		dir = parent
	}
}

// LoadSnapshot reads testdata/api-metadata/<id>.json.
func LoadSnapshot(id string) (*Service, error) {
	dir, err := SnapshotDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, id+".json")) //nolint:gosec // G304: fixed directory, service ID from the registry
	if err != nil {
		return nil, err
	}
	var s Service
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("apimeta: snapshot %s: %w", id, err)
	}
	return &s, nil
}

// WriteSnapshot writes testdata/api-metadata/<id>.json.
func WriteSnapshot(s *Service) error {
	dir, err := SnapshotDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	data, err := Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, s.ID+".json"), data, 0o600)
}
