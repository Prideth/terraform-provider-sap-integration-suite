package apimanagementclassic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

const (
	virtualHostsEntitySet        = "VirtualHosts"
	virtualHostRequestsEntitySet = "VirtualHostRequests"

	// VirtualHostAliasMaxLength is the documented limit of a virtual host
	// alias; a tenant answered a longer one with 400
	// VHR_VIRTUALHOST_ALIAS_LENGTH_GREATER_THAN_63_CHARS.
	VirtualHostAliasMaxLength = 63

	// errVirtualHostUnknown is SAP's answer to an update or delete of a
	// virtual host ID it does not know, also for one deleted already.
	errVirtualHostUnknown = "VHR_NO_COMPLETED_RECORD_FOUND"
)

// VirtualHost is an entity of Management.svc/VirtualHosts: a host name
// under which the API portal exposes API proxies. id equals the
// virtualHostId of the request that created it; name repeats the id for
// hosts created through VirtualHostRequests. virtual_host is the full host
// name, alias first ("prod-apis.<tenant domain>").
type VirtualHost struct {
	ID                  string `json:"id"`
	Name                string `json:"name,omitempty"`
	HostName            string `json:"virtual_host,omitempty"`
	Port                int    `json:"virtual_port,omitempty"`
	IsDefault           bool   `json:"isDefault,omitempty"`
	IsSSL               bool   `json:"isSSL,omitempty"`
	IsForCustomDomain   bool   `json:"isForCustomDomain,omitempty"`
	IsClientAuthEnabled bool   `json:"isClientAuthEnabled,omitempty"`
	KeyStoreName        string `json:"keyStoreName,omitempty"`
	KeyStoreAlias       string `json:"keyStoreAlias,omitempty"`
	TrustStore          string `json:"trustStore,omitempty"`
	ProjectPath         string `json:"projectPath,omitempty"`
}

// Alias is the first label of the host name: the alias a default domain
// virtual host was created with.
func (h VirtualHost) Alias() string {
	alias, _, _ := strings.Cut(h.HostName, ".")
	return alias
}

// virtualHostRequestWire is the body of a POST to
// Configuration.svc/VirtualHostRequests, as SAP Help documents it in
// "Configuring a Default Domain for a Virtual Host": one request type for
// create, update and delete, told apart by operation. Configuration.svc
// does not serve its $metadata to the apiportal-apiaccess keys (403 for
// both roles on a tenant, October 2026), so the field names come from the
// documented requests and the tenant's answers, not from a schema.
type virtualHostRequestWire struct {
	AccountID                   string `json:"accountId,omitempty"`
	VirtualHostURL              string `json:"virtualHostUrl,omitempty"`
	IsDefaultVirtualHostRequest *bool  `json:"isDefaultVirtualHostRequest,omitempty"`
	Operation                   string `json:"operation"`
	VirtualHostID               string `json:"virtualHostId,omitempty"`
}

// VirtualHostRequest is SAP's answer to a request: a record of the request
// itself (ID), naming the virtual host it acted on (VirtualHostID). On a
// tenant every create and update answered COMPLETE at once; a delete
// answered without an allocation status.
type VirtualHostRequest struct {
	ID               string `json:"id"`
	VirtualHostID    string `json:"virtualHostId"`
	VirtualHostURL   string `json:"virtualHostUrl"`
	AllocationStatus string `json:"allocationStatus"`
	AllocatedPort    int    `json:"allocatedPort"`
	Operation        string `json:"operation"`
}

// ListVirtualHosts reads every virtual host of the API portal. A key with
// only APIManagement.SelfService.Administrator may read the list but not a
// single entity (VirtualHosts('<id>') answered 403 on a tenant), so a
// single host is always looked up in the list.
func (c *Client) ListVirtualHosts(ctx context.Context) ([]VirtualHost, error) {
	return v2.GetAllPages[VirtualHost](ctx, c.odata, virtualHostsEntitySet)
}

// FindVirtualHost returns the virtual host with this id, or nil when the
// API portal has none.
func (c *Client) FindVirtualHost(ctx context.Context, id string) (*VirtualHost, error) {
	hosts, err := c.ListVirtualHosts(ctx)
	if err != nil {
		return nil, err
	}
	for i := range hosts {
		if hosts[i].ID == id {
			return &hosts[i], nil
		}
	}
	return nil, nil
}

// WaitForVirtualHost reads the list until ready accepts the host with this
// id (nil while the list has none), bounded by ctx. On a tenant every
// request was answered COMPLETE and the list showed the change at once;
// the wait covers a slower allocation SAP does not document.
func (c *Client) WaitForVirtualHost(ctx context.Context, id string, ready func(*VirtualHost) bool) (*VirtualHost, error) {
	var last *VirtualHost
	err := pollUntilVisible(ctx, func(ctx context.Context) (bool, error) {
		found, err := c.FindVirtualHost(ctx, id)
		if err != nil {
			return false, err
		}
		last = found
		return ready(found), nil
	})
	return last, err
}

// CreateVirtualHost requests a virtual host with this alias on the
// tenant's default domain, not as the default host: the documented sample
// body. accountID is the subdomain of the subaccount.
func (c *Client) CreateVirtualHost(ctx context.Context, accountID, alias string) (*VirtualHostRequest, error) {
	isDefault := false
	return c.requestVirtualHost(ctx, virtualHostRequestWire{
		AccountID:                   accountID,
		VirtualHostURL:              alias,
		IsDefaultVirtualHostRequest: &isDefault,
		Operation:                   "CREATE",
	})
}

// UpdateVirtualHost changes the alias of a default domain virtual host.
// The body is the documented one; isDefault is sent as it is, so pass the
// host's current value to leave the default host unchanged.
func (c *Client) UpdateVirtualHost(ctx context.Context, accountID, id, alias string, isDefault bool) (*VirtualHostRequest, error) {
	return c.requestVirtualHost(ctx, virtualHostRequestWire{
		AccountID:                   accountID,
		VirtualHostURL:              alias,
		IsDefaultVirtualHostRequest: &isDefault,
		Operation:                   "UPDATE",
		VirtualHostID:               id,
	})
}

// DeleteVirtualHost deletes a virtual host. SAP refuses it while the host
// is the default one or while API proxies, drafts or revisions refer to
// it. A host deleted already fails with an error IsUnknownVirtualHost
// recognizes.
func (c *Client) DeleteVirtualHost(ctx context.Context, id string) error {
	_, err := c.requestVirtualHost(ctx, virtualHostRequestWire{Operation: "DELETE", VirtualHostID: id})
	return err
}

// IsUnknownVirtualHost reports whether err is SAP's answer to an update or
// delete of a virtual host ID it does not know: 400
// VHR_NO_COMPLETED_RECORD_FOUND, also for a host deleted before.
func IsUnknownVirtualHost(err error) bool {
	var apiErr *apierror.Error
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusBadRequest && apiErr.Code == errVirtualHostUnknown
}

func (c *Client) requestVirtualHost(ctx context.Context, request virtualHostRequestWire) (*VirtualHostRequest, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("apimanagementclassic: encoding virtual host request: %w", err)
	}
	body, err := c.configuration.Post(ctx, virtualHostRequestsEntitySet, payload)
	if err != nil {
		return nil, err
	}
	var answer VirtualHostRequest
	if err := v2.DecodeEntity(body, &answer); err != nil {
		return nil, err
	}
	return &answer, nil
}
