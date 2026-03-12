package scanner

import (
	"go/ast"
	"go/token"
	"log"
	"strconv"

	"golang.org/x/tools/go/packages"
)

// processEnums processes swagger:enum directives in two passes:
// 1. First pass: register enum type declarations
// 2. Second pass: process const declarations to extract enum values
func (s *Scanner) processEnums(filePath string, file *ast.File, pkg *packages.Package) error {
	// First pass: process enum type declarations
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// Check for swagger:enum directive in either genDecl.Doc or typeSpec.Doc
			doc := genDecl.Doc
			if doc == nil {
				doc = typeSpec.Doc
			}
			if doc == nil || !hasDirective(doc, EnumDirective) {
				continue
			}

			enumInfo := parseEnumTypeDeclaration(typeSpec, filePath, doc)
			if enumInfo != nil {
				s.Enums[enumInfo.TypeName] = enumInfo
				s.TypeToEnum[typeSpec.Name.Name] = enumInfo.TypeName
				if pkg != nil && pkg.Types != nil {
					s.TypeToEnum[pkg.Types.Name()+"."+typeSpec.Name.Name] = enumInfo.TypeName
				}
				s.EnumSources[enumInfo.TypeName] = filePath
			}
		}
	}

	// Second pass: process const declarations to extract enum values
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}

		s.processEnumConstDeclaration(filePath, genDecl)
	}

	return nil
}

// processEnumConstDeclaration processes a const declaration block to extract enum values.
func (s *Scanner) processEnumConstDeclaration(filePath string, genDecl *ast.GenDecl) {
	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok || valueSpec.Type == nil || len(valueSpec.Values) == 0 {
			continue
		}

		// Get the type name
		typeIdent, ok := valueSpec.Type.(*ast.Ident)
		if !ok {
			continue
		}

		// Check if this type is a registered enum
		enumName, ok := s.TypeToEnum[typeIdent.Name]
		if !ok {
			continue
		}

		// Check if this enum was defined in the same file
		// This prevents mixing enum values from different packages with same enum name
		if enumSource, exists := s.EnumSources[enumName]; exists && enumSource != filePath {
			continue
		}

		enumInfo := s.Enums[enumName]
		if enumInfo == nil {
			continue
		}

		// Process each constant in this spec
		for i, name := range valueSpec.Names {
			if i >= len(valueSpec.Values) {
				break
			}

			// Extract the value
			value := extractConstValue(valueSpec.Values[i])
			if value == nil {
				log.Printf("warning: skipping unparseable enum value for %q in %s", name.Name, filePath)
				continue
			}

			// Add the value to the enum
			enumInfo.Values[name.Name] = value

			// Set as example if not set yet
			if enumInfo.Example == nil {
				enumInfo.Example = value
			}
		}
	}
}

// parseEnumTypeDeclaration parses an enum type declaration (without const values).
func parseEnumTypeDeclaration(typeSpec *ast.TypeSpec, filePath string, doc *ast.CommentGroup) *EnumInfo {
	// Get the base type (e.g., string, int)
	baseType := "string"
	if ident, ok := typeSpec.Type.(*ast.Ident); ok {
		baseType = ident.Name
	}

	// Get optional name from directive
	name := extractDirectiveValue(doc, EnumDirective)
	if name == "" {
		name = typeSpec.Name.Name
	}

	// Extract example from doc
	example := extractDirectiveValue(doc, ExampleDirective)

	enumInfo := &EnumInfo{
		TypeName:    name,
		BaseType:    baseType,
		Values:      make(map[string]any),
		Description: extractDescription(doc, []string{SwaggerPrefix, ExampleDirective}),
		SourceFile:  filePath,
	}

	if example != "" {
		enumInfo.Example = example
	}

	return enumInfo
}

// extractConstValue extracts the value from a const expression.
func extractConstValue(expr ast.Expr) any {
	switch v := expr.(type) {
	case *ast.BasicLit:
		switch v.Kind {
		case token.STRING:
			s, _ := strconv.Unquote(v.Value)
			return s
		case token.INT:
			i, _ := strconv.ParseInt(v.Value, 10, 64)
			return i
		case token.FLOAT:
			f, _ := strconv.ParseFloat(v.Value, 64)
			return f
		default:
			return nil
		}
	case *ast.Ident:
		return v.Name
	}
	return nil
}
