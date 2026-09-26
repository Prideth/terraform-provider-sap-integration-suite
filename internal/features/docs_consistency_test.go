package features

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// denial matches wording that says a resource type does not exist or is not
// managed. A sentence that names a registered resource type together with
// such wording contradicts the catalog.
var denial = regexp.MustCompile(`(?i)\b(does not exist|doesn't exist|still does not exist|does not implement|no resource|does not manage|are not managed|is not managed|is not offered|would be named|a future)\b`)

// repoRoot walks up from the package directory to go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

// userDocs lists the hand-written documents that describe support: README,
// ROADMAP, CHANGELOG's unreleased part is left out (it narrates history),
// everything under docs/ except generated reference pages, and the guide
// templates.
func userDocs(t *testing.T, root string) []string {
	t.Helper()
	files := []string{filepath.Join(root, "README.md"), filepath.Join(root, "ROADMAP.md")}
	for _, dir := range []string{"docs", "templates"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel := filepath.ToSlash(strings.TrimPrefix(path, root))
			if d.IsDir() || (!strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".md.tmpl")) {
				return nil
			}
			// Generated from schemas and the catalog; checked by their generators.
			if strings.Contains(rel, "/docs/resources/") || strings.Contains(rel, "/docs/data-sources/") ||
				strings.HasSuffix(rel, "/docs/feature-support.md") || strings.HasSuffix(rel, "/docs/api-discovery-report.md") {
				return nil
			}
			// Generated guides duplicate their templates.
			if strings.Contains(rel, "/docs/guides/") {
				return nil
			}
			files = append(files, path)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return files
}

// sentences splits markdown text into rough sentences, keeping line breaks
// inside a paragraph together.
func sentences(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var out []string
	for _, para := range strings.Split(text, "\n\n") {
		para = strings.ReplaceAll(para, "\n", " ")
		for _, s := range regexp.MustCompile(`[.!?]\s+|\|`).Split(para, -1) {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

// TestDocs_DoNotDenyRegisteredResources fails when a hand-written document
// says a resource type the catalog registers does not exist or is not
// managed, the kind of contradiction a new resource leaves behind in older
// prose.
func TestDocs_DoNotDenyRegisteredResources(t *testing.T) {
	root := repoRoot(t)
	var types []string
	for _, f := range Catalog {
		types = append(types, f.ResourceTypes...)
	}
	for _, file := range userDocs(t, root) {
		data, err := os.ReadFile(file) //nolint:gosec // G304: files under the repository root
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range sentences(string(data)) {
			if !denial.MatchString(s) {
				continue
			}
			for _, typ := range types {
				if regexp.MustCompile(`\b` + typ + `\b`).MatchString(s) {
					rel, _ := filepath.Rel(root, file)
					t.Errorf("%s: %q is registered, but this sentence denies it: %s", filepath.ToSlash(rel), typ, s)
				}
			}
		}
	}
}
