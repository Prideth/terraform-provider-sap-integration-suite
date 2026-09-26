package samples

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// EnvLocalContent names a directory with integration content exported from
// a tenant (package exports and artifact downloads, nested ZIPs included).
// Such content is often private: it is only read from that directory, never
// copied into the repository, and tests that need it skip when it is unset.
const EnvLocalContent = "SAP_LOCAL_CONTENT_DIR"

// LocalArtifact is one design-time artifact found in the local content.
type LocalArtifact struct {
	Source string // file (and nested file) it came from
	// Type is the export's resourceType ("IFlow", "MessageMapping",
	// "ScriptCollection", "ValueMapping"), or "Folder" for an artifact from
	// a download, whose type only its MANIFEST tells.
	Type    string
	Name    string
	Content []byte
}

// LocalArtifacts reads every artifact from the directory in
// SAP_LOCAL_CONTENT_DIR and skips the test when it is not set.
func LocalArtifacts(t testing.TB) []LocalArtifact {
	t.Helper()
	dir := os.Getenv(EnvLocalContent)
	if dir == "" {
		t.Skipf("%s is not set", EnvLocalContent)
	}
	var artifacts []LocalArtifact
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error { //nolint:gosec // G703: dir is the developer's own content directory from SAP_LOCAL_CONTENT_DIR
		if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".zip") {
			return err
		}
		data, err := os.ReadFile(path) //nolint:gosec // G304: files under the developer's own content directory
		if err != nil {
			return err
		}
		artifacts = append(artifacts, collect(filepath.Base(path), data, 0)...)
		return nil
	})
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	return artifacts
}

// LocalArtifactOfType returns the first local artifact of an export type,
// skipping the test when there is none.
func LocalArtifactOfType(t testing.TB, exportType string) LocalArtifact {
	t.Helper()
	for _, a := range LocalArtifacts(t) {
		if a.Type == exportType {
			return a
		}
	}
	t.Skipf("no %s in %s", exportType, os.Getenv(EnvLocalContent))
	return LocalArtifact{}
}

func collect(source string, data []byte, depth int) []LocalArtifact {
	if depth > 3 {
		return nil
	}
	files, err := zipFiles(data)
	if err != nil {
		return nil
	}
	if _, ok := files["resources.cnt"]; ok {
		exported, err := ExportArtifacts(data)
		if err != nil {
			return []LocalArtifact{{Source: source, Type: "UnreadableExport: " + err.Error()}}
		}
		var out []LocalArtifact
		for _, a := range exported {
			if len(a.Content) > 0 && a.Type != "ContentPackage" && a.Type != "File" {
				out = append(out, LocalArtifact{Source: source, Type: a.Type, Name: a.DisplayName, Content: a.Content})
			}
		}
		return out
	}
	var out []LocalArtifact
	for _, name := range slices.Sorted(maps.Keys(files)) {
		content := files[name]
		switch {
		case strings.EqualFold(filepath.Ext(name), ".zip"):
			out = append(out, collect(source+"/"+name, content, depth+1)...)
		case name == "META-INF/MANIFEST.MF":
			out = append(out, LocalArtifact{Source: source, Type: "Folder", Name: source, Content: data})
		case strings.HasSuffix(name, "/META-INF/MANIFEST.MF"):
			prefix := strings.TrimSuffix(name, "META-INF/MANIFEST.MF")
			if folder, err := subZip(files, prefix); err == nil {
				out = append(out, LocalArtifact{Source: source + "/" + prefix, Type: "Folder", Name: strings.TrimSuffix(prefix, "/"), Content: folder})
			}
		}
	}
	return out
}

// subZip builds a ZIP of every file under prefix, relative to it: an
// artifact folder of a download as the provider uploads it.
func subZip(files map[string][]byte, prefix string) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, name := range slices.Sorted(maps.Keys(files)) {
		content := files[name]
		if !strings.HasPrefix(name, prefix) || strings.HasSuffix(name, "/") {
			continue
		}
		f, err := w.Create(strings.TrimPrefix(name, prefix))
		if err != nil {
			return nil, err
		}
		if _, err := f.Write(content); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var valueMappingEntries = regexp.MustCompile(`(?s)<group .*</group>`)

// SyntheticValueMapping replaces the entries of a value mapping artifact
// with made-up ones in the same XML format (<vm version="2.0"> with groups of
// agency/schema/value entries), so a live test never uploads the business
// data of a real value mapping.
func SyntheticValueMapping(content []byte, suffix string) ([]byte, error) {
	files, err := zipFiles(content)
	if err != nil {
		return nil, err
	}
	xml, ok := files["value_mapping.xml"]
	if !ok || !valueMappingEntries.Match(xml) {
		return nil, fmt.Errorf("samples: no value_mapping.xml with groups in the artifact")
	}
	groups := ""
	for i, pair := range [][2]string{{"A", "1"}, {"B", "2"}} {
		groups += fmt.Sprintf(`<group id="%032x"><entry><agency>tfacc_source</agency><schema>code_%s</schema><value>%s</value></entry>`+
			`<entry><agency>tfacc_target</agency><schema>code_%s</schema><value>%s</value></entry></group>`,
			i+1, suffix, pair[0], suffix, pair[1])
	}
	files["value_mapping.xml"] = valueMappingEntries.ReplaceAll(xml, []byte(groups))
	return subZip(files, "")
}
