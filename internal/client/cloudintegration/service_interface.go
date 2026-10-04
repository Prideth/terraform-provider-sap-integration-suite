package cloudintegration

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

const (
	serviceInterfaceDesigntimeArtifactsEntitySet = "ServiceInterfaceDesigntimeArtifacts"
	serviceInterfaceSaveAsVersion                = "ServiceInterfaceDesigntimeArtifactSaveAsVersion"
)

// ServiceInterface is the wire representation of a ServiceInterfaceDesigntimeArtifacts
// entity. SAP documents no request for it; the entity set and its
// SaveAsVersion function import come from the Integration Content API's
// $metadata. A create without content works (tenant, 2026-10-03): SAP then
// generates a bundle with one outbound asynchronous operation that names no
// message type. Model holds the operation model read from the bundle.
type ServiceInterface struct {
	ID          string `json:"Id"`
	Name        string `json:"Name"`
	PackageID   string `json:"PackageId"`
	Version     string `json:"Version,omitempty"`
	Description string `json:"Description,omitempty"`
	Namespace   string `json:"Namespace,omitempty"`

	Model *ServiceInterfaceModel `json:"-"`
}

// serviceInterfaceCreate is the create body; it carries no content.
type serviceInterfaceCreate struct {
	ID          string `json:"Id"`
	Name        string `json:"Name"`
	PackageID   string `json:"PackageId"`
	Namespace   string `json:"Namespace"`
	Description string `json:"Description,omitempty"`
}

// ServiceInterfaceModel is the operation model of a service interface, the
// JSON file src/main/resources/json/<name>.json in its bundle. The field
// names follow the bundles of two service interfaces SAP's editor created
// (package exports, 2026-10-03 and 2026-10-04): outbound and stateless, one
// with an asynchronous operation (request and fault message), one with a
// synchronous operation (request, response and fault message).
type ServiceInterfaceModel struct {
	Name             string                      `json:"name"`
	Namespace        string                      `json:"namespace"`
	Category         string                      `json:"category"`
	InterfacePattern string                      `json:"interfacePattern"`
	Operations       []ServiceInterfaceOperation `json:"operations"`
}

// ServiceInterfaceOperation is one operation of the model.
type ServiceInterfaceOperation struct {
	Name          string                    `json:"name"`
	OperationMode string                    `json:"operationMode"`
	IsSynchronous bool                      `json:"isSynchronous"`
	Request       *ServiceInterfaceMessage  `json:"request,omitempty"`
	Response      *ServiceInterfaceMessage  `json:"response,omitempty"`
	Faults        []ServiceInterfaceMessage `json:"faults,omitempty"`
}

// ServiceInterfaceMessage is a message an operation refers to.
// BundleSymbolicName is the referenced artifact's ID.
type ServiceInterfaceMessage struct {
	TypeID               string `json:"typeId"`
	Role                 string `json:"role"`
	Name                 string `json:"name"`
	PackageName          string `json:"packageName"`
	PackageTechnicalName string `json:"packageTechnicalName"`
	BundleSymbolicName   string `json:"bundleSymbolicName"`
	Namespace            string `json:"namespace"`
	Version              string `json:"version,omitempty"`
	XMLNS                string `json:"xmlns,omitempty"`
}

// The typeId values of the messages an operation names.
const (
	serviceInterfaceRequestTypeID = "ifmmessage"
	serviceInterfaceFaultTypeID   = "ifmfaultm"
)

func serviceInterfacePath(id string) (string, error) {
	key, err := designtimeArtifactKey(id, activeVersion)
	if err != nil {
		return "", err
	}
	return v2.BuildPath(serviceInterfaceDesigntimeArtifactsEntitySet, key, ""), nil
}

// GetServiceInterface reads the active version of a service interface with
// the operation model of its bundle.
func (c *Client) GetServiceInterface(ctx context.Context, id string) (*ServiceInterface, error) {
	path, err := serviceInterfacePath(id)
	if err != nil {
		return nil, err
	}
	body, err := c.odata.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var si ServiceInterface
	if err := v2.DecodeEntity(body, &si); err != nil {
		return nil, err
	}
	content, err := c.GetServiceInterfaceContent(ctx, id)
	if err != nil {
		return nil, err
	}
	model, err := ParseServiceInterfaceBundle(content)
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: reading the service interface bundle: %w", err)
	}
	si.Model = model
	return &si, nil
}

// GetServiceInterfaceContent returns the bundle ($value) of a service
// interface: an archive holding the interface's own bundle and a resolved
// child, both archives themselves.
func (c *Client) GetServiceInterfaceContent(ctx context.Context, id string) ([]byte, error) {
	path, err := serviceInterfacePath(id)
	if err != nil {
		return nil, err
	}
	content, err := c.odata.Get(ctx, path+"/$value")
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: reading the service interface bundle: %w", err)
	}
	return content, nil
}

// CreateServiceInterface creates a service interface without content and
// reads it back.
func (c *Client) CreateServiceInterface(ctx context.Context, si ServiceInterface) (*ServiceInterface, error) {
	payload, err := json.Marshal(serviceInterfaceCreate{
		ID: si.ID, Name: si.Name, PackageID: si.PackageID, Namespace: si.Namespace, Description: si.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: encoding service interface: %w", err)
	}
	if _, err := c.odata.Post(ctx, serviceInterfaceDesigntimeArtifactsEntitySet, payload); err != nil {
		return nil, err
	}
	return c.GetServiceInterface(ctx, si.ID)
}

// SaveServiceInterfaceAsVersion saves the current content under an
// explicit version.
func (c *Client) SaveServiceInterfaceAsVersion(ctx context.Context, id, version string) (*ServiceInterface, error) {
	return saveAsVersion(ctx, c, serviceInterfaceSaveAsVersion, id, version,
		func() (*ServiceInterface, error) { return c.GetServiceInterface(ctx, id) })
}

// DeleteServiceInterface deletes a service interface.
func (c *Client) DeleteServiceInterface(ctx context.Context, id string) error {
	path, err := serviceInterfacePath(id)
	if err != nil {
		return err
	}
	return c.odata.Delete(ctx, path)
}

var serviceInterfaceModelEntry = regexp.MustCompile(`(^|/)src/main/resources/json/[^/]+\.json$`)

// ParseServiceInterfaceBundle finds the operation model in a service
// interface bundle, looking into nested archives too, and decodes it.
func ParseServiceInterfaceBundle(content []byte) (*ServiceInterfaceModel, error) {
	raw, err := findServiceInterfaceModel(content, 0)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, fmt.Errorf("no operation model (src/main/resources/json/*.json) in the bundle")
	}
	var model ServiceInterfaceModel
	if err := json.Unmarshal(raw, &model); err != nil {
		return nil, fmt.Errorf("operation model: %w", err)
	}
	return &model, nil
}

func findServiceInterfaceModel(content []byte, depth int) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("not a ZIP archive: %w", err)
	}
	var nested [][]byte
	for _, f := range r.File {
		data, err := readZipFile(f)
		if err != nil {
			return nil, err
		}
		if serviceInterfaceModelEntry.MatchString(f.Name) {
			return data, nil
		}
		if depth < 2 && bytes.HasPrefix(data, []byte("PK\x03\x04")) {
			nested = append(nested, data)
		}
	}
	for _, n := range nested {
		if raw, err := findServiceInterfaceModel(n, depth+1); err == nil && raw != nil {
			return raw, nil
		}
	}
	return nil, nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, 16<<20))
}

// ServiceInterfaceOperationSpec is an operation as the provider writes it:
// a request message type, a response message type for a synchronous
// operation (none: asynchronous), and fault message types, each named by
// ID with the name, namespace, version and package SAP's editor records.
type ServiceInterfaceOperationSpec struct {
	Name     string
	Request  *ServiceInterfaceMessageRef
	Response *ServiceInterfaceMessageRef
	Faults   []ServiceInterfaceMessageRef
}

// ServiceInterfaceMessageRef names a message type or fault message type the
// way an operation refers to it.
type ServiceInterfaceMessageRef struct {
	ID                   string
	Name                 string
	Namespace            string
	Version              string
	PackageName          string
	PackageTechnicalName string
}

// wire writes a message the way the 2026-10-04 export does: xmlns repeats
// the namespace, version is the referenced artifact's version, and a fault
// has no role.
func (r ServiceInterfaceMessageRef) wire(typeID, role string) map[string]any {
	m := map[string]any{
		"typeId":               typeID,
		"name":                 r.Name,
		"namespace":            r.Namespace,
		"packageName":          r.PackageName,
		"packageTechnicalName": r.PackageTechnicalName,
		"bundleSymbolicName":   r.ID,
		"xmlns":                r.Namespace,
	}
	if r.Version != "" {
		m["version"] = r.Version
	}
	if role != "" {
		m["role"] = role
	}
	return m
}

// WithServiceInterfaceOperations returns the operation model JSON with its
// operations replaced, keeping every other field SAP wrote. Operations are
// written as SAP's editor writes them: request, response and faults,
// repeated in messageDetails. messageDetailsCount follows the two exports
// (asynchronous request and fault: 1; synchronous request, response and
// fault: 3); what SAP uses it for is not known.
func WithServiceInterfaceOperations(model []byte, ops []ServiceInterfaceOperationSpec) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(model, &doc); err != nil {
		return nil, fmt.Errorf("operation model: %w", err)
	}
	out := make([]any, 0, len(ops))
	for _, op := range ops {
		details := []any{}
		sync := op.Response != nil
		mode := "ASYNCHRONOUS"
		if sync {
			mode = "SYNCHRONOUS"
		}
		o := map[string]any{
			"name":                op.Name,
			"isUnreliable":        false,
			"isSynchronous":       sync,
			"operationIdempotent": false,
			"objectState":         "NOT_RELEASED",
			"operationPattern":    "NORMAL_OPERATION",
			"operationMode":       mode,
		}
		messages := 0
		if op.Request != nil {
			req := op.Request.wire(serviceInterfaceRequestTypeID, "Request")
			o["request"] = req
			details = append(details, req)
			messages++
		}
		if sync {
			resp := op.Response.wire(serviceInterfaceRequestTypeID, "Response")
			o["response"] = resp
			details = append(details, resp)
			messages++
		}
		faults := []any{}
		for _, f := range op.Faults {
			w := f.wire(serviceInterfaceFaultTypeID, "")
			faults = append(faults, w)
			details = append(details, w)
		}
		if len(faults) > 0 {
			o["faults"] = faults
		}
		o["messageDetails"] = details
		if sync {
			o["messageDetailsCount"] = len(details)
		} else {
			o["messageDetailsCount"] = messages
		}
		out = append(out, o)
	}
	doc["operations"] = out
	return json.Marshal(doc)
}

// ServiceInterfaceRequireCapability is the manifest header that names the
// message types and fault message types the operations refer to, in the
// form of SAP's editor: fault message types first, then message types, each
// sorted by ID.
func ServiceInterfaceRequireCapability(ops []ServiceInterfaceOperationSpec) string {
	faultIDs := map[string]bool{}
	messageIDs := map[string]bool{}
	for _, op := range ops {
		for _, f := range op.Faults {
			faultIDs[f.ID] = true
		}
		for _, m := range []*ServiceInterfaceMessageRef{op.Request, op.Response} {
			if m != nil {
				messageIDs[m.ID] = true
			}
		}
	}
	var all []string
	for _, id := range sortedKeys(faultIDs) {
		all = append(all, fmt.Sprintf(`faultmessagetype.%s;resolution:=optional;bundleType:String="FaultMessageType";source:String="reference"`, id))
	}
	for _, id := range sortedKeys(messageIDs) {
		all = append(all, fmt.Sprintf(`messagetype.%s;resolution:=optional;bundleType:String="MessageType";source:String="reference"`, id))
	}
	if len(all) == 0 {
		return ""
	}
	return "Require-Capability: " + strings.Join(all, ",")
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// setManifestHeader replaces a header (with its continuation lines) in a
// MANIFEST.MF, or adds it before the trailing blank line; an empty header
// removes it.
func setManifestHeader(manifest, name, header string) string {
	text := strings.ReplaceAll(manifest, "\r\n", "\n")
	var kept []string
	skipping := false
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		if skipping && strings.HasPrefix(line, " ") {
			continue
		}
		skipping = strings.HasPrefix(line, name+":")
		if !skipping {
			kept = append(kept, line)
		}
	}
	if header != "" {
		kept = append(kept, strings.Split(strings.TrimSuffix(wrapManifestLine(header), "\n"), "\n")...)
	}
	return strings.Join(kept, "\r\n") + "\r\n\r\n"
}

// serviceInterfaceUpdate is the update body. Name is required: a PUT without
// it answered 500 "name is null"; with it, 200 and the uploaded operation
// model stored (tenant, 2026-10-04). The content is the nested bundle as
// $value returns it: the interface's own bundle alone answered 400 "The
// bundle is not of type ServiceInterface".
type serviceInterfaceUpdate struct {
	Name            string `json:"Name"`
	Description     string `json:"Description,omitempty"`
	ArtifactContent string `json:"ArtifactContent"`
}

// UpdateServiceInterface writes the operations into the service interface's
// current bundle and uploads it with the name and description, then reads
// the service interface back.
func (c *Client) UpdateServiceInterface(ctx context.Context, id, name, description string, ops []ServiceInterfaceOperationSpec) (*ServiceInterface, error) {
	path, err := serviceInterfacePath(id)
	if err != nil {
		return nil, err
	}
	current, err := c.GetServiceInterfaceContent(ctx, id)
	if err != nil {
		return nil, err
	}
	bundle, err := WithServiceInterfaceBundleOperations(current, ops)
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: building the service interface bundle: %w", err)
	}
	payload, err := json.Marshal(serviceInterfaceUpdate{
		Name: name, Description: description, ArtifactContent: base64.StdEncoding.EncodeToString(bundle),
	})
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: encoding service interface update: %w", err)
	}
	if _, err := c.odata.Put(ctx, path, payload); err != nil {
		return nil, err
	}
	return c.GetServiceInterface(ctx, id)
}

// ServiceInterfaceMessage resolves a message type or fault message type to
// the reference an operation records: its name, namespace and version, and
// its package's name and ID.
func (c *Client) ServiceInterfaceMessage(ctx context.Context, kind MessageTypeKind, id string) (ServiceInterfaceMessageRef, error) {
	mt, err := c.GetMessageType(ctx, kind, id)
	if err != nil {
		return ServiceInterfaceMessageRef{}, fmt.Errorf("cloudintegration: reading %s %q for a service interface: %w", kind.DisplayName, id, err)
	}
	ref := ServiceInterfaceMessageRef{
		ID: mt.ID, Name: mt.Name, Namespace: mt.Namespace, Version: mt.Version,
		PackageName: mt.PackageID, PackageTechnicalName: mt.PackageID,
	}
	if pkg, err := c.GetPackage(ctx, mt.PackageID); err == nil && pkg.Name != "" {
		ref.PackageName = pkg.Name
	}
	return ref, nil
}

const serviceInterfaceResolvedChildSuffix = "_SI_RESOLVED_CHILD"

// WithServiceInterfaceBundleOperations rewrites a service interface's $value:
// in the interface's own bundle (the nested archive that is not the
// resolved child) it replaces the operations of the model and the manifest's
// Require-Capability header. Every other entry stays as it is.
func WithServiceInterfaceBundleOperations(outer []byte, ops []ServiceInterfaceOperationSpec) ([]byte, error) {
	found := false
	out, err := rewriteZip(outer, func(name string, data []byte) ([]byte, error) {
		if found || strings.HasSuffix(name, serviceInterfaceResolvedChildSuffix) || !bytes.HasPrefix(data, []byte("PK\x03\x04")) {
			return data, nil
		}
		found = true
		return rewriteZip(data, func(inner string, content []byte) ([]byte, error) {
			switch {
			case serviceInterfaceModelEntry.MatchString(inner):
				return WithServiceInterfaceOperations(content, ops)
			case inner == "META-INF/MANIFEST.MF":
				return []byte(setManifestHeader(string(content), "Require-Capability", ServiceInterfaceRequireCapability(ops))), nil
			}
			return content, nil
		})
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("no service interface bundle inside the content")
	}
	return out, nil
}

// rewriteZip copies an archive, passing every file through change.
func rewriteZip(data []byte, change func(name string, content []byte) ([]byte, error)) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("not a ZIP archive: %w", err)
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, f := range r.File {
		content, err := readZipFile(f)
		if err != nil {
			return nil, err
		}
		if content, err = change(f.Name, content); err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
		fw, err := w.CreateHeader(&zip.FileHeader{Name: f.Name, Method: zip.Deflate, Modified: f.Modified})
		if err != nil {
			return nil, err
		}
		if _, err := fw.Write(content); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
