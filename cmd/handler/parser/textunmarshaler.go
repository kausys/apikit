package parser

import (
	"fmt"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// AnnotateTextUnmarshalers marks fields whose Go type implements
// encoding.TextUnmarshaler (value or pointer receiver). It uses packages.Load
// on the handler source file so aliases and methods defined in other packages
// (e.g. kernel.TenantID = ID[P]) are resolved correctly.
func AnnotateTextUnmarshalers(filename string, result *ParseResult) error {
	if result == nil || filename == "" {
		return nil
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps,
	}
	pkgs, err := packages.Load(cfg, "file="+filename)
	if err != nil {
		return fmt.Errorf("loading package for TextUnmarshaler detection: %w", err)
	}
	if len(pkgs) == 0 {
		return nil
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		// Type-check failures should not hard-fail generation; keep cast fallback.
		return nil
	}
	if pkg.Types == nil {
		return nil
	}

	iface := textUnmarshalerIface()
	for i := range result.Handlers {
		annotateStructFields(result.Handlers[i].Struct, pkg, iface)
	}
	return nil
}

// textUnmarshalerIface builds encoding.TextUnmarshaler's method set without
// loading the encoding package (avoids a second packages.Load).
func textUnmarshalerIface() *types.Interface {
	bytes := types.NewSlice(types.Typ[types.Byte])
	params := types.NewTuple(types.NewVar(token.NoPos, nil, "", bytes))
	results := types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Universe.Lookup("error").Type()))
	sig := types.NewSignatureType(nil, nil, nil, params, results, false)
	fn := types.NewFunc(token.NoPos, nil, "UnmarshalText", sig)
	return types.NewInterfaceType([]*types.Func{fn}, nil).Complete()
}

func annotateStructFields(s *Struct, pkg *packages.Package, iface *types.Interface) {
	if s == nil {
		return
	}
	for i := range s.Fields {
		f := &s.Fields[i]
		if f.NestedStruct != nil {
			annotateStructFields(f.NestedStruct, pkg, iface)
		}
		if f.IsEmbedded || f.IsBody || f.IsRawBody || f.IsFile || f.IsRequest || f.IsResponseWriter {
			continue
		}
		t := resolveFieldType(pkg, f)
		if t == nil {
			continue
		}
		if implementsTextUnmarshaler(t, iface) {
			f.ImplementsTextUnmarshaler = true
		}
	}
}

func resolveFieldType(pkg *packages.Package, f *Field) types.Type {
	typeName := f.Type
	if f.IsPointer {
		typeName = trimStar(typeName)
	}
	if f.IsSlice {
		typeName = f.SliceType
	}
	if typeName == "" {
		return nil
	}

	// Qualified: pkg.Type
	if pkgName, name, ok := splitQualified(typeName); ok {
		imp := findImport(pkg, pkgName)
		if imp == nil {
			return nil
		}
		obj := imp.Scope().Lookup(name)
		if obj == nil {
			return nil
		}
		return obj.Type()
	}

	// Same-package type
	obj := pkg.Types.Scope().Lookup(typeName)
	if obj == nil {
		return nil
	}
	return obj.Type()
}

func implementsTextUnmarshaler(t types.Type, iface *types.Interface) bool {
	t = types.Unalias(t)
	if types.Implements(t, iface) || types.Implements(types.NewPointer(t), iface) {
		return true
	}
	return false
}

func trimStar(s string) string {
	if len(s) > 0 && s[0] == '*' {
		return s[1:]
	}
	return s
}

func splitQualified(typeName string) (string, string, bool) {
	for i := range len(typeName) {
		if typeName[i] == '.' {
			return typeName[:i], typeName[i+1:], true
		}
	}
	return "", "", false
}

func findImport(pkg *packages.Package, pkgName string) *types.Package {
	for _, imp := range pkg.Imports {
		if imp.Types != nil && imp.Types.Name() == pkgName {
			return imp.Types
		}
	}
	if pkg.Types != nil {
		for _, imp := range pkg.Types.Imports() {
			if imp != nil && imp.Name() == pkgName {
				return imp
			}
		}
	}
	return nil
}
