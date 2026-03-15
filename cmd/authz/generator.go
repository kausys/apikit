package authz

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

// Config holds the configuration for the authz code generator.
type Config struct {
	InputCSV    string // path to the CSV file
	OutputDir   string // output directory for generated files
	PackageName string // Go package name for generated code
	SchemaFile  string // path to authz.yaml (optional, auto-detected if empty)
}

// csvLine represents a parsed row from the CSV, keyed by column name.
type csvLine struct {
	Fields map[string]string   // column_name -> value
	Groups map[string][]string // group_name -> parsed values
}

// Generate reads a schema and CSV file, then generates Go source files.
func Generate(cfg Config) error {
	schema, err := resolveSchema(cfg)
	if err != nil {
		return err
	}

	lines, err := parseCSV(cfg.InputCSV, schema)
	if err != nil {
		return fmt.Errorf("parsing CSV: %w", err)
	}

	if err := os.MkdirAll(cfg.OutputDir, 0750); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	data := buildTemplateData(cfg, schema, lines)

	files := map[string]string{
		"types":     buildTypesTemplate(schema),
		"resources": buildResourcesTemplate(schema),
		"actions":   actionsTemplateStr,
	}

	for name, tmplStr := range files {
		tmpl, err := template.New(name).Funcs(funcMap).Parse(tmplStr)
		if err != nil {
			return fmt.Errorf("parsing %s template: %w", name, err)
		}
		output, err := executeAndFormat(tmpl, data)
		if err != nil {
			return fmt.Errorf("generating %s: %w", name, err)
		}
		path := filepath.Join(cfg.OutputDir, fmt.Sprintf("zz_generated_%s.go", name))
		if err := os.WriteFile(path, output, 0600); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	// Generate per-group files
	for _, g := range schema.Groups {
		groupData := buildGroupFileData(cfg, schema, g, lines, data)
		tmplStr := buildGroupTemplate(schema, g)
		tmpl, err := template.New(g.Name).Funcs(funcMap).Parse(tmplStr)
		if err != nil {
			return fmt.Errorf("parsing %s template: %w", g.Name, err)
		}
		output, err := executeAndFormat(tmpl, groupData)
		if err != nil {
			return fmt.Errorf("generating %s: %w", g.Name, err)
		}
		path := filepath.Join(cfg.OutputDir, fmt.Sprintf("zz_generated_%s.go", g.Name))
		if err := os.WriteFile(path, output, 0600); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	return nil
}

func resolveSchema(cfg Config) (*Schema, error) {
	if cfg.SchemaFile != "" {
		return LoadSchema(cfg.SchemaFile)
	}
	if nearby := FindSchemaFile(cfg.InputCSV); nearby != "" {
		return LoadSchema(nearby)
	}
	return DefaultSchema(), nil
}

// --- CSV Parsing ---

func parseCSV(path string, schema *Schema) ([]csvLine, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // best-effort close on read-only file

	numCols := len(schema.CSVColumns)
	var lines []csvLine
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}

		parts := strings.SplitN(raw, ",", numCols)
		if len(parts) < numCols {
			continue
		}

		fields := make(map[string]string, numCols)
		for i, colName := range schema.CSVColumns {
			if colName == "_" {
				continue // skip ignored columns
			}
			fields[colName] = strings.TrimSpace(parts[i])
		}

		groups := make(map[string][]string)
		for _, g := range schema.Groups {
			raw := fields[g.Column]
			sep := g.GetSeparator()
			for v := range strings.SplitSeq(raw, sep) {
				v = strings.TrimSpace(v)
				if v != "" {
					groups[g.Name] = append(groups[g.Name], v)
				}
			}
		}

		lines = append(lines, csvLine{Fields: fields, Groups: groups})
	}

	return lines, scanner.Err()
}

// --- Template Data Building ---

// templateData is the universal data passed to all templates.
type templateData struct {
	Package      string
	CustomTypes  []customTypeData
	Fields       []fieldData
	Resources    []resourceData
	Actions      []actionData
	GroupEntries []groupEntryData
	PolicyGroups []policyGroupData
}

type customTypeData struct {
	TypeName string // e.g. "Scope", "Action", "Environment"
	Values   []customTypeValue
	IsAction bool
}

type customTypeValue struct {
	Value     string // e.g. "global"
	ConstName string // e.g. "ScopeGlobal"
}

type fieldData struct {
	Name      string // Go field name: "resource", "action", "scope"
	Type      string // Go type: "string", "Action", "Scope"
	CamelName string // "Resource", "Action", "Scope"
}

type resourceData struct {
	Name      string
	CamelName string
	Fields    map[string]string // field_column -> value (e.g. "scope" -> "global")
	Actions   []actionData
}

type actionData struct {
	Verbose   string
	ConstName string // unexported: actionViewAndListAllAdmins
	FieldName string // exported: ViewAndListAllAdmins
}

type groupEntryData struct {
	Name       string
	ConstName  string
	ScopeValue string // value from scope_column
	ScopeConst string // const name, e.g. "ScopeGlobal"
}

type policyGroupData struct {
	ScopeConst string
	Entries    []policyEntryData
}

type policyEntryData struct {
	EntryName   string
	Permissions []string // e.g. "ResourceAdmins.ViewAndListAllAdmins"
}

func buildTemplateData(cfg Config, schema *Schema, lines []csvLine) templateData {
	actionField := schema.ActionField()

	// Extract resources and actions
	resources := extractResourcesFromLines(lines, actionField)

	// Extract all unique actions (sorted)
	allActions := extractAllActionsFromResources(resources)

	// Extract custom types (non-string fields) with their unique values
	customTypes := extractCustomTypes(schema, lines)

	// Build field data
	fields := make([]fieldData, 0, len(schema.Permission.Fields))
	for _, f := range schema.Permission.Fields {
		fields = append(fields, fieldData{
			Name:      f.Name,
			Type:      f.Type,
			CamelName: toCamelCase(f.Name),
		})
	}

	return templateData{
		Package:     cfg.PackageName,
		CustomTypes: customTypes,
		Fields:      fields,
		Resources:   resources,
		Actions:     allActions,
	}
}

func buildGroupFileData(cfg Config, schema *Schema, g GroupSchema, lines []csvLine, base templateData) templateData {
	return templateData{
		Package:      base.Package,
		CustomTypes:  base.CustomTypes,
		Resources:    base.Resources,
		GroupEntries: extractGroupEntries(lines, schema, g),
		PolicyGroups: extractGroupPolicyGroups(lines, schema, g, base.Resources),
	}
}

func extractResourcesFromLines(lines []csvLine, actionField *FieldSchema) []resourceData {
	seen := map[string]*resourceData{}
	var order []string

	for _, l := range lines {
		resName := l.Fields["resource"]
		r, exists := seen[resName]
		if !exists {
			r = &resourceData{
				Name:      resName,
				CamelName: toCamelCase(resName),
				Fields:    l.Fields,
			}
			seen[resName] = r
			order = append(order, resName)
		}

		actionVerbose := l.Fields[actionField.Column]
		fieldName := verboseToConstName(actionVerbose)
		r.Actions = append(r.Actions, actionData{
			Verbose:   actionVerbose,
			ConstName: "action" + fieldName,
			FieldName: fieldName,
		})
	}

	result := make([]resourceData, 0, len(order))
	for _, key := range order {
		result = append(result, *seen[key])
	}
	return result
}

func extractAllActionsFromResources(resources []resourceData) []actionData {
	seen := map[string]bool{}
	var actions []actionData
	for _, r := range resources {
		for _, a := range r.Actions {
			if !seen[a.ConstName] {
				seen[a.ConstName] = true
				actions = append(actions, a)
			}
		}
	}
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].ConstName < actions[j].ConstName
	})
	return actions
}

func extractCustomTypes(schema *Schema, lines []csvLine) []customTypeData {
	types := make([]customTypeData, 0, len(schema.CustomTypes()))

	for _, f := range schema.CustomTypes() {
		seen := map[string]bool{}
		var values []customTypeValue
		for _, l := range lines {
			v := l.Fields[f.Column]
			if v != "" && !seen[v] {
				seen[v] = true
				values = append(values, customTypeValue{
					Value:     v,
					ConstName: toCamelCase(f.Type) + toCamelCase(v),
				})
			}
		}
		types = append(types, customTypeData{
			TypeName: f.Type,
			Values:   values,
			IsAction: f.Type == "Action",
		})
	}
	return types
}

func extractGroupEntries(lines []csvLine, schema *Schema, g GroupSchema) []groupEntryData {
	seen := map[string]bool{}
	var entries []groupEntryData

	sf := schema.ScopeFieldFor(g)

	for _, l := range lines {
		scopeValue := ""
		scopeConst := ""
		if g.ScopeColumn != "" {
			scopeValue = l.Fields[g.ScopeColumn]
			if sf != nil {
				scopeConst = toCamelCase(sf.Type) + toCamelCase(scopeValue)
			}
		}

		for _, entryName := range l.Groups[g.Name] {
			key := scopeValue + ":" + entryName
			if seen[key] {
				continue
			}
			seen[key] = true

			constName := g.TypeName()
			if scopeValue != "" {
				constName += toCamelCase(scopeValue)
			}
			constName += toCamelCase(entryName)

			entries = append(entries, groupEntryData{
				Name:       entryName,
				ConstName:  constName,
				ScopeValue: scopeValue,
				ScopeConst: scopeConst,
			})
		}
	}
	return entries
}

func extractGroupPolicyGroups(lines []csvLine, schema *Schema, g GroupSchema, resources []resourceData) []policyGroupData {
	// Build action field lookup: resource -> verbose -> fieldName
	resourceActions := map[string]map[string]string{}
	for _, r := range resources {
		m := map[string]string{}
		for _, a := range r.Actions {
			m[a.Verbose] = a.FieldName
		}
		resourceActions[r.Name] = m
	}

	actionField := schema.ActionField()

	type key struct{ scope, entry string }
	policyMap := map[key]*policyEntryData{}
	var order []key

	for _, l := range lines {
		resName := l.Fields["resource"]
		resourceVar := "Resource" + toCamelCase(resName)
		actionVerbose := l.Fields[actionField.Column]
		fieldName := resourceActions[resName][actionVerbose]

		scopeValue := ""
		if g.ScopeColumn != "" {
			scopeValue = l.Fields[g.ScopeColumn]
		}

		for _, entryName := range l.Groups[g.Name] {
			k := key{scopeValue, entryName}
			entry, exists := policyMap[k]
			if !exists {
				entry = &policyEntryData{EntryName: entryName}
				policyMap[k] = entry
				order = append(order, k)
			}
			entry.Permissions = append(entry.Permissions, resourceVar+"."+fieldName)
		}
	}

	// Group by scope value
	if g.ScopeColumn == "" {
		// No scope grouping — single group with empty scope
		entries := make([]policyEntryData, 0, len(order))
		for _, k := range order {
			entries = append(entries, *policyMap[k])
		}
		if len(entries) > 0 {
			return []policyGroupData{{Entries: entries}}
		}
		return nil
	}

	scopeGroups := map[string][]policyEntryData{}
	var scopeOrder []string
	scopeSeen := map[string]bool{}
	for _, k := range order {
		if !scopeSeen[k.scope] {
			scopeSeen[k.scope] = true
			scopeOrder = append(scopeOrder, k.scope)
		}
		scopeGroups[k.scope] = append(scopeGroups[k.scope], *policyMap[k])
	}

	sf := schema.ScopeFieldFor(g)
	var groups []policyGroupData
	for _, sv := range scopeOrder {
		scopeConst := ""
		if sf != nil {
			scopeConst = toCamelCase(sf.Type) + toCamelCase(sv)
		}
		groups = append(groups, policyGroupData{
			ScopeConst: scopeConst,
			Entries:    scopeGroups[sv],
		})
	}
	return groups
}

// --- String Helpers ---

func verboseToConstName(verbose string) string {
	words := strings.Fields(verbose)
	var b strings.Builder
	for _, w := range words {
		if len(w) == 0 {
			continue
		}
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}
	return b.String()
}

func toCamelCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	var b strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		runes := []rune(p)
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}
	return b.String()
}

func toLowerFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

var funcMap = template.FuncMap{
	"toLowerFirst": toLowerFirst,
	"toCamelCase":  toCamelCase,
}

// --- Dynamic Template Builders ---

func buildTypesTemplate(schema *Schema) string {
	var sb strings.Builder
	sb.WriteString(`// Code generated by apikit authz gen; DO NOT EDIT.

package {{ .Package }}

import "fmt"

`)

	// Generate custom types (non-string) with constants
	for _, f := range schema.CustomTypes() {
		if f.Type == "Action" {
			// Action type is special — constants go in actions file
			fmt.Fprintf(&sb, `// %s represents an authorization action that can be performed on a resource.
type %s string

// String returns the string representation of the action.
func (a %s) String() string {
	return string(a)
}

`, f.Type, f.Type, f.Type)
		} else {
			// Other custom types get their constants here
			fmt.Fprintf(&sb, `// %s defines the authorization %s level.
type %s string

const (
{{ range .CustomTypes }}{{ if eq .TypeName "%s" }}{{ range .Values }}	// {{ .ConstName }} represents {{ .Value }}-level authorization %s.
	{{ .ConstName }} %s = "{{ .Value }}"
{{ end }}{{ end }}{{ end }})

`, f.Type, strings.ToLower(f.Name), f.Type, f.Type, strings.ToLower(f.Name), f.Type)
		}
	}

	// Generate Permission struct
	sb.WriteString("// Permission represents a fully-qualified permission on a resource.\ntype Permission struct {\n")
	for _, f := range schema.Permission.Fields {
		goType := f.Type
		if goType == "string" {
			goType = "string"
		}
		fmt.Fprintf(&sb, "\t%s %s\n", f.Name, goType)
	}
	sb.WriteString("}\n\n")

	// Generate NewPermission constructor
	params := make([]string, 0, len(schema.Permission.Fields))
	assignments := make([]string, 0, len(schema.Permission.Fields))
	for _, f := range schema.Permission.Fields {
		goType := f.Type
		if goType == "string" {
			goType = "string"
		}
		params = append(params, f.Name+" "+goType)
		assignments = append(assignments, f.Name+": "+f.Name)
	}
	fmt.Fprintf(&sb, "// NewPermission creates a new Permission.\nfunc NewPermission(%s) Permission {\n\treturn Permission{%s}\n}\n\n",
		strings.Join(params, ", "), strings.Join(assignments, ", "))

	// Generate getters
	for _, f := range schema.Permission.Fields {
		goType := f.Type
		if goType == "string" {
			goType = "string"
		}
		fmt.Fprintf(&sb, "// %s returns the %s.\nfunc (p Permission) %s() %s { return p.%s }\n\n",
			toCamelCase(f.Name), f.Name, toCamelCase(f.Name), goType, f.Name)
	}

	// Generate Validate
	actionField := schema.ActionField()
	fmt.Fprintf(&sb, `// Validate checks that the permission is well-formed.
func (p Permission) Validate() error {
	if p.%s == "" {
		return fmt.Errorf("permission has empty action")
	}
	return nil
}

`, actionField.Name)

	// Generate Resource interface
	sb.WriteString("// Resource is the interface that resource types implement.\ntype Resource interface {\n\tString() string\n")
	// Add getter methods for custom-typed fields (non-string, non-Action)
	for _, f := range schema.Permission.Fields {
		if f.Type != "string" && f.Type != "Action" {
			fmt.Fprintf(&sb, "\t%s() %s\n", toCamelCase(f.Name), f.Type)
		}
	}
	sb.WriteString("\tPermissions() []Permission\n}\n\n")

	// Generate group type structs
	for _, g := range schema.Groups {
		sf := schema.ScopeFieldFor(g)
		typeName := g.TypeName()

		fmt.Fprintf(&sb, "// %s represents an authorization %s.\ntype %s struct {\n\tname  string\n",
			typeName, strings.ToLower(typeName), typeName)
		if sf != nil {
			fmt.Fprintf(&sb, "\t%s %s\n", sf.Name, sf.Type)
		}
		sb.WriteString("}\n\n")

		// Constructor
		fmt.Fprintf(&sb, "// New%s creates a new %s.\nfunc New%s(name string", typeName, typeName, typeName)
		if sf != nil {
			fmt.Fprintf(&sb, ", %s %s", sf.Name, sf.Type)
		}
		fmt.Fprintf(&sb, ") %s {\n\treturn %s{name: name", typeName, typeName)
		if sf != nil {
			fmt.Fprintf(&sb, ", %s: %s", sf.Name, sf.Name)
		}
		sb.WriteString("}\n}\n\n")

		// Getters
		fmt.Fprintf(&sb, "// Name returns the %s name.\nfunc (r %s) Name() string { return r.name }\n\n",
			strings.ToLower(typeName), typeName)
		fmt.Fprintf(&sb, "// String returns the %s name.\nfunc (r %s) String() string { return r.name }\n\n",
			strings.ToLower(typeName), typeName)
		if sf != nil {
			fmt.Fprintf(&sb, "// %s returns the %s %s.\nfunc (r %s) %s() %s { return r.%s }\n\n",
				toCamelCase(sf.Name), strings.ToLower(typeName), sf.Name,
				typeName, toCamelCase(sf.Name), sf.Type, sf.Name)
		}
	}

	return sb.String()
}

func buildResourcesTemplate(schema *Schema) string {
	var sb strings.Builder
	sb.WriteString(`// Code generated by apikit authz gen; DO NOT EDIT.

package {{ .Package }}

{{ range .Resources }}{{ $res := . }}
// {{ toLowerFirst .CamelName }} represents the "{{ .Name }}" resource with its available permissions.
type {{ toLowerFirst .CamelName }} struct {
{{ range .Actions }}	{{ .FieldName }} Permission
{{ end }}}

func (r {{ toLowerFirst $res.CamelName }}) String() string {
	return "{{ $res.Name }}"
}
`)

	// Add scope-like getter methods for custom-typed fields (non-string, non-Action)
	for _, f := range schema.Permission.Fields {
		if f.Type != "string" && f.Type != "Action" {
			fmt.Fprintf(&sb, `
func (r {{ toLowerFirst $res.CamelName }}) %s() %s {
	return {{ index $res.Fields "%s" | toCamelCase | printf "%s%%s" }}
}
`, toCamelCase(f.Name), f.Type, f.Column, toCamelCase(f.Type))
		}
	}

	sb.WriteString(`
func (r {{ toLowerFirst $res.CamelName }}) Permissions() []Permission {
	return []Permission{
{{ range .Actions }}		r.{{ .FieldName }},
{{ end }}	}
}
{{ end }}

var (
{{ range .Resources }}{{ $res := . }}	Resource{{ .CamelName }} = {{ toLowerFirst .CamelName }}{
{{ range .Actions }}		{{ .FieldName }}: Permission{`)

	// Build permission field initialization
	var fieldInits []string
	for _, f := range schema.Permission.Fields {
		switch f.Type {
		case "Action":
			fieldInits = append(fieldInits, f.Name+": {{ .ConstName }}")
		case "string":
			fieldInits = append(fieldInits, fmt.Sprintf(`%s: "{{ index $res.Fields "%s" }}"`, f.Name, f.Column))
		default:
			fieldInits = append(fieldInits, fmt.Sprintf(`%s: {{ index $res.Fields "%s" | toCamelCase | printf "%s%%s" }}`, f.Name, f.Column, toCamelCase(f.Type)))
		}
	}
	sb.WriteString(strings.Join(fieldInits, ", "))

	sb.WriteString(`},
{{ end }}	}
{{ end }})

func AllResources() []Resource {
	return []Resource{
{{ range .Resources }}		Resource{{ .CamelName }},
{{ end }}	}
}
`)

	// Scope-filtered helpers for each scope field referenced by groups
	for _, sf := range schema.ScopeFields() {
		fmt.Fprintf(&sb, `
{{ range .CustomTypes }}{{ if eq .TypeName "%s" }}{{ range .Values }}
func {{ .ConstName }}Resources() []Resource {
	var resources []Resource
	for _, r := range AllResources() {
		if r.%s() == {{ .ConstName }} {
			resources = append(resources, r)
		}
	}
	return resources
}
{{ end }}{{ end }}{{ end }}
`, sf.Type, toCamelCase(sf.Name))
	}

	return sb.String()
}

const actionsTemplateStr = `// Code generated by apikit authz gen; DO NOT EDIT.

package {{ .Package }}

import "slices"

const (
{{ range .Actions }}	{{ .ConstName }} Action = "{{ .Verbose }}"
{{ end }})

// AllActions returns a slice of all unique actions across all resources.
func AllActions() []Action {
	return []Action{
{{ range .Actions }}		{{ .ConstName }},
{{ end }}	}
}

// IsValid checks if the action is a valid action type.
func (a Action) IsValid() bool {
	return slices.Contains(AllActions(), a)
}
`

func buildGroupTemplate(schema *Schema, g GroupSchema) string {
	typeName := g.TypeName()
	pluralName := g.PluralName()
	sf := schema.ScopeFieldFor(g)

	var sb strings.Builder
	sb.WriteString(`// Code generated by apikit authz gen; DO NOT EDIT.

package {{ .Package }}

var (
{{ range .GroupEntries }}	{{ .ConstName }} = ` + typeName + `{name: "{{ .Name }}"`)

	if sf != nil {
		fmt.Fprintf(&sb, `, %s: {{ .ScopeConst }}`, sf.Name)
	}

	sb.WriteString(`}
{{ end }})

func All` + pluralName + `() []` + typeName + ` {
	return []` + typeName + `{
{{ range .GroupEntries }}		{{ .ConstName }},
{{ end }}	}
}
`)

	// Scope-filtered helpers
	if sf != nil {
		fmt.Fprintf(&sb, `
{{ range .CustomTypes }}{{ if eq .TypeName "%s" }}{{ range .Values }}
func {{ .ConstName }}%s() []%s {
	var items []%s
	for _, r := range All%s() {
		if r.%s() == {{ .ConstName }} {
			items = append(items, r)
		}
	}
	return items
}
{{ end }}{{ end }}{{ end }}
`, sf.Type, pluralName, typeName, typeName, pluralName, toCamelCase(sf.Name))
	}

	// Default<TypeName>Policies
	if g.ScopeColumn != "" {
		fmt.Fprintf(&sb, `
// Default%sPolicies returns the default %s-to-permission mappings for the given %s.
func Default%sPolicies(%s %s) map[string][]Permission {
	switch %s {
{{ range .PolicyGroups }}	case {{ .ScopeConst }}:
		return map[string][]Permission{
{{ range .Entries }}			"{{ .EntryName }}": {
{{ range .Permissions }}				{{ . }},
{{ end }}			},
{{ end }}		}
{{ end }}	}
	return nil
}
`, typeName, strings.ToLower(typeName), sf.Name, typeName, sf.Name, sf.Type, sf.Name)
	} else {
		fmt.Fprintf(&sb, `
// Default%sPolicies returns the default %s-to-permission mappings.
func Default%sPolicies() map[string][]Permission {
	return map[string][]Permission{
{{ range .PolicyGroups }}{{ range .Entries }}		"{{ .EntryName }}": {
{{ range .Permissions }}			{{ . }},
{{ end }}		},
{{ end }}{{ end }}	}
}
`, typeName, strings.ToLower(typeName), typeName)
	}

	return sb.String()
}

// --- Execution ---

func executeAndFormat(tmpl *template.Template, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("formatting generated code: %w (raw output:\n%s)", err, buf.String())
	}
	return formatted, nil
}
