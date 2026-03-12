package spec

// Holds a set of reusable objects for different aspects of the OAS.
// All objects defined within the Components Object will have no effect on the API unless they are
// explicitly referenced from outside the Components Object.
//
// See: https://spec.openapis.org/oas/v3.1.1.html#components-object
type Components struct {
	// An object to hold reusable Schema Objects.
	Schemas map[string]*Schema `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	// An object to hold reusable Response Objects.
	Responses map[string]*Response `json:"responses,omitempty" yaml:"responses,omitempty"`
	// An object to hold reusable Parameter Objects.
	Parameters map[string]*Parameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	// An object to hold reusable Example Objects.
	Examples map[string]*Example `json:"examples,omitempty" yaml:"examples,omitempty"`
	// An object to hold reusable Request Body Objects.
	RequestBodies map[string]*RequestBody `json:"requestBodies,omitempty" yaml:"requestBodies,omitempty"`
	// An object to hold reusable Header Objects.
	Headers map[string]*Header `json:"headers,omitempty" yaml:"headers,omitempty"`
	// An object to hold reusable Security Scheme Objects.
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
	// An object to hold reusable Link Objects.
	Links map[string]*Link `json:"links,omitempty" yaml:"links,omitempty"`
	// An object to hold reusable Callback Objects.
	Callbacks map[string]*Callback `json:"callbacks,omitempty" yaml:"callbacks,omitempty"`
	// An object to hold reusable Path Item Objects.
	PathItems map[string]*PathItem `json:"pathItems,omitempty" yaml:"pathItems,omitempty"`
}
