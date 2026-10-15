package cloudintegration

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

// Value mapping entries: the source and target value pairs of one agency
// pair (source agency and identifier, target agency and identifier) of a
// value mapping. SAP's Integration Content API specification documents
// reading them (ValMapSchema, ValMaps, DefaultValMaps) and the function
// imports UpsertValMaps, UpdateDefaultValMap and DeleteValMaps. A tenant
// probe (2026-10-15, PENDING #75) showed:
//
//   - UpsertValMaps (POST, no body) answers 202 and adds an entry with a new
//     GUID; the entry is at once in the design-time content, and the
//     artifact's version becomes Draft. The same source and target value
//     again answer 400 "Source and Target value already exist".
//   - With ValMapId, a parameter only the $metadata declares, the upsert
//     changes that entry in place, source and target value alike.
//   - An unknown agency pair answers 404: pairs come only from the value
//     mapping's content.
//   - No request deletes a single entry. DeleteValMaps removes the whole
//     agency pair, which the API cannot create again.
//
// Requests address the artifact as Version='active', which the probe used
// throughout, also once the artifact was a draft.

// ValueMappingAgencyPair names one direction of a value mapping's agency
// pair.
type ValueMappingAgencyPair struct {
	SourceAgency     string
	SourceIdentifier string
	TargetAgency     string
	TargetIdentifier string
}

// ValueMappingEntry is one source and target value pair.
type ValueMappingEntry struct {
	ID          string
	SourceValue string
	TargetValue string
}

// valMap is the wire form of ValMaps and DefaultValMaps entries.
type valMap struct {
	ID    string `json:"Id"`
	Value struct {
		SrcValue string `json:"SrcValue"`
		TgtValue string `json:"TgtValue"`
	} `json:"Value"`
}

// valMapSchema is the wire form of a ValMapSchema entry (an agency pair in
// one direction).
type valMapSchema struct {
	SrcAgency string  `json:"SrcAgency"`
	SrcID     string  `json:"SrcId"`
	TgtAgency string  `json:"TgtAgency"`
	TgtID     string  `json:"TgtId"`
	State     *string `json:"State"`
}

const (
	valMapSchemaNavigation = "ValMapSchema"
	upsertValMapsFunction  = "UpsertValMaps"
)

// schemaPath is the ValMapSchema entity of the pair under the value
// mapping.
func (p ValueMappingAgencyPair) schemaPath(mappingID string) (string, error) {
	key, err := designtimeArtifactKey(mappingID, activeVersion)
	if err != nil {
		return "", err
	}
	pair, err := v2.CompositeKeyPredicate("SrcAgency", p.SourceAgency, "SrcId", p.SourceIdentifier,
		"TgtAgency", p.TargetAgency, "TgtId", p.TargetIdentifier)
	if err != nil {
		return "", err
	}
	return v2.BuildPath(valueMappingDesigntimeArtifactsEntitySet, key, "") + "/" + valMapSchemaNavigation + pair, nil
}

// functionArgs renders the parameters UpsertValMaps and its siblings share.
func (p ValueMappingAgencyPair) functionArgs(mappingID string) string {
	return "Id=" + quotedParam(mappingID) + "&Version=" + quotedParam(activeVersion) +
		"&SrcAgency=" + quotedParam(p.SourceAgency) + "&SrcId=" + quotedParam(p.SourceIdentifier) +
		"&TgtAgency=" + quotedParam(p.TargetAgency) + "&TgtId=" + quotedParam(p.TargetIdentifier)
}

// quotedParam renders a string parameter of a function import: an OData
// string literal (quotes doubled), URL-encoded, so that values with spaces,
// ampersands or quotes reach SAP unchanged (probe #75).
func quotedParam(s string) string {
	return strings.ReplaceAll(url.QueryEscape("'"+v2.EscapeLiteral(s)+"'"), "+", "%20")
}

// HasValueMappingAgencyPair reports whether the value mapping has the agency
// pair in this direction.
func (c *Client) HasValueMappingAgencyPair(ctx context.Context, mappingID string, pair ValueMappingAgencyPair) (bool, error) {
	key, err := designtimeArtifactKey(mappingID, activeVersion)
	if err != nil {
		return false, err
	}
	schemas, err := v2.GetAllPages[valMapSchema](ctx, c.odata,
		v2.BuildPath(valueMappingDesigntimeArtifactsEntitySet, key, "")+"/"+valMapSchemaNavigation)
	if err != nil {
		return false, err
	}
	for _, s := range schemas {
		if s.SrcAgency == pair.SourceAgency && s.SrcID == pair.SourceIdentifier &&
			s.TgtAgency == pair.TargetAgency && s.TgtID == pair.TargetIdentifier {
			return true, nil
		}
	}
	return false, nil
}

// ListValueMappingEntries lists the entries of the agency pair. A pair the
// value mapping does not have answers 404.
func (c *Client) ListValueMappingEntries(ctx context.Context, mappingID string, pair ValueMappingAgencyPair) ([]ValueMappingEntry, error) {
	return c.listValMaps(ctx, mappingID, pair, "ValMaps")
}

// ListDefaultValueMappingEntries lists the entries the agency pair uses by
// default where a source value has more than one target.
func (c *Client) ListDefaultValueMappingEntries(ctx context.Context, mappingID string, pair ValueMappingAgencyPair) ([]ValueMappingEntry, error) {
	return c.listValMaps(ctx, mappingID, pair, "DefaultValMaps")
}

func (c *Client) listValMaps(ctx context.Context, mappingID string, pair ValueMappingAgencyPair, navigation string) ([]ValueMappingEntry, error) {
	path, err := pair.schemaPath(mappingID)
	if err != nil {
		return nil, err
	}
	maps, err := v2.GetAllPages[valMap](ctx, c.odata, path+"/"+navigation)
	if err != nil {
		return nil, err
	}
	entries := make([]ValueMappingEntry, 0, len(maps))
	for _, m := range maps {
		entries = append(entries, ValueMappingEntry{ID: m.ID, SourceValue: m.Value.SrcValue, TargetValue: m.Value.TgtValue})
	}
	return entries, nil
}

// AddValueMappingEntry adds a source and target value pair to the agency
// pair through UpsertValMaps and returns the entry SAP stored. SAP answers
// 400 when the pair exists already and 404 when the agency pair does not.
func (c *Client) AddValueMappingEntry(ctx context.Context, mappingID string, pair ValueMappingAgencyPair, sourceValue, targetValue string) (*ValueMappingEntry, error) {
	if err := c.upsertValMap(ctx, mappingID, pair, "", sourceValue, targetValue); err != nil {
		return nil, err
	}
	// The function import answers 202 without the entry; find it by its values.
	entries, err := c.ListValueMappingEntries(ctx, mappingID, pair)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.SourceValue == sourceValue && e.TargetValue == targetValue {
			return &e, nil
		}
	}
	return nil, fmt.Errorf("cloudintegration: SAP accepted the value mapping entry %q -> %q, but it is not among the entries", sourceValue, targetValue)
}

// ChangeValueMappingEntry changes the source and target value of an
// existing entry in place: UpsertValMaps with ValMapId, which only the
// $metadata declares.
func (c *Client) ChangeValueMappingEntry(ctx context.Context, mappingID string, pair ValueMappingAgencyPair, entryID, sourceValue, targetValue string) error {
	return c.upsertValMap(ctx, mappingID, pair, entryID, sourceValue, targetValue)
}

func (c *Client) upsertValMap(ctx context.Context, mappingID string, pair ValueMappingAgencyPair, entryID, sourceValue, targetValue string) error {
	path := upsertValMapsFunction + "?" + pair.functionArgs(mappingID)
	if entryID != "" {
		path += "&ValMapId=" + quotedParam(entryID)
	}
	path += "&SrcValue=" + quotedParam(sourceValue) + "&TgtValue=" + quotedParam(targetValue) + "&IsConfigured=false"
	_, err := c.odata.Post(ctx, path, nil)
	return err
}
