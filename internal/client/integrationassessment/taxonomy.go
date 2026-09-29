package integrationassessment

import (
	"context"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

// The rest of SAP's ISA-M reference taxonomy, read only. SAP Help lists these
// entities with a description each (integration-assessment-apis-47847b5);
// the fields and links come from the live $metadata, and a tenant read all
// four sets in September 2026. SAP Help's text for Domain Determination is a
// copy of the recommendation degree's; the $metadata shows what it is: the
// integration domain that applies between a source and a target deployment
// model.
const (
	useCasePatternEntitySet         = "UseCasePattern"
	integrationPatternEntitySet     = "IntegrationPattern"
	keyCharacteristicGroupEntitySet = "KeyCharacteristicGroup"
	domainDeterminationEntitySet    = "DomainDetermination"
)

// UseCasePattern refines an integration style.
type UseCasePattern struct {
	ID          string        `json:"Id"`
	Name        string        `json:"Name"`
	Description *string       `json:"Description"`
	Example     *string       `json:"Example"`
	Style       *linkedEntity `json:"Style"`
}

// StyleID is the refined style's Id, or "".
func (u *UseCasePattern) StyleID() string { return u.Style.id() }

// IntegrationPattern combines an integration domain and an integration
// style.
type IntegrationPattern struct {
	ID     string        `json:"Id"`
	Name   string        `json:"Name"`
	Domain *linkedEntity `json:"Domain"`
	Style  *linkedEntity `json:"Style"`
}

// DomainID is the pattern's domain Id, or "".
func (p *IntegrationPattern) DomainID() string { return p.Domain.id() }

// StyleID is the pattern's style Id, or "".
func (p *IntegrationPattern) StyleID() string { return p.Style.id() }

// KeyCharacteristicGroup groups key characteristics.
type KeyCharacteristicGroup struct {
	ID          string  `json:"Id"`
	Name        string  `json:"Name"`
	Description *string `json:"Description"`
}

// DomainDetermination says which integration domain applies between a
// source and a target deployment model. It has no name.
type DomainDetermination struct {
	ID                    string        `json:"Id"`
	Domain                *linkedEntity `json:"Domain"`
	SourceDeploymentModel *linkedEntity `json:"SourceDeploymentModel"`
	TargetDeploymentModel *linkedEntity `json:"TargetDeploymentModel"`
}

// DomainID is the determined domain's Id, or "".
func (d *DomainDetermination) DomainID() string { return d.Domain.id() }

// SourceDeploymentModelID is the source deployment model's Id, or "".
func (d *DomainDetermination) SourceDeploymentModelID() string { return d.SourceDeploymentModel.id() }

// TargetDeploymentModelID is the target deployment model's Id, or "".
func (d *DomainDetermination) TargetDeploymentModelID() string { return d.TargetDeploymentModel.id() }

// ListUseCasePatterns lists the use case patterns with their style.
func (c *Client) ListUseCasePatterns(ctx context.Context) ([]UseCasePattern, error) {
	return v2.GetAllPages[UseCasePattern](ctx, c.odata, v2.BuildPath(useCasePatternEntitySet, "", "$expand=Style"))
}

// ListIntegrationPatterns lists the integration patterns with their domain
// and style.
func (c *Client) ListIntegrationPatterns(ctx context.Context) ([]IntegrationPattern, error) {
	return v2.GetAllPages[IntegrationPattern](ctx, c.odata, v2.BuildPath(integrationPatternEntitySet, "", "$expand=Domain,Style"))
}

// ListKeyCharacteristicGroups lists the key characteristic groups.
func (c *Client) ListKeyCharacteristicGroups(ctx context.Context) ([]KeyCharacteristicGroup, error) {
	return v2.GetAllPages[KeyCharacteristicGroup](ctx, c.odata, keyCharacteristicGroupEntitySet)
}

// ListDomainDeterminations lists the domain determinations with their
// domain and both deployment models.
func (c *Client) ListDomainDeterminations(ctx context.Context) ([]DomainDetermination, error) {
	return v2.GetAllPages[DomainDetermination](ctx, c.odata,
		v2.BuildPath(domainDeterminationEntitySet, "", "$expand=Domain,SourceDeploymentModel,TargetDeploymentModel"))
}
