package apimanagementclassic

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

const apiProxiesEntitySet = "APIProxies"

// transportImportPath is the import call of SAP's API Management Client SDK
// 3.0.6 (StandardAPIProxyClient.importAPIProxy), relative to Transport.svc,
// sent exactly as the SDK sends it: the raw bundle ZIP as
// application/octet-stream. The odd query ("name" carrying
// "?virtualhost=default") is the SDK's own literal; the bundle names the
// proxy, and SAP imports it onto the default virtual host.
const transportImportPath = "APIProxies?name=?virtualhost=default"

// APIProxy is the wire representation of a Management.svc APIProxies entity,
// limited to the properties the provider shows. The proxy's endpoints,
// policies and resources are navigation properties and come from the bundle.
type APIProxy struct {
	Name         string `json:"name"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Version      string `json:"version"`
	ServiceCode  string `json:"service_code"`
	ProviderName string `json:"provider_name"`
	State        string `json:"state"`
	StatusCode   string `json:"status_code"`
	IsPublished  bool   `json:"isPublished"`
	IsVersioned  bool   `json:"isVersioned"`
}

// ImportAPIProxy uploads an API proxy bundle through the Transport API, as
// SAP's Client SDK does. The proxy's name comes from the bundle. SAP's
// documentation says a proxy imported this way is deployed by default.
func (c *Client) ImportAPIProxy(ctx context.Context, bundle []byte) error {
	_, err := c.transport.PostRaw(ctx, transportImportPath, "application/octet-stream", bundle)
	return err
}

// WaitForAPIProxy reads a proxy until Management.svc returns it, bounded by
// ctx: the API portal answers reads of freshly written objects from a cache
// (SAP documents this for API providers), so an import may not be visible at
// once.
func (c *Client) WaitForAPIProxy(ctx context.Context, name string) (*APIProxy, error) {
	var proxy *APIProxy
	err := pollUntilVisible(ctx, func(ctx context.Context) (bool, error) {
		found, err := c.GetAPIProxy(ctx, name)
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		proxy = found
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("apimanagementclassic: API proxy %q was imported but did not become readable: %w", name, err)
	}
	return proxy, nil
}

// GetAPIProxy reads a single API proxy by name.
func (c *Client) GetAPIProxy(ctx context.Context, name string) (*APIProxy, error) {
	body, err := c.odata.Get(ctx, v2.BuildPath(apiProxiesEntitySet, v2.KeyPredicate(name), ""))
	if err != nil {
		return nil, err
	}
	var proxy APIProxy
	if err := v2.DecodeEntity(body, &proxy); err != nil {
		return nil, err
	}
	return &proxy, nil
}

// DeleteAPIProxy deletes an API proxy by name, which also removes its
// deployment.
func (c *Client) DeleteAPIProxy(ctx context.Context, name string) error {
	return c.odata.Delete(ctx, v2.BuildPath(apiProxiesEntitySet, v2.KeyPredicate(name), ""))
}

// maxBundleEntry bounds the proxy descriptor read from a bundle.
const maxBundleEntry = 1 << 20

// APIProxyBundleName returns the proxy name an API proxy bundle declares: the
// <name> of the single descriptor APIProxy/<name>.xml at the top of the
// APIProxy folder. The import names the proxy after it, so the provider
// checks it against the configured name before uploading.
func APIProxyBundleName(bundle []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(bundle), int64(len(bundle)))
	if err != nil {
		return "", fmt.Errorf("reading the API proxy bundle: %w", err)
	}
	var descriptors []*zip.File
	for _, f := range zr.File {
		dir, file := path.Split(f.Name)
		if dir == "APIProxy/" && strings.HasSuffix(file, ".xml") {
			descriptors = append(descriptors, f)
		}
	}
	if len(descriptors) != 1 {
		return "", fmt.Errorf("an API proxy bundle has exactly one APIProxy/<name>.xml descriptor, found %d", len(descriptors))
	}
	rc, err := descriptors[0].Open()
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", descriptors[0].Name, err)
	}
	defer func() { _ = rc.Close() }()
	data, err := io.ReadAll(io.LimitReader(rc, maxBundleEntry))
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", descriptors[0].Name, err)
	}
	var descriptor struct {
		XMLName xml.Name
		Name    string `xml:"name"`
	}
	if err := xml.Unmarshal(data, &descriptor); err != nil {
		return "", fmt.Errorf("parsing %s: %w", descriptors[0].Name, err)
	}
	if descriptor.XMLName.Local != "APIProxy" || descriptor.Name == "" {
		return "", fmt.Errorf("%s is not an API proxy descriptor with a name", descriptors[0].Name)
	}
	return descriptor.Name, nil
}
