package v2

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	sapthttp "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/http"
)

// doer is the minimal interface the shared retrying HTTP client satisfies;
// tests can substitute their own implementation.
type doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is a small OData V2 request helper: it joins a base URL with an
// entity path, sets the headers SAP's OData V2 services expect, and turns
// non-2xx responses into a *apierror.Error via ParseError.
type Client struct {
	http    doer
	baseURL string
}

// New builds an OData V2 client rooted at baseURL (for example
// "https://tenant.example/api/v1"). http is typically a
// *sapthttp.Client wrapping an OAuth2-authenticated transport.
func New(httpClient doer, baseURL string) *Client {
	return &Client{http: httpClient, baseURL: baseURL}
}

// Get issues a GET request against path (an entity set, optionally with a
// key predicate and query string) and returns the raw response body on
// success.
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return c.do(ctx, http.MethodGet, path, nil)
}

// Post issues a POST request with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.do(ctx, http.MethodPost, path, body)
}

// Put issues a PUT request with a JSON body. PUT replaces the entire entity
// with the fields provided; any field the caller omits may be reset to its
// default by the server. For a partial update of a few fields, use Patch
// instead.
func (c *Client) Put(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.do(ctx, http.MethodPut, path, body)
}

// PutRaw issues a PUT request with a raw, non-JSON body and an explicit
// Content-Type — for the handful of OData V2 "$value" raw-media-stream
// endpoints that accept the resource's literal bytes (for example a PEM
// certificate) rather than a JSON-wrapped entity. Unlike Put, this never
// sets "Content-Type: application/json" or "Accept: application/json",
// since the caller's contentType is what the target endpoint actually
// expects.
func (c *Client) PutRaw(ctx context.Context, path, contentType string, body []byte) ([]byte, error) {
	return c.doRaw(ctx, http.MethodPut, path, contentType, body)
}

// PostRaw issues a POST request with a raw, non-JSON body and an explicit
// Content-Type, for upload endpoints that take a file (for example an API
// proxy ZIP bundle) instead of a JSON entity.
func (c *Client) PostRaw(ctx context.Context, path, contentType string, body []byte) ([]byte, error) {
	return c.doRaw(ctx, http.MethodPost, path, contentType, body)
}

// Patch issues a PATCH request with a JSON body. SAP's OData V2 services on
// Cloud Foundry/BTP accept PATCH as the modern equivalent of the legacy
// OData MERGE verb: only the fields present in body are changed, and every
// other field on the entity is left untouched. This is almost always the
// right choice for a Terraform resource's Update, since Terraform only
// tracks the fields declared in its schema and must not clobber the rest of
// the entity.
func (c *Client) Patch(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.do(ctx, http.MethodPatch, path, body)
}

// Delete issues a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	_, err := c.do(ctx, http.MethodDelete, path, nil)
	return err
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	return c.send(ctx, method, path, "application/json", true, body)
}

// doRaw is do's counterpart for a non-JSON request body: it sends
// contentType as-is instead of "application/json", and never sets an
// "Accept: application/json" header (the response is expected to be a raw
// byte stream too, not a JSON-wrapped entity).
func (c *Client) doRaw(ctx context.Context, method, path, contentType string, body []byte) ([]byte, error) {
	return c.send(ctx, method, path, contentType, false, body)
}

func (c *Client) send(ctx context.Context, method, path, contentType string, acceptJSON bool, body []byte) ([]byte, error) {
	// path is normally relative to baseURL, but a server-driven paging
	// "__next" link (see GetAllPages) is already a complete absolute URL
	// that must be followed exactly as SAP returned it, not rejoined with
	// baseURL.
	url := path
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		url = c.baseURL + "/" + path
	}

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("odata: building request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", contentType)
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(body)), nil
		}
	}
	if acceptJSON {
		req.Header.Set("Accept", "application/json")
	}

	resp, err := c.http.Do(req) //nolint:bodyclose // resp.Body is always closed inside sapthttp.ReadLimited below
	if err != nil {
		return nil, fmt.Errorf("odata: request failed: %w", err)
	}

	respBody, err := sapthttp.ReadLimited(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, ParseError(resp.StatusCode, respBody)
	}

	return respBody, nil
}
