package limenhuma

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/ragokan/limen"
)

func Merge(api huma.API, auth *limen.Limen, opts ...limen.OpenAPIOption) (err error) {
	if api == nil {
		return fmt.Errorf("limenhuma: missing Huma API")
	}
	if auth == nil {
		return fmt.Errorf("limenhuma: missing Limen instance")
	}
	return MergeDocument(api.OpenAPI(), auth.OpenAPI(opts...))
}

func MergeDocument(target *huma.OpenAPI, source *limen.OpenAPIDocument) (err error) {
	if target == nil {
		return fmt.Errorf("limenhuma: missing target OpenAPI document")
	}
	if source == nil {
		return fmt.Errorf("limenhuma: missing source OpenAPI document")
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("limenhuma: merge OpenAPI document: %v", recovered)
		}
	}()

	if err := validateOperationConflicts(target, source.Paths); err != nil {
		return err
	}
	if err := mergeSecuritySchemes(target, source.Components.SecuritySchemes); err != nil {
		return err
	}
	for path, pathItem := range source.Paths {
		for method, operation := range pathItem {
			target.AddOperation(toHumaOperation(path, method, operation))
		}
	}
	return nil
}

func validateOperationConflicts(target *huma.OpenAPI, paths map[string]limen.OpenAPIPath) error {
	if len(paths) == 0 || len(target.Paths) == 0 {
		return nil
	}
	for path, pathItem := range paths {
		targetItem := target.Paths[path]
		if targetItem == nil {
			continue
		}
		for method := range pathItem {
			if humaOperationForMethod(targetItem, method) != nil {
				return fmt.Errorf("limenhuma: conflicting operation %s %s", strings.ToUpper(method), path)
			}
		}
	}
	return nil
}

func humaOperationForMethod(pathItem *huma.PathItem, method string) *huma.Operation {
	if pathItem == nil {
		return nil
	}
	switch strings.ToUpper(method) {
	case "GET":
		return pathItem.Get
	case "POST":
		return pathItem.Post
	case "PUT":
		return pathItem.Put
	case "PATCH":
		return pathItem.Patch
	case "DELETE":
		return pathItem.Delete
	case "HEAD":
		return pathItem.Head
	case "OPTIONS":
		return pathItem.Options
	case "TRACE":
		return pathItem.Trace
	default:
		return nil
	}
}

func mergeSecuritySchemes(target *huma.OpenAPI, schemes map[string]limen.OpenAPISecurityScheme) error {
	if len(schemes) == 0 {
		return nil
	}
	if target.Components == nil {
		target.Components = &huma.Components{}
	}
	if target.Components.SecuritySchemes == nil {
		target.Components.SecuritySchemes = make(map[string]*huma.SecurityScheme, len(schemes))
	}
	for name, scheme := range schemes {
		converted := toHumaSecurityScheme(scheme)
		if existing, ok := target.Components.SecuritySchemes[name]; ok && !reflect.DeepEqual(existing, converted) {
			return fmt.Errorf("limenhuma: conflicting security scheme %q", name)
		}
		target.Components.SecuritySchemes[name] = converted
	}
	return nil
}

func toHumaSecurityScheme(scheme limen.OpenAPISecurityScheme) *huma.SecurityScheme {
	return &huma.SecurityScheme{
		Type:         scheme.Type,
		Description:  scheme.Description,
		Name:         scheme.Name,
		In:           scheme.In,
		Scheme:       scheme.Scheme,
		BearerFormat: scheme.BearerFormat,
	}
}

func toHumaOperation(path string, method string, operation limen.OpenAPIOperation) *huma.Operation {
	return &huma.Operation{
		Method:      strings.ToUpper(method),
		Path:        path,
		Tags:        append([]string(nil), operation.Tags...),
		Summary:     operation.Summary,
		Description: operation.Description,
		OperationID: operation.OperationID,
		Parameters:  toHumaParams(operation.Parameters),
		RequestBody: toHumaRequestBody(operation.RequestBody),
		Responses:   toHumaResponses(operation.Responses),
		Deprecated:  operation.Deprecated,
		Security:    toHumaSecurity(operation.Security),
	}
}

func toHumaParams(parameters []limen.OpenAPIParameter) []*huma.Param {
	if len(parameters) == 0 {
		return nil
	}
	out := make([]*huma.Param, 0, len(parameters))
	for _, parameter := range parameters {
		out = append(out, &huma.Param{
			Name:        parameter.Name,
			In:          parameter.In,
			Description: parameter.Description,
			Required:    parameter.Required,
			Schema:      toHumaSchema(parameter.Schema),
		})
	}
	return out
}

func toHumaRequestBody(body *limen.OpenAPIRequestBody) *huma.RequestBody {
	if body == nil {
		return nil
	}
	return &huma.RequestBody{
		Description: body.Description,
		Required:    body.Required,
		Content:     toHumaMediaTypes(body.Content),
	}
}

func toHumaResponses(responses map[string]limen.OpenAPIResponse) map[string]*huma.Response {
	if len(responses) == 0 {
		return nil
	}
	out := make(map[string]*huma.Response, len(responses))
	for status, response := range responses {
		out[status] = &huma.Response{
			Description: response.Description,
			Content:     toHumaMediaTypes(response.Content),
		}
	}
	return out
}

func toHumaMediaTypes(mediaTypes map[string]limen.OpenAPIMediaType) map[string]*huma.MediaType {
	if len(mediaTypes) == 0 {
		return nil
	}
	out := make(map[string]*huma.MediaType, len(mediaTypes))
	for contentType, mediaType := range mediaTypes {
		out[contentType] = &huma.MediaType{
			Schema:  toHumaSchema(mediaType.Schema),
			Example: mediaType.Example,
		}
	}
	return out
}

func toHumaSecurity(security []limen.OpenAPISecurityRequirement) []map[string][]string {
	if len(security) == 0 {
		return nil
	}
	out := make([]map[string][]string, 0, len(security))
	for _, requirement := range security {
		item := make(map[string][]string, len(requirement))
		for name, scopes := range requirement {
			item[name] = append([]string(nil), scopes...)
		}
		out = append(out, item)
	}
	return out
}

func toHumaSchema(schema limen.OpenAPISchema) *huma.Schema {
	if len(schema) == 0 {
		return nil
	}

	out := &huma.Schema{}
	for key, value := range schema {
		switch key {
		case "$ref":
			out.Ref, _ = value.(string)
		case "type":
			out.Type, _ = value.(string)
		case "title":
			out.Title, _ = value.(string)
		case "description":
			out.Description, _ = value.(string)
		case "format":
			out.Format, _ = value.(string)
		case "properties":
			out.Properties = toHumaProperties(value)
		case "required":
			out.Required = toStringSlice(value)
		case "items":
			out.Items = toHumaSchemaValue(value)
		case "enum":
			out.Enum = toAnySlice(value)
		case "additionalProperties":
			out.AdditionalProperties = toAdditionalProperties(value)
		default:
			if out.Extensions == nil {
				out.Extensions = make(map[string]any)
			}
			out.Extensions[key] = value
		}
	}
	return out
}

func toHumaSchemaValue(value any) *huma.Schema {
	switch typed := value.(type) {
	case limen.OpenAPISchema:
		return toHumaSchema(typed)
	case map[string]any:
		return toHumaSchema(limen.OpenAPISchema(typed))
	default:
		return nil
	}
}

func toHumaProperties(value any) map[string]*huma.Schema {
	switch typed := value.(type) {
	case map[string]limen.OpenAPISchema:
		out := make(map[string]*huma.Schema, len(typed))
		for name, schema := range typed {
			out[name] = toHumaSchema(schema)
		}
		return out
	case map[string]any:
		out := make(map[string]*huma.Schema, len(typed))
		for name, schema := range typed {
			out[name] = toHumaSchemaValue(schema)
		}
		return out
	default:
		return nil
	}
}

func toAdditionalProperties(value any) any {
	switch typed := value.(type) {
	case limen.OpenAPISchema:
		return toHumaSchema(typed)
	case map[string]any:
		return toHumaSchema(limen.OpenAPISchema(typed))
	default:
		return value
	}
}

func toStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func toAnySlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return append([]any(nil), typed...)
	case []string:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = item
		}
		return out
	default:
		return nil
	}
}
