package generator

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/kausys/apikit/cmd/openapi/spec"
)

// pathTemplateParam matches one {name} of a path template.
var pathTemplateParam = regexp.MustCompile(`\{([^{}]+)\}`)

// ValidateDocument checks a generated document for what makes it invalid
// OpenAPI, or a promise the API does not keep, and returns one line per
// problem, sorted. It checks what the generator can get wrong, not the whole
// specification: an operation with no response, a path parameter the template
// does not name or a template parameter nothing declares, a parameter declared
// twice, an operation id used twice, a security requirement or a $ref that
// names nothing.
func ValidateDocument(doc *spec.OpenAPI) []string {
	var problems []string
	add := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}

	schemes := map[string]bool{}
	schemas := map[string]*spec.Schema{}
	if doc.Components != nil {
		for name := range doc.Components.SecuritySchemes {
			schemes[name] = true
		}
		schemas = doc.Components.Schemas
	}

	checkSecurity := func(where string, reqs []*spec.SecurityRequirement) {
		for _, req := range reqs {
			for _, name := range slices.Sorted(maps.Keys(req.Requirements)) {
				if !schemes[name] {
					add("%s: security requirement names %q, which is not a declared security scheme", where, name)
				}
			}
		}
	}
	checkSecurity("document", doc.Security)

	refs := map[string][]string{}
	operationIDs := map[string]string{}

	if doc.Paths != nil {
		for _, path := range slices.Sorted(maps.Keys(doc.Paths.PathItems)) {
			item := doc.Paths.PathItems[path]
			templated := map[string]bool{}
			for _, m := range pathTemplateParam.FindAllStringSubmatch(path, -1) {
				templated[m[1]] = true
			}

			for _, op := range operations(item) {
				where := op.method + " " + path
				if op.OperationID != "" {
					if prev, dup := operationIDs[op.OperationID]; dup {
						add("%s: operationId %q is also used by %s", where, op.OperationID, prev)
					}
					operationIDs[op.OperationID] = where
				}
				if op.Responses == nil || (op.Responses.Default == nil && len(op.Responses.StatusCodes) == 0) {
					add("%s: declares no response; an operation needs at least one", where)
				}
				checkSecurity(where, op.Security)

				declared := map[string]bool{}
				seen := map[string]bool{}
				for _, p := range append(slices.Clone(item.Parameters), op.Parameters...) {
					if p == nil || p.Ref != "" {
						continue
					}
					key := p.In + ":" + p.Name
					if seen[key] {
						add("%s: parameter %q in %s is declared twice", where, p.Name, p.In)
					}
					seen[key] = true
					if p.In == "path" {
						declared[p.Name] = true
						if !templated[p.Name] {
							add("%s: path parameter %q does not appear in the path", where, p.Name)
						}
						if !p.Required {
							add("%s: path parameter %q must be required", where, p.Name)
						}
					}
					collectRefs(p.Schema, where, refs)
				}
				for _, name := range slices.Sorted(maps.Keys(templated)) {
					if !declared[name] {
						add("%s: the path names {%s}, but no path parameter declares it", where, name)
					}
				}

				if op.RequestBody != nil {
					for _, mt := range op.RequestBody.Content {
						collectRefs(mt.Schema, where, refs)
					}
				}
				if op.Responses != nil {
					responses := slices.Collect(maps.Values(op.Responses.StatusCodes))
					if op.Responses.Default != nil {
						responses = append(responses, op.Responses.Default)
					}
					for _, resp := range responses {
						for _, mt := range resp.Content {
							collectRefs(mt.Schema, where, refs)
						}
					}
				}
			}
		}
	}

	for _, name := range slices.Sorted(maps.Keys(schemas)) {
		collectRefs(schemas[name], "schema "+name, refs)
	}
	for _, ref := range slices.Sorted(maps.Keys(refs)) {
		name, ok := strings.CutPrefix(ref, "#/components/schemas/")
		if ok && schemas[name] != nil {
			continue
		}
		add("%s: $ref %q resolves to nothing", refs[ref][0], ref)
	}

	slices.Sort(problems)
	return problems
}

type namedOperation struct {
	*spec.Operation
	method string
}

// operations returns a path item's operations with their methods.
func operations(item *spec.PathItem) []namedOperation {
	var ops []namedOperation
	for _, op := range []namedOperation{
		{item.Get, "GET"}, {item.Put, "PUT"}, {item.Post, "POST"}, {item.Delete, "DELETE"},
		{item.Options, "OPTIONS"}, {item.Head, "HEAD"}, {item.Patch, "PATCH"}, {item.Trace, "TRACE"},
	} {
		if op.Operation != nil {
			ops = append(ops, op)
		}
	}
	return ops
}

// collectRefs records every $ref under a schema, with where it was found.
func collectRefs(schema *spec.Schema, where string, refs map[string][]string) {
	if schema == nil {
		return
	}
	if schema.Ref != "" {
		refs[schema.Ref] = append(refs[schema.Ref], where)
		return
	}
	for _, name := range slices.Sorted(maps.Keys(schema.Properties)) {
		collectRefs(schema.Properties[name], where, refs)
	}
	for _, s := range [][]*spec.Schema{schema.AllOf, schema.OneOf, schema.AnyOf, schema.PrefixItems} {
		for _, child := range s {
			collectRefs(child, where, refs)
		}
	}
	for _, child := range []*spec.Schema{
		schema.Items, schema.AdditionalProperties, schema.Not, schema.If, schema.Then, schema.Else,
		schema.Contains, schema.UnevaluatedProperties, schema.UnevaluatedItems,
	} {
		collectRefs(child, where, refs)
	}
}
