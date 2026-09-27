package apidiscovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/auth"
)

// maxDocumentBytes bounds a $metadata or specification download.
const maxDocumentBytes = 32 << 20

// Result is what a live fetch of one service returned.
type Result struct {
	Service *apimeta.Service
	// Raw is the $metadata document as received, for callers that keep a
	// local copy (never written to the repository).
	Raw []byte
	// ServiceDocumentIssues lists disagreements between the service
	// document and $metadata.
	ServiceDocumentIssues []string
}

// Fetch reads a service's $metadata (and, for OData, its service document)
// with the service's OAuth credentials from the environment. Errors never
// contain the service root or credentials, only the service ID and the
// HTTP status.
func Fetch(ctx context.Context, svc Service) (*Result, error) {
	if svc.SpecificationOnly() {
		return nil, fmt.Errorf("%s: contract comes only from the official specification (%s)", svc.ID, svc.Specification)
	}
	root, creds, missing := svc.Resolve(os.Getenv)
	if len(missing) > 0 {
		return nil, fmt.Errorf("%s: not configured, missing %s", svc.ID, strings.Join(missing, ", "))
	}
	base := &http.Client{Timeout: 60 * time.Second}
	client, _, err := auth.Config{TokenURL: creds.TokenURL, ClientID: creds.ClientID, ClientSecret: creds.ClientSecret}.HTTPClient(ctx, base)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", svc.ID, err)
	}

	accept := "application/xml"
	if svc.Protocol == apimeta.ProtocolOpenAPI {
		accept = "application/json, application/yaml"
	}
	raw, err := get(ctx, client, root+svc.documentPath(), accept)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", svc.ID, svc.documentPath(), err)
	}
	var parsed *apimeta.Service
	if svc.Protocol == apimeta.ProtocolOpenAPI {
		parsed, err = apimeta.ParseOpenAPI(svc.ID, raw)
	} else {
		parsed, err = apimeta.ParseEDMX(svc.ID, raw)
	}
	if err != nil {
		return nil, err
	}
	parsed.Source = "live " + svc.documentPath()

	res := &Result{Service: parsed, Raw: raw}
	if svc.Protocol != apimeta.ProtocolOpenAPI {
		doc, err := get(ctx, client, root+"/", "application/json")
		if err == nil {
			parsed.ServiceDocument = serviceDocumentCollections(doc)
			apimeta.Normalize(parsed)
			res.ServiceDocumentIssues = CompareServiceDocument(parsed)
		} else {
			res.ServiceDocumentIssues = []string{"service document not readable: " + err.Error()}
		}
	}
	return res, nil
}

func get(ctx context.Context, client *http.Client, url, accept string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", accept)
	resp, err := client.Do(req) //nolint:gosec // G704: documented service root from the operator's own configuration
	if err != nil {
		// The error text of net/http contains the URL; keep only the cause.
		return nil, fmt.Errorf("request failed: %s", apimeta.RedactURLs(err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDocumentBytes))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return body, nil
}

// serviceDocumentCollections reads the collection names of an OData V2
// ({"d": {"EntitySets": [...]}}) or V4 ({"value": [{"name": ...}]}) service
// document.
func serviceDocumentCollections(body []byte) []string {
	var v2 struct {
		D struct {
			EntitySets []string `json:"EntitySets"`
		} `json:"d"`
	}
	if json.Unmarshal(body, &v2) == nil && len(v2.D.EntitySets) > 0 {
		return v2.D.EntitySets
	}
	var v4 struct {
		Value []struct {
			Name string `json:"name"`
			Kind string `json:"kind"`
		} `json:"value"`
	}
	if json.Unmarshal(body, &v4) == nil {
		var out []string
		for _, c := range v4.Value {
			if c.Kind == "" || c.Kind == "EntitySet" || c.Kind == "Singleton" {
				out = append(out, c.Name)
			}
		}
		return out
	}
	return nil
}

// CompareServiceDocument lists collections the service document names but
// $metadata does not declare, and entity sets $metadata declares that the
// service document leaves out.
func CompareServiceDocument(s *apimeta.Service) []string {
	if len(s.ServiceDocument) == 0 {
		return nil
	}
	declared := map[string]bool{}
	for _, es := range s.EntitySets {
		declared[es.Name] = true
	}
	for _, sg := range s.Singletons {
		declared[sg.Name] = true
	}
	listed := map[string]bool{}
	var issues []string
	for _, name := range s.ServiceDocument {
		listed[name] = true
		if !declared[name] {
			issues = append(issues, "service document lists "+name+", which $metadata does not declare")
		}
	}
	for name := range declared {
		if !listed[name] {
			issues = append(issues, "$metadata declares "+name+", which the service document does not list")
		}
	}
	sort.Strings(issues)
	return issues
}
