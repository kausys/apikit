// Package tags provides built-in parsers for common swagger directives.
// Import this package to automatically register all standard parsers:
//
//	import _ "github.com/kausys/openapi/parser/tags"
package tags

import (
	"github.com/kausys/apikit/cmd/openapi/parser"
)

//nolint:gochecknoinits // registers built-in parsers via blank import pattern
func init() {
	// Register all built-in parsers
	registerMetaParsers()
	registerRouteParsers()
	registerModelParsers()
}

func registerMetaParsers() {
	parser.Register(parser.DirectiveMeta, NewTitleParser())
	parser.Register(parser.DirectiveMeta, NewVersionParser())
	parser.Register(parser.DirectiveMeta, NewDescriptionParser())
	parser.Register(parser.DirectiveMeta, NewTermsOfServiceParser())
	parser.Register(parser.DirectiveMeta, NewContactParser())
	parser.Register(parser.DirectiveMeta, NewLicenseParser())
	parser.Register(parser.DirectiveMeta, NewBasepathParser())
	parser.Register(parser.DirectiveMeta, NewHostParser())
	parser.Register(parser.DirectiveMeta, NewSchemesParser())
	parser.Register(parser.DirectiveMeta, NewConsumesParser())
	parser.Register(parser.DirectiveMeta, NewProducesParser())
}

func registerRouteParsers() {
	parser.Register(parser.DirectiveRoute, NewSummaryParser())
	parser.Register(parser.DirectiveRoute, NewDescriptionParser())
	parser.Register(parser.DirectiveRoute, NewTagsParser())
	parser.Register(parser.DirectiveRoute, NewDeprecatedParser())
	parser.Register(parser.DirectiveRoute, NewOperationIDParser())
	parser.Register(parser.DirectiveRoute, NewSecurityParser())
}

func registerModelParsers() {
	parser.Register(parser.DirectiveModel, NewDescriptionParser())
	parser.Register(parser.DirectiveModel, NewExampleParser())
	parser.Register(parser.DirectiveModel, NewRequiredParser())
}
