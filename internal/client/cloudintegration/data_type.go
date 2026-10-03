package cloudintegration

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

const dataTypeDesigntimeArtifactsEntitySet = "DataTypeDesigntimeArtifacts"

// DataType is the wire representation of a DataTypeDesigntimeArtifacts
// entity, a data type of the Cloud Integration ESR-style design-time
// content. SAP Help documents no request for it; the entity set, its
// properties and DataTypeDesigntimeArtifactSaveAsVersion come from the
// Integration Content API's $metadata, and every operation this client uses
// was verified on a tenant (gap probes of September and October 2026).
type DataType struct {
	ID           string `json:"Id"`
	Name         string `json:"Name"`
	PackageID    string `json:"PackageId"`
	Version      string `json:"Version,omitempty"`
	Description  string `json:"Description,omitempty"`
	Namespace    string `json:"Namespace,omitempty"`
	IsSimpleType bool   `json:"IsSimpleType,omitempty"`
}

// dataTypeCreate is the create body. The bundle carries name, namespace
// and description as well; SAP reads them from there.
type dataTypeCreate struct {
	ID        string `json:"Id"`
	Name      string `json:"Name"`
	PackageID string `json:"PackageId"`
	Content   string `json:"ArtifactContent"`
}

// DataTypeBundle describes the content of a complex data type. The provider
// builds the bundle itself because SAP needs files that a package export
// does not contain: a create with an export's bundle failed with 500 "map is
// null", the same bundle plus additionalAttributes.json and metainfo.prop
// was accepted (tenant, 2026-09-29).
type DataTypeBundle struct {
	ID          string
	Name        string
	Namespace   string
	Description string
	// XSD is the schema of the data type. SAP stores its complex type under
	// the data type's name.
	XSD string
}

// Build returns the bundle as SAP's own $value of a data type is laid out
// (2026-10-03): src/main/resources/xsd/<name>.xsd, .project,
// META-INF/MANIFEST.MF, src/main/resources/additionalAttributes.json and
// metainfo.prop.
func (b DataTypeBundle) Build(now time.Time) ([]byte, error) {
	attributes, err := json.Marshal(struct {
		Description        string `json:"Description"`
		QualifySchema      string `json:"qualifySchema"`
		BundleVersion      string `json:"BundleVersion"`
		ContentModel       string `json:"contentmodel"`
		Namespace          string `json:"namespace"`
		BundleSymbolicName string `json:"BundleSymbolicName"`
		Category           string `json:"category"`
		Classification     string `json:"classification"`
	}{
		Description:        b.Description,
		QualifySchema:      "0",
		BundleVersion:      "1.0.0",
		ContentModel:       "S",
		Namespace:          b.Namespace,
		BundleSymbolicName: b.ID,
		Category:           "Complex Type",
		Classification:     "Free-Style Data Type",
	})
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: encoding data type attributes: %w", err)
	}

	files := []struct{ name, content string }{
		{"src/main/resources/xsd/" + b.Name + ".xsd", b.XSD},
		{".project", dataTypeProject(b.ID)},
		{"META-INF/MANIFEST.MF", dataTypeManifest(b.ID, b.Name)},
		{"src/main/resources/additionalAttributes.json", string(attributes)},
		{"metainfo.prop", "#Store metainfo properties\n#" + now.UTC().Format("Mon Jan 02 15:04:05 MST 2006") +
			"\ndescription=" + escapeProperty(b.Description) + "\n"},
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, f := range files {
		entry, err := w.Create(f.name)
		if err != nil {
			return nil, fmt.Errorf("cloudintegration: building data type bundle: %w", err)
		}
		if _, err := entry.Write([]byte(f.content)); err != nil {
			return nil, fmt.Errorf("cloudintegration: building data type bundle: %w", err)
		}
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("cloudintegration: building data type bundle: %w", err)
	}
	return buf.Bytes(), nil
}

func dataTypeProject(id string) string {
	return `<?xml version="1.0" encoding="UTF-8"?><projectDescription>
   <name>` + xmlEscape(id) + `</name>
   <comment/>
   <projects/>
   <buildSpec>
      <buildCommand>
         <name>org.eclipse.jdt.core.javabuilder</name>
         <arguments/>
      </buildCommand>
   </buildSpec>
   <natures>
      <nature>org.eclipse.jdt.core.javanature</nature>
      <nature>com.sap.ide.ifl.project.support.project.nature</nature>
      <nature>com.sap.ide.ifl.bsn</nature>
   </natures>
</projectDescription>
`
}

// dataTypeManifest is the manifest SAP writes for a data type, with lines
// wrapped at 72 bytes as the JAR specification requires.
func dataTypeManifest(id, name string) string {
	headers := []string{
		"Manifest-Version: 1.0",
		"Bundle-ManifestVersion: 2",
		"Bundle-Name: " + name,
		"Bundle-SymbolicName: " + id,
		"Bundle-Version: 1.0.0",
		"SAP-BundleType: DataType",
		"SAP-NodeType: IFLMAP",
		"Import-Package: ",
		`Provide-Capability: datatype.` + id + `;version:Version="1.0.0"`,
	}
	var b strings.Builder
	for _, h := range headers {
		b.WriteString(wrapManifestLine(h))
	}
	b.WriteString("\n")
	return b.String()
}

// wrapManifestLine splits a header into lines of at most 72 bytes; each
// continuation line starts with one space.
func wrapManifestLine(header string) string {
	const limit = 72
	if len(header) <= limit {
		return header + "\n"
	}
	var b strings.Builder
	b.WriteString(header[:limit] + "\n")
	rest := header[limit:]
	for len(rest) > limit-1 {
		b.WriteString(" " + rest[:limit-1] + "\n")
		rest = rest[limit-1:]
	}
	b.WriteString(" " + rest + "\n")
	return b.String()
}

// xmlEscape escapes text for an XML element.
var xmlEscape = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace

// escapeProperty escapes a value for a Java properties file.
func escapeProperty(s string) string {
	r := strings.NewReplacer(`\`, `\\`, "\n", `\n`, "\r", `\r`, "=", `\=`, ":", `\:`)
	return r.Replace(s)
}

// GetDataType reads the active version of a data type.
func (c *Client) GetDataType(ctx context.Context, id string) (*DataType, error) {
	key, err := designtimeArtifactKey(id, activeVersion)
	if err != nil {
		return nil, err
	}
	body, err := c.odata.Get(ctx, v2.BuildPath(dataTypeDesigntimeArtifactsEntitySet, key, ""))
	if err != nil {
		return nil, err
	}
	var dt DataType
	if err := v2.DecodeEntity(body, &dt); err != nil {
		return nil, err
	}
	return &dt, nil
}

// CreateDataType creates a data type in a package from a bundle built with
// DataTypeBundle.Build. The tenant answered with 201 and the entity; without
// a body the data type is read back.
func (c *Client) CreateDataType(ctx context.Context, packageID, id, name string, bundle []byte) (*DataType, error) {
	payload, err := json.Marshal(dataTypeCreate{
		ID: id, Name: name, PackageID: packageID, Content: base64.StdEncoding.EncodeToString(bundle),
	})
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: encoding data type: %w", err)
	}
	body, err := c.odata.Post(ctx, dataTypeDesigntimeArtifactsEntitySet, payload)
	if err != nil {
		return nil, err
	}
	if v2.EmptyBody(body) {
		return c.GetDataType(ctx, id)
	}
	var dt DataType
	if err := v2.DecodeEntity(body, &dt); err != nil {
		return nil, err
	}
	return &dt, nil
}

// UpdateDataType uploads a new bundle with PUT on the active version. A
// tenant answered with 200, and the read-back content had the new element
// (2026-10-03).
func (c *Client) UpdateDataType(ctx context.Context, id, name string, bundle []byte) (*DataType, error) {
	return updateDesigntimeArtifact(ctx, c, dataTypeDesigntimeArtifactsEntitySet, id, name, bundle,
		func() (*DataType, error) { return c.GetDataType(ctx, id) })
}

// SaveDataTypeAsVersion saves the current content under an explicit version.
func (c *Client) SaveDataTypeAsVersion(ctx context.Context, id, version string) (*DataType, error) {
	return saveAsVersion(ctx, c, "DataTypeDesigntimeArtifactSaveAsVersion", id, version,
		func() (*DataType, error) { return c.GetDataType(ctx, id) })
}

// DeleteDataType deletes a data type; the tenant answered with 200.
func (c *Client) DeleteDataType(ctx context.Context, id string) error {
	key, err := designtimeArtifactKey(id, activeVersion)
	if err != nil {
		return err
	}
	return c.odata.Delete(ctx, v2.BuildPath(dataTypeDesigntimeArtifactsEntitySet, key, ""))
}
