package ast

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// Parser provides generic AST parsing with caching
type Parser struct {
	fset *token.FileSet
}

// New creates a new Parser instance
func New() *Parser {
	return &Parser{
		fset: token.NewFileSet(),
	}
}

// Parse parses a Go source file and returns generic AST information
// This method is completely agnostic to any specific directives
func (p *Parser) Parse(filename string) (*ParseResult, error) {
	// Parse file with comments
	file, err := parser.ParseFile(p.fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	result := &ParseResult{
		File:      file,
		Structs:   make(map[string]*Struct),
		Functions: []*Function{},
		Imports:   extractImports(file),
		Package:   file.Name.Name,
		Filename:  filename,
		FileSet:   p.fset,
	}

	// Extract all structs
	ast.Inspect(file, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if structType, ok := typeSpec.Type.(*ast.StructType); ok {
						s := p.parseStruct(typeSpec, structType, genDecl.Doc)
						result.Structs[s.Name] = s
					}
				}
			}
		}
		return true
	})

	// Extract all functions
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			f := p.parseFunction(funcDecl)
			result.Functions = append(result.Functions, f)
		}
		return true
	})

	return result, nil
}

// parseStruct extracts struct information
func (p *Parser) parseStruct(typeSpec *ast.TypeSpec, structType *ast.StructType, doc *ast.CommentGroup) *Struct {
	s := &Struct{
		Name:       typeSpec.Name.Name,
		TypeSpec:   typeSpec,
		StructType: structType,
		Doc:        doc,
		Comment:    typeSpec.Comment,
		Fields:     []*Field{},
		Pos:        p.fset.Position(typeSpec.Pos()),
	}

	// Parse all fields
	for _, field := range structType.Fields.List {
		fields := p.parseField(field)
		s.Fields = append(s.Fields, fields...)
	}

	return s
}

// parseField extracts field information
func (p *Parser) parseField(astField *ast.Field) []*Field {
	var fields []*Field

	fieldType := p.typeToString(astField.Type)
	isPointer := false
	isSlice := false
	sliceType := ""

	// Check if it's a pointer
	if _, ok := astField.Type.(*ast.StarExpr); ok {
		isPointer = true
	}

	// Check if it's a slice
	if arrayType, ok := astField.Type.(*ast.ArrayType); ok && arrayType.Len == nil {
		isSlice = true
		sliceType = p.typeToString(arrayType.Elt)
	}

	// Extract struct tag
	tag := ""
	if astField.Tag != nil {
		tag = strings.Trim(astField.Tag.Value, "`")
	}

	// Handle named fields
	if len(astField.Names) > 0 {
		for _, name := range astField.Names {
			f := &Field{
				Name:       name.Name,
				Type:       fieldType,
				ASTField:   astField,
				ASTType:    astField.Type,
				Doc:        astField.Doc,
				Comment:    astField.Comment,
				Tag:        tag,
				IsPointer:  isPointer,
				IsSlice:    isSlice,
				SliceType:  sliceType,
				IsEmbedded: false,
				Pos:        p.fset.Position(astField.Pos()),
			}
			fields = append(fields, f)
		}
	} else {
		// Embedded field
		typeName := p.getTypeName(astField.Type)
		f := &Field{
			Name:       typeName,
			Type:       fieldType,
			ASTField:   astField,
			ASTType:    astField.Type,
			Doc:        astField.Doc,
			Comment:    astField.Comment,
			Tag:        tag,
			IsPointer:  isPointer,
			IsSlice:    isSlice,
			SliceType:  sliceType,
			IsEmbedded: true,
			Pos:        p.fset.Position(astField.Pos()),
		}
		fields = append(fields, f)
	}

	return fields
}

// parseFunction extracts function information
func (p *Parser) parseFunction(funcDecl *ast.FuncDecl) *Function {
	f := &Function{
		Name:     funcDecl.Name.Name,
		FuncDecl: funcDecl,
		Doc:      funcDecl.Doc,
		Comment:  nil, // Functions don't have inline comments
		Params:   []*Param{},
		Results:  []*Param{},
		Pos:      p.fset.Position(funcDecl.Pos()),
	}

	// Extract receiver for methods
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		f.Receiver = p.typeToString(funcDecl.Recv.List[0].Type)
		f.ReceiverType = funcDecl.Recv.List[0].Type
	}

	// Extract parameters
	if funcDecl.Type.Params != nil {
		for _, param := range funcDecl.Type.Params.List {
			params := p.parseParams(param)
			f.Params = append(f.Params, params...)
		}
	}

	// Extract results
	if funcDecl.Type.Results != nil {
		for _, result := range funcDecl.Type.Results.List {
			results := p.parseParams(result)
			f.Results = append(f.Results, results...)
		}
	}

	return f
}

// parseParams extracts parameter information
func (p *Parser) parseParams(field *ast.Field) []*Param {
	var params []*Param

	paramType := p.typeToString(field.Type)
	isPointer := false
	isSlice := false
	isVariadic := false

	// Check if it's a pointer
	if _, ok := field.Type.(*ast.StarExpr); ok {
		isPointer = true
	}

	// Check if it's a slice
	if _, ok := field.Type.(*ast.ArrayType); ok {
		isSlice = true
	}

	// Check if it's variadic
	if _, ok := field.Type.(*ast.Ellipsis); ok {
		isVariadic = true
		isSlice = true
	}

	// Handle named parameters
	if len(field.Names) > 0 {
		for _, name := range field.Names {
			p := &Param{
				Name:       name.Name,
				Type:       paramType,
				ASTType:    field.Type,
				IsPointer:  isPointer,
				IsSlice:    isSlice,
				IsVariadic: isVariadic,
			}
			params = append(params, p)
		}
	} else {
		// Unnamed parameter
		p := &Param{
			Name:       "",
			Type:       paramType,
			ASTType:    field.Type,
			IsPointer:  isPointer,
			IsSlice:    isSlice,
			IsVariadic: isVariadic,
		}
		params = append(params, p)
	}

	return params
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
	case *ast.Ellipsis:
		return "..." + p.typeToString(e.Elt)
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

// extractImports extracts all imports from the file
func extractImports(file *ast.File) map[string]string {
	imports := make(map[string]string)

	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		alias := ""

		if imp.Name != nil {
			alias = imp.Name.Name
		} else {
			// Use last part of path as default alias
			parts := strings.Split(path, "/")
			alias = parts[len(parts)-1]
		}

		imports[alias] = path
	}

	return imports
}
