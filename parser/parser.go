package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Parser analyzes Go source files to find apikit handlers
type Parser struct {
	fset               *token.FileSet
	structs            map[string]*Struct           // Cache of parsed structs
	externalStructs    map[string]*Struct           // Cache of external structs by "importPath.StructName"
	structImportsCache map[string]map[string]string // Cache of imports for each external struct
	parsedFiles        map[string]bool              // Cache of already parsed files
	fileImportsCache   map[string]map[string]string // Cache of file imports by file path
	importPathCache    map[string]string            // Cache of resolved import paths
	goModPath          string                       // Cached path to go.mod
	moduleName         string                       // Cached module name from go.mod
	currentDir         string                       // Directory of the file being parsed
}

// New creates a new Parser instance
func New() *Parser {
	return &Parser{
		fset:               token.NewFileSet(),
		structs:            make(map[string]*Struct),
		externalStructs:    make(map[string]*Struct),
		structImportsCache: make(map[string]map[string]string),
		parsedFiles:        make(map[string]bool),
		fileImportsCache:   make(map[string]map[string]string),
		importPathCache:    make(map[string]string),
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

	// Collect imports for resolving external types
	imports := make(map[string]string) // alias -> import path
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		alias := filepath.Base(path)
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		imports[alias] = path
	}

	// First pass: collect all struct definitions
	ast.Inspect(file, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				s := p.parseStruct(typeSpec.Name.Name, structType, typeSpec)
				result.Structs[s.Name] = s
				p.structs[s.Name] = s // Cache for nested resolution
			}
		}
		return true
	})

	// Resolve nested structs in all fields with circular reference detection
	for _, s := range result.Structs {
		visited := make(map[string]bool)
		p.resolveNestedStructsRecursive(s, visited, imports)
	}

	// Second pass: find handlers
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if handler := p.parseHandler(funcDecl, file.Name.Name, result); handler != nil {
				result.Handlers = append(result.Handlers, *handler)
			}
		}
		return true
	})

	return result, nil
}

// parseHandler checks if a function is a handler and extracts its information
func (p *Parser) parseHandler(fn *ast.FuncDecl, pkgName string, result *ParseResult) *Handler {
	// Check for apikit:handler comment
	if !hasApikitComment(fn) {
		return nil
	}

	// Validate handler signature
	if !p.isValidHandlerSignature(fn) {
		pos := p.fset.Position(fn.Pos())
		warning := fmt.Sprintf("%s: function %s has apikit:handler comment but invalid signature",
			pos, fn.Name.Name)
		result.Warnings = append(result.Warnings, warning)
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

	// Look up struct info
	structName := p.getTypeName(params[1].Type)
	if s, ok := result.Structs[structName]; ok {
		h.Struct = s
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
// with circular reference detection
func (p *Parser) resolveNestedStructsRecursive(s *Struct, visited map[string]bool, imports map[string]string) {
	// Prevent infinite recursion by tracking visited structs
	if visited[s.Name] {
		return
	}
	visited[s.Name] = true

	for i := range s.Fields {
		field := &s.Fields[i]

		// Skip special fields
		if field.IsRawBody || field.IsResponseWriter || field.IsRequest {
			continue
		}

		// Get the base type name (without pointer/slice)
		typeName := field.Type
		if field.IsPointer {
			typeName = strings.TrimPrefix(typeName, "*")
		}
		if field.IsSlice {
			typeName = field.SliceType
		}

		// Extract package path from type if it contains a dot
		pkgAlias := ""
		structName := typeName
		if strings.Contains(typeName, ".") {
			parts := strings.Split(typeName, ".")
			if len(parts) == 2 {
				pkgAlias = parts[0]
				structName = parts[1]
				field.PackagePath = pkgAlias
			}
		}

		// Try to resolve from cache first
		if nestedStruct, ok := p.structs[typeName]; ok {
			// Check for circular reference before copying
			if visited[typeName] {
				// Circular reference detected - don't resolve further
				// Just set the basic struct info without fields
				field.NestedStruct = &Struct{
					Name:  nestedStruct.Name,
					IsDTO: nestedStruct.IsDTO,
					// Fields intentionally left empty to break the cycle
				}
				continue
			}

			// Make a copy to avoid shared references
			field.NestedStruct = &Struct{
				Name:   nestedStruct.Name,
				Fields: make([]Field, len(nestedStruct.Fields)),
				IsDTO:  nestedStruct.IsDTO,
			}
			copy(field.NestedStruct.Fields, nestedStruct.Fields)

			// Recursively resolve nested structs within this struct
			// Use cached imports if available, otherwise use current imports
			structImports := imports
			if cachedImports, ok := p.structImportsCache[typeName]; ok {
				structImports = cachedImports
			}
			p.resolveNestedStructsRecursive(field.NestedStruct, visited, structImports)
			continue
		}

		// If not in cache and has package prefix, try to load from external package
		if pkgAlias != "" {
			if importPath, ok := imports[pkgAlias]; ok {
				externalStruct, externalImports := p.loadExternalStruct(importPath, structName)
				if externalStruct != nil {
					// Cache it with full name for future lookups
					p.structs[typeName] = externalStruct
					// Cache the imports for this struct
					p.structImportsCache[typeName] = externalImports

					// Check for circular reference
					if visited[typeName] {
						field.NestedStruct = &Struct{
							Name:  externalStruct.Name,
							IsDTO: externalStruct.IsDTO,
						}
						continue
					}

					// Make a copy
					field.NestedStruct = &Struct{
						Name:   externalStruct.Name,
						Fields: make([]Field, len(externalStruct.Fields)),
						IsDTO:  externalStruct.IsDTO,
					}
					copy(field.NestedStruct.Fields, externalStruct.Fields)

					// Recursively resolve nested structs within the external struct
					// Use the imports from the external file
					p.resolveNestedStructsRecursive(field.NestedStruct, visited, externalImports)
				}
			}
		}
	}
}

// loadExternalStruct loads a struct from an external package using fast file-based parsing
// Returns the struct and the imports from the file where it was found
func (p *Parser) loadExternalStruct(importPath, structName string) (*Struct, map[string]string) {
	// Check cache first
	cacheKey := importPath + "." + structName
	if cached, ok := p.externalStructs[cacheKey]; ok {
		// Return cached struct with its cached imports
		cachedImports := p.structImportsCache[cacheKey]
		return cached, cachedImports
	}

	// Resolve the import path to a directory
	pkgDir := p.resolveImportPath(importPath)
	if pkgDir == "" {
		return nil, nil
	}

	// Find all Go files in the package directory
	files, err := filepath.Glob(filepath.Join(pkgDir, "*.go"))
	if err != nil {
		return nil, nil
	}

	// Parse each file looking for the struct
	for _, file := range files {
		// Skip test files
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		// Check if file was already parsed
		if cachedImports, ok := p.fileImportsCache[file]; ok {
			// File already parsed, check if struct is in cache
			structCacheKey := importPath + "." + structName
			if cachedStruct, ok := p.externalStructs[structCacheKey]; ok {
				return cachedStruct, cachedImports
			}
			// Struct not in this file, skip it
			continue
		}

		// Parse the file
		src, err := parser.ParseFile(p.fset, file, nil, parser.ParseComments)
		if err != nil {
			continue
		}

		// Collect imports from this file
		fileImports := make(map[string]string)
		for _, imp := range src.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			alias := filepath.Base(path)
			if imp.Name != nil {
				alias = imp.Name.Name
			}
			fileImports[alias] = path
		}

		// Cache the imports
		p.fileImportsCache[file] = fileImports

		// Parse ALL structs in this file and cache them
		// This way we only parse each file once
		var foundStruct *Struct
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

				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				// Parse and cache this struct
				s := p.parseStruct(typeSpec.Name.Name, structType, typeSpec)
				structCacheKey := importPath + "." + typeSpec.Name.Name
				p.externalStructs[structCacheKey] = s
				// Cache the imports for this struct
				p.structImportsCache[structCacheKey] = fileImports

				// Check if this is the struct we're looking for
				if typeSpec.Name.Name == structName {
					foundStruct = s
				}
			}
		}

		// Mark as parsed AFTER we've cached all structs
		p.parsedFiles[file] = true

		// If we found the struct we were looking for, return it
		if foundStruct != nil {
			return foundStruct, fileImports
		}
	}

	return nil, nil
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
		lines := strings.Split(string(modContent), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "module ") {
				p.moduleName = strings.TrimPrefix(line, "module ")
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
	if strings.HasPrefix(comment, "in:") {
		value := strings.TrimPrefix(comment, "in:")
		value = strings.TrimSpace(value)

		// Check if value contains quoted string (single quotes)
		// Example: "header 'User-Agent'" or "header 'X-Custom Header'"
		if strings.Contains(value, "'") {
			// Find the first space (separates source from quoted name)
			spaceIdx := strings.Index(value, " ")
			if spaceIdx == -1 {
				// No space, just return the source
				return strings.Trim(value, "'"), ""
			}

			source := value[:spaceIdx]
			rest := strings.TrimSpace(value[spaceIdx+1:])

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
	if strings.HasPrefix(comment, "default:") {
		value := strings.TrimPrefix(comment, "default:")
		return strings.TrimSpace(value)
	}

	return ""
}
