package integrationassessment

import (
	"context"
	"encoding/json"
	"fmt"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

const (
	vendorEntitySet              = "Vendor"
	applicationEntitySet         = "Application"
	applicationInstanceEntitySet = "ApplicationInstance"
	technologyEntitySet          = "Technology"
	technologyInstanceEntitySet  = "TechnologyInstance"
	deploymentModelEntitySet     = "DeploymentModel"
)

// Vendor is a supplier of applications and technologies. Id is assigned by
// the service on create.
type Vendor struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
}

// Application is a business application of the landscape, optionally of a
// vendor.
type Application struct {
	ID     string        `json:"Id"`
	Name   string        `json:"Name"`
	Vendor *linkedEntity `json:"Vendor"`
}

// VendorID is the linked vendor's Id, or "".
func (a *Application) VendorID() string { return a.Vendor.id() }

// ApplicationInstance is one installation of an application, with the
// deployment model it runs on.
type ApplicationInstance struct {
	ID              string        `json:"Id"`
	Name            string        `json:"Name"`
	Description     *string       `json:"Description"`
	Application     *linkedEntity `json:"Application"`
	DeploymentModel *linkedEntity `json:"DeploymentModel"`
}

// ApplicationID is the linked application's Id, or "".
func (i *ApplicationInstance) ApplicationID() string { return i.Application.id() }

// DeploymentModelID is the linked deployment model's Id, or "".
func (i *ApplicationInstance) DeploymentModelID() string { return i.DeploymentModel.id() }

// Technology is an integration technology the assessment can recommend,
// for example a middleware product, of a vendor.
type Technology struct {
	ID     string        `json:"Id"`
	Name   string        `json:"Name"`
	Vendor *linkedEntity `json:"Vendor"`
}

// VendorID is the linked vendor's Id, or "".
func (t *Technology) VendorID() string { return t.Vendor.id() }

// TechnologyInstance is one installation of a technology, with the
// deployment model it runs on.
type TechnologyInstance struct {
	ID              string        `json:"Id"`
	Name            string        `json:"Name"`
	Technology      *linkedEntity `json:"Technology"`
	DeploymentModel *linkedEntity `json:"DeploymentModel"`
}

// TechnologyID is the linked technology's Id, or "".
func (i *TechnologyInstance) TechnologyID() string { return i.Technology.id() }

// DeploymentModelID is the linked deployment model's Id, or "".
func (i *TechnologyInstance) DeploymentModelID() string { return i.DeploymentModel.id() }

// DeploymentModel is an entry of SAP's taxonomy, for example cloud or
// on-premise.
type DeploymentModel struct {
	ID          string  `json:"Id"`
	Name        string  `json:"Name"`
	Description *string `json:"Description"`
}

// Write bodies. A create omits links that are not set. An update sends the
// Vendor link always: null removes it, which a tenant accepted (204) and a
// read afterwards confirmed. Changing a required link with PATCH (an
// instance's application or deployment model, a technology's vendor, a
// technology instance's deployment model) was confirmed the same way on
// 2026-09-28.
type vendorWrite struct {
	Name string `json:"Name"`
}

type applicationCreate struct {
	Name   string `json:"Name"`
	Vendor *idRef `json:"Vendor,omitempty"`
}

type applicationUpdate struct {
	Name   string `json:"Name"`
	Vendor *idRef `json:"Vendor"`
}

type applicationInstanceCreate struct {
	Name            string  `json:"Name"`
	Description     *string `json:"Description,omitempty"`
	Application     *idRef  `json:"Application"`
	DeploymentModel *idRef  `json:"DeploymentModel"`
}

type applicationInstanceUpdate struct {
	Name            string  `json:"Name"`
	Description     *string `json:"Description"`
	Application     *idRef  `json:"Application"`
	DeploymentModel *idRef  `json:"DeploymentModel"`
}

type technologyCreate struct {
	Name   string `json:"Name"`
	Vendor *idRef `json:"Vendor"`
}

type technologyUpdate struct {
	Name   string `json:"Name"`
	Vendor *idRef `json:"Vendor"`
}

type technologyInstanceCreate struct {
	Name            string `json:"Name"`
	Technology      *idRef `json:"Technology"`
	DeploymentModel *idRef `json:"DeploymentModel"`
}

// The technology link of a technology instance is not sent: changing it was
// not tested, so the resource replaces the instance instead.
type technologyInstanceUpdate struct {
	Name            string `json:"Name"`
	DeploymentModel *idRef `json:"DeploymentModel"`
}

// get reads one entity with its links expanded.
func (c *Client) get(ctx context.Context, entitySet, id, expand string, v any) error {
	query := ""
	if expand != "" {
		query = "$expand=" + expand
	}
	body, err := c.odata.Get(ctx, entityPath(entitySet, id, query))
	if err != nil {
		return err
	}
	return v2.DecodeEntity(body, v)
}

// create posts a new entity and decodes the entity the service returns
// (201 with the assigned Id on a tenant).
func (c *Client) create(ctx context.Context, entitySet string, payload, v any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("integrationassessment: encoding %s: %w", entitySet, err)
	}
	body, err := c.odata.Post(ctx, entitySet, b)
	if err != nil {
		return err
	}
	return v2.DecodeEntity(body, v)
}

// patch changes the fields in payload (204 on a tenant). The service
// rejects a Content-Type with a charset parameter (400 V122); the OData
// client sends plain application/json.
func (c *Client) patch(ctx context.Context, entitySet, id string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("integrationassessment: encoding %s: %w", entitySet, err)
	}
	_, err = c.odata.Patch(ctx, entityPath(entitySet, id, ""), b)
	return err
}

func (c *Client) remove(ctx context.Context, entitySet, id string) error {
	return c.odata.Delete(ctx, entityPath(entitySet, id, ""))
}

// GetVendor reads a vendor by Id.
func (c *Client) GetVendor(ctx context.Context, id string) (*Vendor, error) {
	var v Vendor
	return &v, c.get(ctx, vendorEntitySet, id, "", &v)
}

// CreateVendor creates a vendor.
func (c *Client) CreateVendor(ctx context.Context, name string) (*Vendor, error) {
	var v Vendor
	return &v, c.create(ctx, vendorEntitySet, vendorWrite{Name: name}, &v)
}

// UpdateVendor renames a vendor.
func (c *Client) UpdateVendor(ctx context.Context, id, name string) error {
	return c.patch(ctx, vendorEntitySet, id, vendorWrite{Name: name})
}

// DeleteVendor deletes a vendor.
func (c *Client) DeleteVendor(ctx context.Context, id string) error {
	return c.remove(ctx, vendorEntitySet, id)
}

// ListVendors lists every vendor, following server-driven paging.
func (c *Client) ListVendors(ctx context.Context) ([]Vendor, error) {
	return v2.GetAllPages[Vendor](ctx, c.odata, vendorEntitySet)
}

// GetApplication reads an application with its vendor link.
func (c *Client) GetApplication(ctx context.Context, id string) (*Application, error) {
	var a Application
	return &a, c.get(ctx, applicationEntitySet, id, "Vendor", &a)
}

// CreateApplication creates an application, linked to vendorID if it is
// not empty.
func (c *Client) CreateApplication(ctx context.Context, name, vendorID string) (*Application, error) {
	var a Application
	return &a, c.create(ctx, applicationEntitySet, applicationCreate{Name: name, Vendor: ref(vendorID)}, &a)
}

// UpdateApplication renames an application and sets or removes its vendor.
func (c *Client) UpdateApplication(ctx context.Context, id, name, vendorID string) error {
	return c.patch(ctx, applicationEntitySet, id, applicationUpdate{Name: name, Vendor: ref(vendorID)})
}

// DeleteApplication deletes an application.
func (c *Client) DeleteApplication(ctx context.Context, id string) error {
	return c.remove(ctx, applicationEntitySet, id)
}

// GetApplicationInstance reads an application instance with its links.
func (c *Client) GetApplicationInstance(ctx context.Context, id string) (*ApplicationInstance, error) {
	var i ApplicationInstance
	return &i, c.get(ctx, applicationInstanceEntitySet, id, "Application,DeploymentModel", &i)
}

// CreateApplicationInstance creates an instance of an application on a
// deployment model.
func (c *Client) CreateApplicationInstance(ctx context.Context, name string, description *string, applicationID, deploymentModelID string) (*ApplicationInstance, error) {
	var i ApplicationInstance
	return &i, c.create(ctx, applicationInstanceEntitySet, applicationInstanceCreate{
		Name: name, Description: description, Application: ref(applicationID), DeploymentModel: ref(deploymentModelID),
	}, &i)
}

// UpdateApplicationInstance changes an instance's name and description and
// moves it to applicationID and deploymentModelID.
func (c *Client) UpdateApplicationInstance(ctx context.Context, id, name string, description *string, applicationID, deploymentModelID string) error {
	return c.patch(ctx, applicationInstanceEntitySet, id, applicationInstanceUpdate{
		Name: name, Description: description, Application: ref(applicationID), DeploymentModel: ref(deploymentModelID),
	})
}

// DeleteApplicationInstance deletes an application instance.
func (c *Client) DeleteApplicationInstance(ctx context.Context, id string) error {
	return c.remove(ctx, applicationInstanceEntitySet, id)
}

// GetTechnology reads a technology with its vendor link.
func (c *Client) GetTechnology(ctx context.Context, id string) (*Technology, error) {
	var t Technology
	return &t, c.get(ctx, technologyEntitySet, id, "Vendor", &t)
}

// CreateTechnology creates a technology of a vendor.
func (c *Client) CreateTechnology(ctx context.Context, name, vendorID string) (*Technology, error) {
	var t Technology
	return &t, c.create(ctx, technologyEntitySet, technologyCreate{Name: name, Vendor: ref(vendorID)}, &t)
}

// UpdateTechnology renames a technology and moves it to vendorID.
func (c *Client) UpdateTechnology(ctx context.Context, id, name, vendorID string) error {
	return c.patch(ctx, technologyEntitySet, id, technologyUpdate{Name: name, Vendor: ref(vendorID)})
}

// DeleteTechnology deletes a technology.
func (c *Client) DeleteTechnology(ctx context.Context, id string) error {
	return c.remove(ctx, technologyEntitySet, id)
}

// ListTechnologies lists every technology, the ones SAP delivers and the
// ones created on the tenant.
func (c *Client) ListTechnologies(ctx context.Context) ([]Technology, error) {
	return v2.GetAllPages[Technology](ctx, c.odata, technologyEntitySet)
}

// GetTechnologyInstance reads a technology instance with its links.
func (c *Client) GetTechnologyInstance(ctx context.Context, id string) (*TechnologyInstance, error) {
	var i TechnologyInstance
	return &i, c.get(ctx, technologyInstanceEntitySet, id, "Technology,DeploymentModel", &i)
}

// CreateTechnologyInstance creates an instance of a technology on a
// deployment model.
func (c *Client) CreateTechnologyInstance(ctx context.Context, name, technologyID, deploymentModelID string) (*TechnologyInstance, error) {
	var i TechnologyInstance
	return &i, c.create(ctx, technologyInstanceEntitySet, technologyInstanceCreate{
		Name: name, Technology: ref(technologyID), DeploymentModel: ref(deploymentModelID),
	}, &i)
}

// UpdateTechnologyInstance renames a technology instance and moves it to
// deploymentModelID.
func (c *Client) UpdateTechnologyInstance(ctx context.Context, id, name, deploymentModelID string) error {
	return c.patch(ctx, technologyInstanceEntitySet, id, technologyInstanceUpdate{Name: name, DeploymentModel: ref(deploymentModelID)})
}

// DeleteTechnologyInstance deletes a technology instance.
func (c *Client) DeleteTechnologyInstance(ctx context.Context, id string) error {
	return c.remove(ctx, technologyInstanceEntitySet, id)
}

// ListDeploymentModels lists SAP's deployment models.
func (c *Client) ListDeploymentModels(ctx context.Context) ([]DeploymentModel, error) {
	return v2.GetAllPages[DeploymentModel](ctx, c.odata, deploymentModelEntitySet)
}
