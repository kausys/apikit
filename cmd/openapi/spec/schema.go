package spec

// The Schema Object allows the definition of input and output data types. These types can be
// objects, but also primitives and arrays. This object is a superset of the JSON Schema
// Specification Draft 2020-12.
//
// See: https://spec.openapis.org/oas/v3.1.1.html#schema-object
type Schema struct {
	// A reference to another schema. If present, other properties are ignored.
	Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`

	// JSON Schema keywords — Core Vocabulary
	// See: https://json-schema.org/draft/2020-12/json-schema-core

	// The JSON Schema dialect for this schema.
	Schema string `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	// Inline schema definitions (replaces "definitions").
	Defs map[string]*Schema `json:"$defs,omitempty" yaml:"$defs,omitempty"`
	// Plain name anchor for this schema.
	Anchor string `json:"$anchor,omitempty" yaml:"$anchor,omitempty"`
	// A comment for schema maintainers.
	Comment string `json:"$comment,omitempty" yaml:"$comment,omitempty"`

	// JSON Schema keywords — Validation Vocabulary
	// See: https://json-schema.org/draft/2020-12/json-schema-validation

	// The title of the schema.
	Title string `json:"title,omitempty" yaml:"title,omitempty"`
	// A number that the value must be a multiple of.
	MultipleOf *float64 `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`
	// The maximum value for a number (inclusive).
	Maximum *float64 `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	// The exclusive maximum value for a number. In JSON Schema 2020-12, this is a numeric value
	// (not a boolean as in draft-04/OpenAPI 3.0).
	ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	// The minimum value for a number (inclusive).
	Minimum *float64 `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	// The exclusive minimum value for a number. In JSON Schema 2020-12, this is a numeric value
	// (not a boolean as in draft-04/OpenAPI 3.0).
	ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	// The maximum length for a string.
	MaxLength *uint64 `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	// The minimum length for a string.
	MinLength uint64 `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	// A regular expression that a string value must match. This string SHOULD be a valid regular
	// expression, according to the Ecma-262 Edition 5.1 regular expression dialect.
	Pattern string `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	// The maximum number of items in an array.
	MaxItems *uint64 `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	// The minimum number of items in an array.
	MinItems *uint64 `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	// Indicates whether items in an array must be unique.
	UniqueItems bool `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	// The maximum number of properties in an object.
	MaxProperties *uint64 `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	// The minimum number of properties in an object.
	MinProperties *uint64 `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	// A list of required properties when type is "object".
	Required []string `json:"required,omitempty" yaml:"required,omitempty"`
	// The enumeration of possible values.
	Enum []any `json:"enum,omitempty" yaml:"enum,omitempty"`
	// A constant value that the instance must be equal to.
	Const any `json:"const,omitempty" yaml:"const,omitempty"`

	// JSON Schema keywords — Applicator Vocabulary

	// Value can be a string or array of strings. In OpenAPI 3.1, type can be
	// ["string", "null"] to express nullable types (replacing the removed nullable keyword).
	Type SchemaType `json:"type,omitzero" yaml:"type,omitempty"`
	// Inline or referenced schema MUST be of a Schema Object.
	AllOf []*Schema `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	// Inline or referenced schema MUST be of a Schema Object.
	OneOf []*Schema `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	// Inline or referenced schema MUST be of a Schema Object.
	AnyOf []*Schema `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	// Inline or referenced schema MUST be of a Schema Object.
	Not *Schema `json:"not,omitempty" yaml:"not,omitempty"`
	// Conditional schema — if this schema validates, apply Then.
	If *Schema `json:"if,omitempty" yaml:"if,omitempty"`
	// Conditional schema — applied when If validates.
	Then *Schema `json:"then,omitempty" yaml:"then,omitempty"`
	// Conditional schema — applied when If does not validate.
	Else *Schema `json:"else,omitempty" yaml:"else,omitempty"`
	// Schemas that apply when a specific property is present.
	DependentSchemas map[string]*Schema `json:"dependentSchemas,omitempty" yaml:"dependentSchemas,omitempty"`
	// Ordered tuple validation for array items (replaces positional items from older drafts).
	PrefixItems []*Schema `json:"prefixItems,omitempty" yaml:"prefixItems,omitempty"`
	// Schema for array items. In 3.1 this applies to items after prefixItems.
	Items *Schema `json:"items,omitempty" yaml:"items,omitempty"`
	// Schema that at least one array item must match.
	Contains *Schema `json:"contains,omitempty" yaml:"contains,omitempty"`
	// Property definitions MUST be a Schema Object (inline or referenced).
	Properties map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	// Value can be boolean or object. Inline or referenced schema MUST be of a Schema Object.
	// Consistent with JSON Schema, additionalProperties defaults to true.
	AdditionalProperties *Schema `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	// Applies to properties not evaluated by other keywords (e.g., allOf/properties).
	UnevaluatedProperties *Schema `json:"unevaluatedProperties,omitempty" yaml:"unevaluatedProperties,omitempty"`
	// Applies to array items not evaluated by other keywords (e.g., prefixItems/items).
	UnevaluatedItems *Schema `json:"unevaluatedItems,omitempty" yaml:"unevaluatedItems,omitempty"`

	// JSON Schema keywords — Content Vocabulary (for string-encoded content)

	// The media type of the string content (e.g., "application/octet-stream" for file uploads).
	// Replaces the OpenAPI 3.0 pattern of using format: binary.
	ContentMediaType string `json:"contentMediaType,omitempty" yaml:"contentMediaType,omitempty"`
	// The encoding of the string content (e.g., "base64").
	ContentEncoding string `json:"contentEncoding,omitempty" yaml:"contentEncoding,omitempty"`

	// JSON Schema keywords — Metadata/Annotation

	// CommonMark syntax MAY be used for rich text representation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// See Data Type Formats for further details. While relying on JSON Schema's defined formats, the
	// OAS offers a few additional predefined formats.
	Format string `json:"format,omitempty" yaml:"format,omitempty"`
	// The default value represents what would be assumed by the consumer of the input as the value of
	// the schema if one is not provided.
	Default any `json:"default,omitempty" yaml:"default,omitempty"`
	// An array of examples. Replaces the singular "example" keyword from OpenAPI 3.0.
	Examples []any `json:"examples,omitempty" yaml:"examples,omitempty"`

	// OpenAPI-specific fixed fields

	// Adds support for polymorphism. The discriminator is used to determine which of a set of schemas
	// a payload is expected to satisfy. See Composition and Inheritance for more details.
	Discriminator *Discriminator `json:"discriminator,omitempty" yaml:"discriminator,omitempty"`
	// Relevant only for Schema Object properties definitions. Declares the property as "read only".
	ReadOnly bool `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	// Relevant only for Schema Object properties definitions. Declares the property as "write only".
	WriteOnly bool `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`
	// This MAY be used only on property schemas. It has no effect on root schemas. Adds additional
	// metadata to describe the XML representation of this property.
	XML *XML `json:"xml,omitempty" yaml:"xml,omitempty"`
	// Additional external documentation for this schema.
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
	// Specifies that a schema is deprecated and SHOULD be transitioned out of usage. Default value is
	// false.
	Deprecated bool `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
}
