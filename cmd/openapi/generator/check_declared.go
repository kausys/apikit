package generator

import (
	"fmt"
	"slices"
	"strings"
)

// checkDeclaredSchemasResolve reports response types a route names that resolve
// to nothing the document can reference.
//
// typeToSchema tries the models, then the enums, then setSchemaType — whose
// default arm prints a warning and emits `type: string`. For a struct FIELD that
// is the right answer: a domain's own id type is a string on the wire, and every
// consumer has fields like it. For a name the author wrote in a `Responses:`
// directive it never is. `200: PaginatedAPIKeyResponse` names a model; a model
// that does not resolve leaves the operation returning a bare string, and
// --clean-unused then prunes the schema nobody references any more. The endpoint
// still works, so the loss surfaces in a generated client months later.
//
// The usual cause is a model the scan never reached while the route did: the two
// live in different packages, and an --ignore pattern or a package that failed to
// type-check took out only one of them. That is how a legacy `spec: dashboard`
// route lost a schema declared in a V3 package.
//
// A warning rather than an error, including under --strict, and deliberately so:
// the 346-handler consumer this was found in has four of these today, all
// predating the run that exposed them. The bar the other conditions were held to
// was that nothing generating cleanly starts failing, and this one does not clear
// it yet. What it does clear is being findable: one block naming the routes,
// instead of the per-type "unknown type" line that scrolls past among the dozens
// a normal run prints for domain id types. Promoting it to a --strict error is a
// follow-up, once consumers have a release to clean up against.
func (g *Generator) checkDeclaredSchemasResolve() error {
	var unresolved []string
	for _, route := range g.scanner.Routes {
		for _, resp := range route.Responses {
			if resp.Type == "" || !looksLikeModelRef(resp.Type) || g.declaredTypeResolves(resp.Type) {
				continue
			}
			unresolved = append(unresolved, fmt.Sprintf("%s %s → %s: %s (%s)",
				route.Method, route.Path, resp.StatusCode, resp.Type, route.SourceFile))
		}
	}
	if len(unresolved) == 0 {
		return nil
	}

	slices.Sort(unresolved)
	fmt.Printf(
		"warning: these routes declare a response type that resolves to no model, "+
			"enum or built-in, so the response is written as a bare `type: string` and "+
			"--clean-unused drops the schema:\n    %s\n  the model is usually in a "+
			"package the scan did not reach — check --ignore and any package that failed "+
			"to type-check — or the swagger:model name does not match\n",
		strings.Join(unresolved, "\n    "))
	return nil
}

// looksLikeModelRef reports whether name is plausibly a model the author meant
// to reference, rather than prose the directive parser read as a type.
//
// A `Responses:` line is free text after the status code, so `- 412: when the
// account is not ready` yields the type "when". Those are already written as
// `type: string` and always have been; treating them as unresolved models would
// fail a build for a comment that reads fine. A swagger:model is a Go type name,
// so the test is an exported identifier: leading upper case, then letters,
// digits, underscores, with one optional package qualifier.
func looksLikeModelRef(name string) bool {
	base := name
	if pkg, rest, ok := strings.Cut(name, "."); ok {
		if pkg == "" || rest == "" || strings.Contains(rest, ".") {
			return false
		}
		base = rest
	}
	if base == "" {
		return false
	}
	for i, r := range base {
		switch {
		case i == 0 && (r < 'A' || r > 'Z'):
			return false
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
		default:
			return false
		}
	}
	return true
}

// declaredTypeResolves mirrors what typeToSchema will accept, so the check and
// the conversion cannot disagree about what resolves.
func (g *Generator) declaredTypeResolves(typeName string) bool {
	if _, ok := g.resolveModelRef(typeName); ok {
		return true
	}
	if g.scanner.GetEnumForType(typeName) != nil {
		return true
	}
	if isKnownGoType(typeName) {
		return true
	}
	// setSchemaType's own escape for a package-qualified name whose bare form is
	// a known model (pkg.User -> User).
	if short := shortTypeName(typeName); short != typeName {
		if _, ok := g.scanner.Structs[short]; ok {
			return true
		}
	}
	return false
}
