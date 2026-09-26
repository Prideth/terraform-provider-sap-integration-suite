// Package apimeta reads, normalizes and compares the published contracts of
// SAP Integration Suite services: OData $metadata documents (V2 and V4) and
// OpenAPI specifications.
//
// It serves three purposes. Contract tests check the provider's wire
// structs against a service's contract (Verify). Discovery compares a live
// contract with a committed snapshot and reports, in semantic terms, what
// SAP added, removed or changed (Diff). And the discovery report lists what
// a service exposes that the provider does not use yet.
//
// Everything here works offline on documents; fetching them from a tenant
// is left to the callers (cmd/apidiscovery and the acceptance tests).
package apimeta

// Protocols of a Service.
const (
	ProtocolODataV2 = "odata-v2"
	ProtocolODataV4 = "odata-v4"
	ProtocolOpenAPI = "openapi"
)

// Service is the normalized contract of one service root. The JSON form is
// the snapshot format; every list is sorted so that two snapshots of the
// same contract are byte-identical.
type Service struct {
	ID       string `json:"id"`
	Protocol string `json:"protocol"`
	// CapturedAt is the date the contract last changed in the snapshot
	// (YYYY-MM-DD); a refresh without semantic changes keeps it.
	CapturedAt string `json:"capturedAt,omitempty"`
	// Source says where the document came from, without host names, for
	// example "live $metadata" or "SAP-samples/…/openapi.json".
	Source string `json:"source,omitempty"`

	Namespaces   []string      `json:"namespaces,omitempty"`
	EntitySets   []EntitySet   `json:"entitySets,omitempty"`
	Singletons   []EntitySet   `json:"singletons,omitempty"`
	EntityTypes  []EntityType  `json:"entityTypes,omitempty"`
	ComplexTypes []EntityType  `json:"complexTypes,omitempty"`
	EnumTypes    []EnumType    `json:"enumTypes,omitempty"`
	Associations []Association `json:"associations,omitempty"`
	Operations   []Operation   `json:"operations,omitempty"`

	// ServiceDocument lists the collections the service root itself names
	// (OData service document), when it was fetched.
	ServiceDocument []string `json:"serviceDocument,omitempty"`

	// REST holds an OpenAPI contract (Protocol openapi).
	REST *RESTContract `json:"rest,omitempty"`

	// Graph summarizes the traversal from the service's entry points.
	Graph *GraphSummary `json:"graph,omitempty"`
}

// EntitySet is an entity set or singleton of the entity container.
type EntitySet struct {
	Name       string `json:"name"`
	EntityType string `json:"entityType"`
	// NavigationBindings maps a navigation path to its target entity set
	// (OData V4).
	NavigationBindings map[string]string `json:"navigationBindings,omitempty"`
}

// EntityType is an entity type or, in ComplexTypes, a complex type.
// Names are qualified with the schema namespace ("com.sap.hci.api.Package").
type EntityType struct {
	Name        string               `json:"name"`
	BaseType    string               `json:"baseType,omitempty"`
	Abstract    bool                 `json:"abstract,omitempty"`
	OpenType    bool                 `json:"openType,omitempty"`
	HasStream   bool                 `json:"hasStream,omitempty"` // media entity
	Key         []string             `json:"key,omitempty"`       // in declared order
	Properties  []Property           `json:"properties,omitempty"`
	Navigation  []NavigationProperty `json:"navigationProperties,omitempty"`
	Annotations map[string]string    `json:"annotations,omitempty"`
}

// Property is a structural property.
type Property struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	// Facets, as written in the document ("" when absent).
	MaxLength   string            `json:"maxLength,omitempty"`
	Precision   string            `json:"precision,omitempty"`
	Scale       string            `json:"scale,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// NavigationProperty links an entity type to another.
type NavigationProperty struct {
	Name string `json:"name"`
	// Target is the qualified target entity type; Collection is true for a
	// to-many relationship.
	Target     string `json:"target"`
	Collection bool   `json:"collection,omitempty"`
	// OData V2: the association and roles.
	Relationship string `json:"relationship,omitempty"`
	FromRole     string `json:"fromRole,omitempty"`
	ToRole       string `json:"toRole,omitempty"`
	// OData V4.
	Partner        string                  `json:"partner,omitempty"`
	ContainsTarget bool                    `json:"containsTarget,omitempty"`
	Constraints    []ReferentialConstraint `json:"referentialConstraints,omitempty"`
}

// ReferentialConstraint pairs a dependent property with the principal one.
type ReferentialConstraint struct {
	Property           string `json:"property"`
	ReferencedProperty string `json:"referencedProperty"`
}

// Association is an OData V2 association.
type Association struct {
	Name        string                  `json:"name"`
	Ends        []AssociationEnd        `json:"ends"`
	Principal   string                  `json:"principalRole,omitempty"`
	Dependent   string                  `json:"dependentRole,omitempty"`
	Constraints []ReferentialConstraint `json:"referentialConstraints,omitempty"`
}

// AssociationEnd is one end of an association.
type AssociationEnd struct {
	Role         string `json:"role"`
	Type         string `json:"type"`
	Multiplicity string `json:"multiplicity"`
}

// EnumType is an OData V4 enumeration.
type EnumType struct {
	Name    string   `json:"name"`
	Members []string `json:"members"`
}

// Operation kinds.
const (
	KindFunctionImport = "FunctionImport"
	KindActionImport   = "ActionImport"
	KindFunction       = "Function"
	KindAction         = "Action"
)

// Operation is a function import (V2 or V4), action import, or a V4
// function or action.
type Operation struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	// HTTPMethod is the V2 m:HttpMethod of a function import.
	HTTPMethod string `json:"httpMethod,omitempty"`
	// Target is the function or action a V4 import exposes.
	Target     string      `json:"target,omitempty"`
	EntitySet  string      `json:"entitySet,omitempty"`
	Bound      bool        `json:"bound,omitempty"`
	Parameters []Parameter `json:"parameters,omitempty"`
	ReturnType string      `json:"returnType,omitempty"`
}

// Parameter is an operation parameter. Order is significant and kept.
type Parameter struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Mode     string `json:"mode,omitempty"`
}

// GraphSummary is the result of traversing the metadata graph from the
// entity container.
type GraphSummary struct {
	// Reachable lists every type reachable from an entity set, singleton or
	// operation, through properties, base types, navigation properties and
	// operation signatures.
	Reachable []string `json:"reachable"`
	// Unreachable lists declared types no entry point leads to.
	Unreachable []string `json:"unreachable,omitempty"`
	// Unresolved lists type references that no schema declares.
	Unresolved []string `json:"unresolved,omitempty"`
}

// RESTContract is the normalized form of an OpenAPI specification.
type RESTContract struct {
	Title           string          `json:"title,omitempty"`
	Version         string          `json:"version,omitempty"`
	Operations      []RESTOperation `json:"operations"`
	Schemas         []RESTSchema    `json:"schemas,omitempty"`
	SecuritySchemes []string        `json:"securitySchemes,omitempty"`
}

// RESTOperation is one method on one path.
type RESTOperation struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	OperationID string            `json:"operationId,omitempty"`
	Parameters  []RESTParameter   `json:"parameters,omitempty"`
	RequestBody string            `json:"requestBody,omitempty"`
	Responses   map[string]string `json:"responses,omitempty"`
	Security    []string          `json:"security,omitempty"`
}

// RESTParameter is a path, query, header or cookie parameter.
type RESTParameter struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Type     string `json:"type,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// RESTSchema is a named component schema.
type RESTSchema struct {
	Name       string         `json:"name"`
	Type       string         `json:"type,omitempty"`
	Properties []RESTProperty `json:"properties,omitempty"`
}

// RESTProperty is a property of a component schema.
type RESTProperty struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required,omitempty"`
}
