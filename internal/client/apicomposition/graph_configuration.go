package apicomposition

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
)

const graphConfigurationResource = "GraphConfiguration"

// Status values of a business data graph during design-time processing, as
// SAP's "Configuration API Specification and Usage" page describes them: a
// new or changed graph is PROCESSING, and ends in DEPLOYMENT_INITIATED on
// success or FAILED on error.
const (
	StatusProcessing          = "PROCESSING"
	StatusDeploymentInitiated = "DEPLOYMENT_INITIATED"
	StatusFailed              = "FAILED"
)

// DataSourceService is one BTP destination a data source reads from. Path
// is appended to the destination URL, so one root destination can serve
// several OData services of the same system.
type DataSourceService struct {
	DestinationName string `json:"destinationName"`
	Path            string `json:"path"`
}

// DataSource is one business system of the landscape. Its name becomes the
// key qualifier in key-based references; Namespace is needed only for
// custom services SAP does not know.
type DataSource struct {
	Name      string              `json:"name"`
	Namespace string              `json:"namespace,omitempty"`
	Services  []DataSourceService `json:"services"`
}

// LocatingCue declares a cue that locating rules can reference. SAP's
// configuration file page documents a cue as {"name", "description"}.
type LocatingCue struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// KeyFormatStrategy translates key values between two data sources. SAP
// supports only the strategy "format", with an RE2 match and a replace
// pattern.
type KeyFormatStrategy struct {
	Name    string `json:"name"`
	Match   string `json:"match"`
	Replace string `json:"replace"`
}

// KeyMappingSide is one side of a key mapping: the entity, the data source
// it lives in, and the key attribute (SAP supports exactly one).
type KeyMappingSide struct {
	DataSource string             `json:"dataSource"`
	EntityName string             `json:"entityName"`
	Attributes []string           `json:"attributes"`
	Strategy   *KeyFormatStrategy `json:"strategy,omitempty"`
}

// KeyMapping links a foreign key in one data source to the key of the same
// entity in another, so references can be followed across systems that use
// different keys.
type KeyMapping struct {
	// Cues scope the mapping to requests with these cues. SAP's pages
	// describe cue-scoped key mappings without naming the property; the
	// name comes from the $metadata (InterSourceForeignKey.cues).
	Cues       []string       `json:"cues,omitempty"`
	ForeignKey KeyMappingSide `json:"foreignKey"`
	References KeyMappingSide `json:"references"`
}

// LocatingRule names the leading data source for one entity, or for a
// namespace through a trailing wildcard. A specific name overrides a
// wildcard and a rule with cues overrides the default rule, so the order of
// rules carries no meaning.
type LocatingRule struct {
	Name         string   `json:"name"`
	Leading      string   `json:"leading"`
	Local        []string `json:"local,omitempty"`
	Cues         []string `json:"cues,omitempty"`
	SourceEntity string   `json:"sourceEntity,omitempty"`
}

// LocatingPolicy is a single object with cues, key mappings and rules. The
// property table on SAP's API page calls it "Array of locating policies",
// but both the API page's Create example and the configuration file page
// show an object, and the object is what this client sends.
type LocatingPolicy struct {
	// Description is in the $metadata only.
	Description string         `json:"description,omitempty"`
	Cues        []LocatingCue  `json:"cues,omitempty"`
	KeyMapping  []KeyMapping   `json:"keyMapping,omitempty"`
	Rules       []LocatingRule `json:"rules,omitempty"`
}

// ExtensionRef names an extension applied to a graph. The $metadata types
// extensions as objects with a name (ExtensionRef), not as plain strings.
type ExtensionRef struct {
	Name string `json:"name"`
}

// ExtensionNames returns the names of the graph's extensions, or nil when
// it has none.
func (g *GraphConfiguration) ExtensionNames() []string {
	var names []string
	for _, e := range g.Extensions {
		names = append(names, e.Name)
	}
	return names
}

// GraphConfiguration is a business data graph as the Configuration API
// returns it. EffectiveGraphModelVersion, StatusDetails, LogMessages and
// Status are read-only. The $metadata types a log message as level,
// message and code, but each one is kept as the raw JSON SAP returned, as
// before the $metadata was known. Extensions cannot be managed through
// this API, according to SAP, and are only read.
type GraphConfiguration struct {
	BusinessDataGraphIdentifier string            `json:"businessDataGraphIdentifier"`
	SchemaVersion               string            `json:"schemaVersion,omitempty"`
	Description                 string            `json:"description,omitempty"`
	GraphModelVersion           string            `json:"graphModelVersion,omitempty"`
	EffectiveGraphModelVersion  string            `json:"effectiveGraphModelVersion,omitempty"`
	ODataContainment            *bool             `json:"odataContainment,omitempty"`
	Exclude                     []string          `json:"exclude,omitempty"`
	DataSources                 []DataSource      `json:"dataSources"`
	LocatingPolicy              LocatingPolicy    `json:"locatingPolicy"`
	Extensions                  []ExtensionRef    `json:"extensions,omitempty"`
	StatusDetails               string            `json:"statusDetails,omitempty"`
	LogMessages                 []json.RawMessage `json:"logMessages,omitempty"`
	Status                      string            `json:"status,omitempty"`
	// Deleted is in the $metadata (default false) but not in SAP's
	// documentation. A graph marked deleted is treated as gone.
	Deleted bool `json:"deleted,omitempty"`
}

// GraphConfigurationInput is the writable part of a business data graph.
// It is the body of both Create and Update. Exclude is always sent, as an
// empty list when there is nothing to exclude, so that an Update clears a
// list that was removed from the configuration. Properties it leaves out,
// such as extensions, are not touched by an Update.
type GraphConfigurationInput struct {
	BusinessDataGraphIdentifier string `json:"businessDataGraphIdentifier"`
	SchemaVersion               string `json:"schemaVersion,omitempty"`
	// Description and ODataContainment are in the $metadata only; SAP's
	// pages describe OData containment without its property name. They are
	// sent only when set, so an update leaves SAP's values alone otherwise.
	Description       string         `json:"description,omitempty"`
	GraphModelVersion string         `json:"graphModelVersion,omitempty"`
	ODataContainment  *bool          `json:"odataContainment,omitempty"`
	Exclude           []string       `json:"exclude"`
	DataSources       []DataSource   `json:"dataSources"`
	LocatingPolicy    LocatingPolicy `json:"locatingPolicy"`
}

// ProcessingFailedError reports a business data graph whose processing
// ended in FAILED. Config is the graph as SAP reported it, so a caller can
// still record that the graph exists.
type ProcessingFailedError struct {
	ID     string
	Config *GraphConfiguration
}

func (e *ProcessingFailedError) Error() string {
	detail := e.Config.StatusDetails
	if detail == "" {
		detail = "SAP reported status FAILED without details"
	}
	msg := fmt.Sprintf("apicomposition: processing of business data graph %q failed: %s", e.ID, detail)
	for _, m := range e.Config.LogMessages {
		msg += "\n" + string(m)
	}
	return msg
}

// CreateGraphConfiguration creates a business data graph with
// POST .../GraphConfiguration and waits until SAP has processed it. The
// graph returned is the one read after processing. When processing fails,
// the error is a *ProcessingFailedError.
func (c *Client) CreateGraphConfiguration(ctx context.Context, in GraphConfigurationInput) (*GraphConfiguration, error) {
	payload, err := encodeInput(in)
	if err != nil {
		return nil, err
	}
	if _, err := c.post(ctx, graphConfigurationResource, payload); err != nil {
		return nil, err
	}
	return c.waitUntilProcessed(ctx, in.BusinessDataGraphIdentifier)
}

// GetGraphConfiguration reads a business data graph with
// GET .../GraphConfiguration/{id}.
func (c *Client) GetGraphConfiguration(ctx context.Context, id string) (*GraphConfiguration, error) {
	body, err := c.get(ctx, graphConfigurationPath(id))
	if err != nil {
		return nil, err
	}

	var cfg GraphConfiguration
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("apicomposition: decoding business data graph configuration: %w", err)
	}
	if cfg.Deleted {
		return nil, &apierror.Error{StatusCode: http.StatusNotFound, Message: "SAP marks the business data graph as deleted"}
	}
	return &cfg, nil
}

// UpdateGraphConfiguration changes a business data graph with
// PATCH .../GraphConfiguration/{id} and waits until SAP has processed the
// change. SAP documents the method and URL but gives no example body; the
// body sent is the writable part of the graph, which a PATCH applies
// property by property.
func (c *Client) UpdateGraphConfiguration(ctx context.Context, id string, in GraphConfigurationInput) (*GraphConfiguration, error) {
	payload, err := encodeInput(in)
	if err != nil {
		return nil, err
	}
	if _, err := c.patch(ctx, graphConfigurationPath(id), payload); err != nil {
		return nil, err
	}
	return c.waitUntilProcessed(ctx, id)
}

// DeleteGraphConfiguration deletes a business data graph. SAP states that
// the API deletes graphs but documents no request; this sends DELETE to the
// same URL that GET and PATCH use.
func (c *Client) DeleteGraphConfiguration(ctx context.Context, id string) error {
	return c.delete(ctx, graphConfigurationPath(id))
}

func graphConfigurationPath(id string) string {
	return graphConfigurationResource + "/" + url.PathEscape(id)
}

func encodeInput(in GraphConfigurationInput) ([]byte, error) {
	if in.Exclude == nil {
		in.Exclude = []string{}
	}
	payload, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("apicomposition: encoding business data graph configuration: %w", err)
	}
	return payload, nil
}

// waitUntilProcessed polls the graph until its status leaves PROCESSING,
// with capped exponential backoff, until ctx ends.
func (c *Client) waitUntilProcessed(ctx context.Context, id string) (*GraphConfiguration, error) {
	const (
		initialDelay = 1 * time.Second
		maxDelay     = 10 * time.Second
	)

	delay := initialDelay
	for {
		cfg, err := c.GetGraphConfiguration(ctx, id)
		if err != nil {
			return nil, err
		}

		switch cfg.Status {
		case StatusProcessing:
		case StatusFailed:
			return cfg, &ProcessingFailedError{ID: id, Config: cfg}
		default:
			return cfg, nil
		}

		jitter := time.Duration(rand.Int63n(int64(delay) / 2)) //nolint:gosec // G404: jitter timing, not security-sensitive
		timer := time.NewTimer(delay + jitter)
		select {
		case <-ctx.Done():
			timer.Stop()
			return cfg, fmt.Errorf("apicomposition: business data graph %q was still processing when the wait ended: %w", id, ctx.Err())
		case <-timer.C:
		}

		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
}
