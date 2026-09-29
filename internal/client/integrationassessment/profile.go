package integrationassessment

import (
	"context"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

// A technology's profile links it to entries of SAP's ISA-M taxonomy: the
// integration domains and styles it serves and how well it meets each key
// characteristic. The assessment compares these profiles when it recommends
// a technology. A tenant created, read and deleted all three association
// types on 2026-09-28; a key characteristic answered PATCH with 400 V101
// ("The operation for this entity is not allowed").
const (
	domainEntitySet                      = "Domain"
	styleEntitySet                       = "Style"
	keyCharacteristicEntitySet           = "KeyCharacteristic"
	keyCharacteristicValueEntitySet      = "KeyCharacteristicValue"
	recommendationDegreeEntitySet        = "KeyCharacteristicRecommendationDegree"
	technologyDomainEntitySet            = "TechnologyDomain"
	technologyStyleEntitySet             = "TechnologyStyle"
	technologyKeyCharacteristicEntitySet = "TechnologyKeyCharacteristic"
)

// Domain is an integration domain of the taxonomy, for example
// cloud-to-cloud integration.
type Domain struct {
	ID          string  `json:"Id"`
	Name        string  `json:"Name"`
	Description *string `json:"Description"`
}

// Style is an integration style of the taxonomy, for example process
// integration or data integration.
type Style struct {
	ID          string  `json:"Id"`
	Name        string  `json:"Name"`
	Description *string `json:"Description"`
}

// KeyCharacteristic is a criterion the assessment rates technologies on.
type KeyCharacteristic struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
}

// KeyCharacteristicValue is one possible value of a key characteristic.
type KeyCharacteristicValue struct {
	ID                string        `json:"Id"`
	Name              string        `json:"Name"`
	Description       *string       `json:"Description"`
	KeyCharacteristic *linkedEntity `json:"KeyCharacteristic"`
}

// KeyCharacteristicID is the Id of the key characteristic the value
// belongs to, or "" when the read did not expand it.
func (v *KeyCharacteristicValue) KeyCharacteristicID() string { return v.KeyCharacteristic.id() }

// RecommendationDegree says how strongly a technology meets a key
// characteristic value.
type RecommendationDegree struct {
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	Score int    `json:"Score"`
}

// TechnologyDomain links a technology to a domain.
type TechnologyDomain struct {
	ID         string        `json:"Id"`
	Technology *linkedEntity `json:"Technology"`
	Domain     *linkedEntity `json:"Domain"`
}

// TechnologyID is the linked technology's Id, or "".
func (d *TechnologyDomain) TechnologyID() string { return d.Technology.id() }

// DomainID is the linked domain's Id, or "".
func (d *TechnologyDomain) DomainID() string { return d.Domain.id() }

// TechnologyStyle links a technology to a style.
type TechnologyStyle struct {
	ID         string        `json:"Id"`
	Technology *linkedEntity `json:"Technology"`
	Style      *linkedEntity `json:"Style"`
}

// TechnologyID is the linked technology's Id, or "".
func (s *TechnologyStyle) TechnologyID() string { return s.Technology.id() }

// StyleID is the linked style's Id, or "".
func (s *TechnologyStyle) StyleID() string { return s.Style.id() }

// TechnologyKeyCharacteristic rates a technology on one key characteristic
// value.
type TechnologyKeyCharacteristic struct {
	ID                     string        `json:"Id"`
	Description            *string       `json:"Description"`
	Technology             *linkedEntity `json:"Technology"`
	KeyCharacteristicValue *linkedEntity `json:"KeyCharacteristicValue"`
	RecommendationDegree   *linkedEntity `json:"RecommendationDegree"`
}

// TechnologyID is the linked technology's Id, or "".
func (k *TechnologyKeyCharacteristic) TechnologyID() string { return k.Technology.id() }

// KeyCharacteristicValueID is the linked value's Id, or "".
func (k *TechnologyKeyCharacteristic) KeyCharacteristicValueID() string {
	return k.KeyCharacteristicValue.id()
}

// RecommendationDegreeID is the linked degree's Id, or "".
func (k *TechnologyKeyCharacteristic) RecommendationDegreeID() string {
	return k.RecommendationDegree.id()
}

// Create bodies, as the tenant accepted them. The associations have no
// update.
type technologyDomainCreate struct {
	Technology *idRef `json:"Technology"`
	Domain     *idRef `json:"Domain"`
}

type technologyStyleCreate struct {
	Technology *idRef `json:"Technology"`
	Style      *idRef `json:"Style"`
}

type technologyKeyCharacteristicCreate struct {
	Technology             *idRef  `json:"Technology"`
	KeyCharacteristicValue *idRef  `json:"KeyCharacteristicValue"`
	RecommendationDegree   *idRef  `json:"RecommendationDegree"`
	Description            *string `json:"Description,omitempty"`
}

// ListDomains lists the integration domains of the taxonomy.
func (c *Client) ListDomains(ctx context.Context) ([]Domain, error) {
	return v2.GetAllPages[Domain](ctx, c.odata, domainEntitySet)
}

// ListStyles lists the integration styles of the taxonomy.
func (c *Client) ListStyles(ctx context.Context) ([]Style, error) {
	return v2.GetAllPages[Style](ctx, c.odata, styleEntitySet)
}

// ListKeyCharacteristics lists the key characteristics of the taxonomy.
func (c *Client) ListKeyCharacteristics(ctx context.Context) ([]KeyCharacteristic, error) {
	return v2.GetAllPages[KeyCharacteristic](ctx, c.odata, keyCharacteristicEntitySet)
}

// ListKeyCharacteristicValues lists every key characteristic value with the
// key characteristic it belongs to.
func (c *Client) ListKeyCharacteristicValues(ctx context.Context) ([]KeyCharacteristicValue, error) {
	return v2.GetAllPages[KeyCharacteristicValue](ctx, c.odata,
		v2.BuildPath(keyCharacteristicValueEntitySet, "", "$expand=KeyCharacteristic"))
}

// ListRecommendationDegrees lists the recommendation degrees.
func (c *Client) ListRecommendationDegrees(ctx context.Context) ([]RecommendationDegree, error) {
	return v2.GetAllPages[RecommendationDegree](ctx, c.odata, recommendationDegreeEntitySet)
}

// GetTechnologyDomain reads a technology's domain link.
func (c *Client) GetTechnologyDomain(ctx context.Context, id string) (*TechnologyDomain, error) {
	var d TechnologyDomain
	return &d, c.get(ctx, technologyDomainEntitySet, id, "Technology,Domain", &d)
}

// CreateTechnologyDomain links a technology to a domain.
func (c *Client) CreateTechnologyDomain(ctx context.Context, technologyID, domainID string) (*TechnologyDomain, error) {
	var d TechnologyDomain
	return &d, c.create(ctx, technologyDomainEntitySet,
		technologyDomainCreate{Technology: ref(technologyID), Domain: ref(domainID)}, &d)
}

// DeleteTechnologyDomain removes a technology's domain link.
func (c *Client) DeleteTechnologyDomain(ctx context.Context, id string) error {
	return c.remove(ctx, technologyDomainEntitySet, id)
}

// GetTechnologyStyle reads a technology's style link.
func (c *Client) GetTechnologyStyle(ctx context.Context, id string) (*TechnologyStyle, error) {
	var s TechnologyStyle
	return &s, c.get(ctx, technologyStyleEntitySet, id, "Technology,Style", &s)
}

// CreateTechnologyStyle links a technology to a style.
func (c *Client) CreateTechnologyStyle(ctx context.Context, technologyID, styleID string) (*TechnologyStyle, error) {
	var s TechnologyStyle
	return &s, c.create(ctx, technologyStyleEntitySet,
		technologyStyleCreate{Technology: ref(technologyID), Style: ref(styleID)}, &s)
}

// DeleteTechnologyStyle removes a technology's style link.
func (c *Client) DeleteTechnologyStyle(ctx context.Context, id string) error {
	return c.remove(ctx, technologyStyleEntitySet, id)
}

// GetTechnologyKeyCharacteristic reads a technology's rating on a key
// characteristic value.
func (c *Client) GetTechnologyKeyCharacteristic(ctx context.Context, id string) (*TechnologyKeyCharacteristic, error) {
	var k TechnologyKeyCharacteristic
	return &k, c.get(ctx, technologyKeyCharacteristicEntitySet, id,
		"Technology,KeyCharacteristicValue,RecommendationDegree", &k)
}

// CreateTechnologyKeyCharacteristic rates a technology on a key
// characteristic value.
func (c *Client) CreateTechnologyKeyCharacteristic(ctx context.Context, technologyID, valueID, degreeID string, description *string) (*TechnologyKeyCharacteristic, error) {
	var k TechnologyKeyCharacteristic
	return &k, c.create(ctx, technologyKeyCharacteristicEntitySet, technologyKeyCharacteristicCreate{
		Technology: ref(technologyID), KeyCharacteristicValue: ref(valueID),
		RecommendationDegree: ref(degreeID), Description: description,
	}, &k)
}

// DeleteTechnologyKeyCharacteristic removes a technology's rating.
func (c *Client) DeleteTechnologyKeyCharacteristic(ctx context.Context, id string) error {
	return c.remove(ctx, technologyKeyCharacteristicEntitySet, id)
}
