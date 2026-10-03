package cloudintegration

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

const (
	messageTypeDesigntimeArtifactsEntitySet      = "MessageTypeDesigntimeArtifacts"
	faultMessageTypeDesigntimeArtifactsEntitySet = "FaultMessageTypeDesigntimeArtifacts"
)

// MessageTypeKind selects one of the two entity sets that share the shape of
// a message type: message types and fault message types.
type MessageTypeKind struct {
	entitySet     string
	saveAsVersion string
	DisplayName   string
}

var (
	// KindMessageType is MessageTypeDesigntimeArtifacts.
	KindMessageType = MessageTypeKind{messageTypeDesigntimeArtifactsEntitySet, "MessageTypeDesigntimeArtifactSaveAsVersion", "message type"}
	// KindFaultMessageType is FaultMessageTypeDesigntimeArtifacts.
	KindFaultMessageType = MessageTypeKind{faultMessageTypeDesigntimeArtifactsEntitySet, "FaultMessageTypeDesigntimeArtifactSaveAsVersion", "fault message type"}
)

// MessageType is the wire representation of a message type or a fault
// message type. SAP documents no request for either; the entity sets come
// from the Integration Content API's $metadata and were tested on a tenant
// (2026-10-03). SAP generates the content itself from DataTypeUsed: the
// schema includes the data type and defines an element of its type (a fault
// message type adds SAP's standard ExchangeFaultData). A read returns
// DataTypeUsed empty even when it took effect, so DataTypeID reads the
// reference from the generated bundle instead.
type MessageType struct {
	ID           string `json:"Id"`
	Name         string `json:"Name"`
	PackageID    string `json:"PackageId"`
	Version      string `json:"Version,omitempty"`
	Description  string `json:"Description,omitempty"`
	Namespace    string `json:"Namespace,omitempty"`
	DataTypeUsed string `json:"DataTypeUsed,omitempty"`
}

// messageTypeCreate is the create body; it carries no content.
type messageTypeCreate struct {
	ID           string `json:"Id"`
	Name         string `json:"Name"`
	PackageID    string `json:"PackageId"`
	Namespace    string `json:"Namespace"`
	Description  string `json:"Description,omitempty"`
	DataTypeUsed string `json:"DataTypeUsed,omitempty"`
}

// messageTypeUpdate is the update body. Name is not part of it: a tenant
// answered a PUT with Name with 400 "Update of Name are not allowed", the
// same PUT without it with 200 and regenerated content (2026-10-03).
type messageTypeUpdate struct {
	Description  string `json:"Description"`
	DataTypeUsed string `json:"DataTypeUsed,omitempty"`
}

func messageTypePath(kind MessageTypeKind, id string) (string, error) {
	key, err := designtimeArtifactKey(id, activeVersion)
	if err != nil {
		return "", err
	}
	return v2.BuildPath(kind.entitySet, key, ""), nil
}

// GetMessageType reads the active version of a message type or fault
// message type, with DataTypeUsed taken from its generated bundle.
func (c *Client) GetMessageType(ctx context.Context, kind MessageTypeKind, id string) (*MessageType, error) {
	path, err := messageTypePath(kind, id)
	if err != nil {
		return nil, err
	}
	body, err := c.odata.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var mt MessageType
	if err := v2.DecodeEntity(body, &mt); err != nil {
		return nil, err
	}
	content, err := c.odata.Get(ctx, path+"/$value")
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: reading the %s bundle: %w", kind.DisplayName, err)
	}
	used, err := dataTypeUsedInBundle(content)
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: reading the %s bundle: %w", kind.DisplayName, err)
	}
	mt.DataTypeUsed = used
	return &mt, nil
}

// dataTypeUsedInBundle returns the data type a generated message type
// bundle refers to (dtUsedinMT in additionalAttributes.json), or "" for
// none.
func dataTypeUsedInBundle(content []byte) (string, error) {
	r, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("not a ZIP archive: %w", err)
	}
	for _, f := range r.File {
		if f.Name != "src/main/resources/additionalAttributes.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		data, err := io.ReadAll(io.LimitReader(rc, 1<<20))
		_ = rc.Close()
		if err != nil {
			return "", err
		}
		var attrs struct {
			DataTypeUsed string `json:"dtUsedinMT"`
		}
		if err := json.Unmarshal(data, &attrs); err != nil {
			return "", fmt.Errorf("additionalAttributes.json: %w", err)
		}
		return attrs.DataTypeUsed, nil
	}
	return "", nil
}

// CreateMessageType creates a message type or fault message type without
// content and reads it back.
func (c *Client) CreateMessageType(ctx context.Context, kind MessageTypeKind, mt MessageType) (*MessageType, error) {
	payload, err := json.Marshal(messageTypeCreate{
		ID: mt.ID, Name: mt.Name, PackageID: mt.PackageID, Namespace: mt.Namespace,
		Description: mt.Description, DataTypeUsed: mt.DataTypeUsed,
	})
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: encoding %s: %w", kind.DisplayName, err)
	}
	if _, err := c.odata.Post(ctx, kind.entitySet, payload); err != nil {
		return nil, err
	}
	return c.GetMessageType(ctx, kind, mt.ID)
}

// UpdateMessageType changes the description and the data type used, and
// reads the message type back.
func (c *Client) UpdateMessageType(ctx context.Context, kind MessageTypeKind, id, description, dataTypeUsed string) (*MessageType, error) {
	path, err := messageTypePath(kind, id)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(messageTypeUpdate{Description: description, DataTypeUsed: dataTypeUsed})
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: encoding %s update: %w", kind.DisplayName, err)
	}
	if _, err := c.odata.Put(ctx, path, payload); err != nil {
		return nil, err
	}
	return c.GetMessageType(ctx, kind, id)
}

// SaveMessageTypeAsVersion saves the current content under an explicit
// version.
func (c *Client) SaveMessageTypeAsVersion(ctx context.Context, kind MessageTypeKind, id, version string) (*MessageType, error) {
	return saveAsVersion(ctx, c, kind.saveAsVersion, id, version,
		func() (*MessageType, error) { return c.GetMessageType(ctx, kind, id) })
}

// DeleteMessageType deletes a message type or fault message type.
func (c *Client) DeleteMessageType(ctx context.Context, kind MessageTypeKind, id string) error {
	path, err := messageTypePath(kind, id)
	if err != nil {
		return err
	}
	return c.odata.Delete(ctx, path)
}
