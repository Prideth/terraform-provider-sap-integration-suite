package cloudintegration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Content updates send only Name and ArtifactContent, and read the artifact
// back when SAP answers 200 without a body. On a tenant in September 2026 a
// message mapping update that also sent empty Id and PackageId failed with
// 500 "Update of PackageId and Id are not allowed", and flow updates
// answered 200 with no body.
func TestContentUpdates_BodyAndEmptyResponse(t *testing.T) {
	cases := []struct {
		name   string
		path   string
		update func(c *Client) (string, error)
	}{
		{"integration flow", "/api/v1/IntegrationDesigntimeArtifacts(Id='A',Version='active')", func(c *Client) (string, error) {
			f, err := c.UpdateIntegrationFlow(context.Background(), "A", "Name A", []byte("zip"))
			if err != nil {
				return "", err
			}
			return f.Version, nil
		}},
		{"message mapping", "/api/v1/MessageMappingDesigntimeArtifacts(Id='A',Version='active')", func(c *Client) (string, error) {
			m, err := c.UpdateMessageMapping(context.Background(), "A", "Name A", []byte("zip"))
			if err != nil {
				return "", err
			}
			return m.Version, nil
		}},
		{"script collection", "/api/v1/ScriptCollectionDesigntimeArtifacts(Id='A',Version='active')", func(c *Client) (string, error) {
			s, err := c.UpdateScriptCollection(context.Background(), "A", "Name A", []byte("zip"))
			if err != nil {
				return "", err
			}
			return s.Version, nil
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var sent map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tc.path {
					t.Errorf("path = %q, want %q", r.URL.Path, tc.path)
				}
				if r.Method == http.MethodPut {
					body, _ := io.ReadAll(r.Body)
					_ = json.Unmarshal(body, &sent)
					w.WriteHeader(http.StatusOK) // no body, as on the tenant
					return
				}
				_, _ = w.Write([]byte(`{"d": {"Id": "A", "Version": "1.0.2", "Name": "Name A", "PackageId": "P"}}`))
			}))
			defer server.Close()

			version, err := tc.update(New(http.DefaultClient, server.URL))
			if err != nil {
				t.Fatalf("update: %v", err)
			}
			if version != "1.0.2" {
				t.Errorf("version = %q, want 1.0.2 from the read-back", version)
			}
			if len(sent) != 2 || sent["Name"] != "Name A" || sent["ArtifactContent"] == nil {
				t.Errorf("update body = %v, want exactly Name and ArtifactContent", sent)
			}
		})
	}
}
