package provider

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// exampleSchema is what an example block may use: its attributes (and which
// are required) and its nested blocks.
type exampleSchema struct {
	attributes map[string]bool // name -> required
	blocks     map[string]bool
}

// metaArguments are Terraform's own arguments, valid in every resource and
// data block.
var metaArguments = map[string]bool{"count": true, "for_each": true, "depends_on": true, "provider": true}
var metaBlocks = map[string]bool{"lifecycle": true, "provisioner": true, "connection": true}

func exampleSchemas(t *testing.T) (resources, dataSources map[string]exampleSchema, providerSchema exampleSchema) {
	t.Helper()
	ctx := context.Background()
	p := &sapIntegrationSuiteProvider{version: "test"}
	resources, dataSources = map[string]exampleSchema{}, map[string]exampleSchema{}
	for _, newResource := range p.Resources(ctx) {
		r := newResource()
		var meta resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "sapintegrationsuite"}, &meta)
		var s resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &s)
		es := exampleSchema{attributes: map[string]bool{}, blocks: map[string]bool{}}
		for name, a := range s.Schema.Attributes {
			es.attributes[name] = a.IsRequired()
		}
		for name := range s.Schema.Blocks {
			es.blocks[name] = true
		}
		resources[meta.TypeName] = es
	}
	for _, newDataSource := range p.DataSources(ctx) {
		d := newDataSource()
		var meta datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "sapintegrationsuite"}, &meta)
		var s datasource.SchemaResponse
		d.Schema(ctx, datasource.SchemaRequest{}, &s)
		es := exampleSchema{attributes: map[string]bool{}, blocks: map[string]bool{}}
		for name, a := range s.Schema.Attributes {
			es.attributes[name] = a.IsRequired()
		}
		for name := range s.Schema.Blocks {
			es.blocks[name] = true
		}
		dataSources[meta.TypeName] = es
	}
	var ps provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &ps)
	providerSchema = exampleSchema{attributes: map[string]bool{}, blocks: map[string]bool{}}
	for name := range ps.Schema.Attributes {
		providerSchema.attributes[name] = false
	}
	for name := range ps.Schema.Blocks {
		providerSchema.blocks[name] = true
	}
	return resources, dataSources, providerSchema
}

// checkExampleBody reports arguments and blocks the schema does not have,
// and required arguments that are missing.
func checkExampleBody(t *testing.T, where string, body *hclsyntax.Body, s exampleSchema, requireAll bool) {
	t.Helper()
	for name := range body.Attributes {
		if _, ok := s.attributes[name]; !ok && !metaArguments[name] {
			t.Errorf("%s: unknown argument %q", where, name)
		}
	}
	for _, b := range body.Blocks {
		if !s.blocks[b.Type] && !metaBlocks[b.Type] {
			t.Errorf("%s: unknown block %q", where, b.Type)
		}
	}
	for name, required := range s.attributes {
		if _, set := body.Attributes[name]; requireAll && required && !set {
			t.Errorf("%s: required argument %q is missing", where, name)
		}
	}
}

// checkReferences reports references to attributes that a resource or data
// source of this provider does not have, for example
// sapintegrationsuite_integration_flow.orders.flow_version.
func checkReferences(t *testing.T, where string, body *hclsyntax.Body, resources, dataSources map[string]exampleSchema) {
	t.Helper()
	for _, attr := range body.Attributes {
		for _, traversal := range attr.Expr.Variables() {
			checkReference(t, where, traversal, resources, dataSources)
		}
	}
	for _, nested := range body.Blocks {
		checkReferences(t, where, nested.Body, resources, dataSources)
	}
}

// checkReference checks one reference such as
// sapintegrationsuite_integration_flow.orders.version or
// data.sapintegrationsuite_partners.all.pids.
func checkReference(t *testing.T, where string, traversal hcl.Traversal, resources, dataSources map[string]exampleSchema) {
	t.Helper()
	root := traversal.RootName()
	schemas, typeName, attrName, prefix := resources, root, "", ""
	switch {
	case strings.HasPrefix(root, "sapintegrationsuite_") && len(traversal) >= 3:
		attrName = stepName(traversal[2])
	case root == "data" && len(traversal) >= 4:
		schemas, typeName, attrName, prefix = dataSources, stepName(traversal[1]), stepName(traversal[3]), "data."
	default:
		return
	}
	s, ok := schemas[typeName]
	if !ok || attrName == "" {
		return
	}
	if _, has := s.attributes[attrName]; !has {
		t.Errorf("%s: %s%s has no attribute %q", where, prefix, typeName, attrName)
	}
}

func stepName(step hcl.Traverser) string {
	if a, ok := step.(hcl.TraverseAttr); ok {
		return a.Name
	}
	return ""
}

// TestExamples_MatchSchema checks every example that the Registry
// documentation shows against the provider's schemas, so an example can
// neither use an argument that does not exist nor leave out a required one.
func TestExamples_MatchSchema(t *testing.T) {
	resources, dataSources, providerSchema := exampleSchemas(t)
	root := filepath.Join("..", "..", "examples")
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".tf") {
			files = append(files, path)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatal("no examples found")
	}
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		checkExampleSource(t, file, src, true, resources, dataSources, providerSchema)
	}
}

// TestGuideSnippets_MatchSchema applies the same check to the HCL snippets
// in the guide and page templates. Snippets are often deliberately partial
// ("# ..."), so a missing required argument is not reported there.
func TestGuideSnippets_MatchSchema(t *testing.T) {
	resources, dataSources, providerSchema := exampleSchemas(t)
	root := filepath.Join("..", "..", "templates")
	fence := regexp.MustCompile("(?s)```(?:terraform|hcl)\r?\n(.*?)```")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, m := range fence.FindAllSubmatch(src, -1) {
			checkExampleSource(t, fmt.Sprintf("%s (snippet %d)", path, i+1), m[1], false, resources, dataSources, providerSchema)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	readme, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for i, m := range fence.FindAllSubmatch(readme, -1) {
		checkExampleSource(t, fmt.Sprintf("README.md (snippet %d)", i+1), m[1], false, resources, dataSources, providerSchema)
	}
}

func checkExampleSource(t *testing.T, file string, src []byte, requireAll bool, resources, dataSources map[string]exampleSchema, providerSchema exampleSchema) {
	t.Helper()
	{
		f, diags := hclsyntax.ParseConfig(src, file, hcl.InitialPos)
		if diags.HasErrors() {
			t.Errorf("%s: %s", file, diags.Error())
			return
		}
		body := f.Body.(*hclsyntax.Body)
		for _, b := range body.Blocks {
			where := file + ": " + b.Type + " " + strings.Join(b.Labels, ".")
			switch {
			case b.Type == "resource" && len(b.Labels) == 2 && strings.HasPrefix(b.Labels[0], "sapintegrationsuite_"):
				s, ok := resources[b.Labels[0]]
				if !ok {
					t.Errorf("%s: unknown resource type", where)
					continue
				}
				checkExampleBody(t, where, b.Body, s, requireAll)
			case b.Type == "data" && len(b.Labels) == 2 && strings.HasPrefix(b.Labels[0], "sapintegrationsuite_"):
				s, ok := dataSources[b.Labels[0]]
				if !ok {
					t.Errorf("%s: unknown data source type", where)
					continue
				}
				checkExampleBody(t, where, b.Body, s, requireAll)
			case b.Type == "provider" && len(b.Labels) == 1 && b.Labels[0] == "sapintegrationsuite":
				checkExampleBody(t, where, b.Body, providerSchema, requireAll)
			}
			checkReferences(t, where, b.Body, resources, dataSources)
		}
	}
}
