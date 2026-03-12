package sdkgen

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/kausys/apikit/openapi/sdkgen/templates"
)

// templateData is the data passed to each template.
type templateData struct {
	*SDKData
	Service   *ServiceData   // Set for service templates
	ModelFile *ModelFileData // Set for model templates
}

// render executes all templates with the SDKData and returns a map of filename → content.
func render(data *SDKData) (map[string][]byte, error) {
	funcMap := template.FuncMap{
		"pascal":     toPascalCase,
		"camel":      toCamelCase,
		"snake":      toSnakeCase,
		"lower":      strings.ToLower,
		"upper":      strings.ToUpper,
		"methodFunc": httpMethodToFunc,
		"zeroValue":  goZeroValue,
		"baseType":   goBaseType,
		"isPointer":  isPointerType,
		"isSlice":    isSliceType,
		"comment":    formatGoComment,
		"formatParam": func(p ParamData) string {
			return formatQueryValue(p.GoName, p.GoType)
		},
		"formatParamStruct": func(p ParamData) string {
			return formatQueryValue("params."+toPascalCase(p.GoName), p.GoType)
		},
		"isPrimitiveSlice": isPrimitiveSlice,
		"zapField":         zapFieldFunc,
		"zapFieldStruct": func(p ParamData) string {
			return zapFieldExpr(p.Name, "params."+toPascalCase(p.GoName), p.GoType)
		},
	}

	files := make(map[string][]byte)

	// 1. Config: config/config.go
	if err := renderTemplate(funcMap, "config.go.tmpl", "config/config.go", &templateData{SDKData: data}, files); err != nil {
		return nil, err
	}

	// 2. Models: models/<tag>.go
	for i := range data.Models {
		mf := &data.Models[i]
		fileName := "models/" + mf.FileName + ".go"
		td := &templateData{SDKData: data, ModelFile: mf}
		if err := renderTemplate(funcMap, "models.go.tmpl", fileName, td, files); err != nil {
			return nil, err
		}
	}

	// 3. Services: services/<tag>_service.go
	for i := range data.Services {
		svc := &data.Services[i]
		fileName := "services/" + svc.FileName + ".go"
		td := &templateData{SDKData: data, Service: svc}
		if err := renderTemplate(funcMap, "service.go.tmpl", fileName, td, files); err != nil {
			return nil, err
		}
	}

	// 4. SDK root: {provider}.go
	if err := renderTemplate(funcMap, "sdk.go.tmpl", data.Provider.Name+".go", &templateData{SDKData: data}, files); err != nil {
		return nil, err
	}

	// 5. Types: types.go
	if err := renderTemplate(funcMap, "types.go.tmpl", "types.go", &templateData{SDKData: data}, files); err != nil {
		return nil, err
	}

	return files, nil
}

// renderTemplate parses and executes a single template, adding the result to files.
func renderTemplate(funcMap template.FuncMap, tmplName, outPath string, data *templateData, files map[string][]byte) error {
	tmplBytes, err := templates.FS.ReadFile(tmplName)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", tmplName, err)
	}

	tmpl, err := template.New(tmplName).Funcs(funcMap).Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", tmplName, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", tmplName, err)
	}

	files[outPath] = buf.Bytes()
	return nil
}

// httpMethodToFunc converts HTTP method to resty method name.
func httpMethodToFunc(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return "Get"
	case "POST":
		return "Post"
	case "PUT":
		return "Put"
	case "DELETE":
		return "Delete"
	case "PATCH":
		return "Patch"
	case "HEAD":
		return "Head"
	case "OPTIONS":
		return "Options"
	default:
		return "Get"
	}
}

// goZeroValue returns the zero value for a Go type string.
func goZeroValue(goType string) string {
	if goType == "" {
		return ""
	}
	if strings.HasPrefix(goType, "models.") {
		return goType + "{}"
	}
	if strings.HasPrefix(goType, "*") || strings.HasPrefix(goType, "[]") || strings.HasPrefix(goType, "map[") {
		return "nil"
	}
	switch goType {
	case "string":
		return `""`
	case "int", "int32", "int64", "float32", "float64":
		return "0"
	case "bool":
		return "false"
	default:
		return "nil"
	}
}

// goBaseType strips pointer prefix from a type.
func goBaseType(goType string) string {
	return strings.TrimPrefix(goType, "*")
}

// isPointerType returns true if the type starts with *.
func isPointerType(goType string) bool {
	return strings.HasPrefix(goType, "*")
}

// isSliceType returns true if the type starts with [].
func isSliceType(goType string) bool {
	return strings.HasPrefix(goType, "[]")
}

// isPrimitiveSlice returns true if the type is a slice of primitive types (no .Raw field).
func isPrimitiveSlice(goType string) bool {
	if !strings.HasPrefix(goType, "[]") {
		return false
	}
	elemType := goType[2:]
	switch elemType {
	case "string", "int", "int32", "int64", "float32", "float64", "bool", "any":
		return true
	default:
		return false
	}
}

// formatGoComment converts a multi-line string into Go comment lines.
// Each line is prefixed with "// ", and empty lines become "//".
func formatGoComment(prefix, text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			out = append(out, "//")
		} else {
			out = append(out, "// "+prefix+line)
		}
		prefix = "" // only apply prefix to the first line
	}
	return strings.Join(out, "\n")
}

// zapFieldFunc returns a zap field expression for a ParamData using its GoName.
func zapFieldFunc(p ParamData) string {
	return zapFieldExpr(p.Name, p.GoName, p.GoType)
}

// zapFieldExpr returns the appropriate zap.Field call for a given type.
func zapFieldExpr(name, expr, goType string) string {
	switch goType {
	case "int":
		return fmt.Sprintf("zap.Int(%q, %s)", name, expr)
	case "int32":
		return fmt.Sprintf("zap.Int32(%q, %s)", name, expr)
	case "int64":
		return fmt.Sprintf("zap.Int64(%q, %s)", name, expr)
	case "float32":
		return fmt.Sprintf("zap.Float32(%q, %s)", name, expr)
	case "float64":
		return fmt.Sprintf("zap.Float64(%q, %s)", name, expr)
	case "bool":
		return fmt.Sprintf("zap.Bool(%q, %s)", name, expr)
	case "string":
		return fmt.Sprintf("zap.String(%q, %s)", name, expr)
	default:
		return fmt.Sprintf("zap.Any(%q, %s)", name, expr)
	}
}

// formatQueryValue returns a Go expression that converts the given variable to a string
// suitable for use as a query parameter value.
func formatQueryValue(expr, goType string) string {
	switch goType {
	case "string":
		return expr
	case "int":
		return "strconv.Itoa(" + expr + ")"
	case "int32":
		return "strconv.FormatInt(int64(" + expr + "), 10)"
	case "int64":
		return "strconv.FormatInt(" + expr + ", 10)"
	case "bool":
		return "strconv.FormatBool(" + expr + ")"
	default:
		return "fmt.Sprintf(\"%v\", " + expr + ")"
	}
}
