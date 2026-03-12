package tags

import (
	"fmt"

	"github.com/kausys/apikit/cmd/openapi/parser"
	"github.com/kausys/apikit/cmd/openapi/spec"
)

// NewTitleParser creates a parser for the Title directive.
func NewTitleParser() *SingleLineParser {
	return NewSingleLineParser("title", "title:", []parser.Context{parser.ContextMeta}, parser.SetterMap{
		parser.ContextMeta: func(target any, value any) error {
			if info, ok := target.(*spec.Info); ok {
				info.Title = value.(string)
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "title", ExpectedType: "*spec.Info", ActualType: typeOf(target)}
		},
	})
}

// NewVersionParser creates a parser for the Version directive.
func NewVersionParser() *SingleLineParser {
	return NewSingleLineParser("version", "version:", []parser.Context{parser.ContextMeta}, parser.SetterMap{
		parser.ContextMeta: func(target any, value any) error {
			if info, ok := target.(*spec.Info); ok {
				info.Version = value.(string)
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "version", ExpectedType: "*spec.Info", ActualType: typeOf(target)}
		},
	})
}

// NewTermsOfServiceParser creates a parser for the Terms of Service directive.
func NewTermsOfServiceParser() *SingleLineParser {
	return NewSingleLineParser("termsOfService", "terms of service:", []parser.Context{parser.ContextMeta}, parser.SetterMap{
		parser.ContextMeta: func(target any, value any) error {
			if info, ok := target.(*spec.Info); ok {
				info.TermsOfService = value.(string)
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "termsOfService", ExpectedType: "*spec.Info", ActualType: typeOf(target)}
		},
	})
}

// NewContactParser creates a parser for the Contact directive.
func NewContactParser() *SingleLineParser {
	return NewSingleLineParser("contact", "contact:", []parser.Context{parser.ContextMeta}, parser.SetterMap{
		parser.ContextMeta: func(target any, value any) error {
			if info, ok := target.(*spec.Info); ok {
				if info.Contact == nil {
					info.Contact = &spec.Contact{}
				}
				info.Contact.Email = value.(string)
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "contact", ExpectedType: "*spec.Info", ActualType: typeOf(target)}
		},
	})
}

// NewLicenseParser creates a parser for the License directive.
func NewLicenseParser() *SingleLineParser {
	return NewSingleLineParser("license", "license:", []parser.Context{parser.ContextMeta}, parser.SetterMap{
		parser.ContextMeta: func(target any, value any) error {
			if info, ok := target.(*spec.Info); ok {
				if info.License == nil {
					info.License = &spec.License{}
				}
				info.License.Name = value.(string)
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "license", ExpectedType: "*spec.Info", ActualType: typeOf(target)}
		},
	})
}

// NewBasepathParser creates a parser for the BasePath directive.
func NewBasepathParser() *SingleLineParser {
	return NewSingleLineParser("basepath", "basepath:", []parser.Context{parser.ContextMeta}, parser.SetterMap{
		parser.ContextMeta: func(target any, value any) error {
			if openAPI, ok := target.(*spec.OpenAPI); ok {
				if len(openAPI.Servers) == 0 {
					openAPI.Servers = append(openAPI.Servers, &spec.Server{})
				}
				openAPI.Servers[0].URL = value.(string)
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "basepath", ExpectedType: "*spec.OpenAPI", ActualType: typeOf(target)}
		},
	})
}

// NewHostParser creates a parser for the Host directive.
func NewHostParser() *SingleLineParser {
	return NewSingleLineParser("host", "host:", []parser.Context{parser.ContextMeta}, parser.SetterMap{
		parser.ContextMeta: func(target any, value any) error {
			if openAPI, ok := target.(*spec.OpenAPI); ok {
				if len(openAPI.Servers) == 0 {
					openAPI.Servers = append(openAPI.Servers, &spec.Server{})
				}
				// Prepend https:// if no scheme
				host := value.(string)
				if openAPI.Servers[0].URL != "" {
					openAPI.Servers[0].URL = "https://" + host + openAPI.Servers[0].URL
				} else {
					openAPI.Servers[0].URL = "https://" + host
				}
				return nil
			}
			return &parser.ErrInvalidTarget{ParserName: "host", ExpectedType: "*spec.OpenAPI", ActualType: typeOf(target)}
		},
	})
}

// NewSchemesParser creates a parser for the Schemes directive (http, https).
func NewSchemesParser() *ListParser {
	return NewListParser("schemes", "schemes:", " ", []parser.Context{parser.ContextMeta}, parser.SetterMap{
		parser.ContextMeta: func(target any, value any) error {
			// Schemes are handled via server URLs in OpenAPI 3.0
			return nil
		},
	})
}

// NewConsumesParser creates a parser for the Consumes directive.
func NewConsumesParser() *ListParser {
	return NewListParser("consumes", "consumes:", " ", []parser.Context{parser.ContextMeta}, parser.SetterMap{
		parser.ContextMeta: func(target any, value any) error {
			// Consumes is handled at operation level in OpenAPI 3.0
			return nil
		},
	})
}

// NewProducesParser creates a parser for the Produces directive.
func NewProducesParser() *ListParser {
	return NewListParser("produces", "produces:", " ", []parser.Context{parser.ContextMeta}, parser.SetterMap{
		parser.ContextMeta: func(target any, value any) error {
			// Produces is handled at operation level in OpenAPI 3.0
			return nil
		},
	})
}

// typeOf returns the type name of a value for error messages.
func typeOf(v any) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprintf("%T", v)
}
