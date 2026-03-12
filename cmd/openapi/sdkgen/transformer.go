package sdkgen

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/kausys/apikit/cmd/openapi/spec"
)

// SDKData is the intermediate representation bridging spec+config and templates.
type SDKData struct {
	Provider   ProviderData
	Config     ConfigData
	Services   []ServiceData
	Models     []ModelFileData
	ModulePath string // Go module path for the SDK (e.g., "api/pkg/sdk/pokemon")
}

// ProviderData holds provider naming info.
type ProviderData struct {
	Name        string // lowercase (e.g., "pokemon")
	DisplayName string // title case (e.g., "Pokemon")
}

// ConfigData holds config generation info.
type ConfigData struct {
	Prefix string
	Fields []ConfigFieldData
}

// ConfigFieldData is a config field for template rendering.
type ConfigFieldData struct {
	Name       string // Go field name (e.g., "IsProduction")
	Type       string // Go type (e.g., "bool")
	ConfigKey  string // Full gookit key (e.g., "pokemon.production")
	GookitFunc string // gookit function (e.g., "Bool")
	Default    string // Default value expression (e.g., "true")
}

// ServiceData represents a single service (one per tag).
type ServiceData struct {
	Name      string // PascalCase (e.g., "Balance")
	FileName  string // snake_case file name (e.g., "balance_service")
	FieldName string // camelCase field name (e.g., "balanceService")
	Methods   []MethodData
}

// MethodData represents a single API operation.
type MethodData struct {
	Name             string // PascalCase from operationId (e.g., "GetBalances")
	HTTPMethod       string // GET, POST, etc.
	Path             string // e.g., "/api/v3/balance"
	TracerSpan       string // e.g., "pokemon.sdk.GetBalances"
	PathParams       []ParamData
	QueryParams      []ParamData
	HasRequestBody   bool
	RequestBodyType  string // e.g., "models.WithdrawalRequest"
	ResponseType     string // e.g., "*models.Balance" or "[]models.Balance"
	ResponseWrapper  string // gjson path or ""
	Comment          string
	UseParamsStruct  bool
	ParamsStructName string // e.g., "GetWalletParams"
}

// ParamData represents a path or query parameter.
type ParamData struct {
	Name     string // Original API name (e.g., "hash")
	GoName   string // Go parameter name (e.g., "hash")
	GoType   string // Go type (e.g., "string")
	Required bool
}

// ModelFileData represents a single models file (grouped by tag).
type ModelFileData struct {
	FileName    string // e.g., "balance"
	Tag         string
	Structs     []StructData
	Enums       []EnumData
	TypeAliases []TypeAliasData
	Imports     []ImportData
}

// TypeAliasData represents a Go type alias (e.g., type Foo = []Bar).
type TypeAliasData struct {
	Name    string
	Type    string
	Comment string
}

// StructData represents a Go struct to generate.
type StructData struct {
	Name    string
	Comment string
	Fields  []FieldData
}

// FieldData represents a struct field.
type FieldData struct {
	Name     string
	Type     string
	JSONTag  string
	Comment  string
	Required bool
}

// EnumData represents a Go enum (type + const block).
type EnumData struct {
	Name    string
	Type    string // underlying type (always "string")
	Comment string
	Values  []EnumValueData
}

// EnumValueData represents a single enum constant.
type EnumValueData struct {
	Name  string
	Value string
}

// ImportData represents a Go import.
type ImportData struct {
	Path  string
	Alias string
}

// transform converts the parsed OpenAPI spec and config into SDKData.
func transform(cfg *SDKGenConfig, openAPI *spec.OpenAPI) (*SDKData, error) {
	sc := newSchemaConverter(cfg.Models.CustomTypes)
	sc.schemas = openAPI.Components.Schemas

	data := &SDKData{
		Provider: ProviderData{
			Name:        cfg.Provider.Name,
			DisplayName: cfg.Provider.DisplayName,
		},
		Config:     transformConfig(cfg),
		ModulePath: cfg.Output.ModulePath,
	}

	// Transform schemas → models (grouped by tag)
	models, err := transformModels(sc, cfg, openAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to transform models: %w", err)
	}
	data.Models = models

	// Transform paths → services
	services, err := transformServices(sc, cfg, openAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to transform services: %w", err)
	}
	data.Services = services

	return data, nil
}

// transformConfig builds ConfigData from the config.
func transformConfig(cfg *SDKGenConfig) ConfigData {
	fields := make([]ConfigFieldData, 0, len(cfg.Config.Fields))
	for _, f := range cfg.Config.Fields {
		goType, gookitFunc := mapConfigType(f.Type)
		fields = append(fields, ConfigFieldData{
			Name:       f.Name,
			Type:       goType,
			ConfigKey:  cfg.Config.Prefix + "." + f.Key,
			GookitFunc: gookitFunc,
			Default:    f.Default,
		})
	}
	return ConfigData{
		Prefix: cfg.Config.Prefix,
		Fields: fields,
	}
}

// mapConfigType maps config field types to Go types and gookit functions.
func mapConfigType(typ string) (string, string) {
	switch typ {
	case "bool":
		return "bool", "Bool"
	case "int":
		return "int", "Int"
	case "duration":
		return "time.Duration", "Duration"
	case "string", "":
		return "string", "String"
	default:
		return "string", "String"
	}
}

// transformModels walks components/schemas and groups structs/enums by tag.
func transformModels(sc *schemaConverter, _ *SDKGenConfig, openAPI *spec.OpenAPI) ([]ModelFileData, error) {
	if openAPI.Components == nil || len(openAPI.Components.Schemas) == 0 {
		return nil, nil
	}

	// Build a map of schema name → tags (from operations that reference it)
	schemaTagMap := buildSchemaTagMap(openAPI)

	// Group schemas by tag
	tagModels := make(map[string]*ModelFileData)

	schemaNames := sortedKeys(openAPI.Components.Schemas)
	for _, name := range schemaNames {
		schema := openAPI.Components.Schemas[name]

		// Determine which tag this schema belongs to
		tags := schemaTagMap[name]
		tag := "common"
		if len(tags) == 1 {
			tag = tags[0]
		} else if len(tags) > 1 {
			tag = tags[0] // Use first tag if multiple
		}

		tagLower := strings.ToLower(tag)
		if _, ok := tagModels[tagLower]; !ok {
			tagModels[tagLower] = &ModelFileData{
				FileName: tagLower,
				Tag:      tag,
			}
		}
		mf := tagModels[tagLower]

		// Convert schema to struct or enum
		switch {
		case len(schema.Enum) > 0:
			enumData := sc.schemaToEnum(name, schema)
			if enumData != nil {
				mf.Enums = append(mf.Enums, *enumData)
			}
		case len(schema.Properties) > 0 || len(schema.AllOf) > 0:
			structData := sc.schemaToStruct(name, schema)
			if structData != nil {
				mf.Structs = append(mf.Structs, *structData)
			}
		case schema.Type.Value() == "object":
			// Empty object schema — generate an empty struct
			mf.Structs = append(mf.Structs, StructData{
				Name:    name,
				Comment: schema.Description,
			})
		case schema.Type.Value() == "array" && schema.Items != nil:
			// Top-level array schema — generate a type alias
			itemType := sc.goType(schema.Items, true)
			mf.TypeAliases = append(mf.TypeAliases, TypeAliasData{
				Name:    name,
				Type:    "[]" + itemType,
				Comment: schema.Description,
			})
		}
	}

	// Collect imports from schema converter
	commonImports := sc.sortedImports()

	// Build sorted model file list
	var models []ModelFileData
	tagNames := sortedKeys(tagModels)
	for _, tag := range tagNames {
		mf := tagModels[tag]
		if len(mf.Structs) > 0 || len(mf.Enums) > 0 || len(mf.TypeAliases) > 0 {
			mf.Imports = commonImports
			models = append(models, *mf)
		}
	}

	return models, nil
}

// buildSchemaTagMap maps schema names to the tags of operations that use them.
func buildSchemaTagMap(openAPI *spec.OpenAPI) map[string][]string {
	schemaTagMap := make(map[string][]string)

	if openAPI.Paths == nil {
		return schemaTagMap
	}

	for _, pathItem := range openAPI.Paths.PathItems {
		ops := operationsFromPathItem(pathItem)
		for _, op := range ops {
			if len(op.Tags) == 0 {
				continue
			}
			tag := op.Tags[0]

			refs := collectOperationSchemaRefs(op)
			for _, ref := range refs {
				name := extractRefName(ref)
				existing := schemaTagMap[name]
				found := slices.Contains(existing, tag)
				if !found {
					schemaTagMap[name] = append(schemaTagMap[name], tag)
				}
			}
		}
	}

	return schemaTagMap
}

// collectOperationSchemaRefs collects all $ref strings from an operation.
func collectOperationSchemaRefs(op *spec.Operation) []string {
	var refs []string

	if op.RequestBody != nil {
		for _, mt := range op.RequestBody.Content {
			if mt.Schema != nil && mt.Schema.Ref != "" {
				refs = append(refs, mt.Schema.Ref)
			}
		}
	}

	if op.Responses != nil {
		allResponses := make(map[string]*spec.Response)
		maps.Copy(allResponses, op.Responses.StatusCodes)
		if op.Responses.Default != nil {
			allResponses["default"] = op.Responses.Default
		}

		for _, resp := range allResponses {
			for _, mt := range resp.Content {
				if mt.Schema == nil {
					continue
				}
				if mt.Schema.Ref != "" {
					refs = append(refs, mt.Schema.Ref)
				}
				if mt.Schema.Items != nil && mt.Schema.Items.Ref != "" {
					refs = append(refs, mt.Schema.Items.Ref)
				}
			}
		}
	}

	return refs
}

// operationsFromPathItem returns all non-nil operations from a path item.
func operationsFromPathItem(pi *spec.PathItem) []*spec.Operation {
	var ops []*spec.Operation
	if pi.Get != nil {
		ops = append(ops, pi.Get)
	}
	if pi.Post != nil {
		ops = append(ops, pi.Post)
	}
	if pi.Put != nil {
		ops = append(ops, pi.Put)
	}
	if pi.Delete != nil {
		ops = append(ops, pi.Delete)
	}
	if pi.Patch != nil {
		ops = append(ops, pi.Patch)
	}
	if pi.Head != nil {
		ops = append(ops, pi.Head)
	}
	if pi.Options != nil {
		ops = append(ops, pi.Options)
	}
	return ops
}

// resolveParameter resolves a $ref parameter to its definition in components/parameters.
func resolveParameter(param *spec.Parameter, openAPI *spec.OpenAPI) *spec.Parameter {
	if param.Ref == "" {
		return param
	}
	refName := extractRefName(param.Ref)
	if openAPI.Components != nil && openAPI.Components.Parameters != nil {
		if resolved, ok := openAPI.Components.Parameters[refName]; ok {
			return resolved
		}
	}
	return nil
}

// shouldUseParamsStruct determines if a method should use a params struct based on config.
func shouldUseParamsStruct(cfg *SDKGenConfig, operationID string) bool {
	if cfg.Services.Operations != nil {
		if override, ok := cfg.Services.Operations[operationID]; ok {
			if override.ParamsStyle != "" {
				return override.ParamsStyle == "struct"
			}
		}
	}
	return cfg.Services.ParamsStyle == "struct"
}

// transformServices walks paths in the spec and groups operations by first tag.
func transformServices(sc *schemaConverter, cfg *SDKGenConfig, openAPI *spec.OpenAPI) ([]ServiceData, error) {
	if openAPI.Paths == nil {
		return nil, nil
	}

	tagServices := make(map[string]*ServiceData)

	paths := sortedKeys(openAPI.Paths.PathItems)
	for _, path := range paths {
		pathItem := openAPI.Paths.PathItems[path]

		type opEntry struct {
			method string
			op     *spec.Operation
		}
		entries := []opEntry{
			{"GET", pathItem.Get},
			{"POST", pathItem.Post},
			{"PUT", pathItem.Put},
			{"DELETE", pathItem.Delete},
			{"PATCH", pathItem.Patch},
			{"HEAD", pathItem.Head},
			{"OPTIONS", pathItem.Options},
		}

		for _, entry := range entries {
			if entry.op == nil {
				continue
			}

			tag := "default"
			if len(entry.op.Tags) > 0 {
				tag = entry.op.Tags[0]
			}

			tagLower := strings.ToLower(tag)
			if _, ok := tagServices[tagLower]; !ok {
				pascalTag := toPascalCase(tag)
				tagServices[tagLower] = &ServiceData{
					Name:      pascalTag,
					FileName:  toSnakeCase(tag) + "_service",
					FieldName: toCamelCase(tag) + "Service",
				}
			}

			method, err := transformOperation(sc, cfg, openAPI, path, entry.method, entry.op, pathItem.Parameters)
			if err != nil {
				return nil, fmt.Errorf("failed to transform operation %s %s: %w", entry.method, path, err)
			}

			svc := tagServices[tagLower]
			svc.Methods = append(svc.Methods, method)
		}
	}

	var services []ServiceData
	svcNames := sortedKeys(tagServices)
	for _, name := range svcNames {
		services = append(services, *tagServices[name])
	}

	return services, nil
}

// transformOperation converts a single OpenAPI operation into a MethodData.
func transformOperation(sc *schemaConverter, cfg *SDKGenConfig, openAPI *spec.OpenAPI, path, httpMethod string, op *spec.Operation, pathItemParams []*spec.Parameter) (MethodData, error) {
	methodName := toPascalCase(op.OperationID)
	if methodName == "" {
		methodName = toPascalCase(httpMethod + "_" + strings.ReplaceAll(strings.Trim(path, "/"), "/", "_"))
	}

	method := MethodData{
		Name:            methodName,
		HTTPMethod:      httpMethod,
		Path:            path,
		TracerSpan:      cfg.Provider.Name + ".sdk." + methodName,
		ResponseWrapper: cfg.Services.ResponseWrapper,
		Comment:         operationComment(op),
	}

	allParams := append([]*spec.Parameter{}, pathItemParams...)
	allParams = append(allParams, op.Parameters...)

	for _, param := range allParams {
		if param == nil {
			continue
		}
		if param.Ref != "" {
			param = resolveParameter(param, openAPI)
			if param == nil {
				continue
			}
		}
		paramType := "string"
		if param.Schema != nil {
			paramType = sc.goType(param.Schema, param.Required)
		}
		pd := ParamData{
			Name:     param.Name,
			GoName:   toCamelCase(param.Name),
			GoType:   paramType,
			Required: param.Required,
		}
		switch param.In {
		case "path":
			method.PathParams = append(method.PathParams, pd)
		case "query":
			method.QueryParams = append(method.QueryParams, pd)
		}
	}

	if op.RequestBody != nil {
		method.HasRequestBody = true
		for _, mt := range op.RequestBody.Content {
			if mt.Schema != nil {
				if mt.Schema.Ref != "" {
					method.RequestBodyType = "models." + extractRefName(mt.Schema.Ref)
				} else {
					method.RequestBodyType = sc.goType(mt.Schema, true)
				}
				break
			}
		}
	}

	method.ResponseType = extractResponseType(sc, op)

	method.UseParamsStruct = shouldUseParamsStruct(cfg, op.OperationID)
	if method.UseParamsStruct {
		method.ParamsStructName = method.Name + "Params"
	}

	return method, nil
}

// extractResponseType determines the Go return type from the operation's success response.
func extractResponseType(sc *schemaConverter, op *spec.Operation) string {
	if op.Responses == nil {
		return ""
	}

	successCodes := []string{"200", "201", "202"}
	for _, code := range successCodes {
		resp, ok := op.Responses.StatusCodes[code]
		if !ok || resp == nil {
			continue
		}
		for _, mt := range resp.Content {
			if mt.Schema == nil {
				continue
			}
			schema := mt.Schema

			if schema.Type.Value() == "array" && schema.Items != nil {
				if schema.Items.Ref != "" {
					return "[]models." + extractRefName(schema.Items.Ref)
				}
				return "[]" + sc.goType(schema.Items, true)
			}

			if schema.Ref != "" {
				refName := extractRefName(schema.Ref)
				// If the referenced schema is a top-level array, expand to slice type
				// so the service template handles it as a slice (nil zero value, no .Raw).
				if sc.schemas != nil {
					if refSchema, ok := sc.schemas[refName]; ok && refSchema.Type.Value() == "array" && refSchema.Items != nil {
						if refSchema.Items.Ref != "" {
							return "[]models." + extractRefName(refSchema.Items.Ref)
						}
						return "[]" + sc.goType(refSchema.Items, true)
					}
				}
				return "models." + refName
			}

			goType := sc.goType(schema, true)
			if goType != "any" {
				return goType
			}
		}
	}

	return ""
}

// operationComment builds the comment for a method from the operation's summary and description.
// If description is present, it combines summary + description. Otherwise, just summary.
func operationComment(op *spec.Operation) string {
	if op.Description != "" && op.Summary != "" {
		return op.Summary + "\n\n" + op.Description
	}
	if op.Description != "" {
		return op.Description
	}
	return op.Summary
}

// sortedKeys returns map keys sorted alphabetically.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
