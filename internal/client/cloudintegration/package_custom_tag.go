package cloudintegration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

// Custom tag values of an integration package. SAP's Integration Content API
// specification documents reading them (GET IntegrationPackages('<id>')/
// CustomTags) and setting one (PUT IntegrationPackages('<id>')/$links/
// CustomTags('<name>') with {"Value": ...}, 202). A tenant probe (2026-10-17,
// PENDING #76) showed:
//
//   - A package lists every tag of the tenant's custom tag configuration, an
//     unset one with an empty value. $expand=CustomTags on the package
//     answers 501, and $filter on the list is ignored.
//   - A tag the configuration does not have answers 400 "CustomTag '<name>'
//     doesn't exist"; names are case-sensitive.
//   - Several values are separated by "|". SAP trims them, drops duplicates,
//     keeps the order and stores them joined with " | ".
//   - An empty value (or null) clears the tag.
//   - The package PUT the provider sends keeps the tag values.

// PackageCustomTag is one custom tag of a package with its values.
type PackageCustomTag struct {
	Name   string
	Values []string
}

// customTag is the wire form of a CustomTags entry.
type customTag struct {
	Name  string  `json:"Name"`
	Value *string `json:"Value"`
}

// customTagWrite is the body of the documented update request.
type customTagWrite struct {
	Value string `json:"Value"`
}

const (
	customTagsEntitySet = "CustomTags"
	// customTagValueSeparator separates several values of a tag; SAP stores
	// them joined with " | ".
	customTagValueSeparator = "|"
)

// SplitCustomTagValue turns a stored tag value into its values, the way SAP
// separates them. An empty value has none.
func SplitCustomTagValue(value string) []string {
	var values []string
	for _, v := range strings.Split(value, customTagValueSeparator) {
		if v = strings.TrimSpace(v); v != "" {
			values = append(values, v)
		}
	}
	return values
}

// ListPackageCustomTags reads every custom tag of the package, in the order
// SAP returns them. A missing package answers 404.
func (c *Client) ListPackageCustomTags(ctx context.Context, packageID string) ([]PackageCustomTag, error) {
	tags, err := v2.GetAllPages[customTag](ctx, c.odata,
		v2.BuildPath(integrationPackagesEntitySet, v2.KeyPredicate(packageID), "")+"/"+customTagsEntitySet)
	if err != nil {
		return nil, err
	}
	out := make([]PackageCustomTag, 0, len(tags))
	for _, t := range tags {
		value := ""
		if t.Value != nil {
			value = *t.Value
		}
		out = append(out, PackageCustomTag{Name: t.Name, Values: SplitCustomTagValue(value)})
	}
	return out, nil
}

// SetPackageCustomTag sets the values of one custom tag of the package. No
// values clear the tag. SAP answers 400 for a tag its configuration does not
// have.
func (c *Client) SetPackageCustomTag(ctx context.Context, packageID, name string, values []string) error {
	payload, err := json.Marshal(customTagWrite{Value: strings.Join(values, customTagValueSeparator)})
	if err != nil {
		return fmt.Errorf("cloudintegration: encoding custom tag: %w", err)
	}
	path := v2.BuildPath(integrationPackagesEntitySet, v2.KeyPredicate(packageID), "") +
		"/$links/" + customTagsEntitySet + v2.KeyPredicate(name)
	_, err = c.odata.Put(ctx, path, payload)
	return err
}
