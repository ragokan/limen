package limen

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

const (
	openAPIVersion             = "3.1.0"
	defaultOpenAPITitle        = "Limen Auth API"
	defaultOpenAPIDocumentVer  = "1.0.0"
	openAPISessionCookieScheme = "sessionCookie"
	openAPIBearerSessionScheme = "bearerAuth"
	defaultOpenAPIResponseCode = "200"
	defaultOpenAPIResponseDesc = "OK"
	defaultOpenAPIContentType  = "application/json"
)

type OpenAPIConfig struct {
	Title           string
	Version         string
	Description     string
	Servers         []OpenAPIServer
	SecuritySchemes map[string]OpenAPISecurityScheme
}

type OpenAPIOption func(*OpenAPIConfig)

type OpenAPIDocument struct {
	OpenAPI    string                 `json:"openapi"`
	Info       OpenAPIInfo            `json:"info"`
	Servers    []OpenAPIServer        `json:"servers,omitempty"`
	Paths      map[string]OpenAPIPath `json:"paths"`
	Components OpenAPIComponents      `json:"components,omitempty"`
}

type OpenAPIInfo struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type OpenAPIComponents struct {
	SecuritySchemes map[string]OpenAPISecurityScheme `json:"securitySchemes,omitempty"`
}

type OpenAPISecurityScheme struct {
	Type         string `json:"type"`
	Description  string `json:"description,omitempty"`
	Name         string `json:"name,omitempty"`
	In           string `json:"in,omitempty"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
}

type OpenAPIPath map[string]OpenAPIOperation

type OpenAPIOperation struct {
	OperationID string                       `json:"operationId,omitempty"`
	Summary     string                       `json:"summary,omitempty"`
	Description string                       `json:"description,omitempty"`
	Tags        []string                     `json:"tags,omitempty"`
	Deprecated  bool                         `json:"deprecated,omitempty"`
	Parameters  []OpenAPIParameter           `json:"parameters,omitempty"`
	RequestBody *OpenAPIRequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse   `json:"responses"`
	Security    []OpenAPISecurityRequirement `json:"security,omitempty"`
}

type OpenAPIParameter struct {
	Name        string        `json:"name"`
	In          string        `json:"in"`
	Description string        `json:"description,omitempty"`
	Required    bool          `json:"required,omitempty"`
	Schema      OpenAPISchema `json:"schema,omitempty"`
}

type OpenAPIRequestBody struct {
	Description string                      `json:"description,omitempty"`
	Required    bool                        `json:"required,omitempty"`
	Content     map[string]OpenAPIMediaType `json:"content,omitempty"`
}

type OpenAPIResponse struct {
	Description string                      `json:"description"`
	Content     map[string]OpenAPIMediaType `json:"content,omitempty"`
}

type OpenAPIMediaType struct {
	Schema  OpenAPISchema `json:"schema,omitempty"`
	Example any           `json:"example,omitempty"`
}

type OpenAPISchema map[string]any

type OpenAPISecurityRequirement map[string][]string

func WithOpenAPITitle(title string) OpenAPIOption {
	return func(c *OpenAPIConfig) {
		c.Title = title
	}
}

func WithOpenAPIVersion(version string) OpenAPIOption {
	return func(c *OpenAPIConfig) {
		c.Version = version
	}
}

func WithOpenAPIDescription(description string) OpenAPIOption {
	return func(c *OpenAPIConfig) {
		c.Description = description
	}
}

func WithOpenAPIServers(servers ...OpenAPIServer) OpenAPIOption {
	return func(c *OpenAPIConfig) {
		c.Servers = append([]OpenAPIServer(nil), servers...)
	}
}

func WithOpenAPISecurityScheme(name string, scheme OpenAPISecurityScheme) OpenAPIOption {
	return func(c *OpenAPIConfig) {
		if c.SecuritySchemes == nil {
			c.SecuritySchemes = make(map[string]OpenAPISecurityScheme)
		}
		c.SecuritySchemes[name] = scheme
	}
}

func OpenAPIStringSchema() OpenAPISchema {
	return OpenAPISchema{"type": "string"}
}

func OpenAPIObjectSchema(properties map[string]OpenAPISchema, required ...string) OpenAPISchema {
	schema := OpenAPISchema{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func (a *Limen) OpenAPI(opts ...OpenAPIOption) *OpenAPIDocument {
	config := a.defaultOpenAPIConfig()
	for _, opt := range opts {
		opt(config)
	}

	router := a.buildRouter(routerBuildOptions{})
	return buildOpenAPIDocument(config, router.Routes())
}

func (a *Limen) OpenAPIJSON(opts ...OpenAPIOption) ([]byte, error) {
	return json.MarshalIndent(a.OpenAPI(opts...), "", "  ")
}

func (a *Limen) OpenAPIHandler(opts ...OpenAPIOption) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		document := a.OpenAPI(opts...)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(document); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func (a *Limen) defaultOpenAPIConfig() *OpenAPIConfig {
	config := &OpenAPIConfig{
		Title:           defaultOpenAPITitle,
		Version:         defaultOpenAPIDocumentVer,
		SecuritySchemes: make(map[string]OpenAPISecurityScheme),
	}
	if a != nil && a.core != nil {
		if a.core.baseURL != "" {
			config.Servers = []OpenAPIServer{{URL: a.core.baseURL}}
		}
		if a.core.config != nil && a.core.config.HTTP != nil && a.core.config.HTTP.cookieConfig != nil {
			config.SecuritySchemes[openAPISessionCookieScheme] = OpenAPISecurityScheme{
				Type: "apiKey",
				In:   "cookie",
				Name: a.core.config.HTTP.cookieConfig.sessionCookieName,
			}
		}
		if a.core.config != nil && a.core.config.Session != nil && a.core.config.Session.BearerEnabled {
			config.SecuritySchemes[openAPIBearerSessionScheme] = OpenAPISecurityScheme{
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "opaque",
			}
		}
	}
	return config
}

func buildOpenAPIDocument(config *OpenAPIConfig, routes []RegisteredRoute) *OpenAPIDocument {
	document := &OpenAPIDocument{
		OpenAPI: openAPIVersion,
		Info: OpenAPIInfo{
			Title:       config.Title,
			Version:     config.Version,
			Description: config.Description,
		},
		Servers: config.Servers,
		Paths:   make(map[string]OpenAPIPath),
		Components: OpenAPIComponents{
			SecuritySchemes: config.SecuritySchemes,
		},
	}

	for _, route := range routes {
		if route.Method == MethodANY {
			continue
		}

		openAPIPath, pathParameters := openAPIPathAndParams(route.Pattern)
		method := strings.ToLower(string(route.Method))
		if document.Paths[openAPIPath] == nil {
			document.Paths[openAPIPath] = make(OpenAPIPath)
		}
		document.Paths[openAPIPath][method] = openAPIOperationForRoute(config, route, pathParameters)
	}

	return document
}

func openAPIOperationForRoute(config *OpenAPIConfig, route RegisteredRoute, pathParameters []OpenAPIParameter) OpenAPIOperation {
	metadata := route.Metadata.clone()
	parameters := append([]OpenAPIParameter(nil), pathParameters...)
	parameters = appendMissingParameters(parameters, metadata.Parameters...)

	operationID := metadata.OperationID
	if operationID == "" {
		operationID = string(route.RouteID)
	}

	requestBody := metadata.RequestBody
	if requestBody == nil && methodAllowsRequestBody(route.Method) && len(metadata.AllowedContentTypes) > 0 {
		requestBody = requestBodyForContentTypes(metadata.AllowedContentTypes)
	}

	responses := openAPIResponses(metadata.Responses)

	security := append([]OpenAPISecurityRequirement(nil), metadata.Security...)
	if len(security) == 0 && metadata.AuthRequired {
		security = defaultOpenAPISecurityRequirements(config)
	}

	tags := append([]string(nil), metadata.Tags...)
	if len(tags) == 0 {
		tags = []string{"auth"}
	}

	return OpenAPIOperation{
		OperationID: operationID,
		Summary:     metadata.Summary,
		Description: metadata.Description,
		Tags:        tags,
		Deprecated:  metadata.Deprecated,
		Parameters:  parameters,
		RequestBody: requestBody,
		Responses:   responses,
		Security:    security,
	}
}

func defaultOpenAPISecurityRequirements(config *OpenAPIConfig) []OpenAPISecurityRequirement {
	if config == nil {
		return []OpenAPISecurityRequirement{{openAPISessionCookieScheme: []string{}}}
	}

	requirements := make([]OpenAPISecurityRequirement, 0, 2)
	if _, ok := config.SecuritySchemes[openAPISessionCookieScheme]; ok {
		requirements = append(requirements, OpenAPISecurityRequirement{openAPISessionCookieScheme: []string{}})
	}
	if _, ok := config.SecuritySchemes[openAPIBearerSessionScheme]; ok {
		requirements = append(requirements, OpenAPISecurityRequirement{openAPIBearerSessionScheme: []string{}})
	}
	if len(requirements) == 0 {
		requirements = append(requirements, OpenAPISecurityRequirement{openAPISessionCookieScheme: []string{}})
	}
	return requirements
}

func appendMissingParameters(parameters []OpenAPIParameter, additions ...OpenAPIParameter) []OpenAPIParameter {
	seen := make(map[string]struct{}, len(parameters)+len(additions))
	for _, parameter := range parameters {
		seen[parameter.In+":"+parameter.Name] = struct{}{}
	}
	for _, parameter := range additions {
		key := parameter.In + ":" + parameter.Name
		if _, ok := seen[key]; ok {
			continue
		}
		parameters = append(parameters, parameter)
		seen[key] = struct{}{}
	}
	return parameters
}

func openAPIResponses(responses map[int]OpenAPIResponse) map[string]OpenAPIResponse {
	if len(responses) == 0 {
		return map[string]OpenAPIResponse{
			defaultOpenAPIResponseCode: {Description: defaultOpenAPIResponseDesc},
		}
	}

	out := make(map[string]OpenAPIResponse, len(responses))
	for status, response := range responses {
		if response.Description == "" {
			response.Description = http.StatusText(status)
			if response.Description == "" {
				response.Description = defaultOpenAPIResponseDesc
			}
		}
		out[strconv.Itoa(status)] = response
	}
	return out
}

func requestBodyForContentTypes(contentTypes []string) *OpenAPIRequestBody {
	content := make(map[string]OpenAPIMediaType, len(contentTypes))
	for _, contentType := range contentTypes {
		content[contentType] = OpenAPIMediaType{
			Schema: OpenAPIObjectSchema(map[string]OpenAPISchema{}),
		}
	}
	return &OpenAPIRequestBody{Content: content}
}

func methodAllowsRequestBody(method HTTPMethod) bool {
	return method == MethodPOST || method == MethodPUT || method == MethodPATCH
}

func openAPIPathAndParams(pattern string) (string, []OpenAPIParameter) {
	if pattern == "" || pattern == "/" {
		return "/", nil
	}

	segments := strings.Split(pattern, "/")
	parameters := make([]OpenAPIParameter, 0)
	for i, segment := range segments {
		if !strings.HasPrefix(segment, ":") {
			continue
		}
		name := strings.TrimPrefix(segment, ":")
		segments[i] = "{" + name + "}"
		parameters = append(parameters, OpenAPIParameter{
			Name:     name,
			In:       "path",
			Required: true,
			Schema:   OpenAPIStringSchema(),
		})
	}
	return strings.Join(segments, "/"), parameters
}

var _ http.Handler = http.HandlerFunc(nil)
