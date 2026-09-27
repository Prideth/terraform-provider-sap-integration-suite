package provider

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/features"
)

var statusLine = regexp.MustCompile(`\*\*Status:\*\* ([a-z_]+)`)

// primaryFeature returns the catalog feature a type belongs to. A type can
// appear in more than one feature (for example in design-time versioning as
// well as in its own entry); the feature whose key ends in the type's name
// wins.
func primaryFeature(typeName string, dataSource bool) (features.Feature, bool) {
	short := strings.TrimPrefix(typeName, "sapintegrationsuite_")
	var first *features.Feature
	for i, f := range features.Catalog {
		types := f.ResourceTypes
		if dataSource {
			types = f.DataSourceTypes
		}
		for _, t := range types {
			if t != typeName {
				continue
			}
			if strings.HasSuffix(f.Key, "."+short) {
				return f, true
			}
			if first == nil {
				first = &features.Catalog[i]
			}
		}
	}
	if first != nil {
		return *first, true
	}
	return features.Feature{}, false
}

// checkDocTemplate verifies one Registry page template: it exists, has the
// sections a practitioner needs, embeds its own example, and states the
// support status the catalog records.
func checkDocTemplate(t *testing.T, typeName, dir, exampleDir string, sections []string, dataSource bool) {
	t.Helper()
	short := strings.TrimPrefix(typeName, "sapintegrationsuite_")
	file := filepath.Join("..", "..", "templates", dir, short+".md.tmpl")
	src, err := os.ReadFile(file)
	if err != nil {
		t.Errorf("%s: no page template %s", typeName, file)
		return
	}
	text := string(src)
	for _, s := range sections {
		if !strings.Contains(text, s) {
			t.Errorf("%s: %s has no %q", typeName, file, s)
		}
	}
	example := "examples/" + exampleDir + "/" + typeName + "/"
	if !strings.Contains(text, example) {
		t.Errorf("%s: %s does not embed its example from %s", typeName, file, example)
	}
	m := statusLine.FindStringSubmatch(text)
	if m == nil {
		t.Errorf("%s: %s has no **Status:** line", typeName, file)
		return
	}
	f, ok := primaryFeature(typeName, dataSource)
	if !ok {
		return // provider metadata data sources have no catalog entry
	}
	if m[1] != string(f.SupportStatus) {
		t.Errorf("%s: page says status %q, the catalog says %q", typeName, m[1], f.SupportStatus)
	}
	switch f.SupportStatus {
	case features.StatusExperimental:
		if !strings.Contains(text, "enable_experimental") {
			t.Errorf("%s: experimental, but the page does not name enable_experimental", typeName)
		}
	case features.StatusUnofficial:
		if !strings.Contains(text, "enable_unofficial") {
			t.Errorf("%s: unofficial, but the page does not name enable_unofficial", typeName)
		}
	}
	if !dataSource && len(f.UndocumentedOperations) > 0 && !strings.Contains(text, "enable_unofficial") {
		t.Errorf("%s: the catalog lists undocumented operations, but the page does not name enable_unofficial", typeName)
	}
}

// TestDocTemplates_CoverEveryType keeps the Registry documentation complete:
// every registered resource and data source has a hand-written page
// template with the same structure, and the status it states cannot drift
// from the feature catalog.
func TestDocTemplates_CoverEveryType(t *testing.T) {
	ctx := context.Background()
	p := &sapIntegrationSuiteProvider{version: "test"}
	for _, newResource := range p.Resources(ctx) {
		var meta resource.MetadataResponse
		newResource().Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "sapintegrationsuite"}, &meta)
		checkDocTemplate(t, meta.TypeName, "resources", "resources", []string{
			"## Prerequisites", "## Lifecycle", "## Example Usage", "{{ .SchemaMarkdown", "## Import", "## Limitations", "## Related",
		}, false)
	}
	for _, newDataSource := range p.DataSources(ctx) {
		var meta datasource.MetadataResponse
		newDataSource().Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "sapintegrationsuite"}, &meta)
		checkDocTemplate(t, meta.TypeName, "data-sources", "data-sources", []string{
			"## Example Usage", "{{ .SchemaMarkdown", "## Related",
		}, true)
	}
}

var markdownLink = regexp.MustCompile(`\]\(([^)#\s]+\.md)(#[^)]*)?\)`)

// TestDocs_RelativeLinksResolve fails when a page of the generated
// documentation links to another page that does not exist, for example after
// a resource or guide was renamed.
func TestDocs_RelativeLinksResolve(t *testing.T) {
	root := filepath.Join("..", "..", "docs")
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range markdownLink.FindAllStringSubmatch(string(src), -1) {
			target := m[1]
			if strings.Contains(target, "://") {
				continue
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(path), filepath.FromSlash(target))); err != nil {
				t.Errorf("%s links to %s, which does not exist", path, target)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
