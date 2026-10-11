package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Parser analyzes Go source files to find apikit handlers
type Parser struct {
	fset            *token.FileSet
	packages        map[string]*pkgScope         // Structs of each package parsed so far, by directory
	importPathCache map[string]string            // Cache of resolved import paths
	goModPath       string                       // Cached path to go.mod
	moduleName      string                       // Cached module name from go.mod
	currentDir      string                       // Directory of the file being parsed
	consulted       map[string]bool              // Packages the file being parsed read a struct from
	typed           map[string]*packages.Package // Type-checked packages, by directory
}

// pkgScope is what one package declares: its structs by name, and for each
// struct the imports of the file it is declared in, which is what the
// qualified types among its fields resolve against.
//
// Every lookup goes through the package a type belongs to. Structs used to be
// cached by bare name across every file of a run, so a field of type Body in
// one package could resolve to another package's Body — whichever was parsed
// last — and its wrapper bound the wrong fields with nothing reported.
type pkgScope struct {
	dir     string
	structs map[string]*Struct
	imports map[string]map[string]string
}

// New creates a new Parser instance
func New() *Parser {
	return &Parser{
		fset:            token.NewFileSet(),
		packages:        make(map[string]*pkgScope),
		importPathCache: make(map[string]string),
		typed:           make(map[string]*packages.Package),
	}
}

// ParseFile analyzes a single Go file and extracts handler information
func (p *Parser) ParseFile(filename string) (*ParseResult, error) {
	// Store the directory of the file being parsed
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("getting absolute path: %w", err)
	}
	p.currentDir = filepath.Dir(absPath)
	p.consulted = map[string]bool{p.currentDir: true}

	// Parse the file
	file, err := parser.ParseFile(p.fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}

	result := &ParseResult{
		Handlers: []Handler{},
		Structs:  make(map[string]*Struct),
		Source: Source{
			Filename: filename,
			Package:  file.Name.Name,
		},
		Warnings: []string{},
	}

	imports := fileImports(file)

	// The structs this file declares, as before; the payload of a handler is
	// looked up in the whole package below.
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			if typeSpec, ok := spec.(*ast.TypeSpec); ok {
				if structType, ok := typeSpec.Type.(*ast.StructType); ok {
					st := p.parseStruct(typeSpec.Name.Name, structType, typeSpec)
					p.resolveNestedStructsRecursive(st, p.packageScope(p.currentDir), imports, map[string]bool{})
					result.Structs[st.Name] = st
				}
			}
		}
	}

	// Second pass: find handlers
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if handler := p.parseHandler(funcDecl, file.Name.Name, imports, result); handler != nil {
				result.Handlers = append(result.Handlers, *handler)
			}
		}
		return true
	})

	result.Packages = slices.Sorted(maps.Keys(p.consulted))
	return result, nil
}

// parseHandler checks if a function is a handler and extracts its information
func (p *Parser) parseHandler(fn *ast.FuncDecl, pkgName string, imports map[string]string, result *ParseResult) *Handler {
	// Check for apikit:handler comment
	if !hasApikitComment(fn) {
		return nil
	}

	// Validate handler signature.
	//
	// An ERROR, not a warning. The author asked for a wrapper by writing the
	// annotation; dropping the handler silently means the call site
	// (`fooAPIKit(h.foo)`) has nothing to refer to, and that surfaces as an
	// undefined-symbol error in a different file with nothing pointing back to
	// the cause. It was also only ever printed under --verbose.
	if !p.isValidHandlerSignature(fn) {
		pos := p.fset.Position(fn.Pos())
		result.Errors = append(result.Errors, fmt.Sprintf(
			"%s: %s has an apikit:handler comment but an invalid signature "+
				"(want func(context.Context, Payload) (T, error), optionally "+
				"followed by http.ResponseWriter and/or *http.Request)",
			pos, fn.Name.Name))
		return nil
	}

	h := &Handler{
		Name:    fn.Name.Name,
		Package: pkgName,
		Pos:     p.fset.Position(fn.Pos()),
	}

	// Handle receiver for methods
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		h.Receiver = p.typeToString(fn.Recv.List[0].Type)
	}

	// Get parameter type (second parameter)
	// Note: isValidHandlerSignature already verified len(params.List) >= 2
	// but we add defensive check for robustness
	params := fn.Type.Params.List
	if len(params) < 2 {
		// This should never happen due to isValidHandlerSignature check
		// but we handle it defensively
		pos := p.fset.Position(fn.Pos())
		warning := fmt.Sprintf("%s: function %s has insufficient parameters",
			pos, fn.Name.Name)
		result.Warnings = append(result.Warnings, warning)
		return nil
	}
	h.ParamType = p.typeToString(params[1].Type)

	// Check for optional http.ResponseWriter and *http.Request parameters
	if len(params) > 2 {
		for i := 2; i < len(params); i++ {
			if p.isResponseWriterType(params[i].Type) {
				h.HasResponseWriter = true
			} else if p.isRequestType(params[i].Type) {
				h.HasRequest = true
			}
		}
	}

	// Look up struct info, in the package the payload type belongs to: this
	// one for a bare name, the imported one for a qualified name.
	//
	// Not finding it is an ERROR, and it is the most dangerous condition this
	// parser can hit. With h.Struct nil, prepareHandlerData returns early and the
	// template emits a parse function whose whole body is `return nil`: no path
	// or query extraction, no body unmarshal, no validation. That wrapper
	// COMPILES and registers on its route, so the failure is a handler receiving
	// a zero payload in production with nothing having gone wrong at build time.
	payloadType := strings.TrimPrefix(h.ParamType, "*")
	s, importPath, alias := p.lookupStruct(payloadType, p.packageScope(p.currentDir), imports)
	if s == nil {
		result.Errors = append(result.Errors, fmt.Sprintf(
			"%s: %s takes payload %q, which is not a struct apikit can find — "+
				"it must be declared in this package, or in a package of this "+
				"module imported by this file (a type it cannot read would "+
				"silently generate a wrapper that parses nothing)",
			p.fset.Position(fn.Pos()), fn.Name.Name, payloadType))
		return nil
	}
	h.Struct = s

	// A handler that takes the *http.Request may read the body itself; one
	// that does not has only what the wrapper fills.
	if !h.HasRequest {
		if unbound := unboundJSONFields(s); len(unbound) > 0 {
			for _, name := range unbound {
				result.Errors = append(result.Errors, fmt.Sprintf(
					"%s: %s takes %s, whose field %s has a json name but nothing fills it — "+
						"the wrapper decodes the body only into a field tagged json:\"body\" "+
						"(or marked // in: body); move it into that field's struct, or bind "+
						"it with a path, query, header, cookie or form tag",
					p.fset.Position(fn.Pos()), fn.Name.Name, payloadType, name))
			}
			return nil
		}
	}
	if importPath != "" {
		h.ParamImport = importPath
		h.ParamImportAlias = alias
	}

	// Get return type (first return value)
	// Note: isValidHandlerSignature already verified len(results.List) == 2
	// but we add defensive check for robustness
	results := fn.Type.Results.List
	if len(results) < 1 {
		// This should never happen due to isValidHandlerSignature check
		pos := p.fset.Position(fn.Pos())
		warning := fmt.Sprintf("%s: function %s has no return values",
			pos, fn.Name.Name)
		result.Warnings = append(result.Warnings, warning)
		return nil
	}
	h.ReturnType = p.typeToString(results[0].Type)

	return h
}

// parseStruct extracts struct field information
func (p *Parser) parseStruct(name string, st *ast.StructType, typeSpec *ast.TypeSpec) *Struct {
	s := &Struct{
		Name:   name,
		Fields: []Field{},
	}

	// Check for apikit:dto comment
	if typeSpec != nil && typeSpec.Doc != nil {
		for _, comment := range typeSpec.Doc.List {
			if strings.Contains(comment.Text, "apikit:dto") {
				s.IsDTO = true
				break
			}
		}
	}

	// Parse fields
	for _, field := range st.Fields.List {
		fields := p.parseField(field)
		s.Fields = append(s.Fields, fields...)
	}

	return s
}

// parseField extracts field information including tags
func (p *Parser) parseField(field *ast.Field) []Field {
	var fields []Field

	fieldType := p.typeToString(field.Type)
	isPointer := false
	isSlice := false
	sliceType := ""

	// Check if it's a pointer
	if _, ok := field.Type.(*ast.StarExpr); ok {
		isPointer = true
	}

	// Check if it's a slice
	if arrayType, ok := field.Type.(*ast.ArrayType); ok && arrayType.Len == nil {
		isSlice = true
		sliceType = p.typeToString(arrayType.Elt)
	}

	// Extract "// in:xxx" and "// default:xxx" comments
	inComment := ""
	inCommentName := ""
	defaultFromComment := ""
	isBody := false
	if field.Comment != nil {
		for _, comment := range field.Comment.List {
			// Extract "// in:xxx"
			if source, name := extractInComment(comment.Text); source != "" {
				inComment = source
				inCommentName = name
				if source == "body" {
					isBody = true
				}
			}
			// Extract "// default:xxx"
			if defaultVal := extractDefaultComment(comment.Text); defaultVal != "" {
				defaultFromComment = defaultVal
			}
		}
	}
	if field.Doc != nil {
		for _, comment := range field.Doc.List {
			// Extract "// in:xxx" (only if not found in Comment)
			if inComment == "" {
				if source, name := extractInComment(comment.Text); source != "" {
					inComment = source
					inCommentName = name
					if source == "body" {
						isBody = true
					}
				}
			}
			// Extract "// default:xxx" (only if not found in Comment)
			if defaultFromComment == "" {
				if defaultVal := extractDefaultComment(comment.Text); defaultVal != "" {
					defaultFromComment = defaultVal
				}
			}
		}
	}

	// Handle named fields
	if len(field.Names) > 0 {
		for _, name := range field.Names {
			f := Field{
				Name:          name.Name,
				Type:          fieldType,
				IsPointer:     isPointer,
				IsSlice:       isSlice,
				SliceType:     sliceType,
				IsBody:        isBody,
				InComment:     inComment,
				InCommentName: inCommentName,
			}

			// Check for special field types
			// More flexible RawBody detection: any []byte field with "body" in the name (case-insensitive)
			f.IsRawBody = fieldType == "[]byte" && (name.Name == FieldNameRawBody ||
				name.Name == FieldNameRaw)

			// http.ResponseWriter aliases: ResponseWriter, Response, Writer, Res
			f.IsResponseWriter = (name.Name == FieldNameResponseWriter ||
				name.Name == FieldNameResponse ||
				name.Name == FieldNameWriter ||
				name.Name == FieldNameRes) &&
				fieldType == "http.ResponseWriter"

			// *http.Request aliases: Request, Req
			f.IsRequest = (name.Name == FieldNameRequest ||
				name.Name == FieldNameReq) &&
				fieldType == "*http.Request"

			// File upload fields: *multipart.FileHeader or []*multipart.FileHeader
			f.IsFile = fieldType == TypeFileHeader ||
				fieldType == TypeFileHeaderSlice

			// Store the complete struct tag
			if field.Tag != nil {
				f.StructTag = strings.Trim(field.Tag.Value, "`")
			}

			// Check if this is an inline struct type
			if structType, ok := field.Type.(*ast.StructType); ok {
				// Parse the inline struct
				inlineStruct := p.parseStruct(name.Name+"_inline", structType, nil)
				f.NestedStruct = inlineStruct
			}

			fields = append(fields, f)
		}
	} else {
		// Embedded field
		typeName := p.getTypeName(field.Type)
		f := Field{
			Name:          typeName,
			Type:          fieldType, // This contains the full type with package prefix
			IsEmbedded:    true,
			IsSlice:       isSlice,
			SliceType:     sliceType,
			IsBody:        isBody,
			InComment:     inComment,
			InCommentName: inCommentName,
		}

		// Store the complete struct tag
		if field.Tag != nil {
			f.StructTag = strings.Trim(field.Tag.Value, "`")
		}

		fields = append(fields, f)
	}

	return fields
}

// isValidHandlerSignature checks if function has the correct signature:
// func(context.Context, T) (R, error)
// func(context.Context, T, http.ResponseWriter) (R, error)
// func(context.Context, T, *http.Request) (R, error)
// func(context.Context, T, http.ResponseWriter, *http.Request) (R, error)
func (p *Parser) isValidHandlerSignature(fn *ast.FuncDecl) bool {
	// Check parameters: minimum (context.Context, T)
	params := fn.Type.Params
	if params == nil || len(params.List) < 2 || len(params.List) > 4 {
		return false
	}

	// First param must be context.Context
	if !p.isContextType(params.List[0].Type) {
		return false
	}

	// Optional third and fourth params can be http.ResponseWriter or *http.Request
	if len(params.List) > 2 {
		for i := 2; i < len(params.List); i++ {
			if !p.isResponseWriterType(params.List[i].Type) && !p.isRequestType(params.List[i].Type) {
				return false
			}
		}
	}

	// Check results: (T, error)
	results := fn.Type.Results
	if results == nil || len(results.List) != 2 {
		return false
	}

	// Second result must be error
	if !p.isErrorType(results.List[1].Type) {
		return false
	}

	return true
}

// isContextType checks if the type is context.Context
func (p *Parser) isContextType(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	x, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return x.Name == "context" && sel.Sel.Name == "Context"
}

// isErrorType checks if the type is error
func (p *Parser) isErrorType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "error"
}

// isResponseWriterType checks if the type is http.ResponseWriter
func (p *Parser) isResponseWriterType(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	x, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return x.Name == "http" && sel.Sel.Name == "ResponseWriter"
}

// isRequestType checks if the type is *http.Request
func (p *Parser) isRequestType(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}

	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	x, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return x.Name == "http" && sel.Sel.Name == "Request"
}

// typeToString converts an AST type expression to a string
func (p *Parser) typeToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + p.typeToString(e.X)
	case *ast.SelectorExpr:
		return p.typeToString(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		if e.Len == nil {
			return "[]" + p.typeToString(e.Elt)
		}
		return "[" + p.exprToString(e.Len) + "]" + p.typeToString(e.Elt)
	case *ast.MapType:
		return "map[" + p.typeToString(e.Key) + "]" + p.typeToString(e.Value)
	case *ast.InterfaceType:
		return "any"
	default:
		return ""
	}
}

// getTypeName extracts just the type name without package prefix or pointer
func (p *Parser) getTypeName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return p.getTypeName(e.X)
	case *ast.SelectorExpr:
		return e.Sel.Name
	default:
		return ""
	}
}

// exprToString converts an expression to string (for array lengths)
func (p *Parser) exprToString(expr ast.Expr) string {
	if lit, ok := expr.(*ast.BasicLit); ok {
		return lit.Value
	}
	return ""
}

// resolveNestedStructsRecursive resolves nested struct types for all fields recursively
// with circular reference detection. scope is the package s is declared in,
// imports the imports of its file.
func (p *Parser) resolveNestedStructsRecursive(s *Struct, scope *pkgScope, imports map[string]string, visited map[string]bool) {
	key := scope.dir + "." + s.Name
	if visited[key] {
		return
	}
	visited[key] = true
	defer delete(visited, key)

	for i := range s.Fields {
		field := &s.Fields[i]

		// Skip special fields
		if field.IsRawBody || field.IsResponseWriter || field.IsRequest {
			continue
		}

		// An inline struct is declared right here: its fields resolve in
		// the same scope.
		if field.NestedStruct != nil {
			p.resolveNestedStructsRecursive(field.NestedStruct, scope, imports, visited)
			continue
		}

		// Get the base type name (without pointer/slice)
		typeName := strings.TrimPrefix(field.Type, "*")
		if field.IsSlice {
			typeName = strings.TrimPrefix(field.SliceType, "*")
		}
		if alias, _, ok := strings.Cut(typeName, "."); ok {
			field.PackagePath = alias
		}

		nested, nestedScope, nestedImports := p.findStruct(typeName, scope, imports)
		if nested == nil {
			continue
		}

		// Circular reference: name the struct, leave its fields out.
		if visited[nestedScope.dir+"."+nested.Name] {
			field.NestedStruct = &Struct{Name: nested.Name, IsDTO: nested.IsDTO}
			continue
		}

		field.NestedStruct = nested.clone()
		p.resolveNestedStructsRecursive(field.NestedStruct, nestedScope, nestedImports, visited)
	}
}

// unboundJSONFields returns the payload fields that carry a json name and that
// no binding fills. The wrapper decodes the body into one field, the one named
// "body"; a json name anywhere else on the payload reads like a binding and is
// not one, so a struct of them produced a wrapper that parsed nothing.
func unboundJSONFields(s *Struct) []string {
	var names []string
	for _, f := range s.Fields {
		if f.IsEmbedded {
			if f.NestedStruct != nil {
				names = append(names, unboundJSONFields(f.NestedStruct)...)
			}
			continue
		}
		if f.IsBody || f.IsRawBody || f.IsRequest || f.IsResponseWriter || f.IsFile || f.InComment != "" {
			continue
		}
		tag := reflect.StructTag(f.StructTag)
		name, _, _ := strings.Cut(tag.Get(TagJSON), ",")
		if name == "" || name == "-" || name == "body" {
			continue
		}
		bound := false
		for _, source := range []string{TagPath, TagQuery, TagHeader, TagCookie, TagForm} {
			if _, ok := tag.Lookup(source); ok {
				bound = true
				break
			}
		}
		if !bound {
			names = append(names, f.Name)
		}
	}
	return names
}

// lookupStruct returns a resolved copy of the struct typeName names, seen from
// a file of scope with the given imports, and for a qualified name the import
// path and alias it came through.
func (p *Parser) lookupStruct(typeName string, scope *pkgScope, imports map[string]string) (*Struct, string, string) {
	found, foundScope, foundImports := p.findStruct(typeName, scope, imports)
	if found == nil {
		return nil, "", ""
	}
	s := found.clone()
	p.resolveNestedStructsRecursive(s, foundScope, foundImports, map[string]bool{})
	alias, _, qualified := strings.Cut(typeName, ".")
	if !qualified {
		return s, "", ""
	}
	return s, imports[alias], alias
}

// findStruct finds the declaration typeName names: in scope for a bare name,
// in the imported package for a qualified one. It returns the declaration, the
// package it is in and the imports of its file.
func (p *Parser) findStruct(typeName string, scope *pkgScope, imports map[string]string) (*Struct, *pkgScope, map[string]string) {
	if alias, name, ok := strings.Cut(typeName, "."); ok {
		importPath, known := imports[alias]
		if !known {
			return nil, nil, nil
		}
		dir := p.resolveImportPath(importPath)
		if dir == "" {
			return nil, nil, nil
		}
		scope = p.packageScope(dir)
		typeName = name
	}
	if p.consulted != nil {
		p.consulted[scope.dir] = true
	}
	s, ok := scope.structs[typeName]
	if !ok {
		return nil, nil, nil
	}
	return s, scope, scope.imports[typeName]
}

// packageScope parses the package in dir once and returns what it declares.
// Tests and generated wrappers are left out: neither declares a payload.
func (p *Parser) packageScope(dir string) *pkgScope {
	if scope, ok := p.packages[dir]; ok {
		return scope
	}
	scope := &pkgScope{
		dir:     dir,
		structs: make(map[string]*Struct),
		imports: make(map[string]map[string]string),
	}
	p.packages[dir] = scope

	files, _ := filepath.Glob(filepath.Join(dir, "*.go"))
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") || strings.HasSuffix(file, "_apikit.go") {
			continue
		}
		src, err := parser.ParseFile(p.fset, file, nil, parser.ParseComments)
		if err != nil {
			continue
		}
		imports := fileImports(src)
		for _, decl := range src.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if structType, ok := typeSpec.Type.(*ast.StructType); ok {
					scope.structs[typeSpec.Name.Name] = p.parseStruct(typeSpec.Name.Name, structType, typeSpec)
					scope.imports[typeSpec.Name.Name] = imports
				}
			}
		}
	}
	return scope
}

// fileImports maps each import of a file to its path, by the name the file
// uses for it.
func fileImports(file *ast.File) map[string]string {
	imports := make(map[string]string)
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		alias := filepath.Base(path)
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		imports[alias] = path
	}
	return imports
}

// clone copies a struct deeply enough that resolving its fields does not
// change the declaration it was copied from.
func (s *Struct) clone() *Struct {
	c := *s
	c.Fields = make([]Field, len(s.Fields))
	for i, f := range s.Fields {
		if f.NestedStruct != nil {
			f.NestedStruct = f.NestedStruct.clone()
		}
		c.Fields[i] = f
	}
	return &c
}

// resolveImportPath converts an import path to a filesystem directory
func (p *Parser) resolveImportPath(importPath string) string {
	// Check cache first
	if cached, ok := p.importPathCache[importPath]; ok {
		return cached
	}

	if p.currentDir == "" {
		return ""
	}

	// Find go.mod once and cache it
	if p.goModPath == "" {
		p.goModPath = p.findGoMod(p.currentDir)
		if p.goModPath == "" {
			return ""
		}
	}

	// Read module name once and cache it
	if p.moduleName == "" {
		modContent, err := os.ReadFile(p.goModPath)
		if err != nil {
			return ""
		}

		// Extract module name from first line: "module github.com/user/repo"
		lines := strings.SplitSeq(string(modContent), "\n")
		for line := range lines {
			line = strings.TrimSpace(line)
			if after, ok := strings.CutPrefix(line, "module "); ok {
				p.moduleName = after
				p.moduleName = strings.TrimSpace(p.moduleName)
				break
			}
		}

		if p.moduleName == "" {
			return ""
		}
	}

	// Check if import path starts with module name
	if !strings.HasPrefix(importPath, p.moduleName) {
		return ""
	}

	// Convert import path to relative path
	relPath := strings.TrimPrefix(importPath, p.moduleName)
	relPath = strings.TrimPrefix(relPath, "/")

	// Build full path
	moduleDir := filepath.Dir(p.goModPath)
	result := filepath.Join(moduleDir, relPath)

	// Cache the result
	p.importPathCache[importPath] = result

	return result
}

// findGoMod searches for go.mod file starting from dir and going up
func (p *Parser) findGoMod(dir string) string {
	for {
		modPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(modPath); err == nil {
			return modPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// hasApikitComment checks if a function has the apikit:handler comment
func hasApikitComment(fn *ast.FuncDecl) bool {
	if fn.Doc == nil {
		return false
	}

	for _, comment := range fn.Doc.List {
		if strings.Contains(comment.Text, DirectiveHandler) {
			return true
		}
	}

	return false
}

// extractInComment extracts the source and optional name from "// in:xxx" comment
// Returns: (source, name)
// Examples:
//   - "// in:query" -> ("query", "")
//   - "// in:path userId" -> ("path", "userId")
//   - "// in:header X-API-Key" -> ("header", "X-API-Key")
//   - "// in:header 'User-Agent'" -> ("header", "User-Agent")
//   - "// in:header 'X-Custom Header'" -> ("header", "X-Custom Header")
//   - "// in: body" -> ("body", "")
func extractInComment(comment string) (string, string) {
	// Remove comment markers
	comment = strings.TrimPrefix(comment, "//")
	comment = strings.TrimPrefix(comment, "/*")
	comment = strings.TrimSuffix(comment, "*/")
	comment = strings.TrimSpace(comment)

	// Check for "in:" prefix
	if after, ok := strings.CutPrefix(comment, "in:"); ok {
		value := after
		value = strings.TrimSpace(value)

		// Check if value contains quoted string (single quotes)
		// Example: "header 'User-Agent'" or "header 'X-Custom Header'"
		if strings.Contains(value, "'") {
			// Find the first space (separates source from quoted name)
			before, after, ok := strings.Cut(value, " ")
			if !ok {
				// No space, just return the source
				return strings.Trim(value, "'"), ""
			}

			source := before
			rest := strings.TrimSpace(after)

			// Extract quoted name
			if strings.HasPrefix(rest, "'") && strings.HasSuffix(rest, "'") {
				name := strings.Trim(rest, "'")
				return source, name
			}

			// If quotes are not properly closed, fall back to regular parsing
		}

		// Regular parsing (no quotes)
		// Split by space to get source and optional name
		parts := strings.Fields(value)
		if len(parts) == 0 {
			return "", ""
		}
		if len(parts) == 1 {
			return parts[0], ""
		}
		// parts[0] = source (query, path, header, etc.)
		// parts[1] = parameter name
		return parts[0], parts[1]
	}

	return "", ""
}

// extractDefaultComment extracts the default value from "// default:xxx" comment
// Returns: default value (empty string if not found)
// Examples:
//   - "// default:10" -> "10"
//   - "// default:true" -> "true"
//   - "// default:hello world" -> "hello world"
//   - "// default: 10" -> "10"
func extractDefaultComment(comment string) string {
	// Remove comment markers
	comment = strings.TrimPrefix(comment, "//")
	comment = strings.TrimPrefix(comment, "/*")
	comment = strings.TrimSuffix(comment, "*/")
	comment = strings.TrimSpace(comment)

	// Check for "default:" prefix
	if after, ok := strings.CutPrefix(comment, "default:"); ok {
		value := after
		return strings.TrimSpace(value)
	}

	return ""
}
