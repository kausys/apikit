package scanner

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

// indexDeclared records every struct type a file declares.
func (s *Scanner) indexDeclared(filePath string, file *ast.File, pkg *packages.Package) {
	if pkg == nil {
		return
	}
	imports := extractFileImports(file)
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
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			doc := typeSpec.Doc
			if doc == nil {
				doc = genDecl.Doc
			}
			s.Declared[pkg.PkgPath+"."+typeSpec.Name.Name] = &DeclaredStruct{
				Name:    typeSpec.Name.Name,
				PkgPath: pkg.PkgPath,
				File:    filePath,
				Doc:     doc,
				Type:    structType,
				Imports: imports,
			}
		}
	}
}

// RequestStruct returns the parameters struct of a route declared on a
// handler: its request type, read with the binding tags a generated handler
// uses. Nil when the route names no request type or the scan did not see it.
func (s *Scanner) RequestStruct(route *RouteInfo) *StructInfo {
	if route.RequestType == "" {
		return nil
	}
	if cached, ok := s.requestStructs[route.RequestType]; ok {
		return cached
	}
	decl, ok := s.Declared[route.RequestType]
	if !ok {
		return nil
	}
	info := &StructInfo{
		Name:        decl.Name,
		Fields:      []*FieldInfo{},
		Description: extractDescription(decl.Doc, []string{SwaggerPrefix}),
		IsParameter: true,
		SourceFile:  decl.File,
	}
	processStructFields(info, decl.Type, decl.Imports)
	s.qualifyFields(info, decl.PkgPath)
	s.requestStructs[route.RequestType] = info
	return info
}

// EnsureModel makes the struct a field refers to a model, when the scan saw
// its declaration and nothing names it yet, and returns the model's name. A
// request body's type, and the structs its fields refer to, need no
// swagger:model directive: being the body of a documented route is reason
// enough to document them. "" when the type is not a struct the scan saw.
func (s *Scanner) EnsureModel(f *FieldInfo) string {
	return s.ensureModel(f.QualifiedType, map[string]bool{})
}

func (s *Scanner) ensureModel(qualified string, visiting map[string]bool) string {
	decl, ok := s.Declared[qualified]
	if !ok {
		return ""
	}
	if name, ok := s.TypeToStruct[decl.Name]; ok {
		if existing := s.Structs[name]; existing != nil && existing.SourceFile == decl.File {
			return name
		}
	}
	if visiting[qualified] {
		return decl.Name
	}
	visiting[qualified] = true

	name := decl.Name
	if existing, taken := s.Structs[name]; taken && existing.SourceFile != decl.File {
		// Another package's model has the name: qualify this one.
		name = lastSegment(decl.PkgPath) + decl.Name
	}
	info := &StructInfo{
		Name:           name,
		Fields:         []*FieldInfo{},
		Description:    extractDescription(decl.Doc, []string{SwaggerPrefix}),
		IsModel:        true,
		SourceFile:     decl.File,
		UnderlyingKind: KindStruct,
	}
	processStructFields(info, decl.Type, decl.Imports)
	s.qualifyFields(info, decl.PkgPath)
	s.Structs[name] = info
	s.StructSources[name] = decl.File
	if _, taken := s.TypeToStruct[decl.Name]; !taken {
		s.TypeToStruct[decl.Name] = name
	}

	// The structs its fields refer to are documented with it.
	for _, field := range info.Fields {
		if field.QualifiedType == "" {
			continue
		}
		if ref := s.ensureModel(field.QualifiedType, visiting); ref != "" && ref != shortTypeName(field.Type) {
			field.Type = ref
		}
	}
	return name
}

// qualifyFields gives a field of a bare type the qualified name of the type
// in the package that declares it, so the struct it refers to can be found.
func (s *Scanner) qualifyFields(info *StructInfo, pkgPath string) {
	for _, field := range info.Fields {
		if field.QualifiedType == "" && field.Type != "" && !strings.Contains(field.Type, ".") {
			if _, ok := s.Declared[pkgPath+"."+field.Type]; ok {
				field.QualifiedType = pkgPath + "." + field.Type
			}
		}
	}
}

func lastSegment(path string) string {
	if _, last, ok := strings.CutLast(path, "/"); ok {
		return last
	}
	return path
}

func shortTypeName(typeName string) string {
	if _, name, ok := strings.Cut(typeName, "."); ok {
		return name
	}
	return typeName
}
