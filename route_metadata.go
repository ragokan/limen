package limen

type RouteMetadataOption func(*RouteMetadata)

func NewRouteMetadata(opts ...RouteMetadataOption) *RouteMetadata {
	metadata := &RouteMetadata{}
	for _, opt := range opts {
		opt(metadata)
	}
	return metadata
}

func WithRouteAllowedContentTypes(contentTypes ...string) RouteMetadataOption {
	return func(m *RouteMetadata) {
		m.AllowedContentTypes = append([]string(nil), contentTypes...)
	}
}

func WithRouteOperationID(operationID string) RouteMetadataOption {
	return func(m *RouteMetadata) {
		m.OperationID = operationID
	}
}

func WithRouteSummary(summary string) RouteMetadataOption {
	return func(m *RouteMetadata) {
		m.Summary = summary
	}
}

func WithRouteDescription(description string) RouteMetadataOption {
	return func(m *RouteMetadata) {
		m.Description = description
	}
}

func WithRouteTags(tags ...string) RouteMetadataOption {
	return func(m *RouteMetadata) {
		m.Tags = append([]string(nil), tags...)
	}
}

func WithRouteAuthRequired(required bool) RouteMetadataOption {
	return func(m *RouteMetadata) {
		m.AuthRequired = required
	}
}

func WithRouteDeprecated(deprecated bool) RouteMetadataOption {
	return func(m *RouteMetadata) {
		m.Deprecated = deprecated
	}
}

func WithRouteParameters(parameters ...OpenAPIParameter) RouteMetadataOption {
	return func(m *RouteMetadata) {
		m.Parameters = append([]OpenAPIParameter(nil), parameters...)
	}
}

func WithRouteRequestBody(body *OpenAPIRequestBody) RouteMetadataOption {
	return func(m *RouteMetadata) {
		m.RequestBody = body
	}
}

func WithRouteResponse(status int, response OpenAPIResponse) RouteMetadataOption {
	return func(m *RouteMetadata) {
		if m.Responses == nil {
			m.Responses = make(map[int]OpenAPIResponse)
		}
		m.Responses[status] = response
	}
}

func WithRouteSecurity(security ...OpenAPISecurityRequirement) RouteMetadataOption {
	return func(m *RouteMetadata) {
		m.Security = append([]OpenAPISecurityRequirement(nil), security...)
	}
}

func (m *RouteMetadata) clone() *RouteMetadata {
	if m == nil {
		return &RouteMetadata{}
	}

	clone := *m
	clone.AllowedContentTypes = append([]string(nil), m.AllowedContentTypes...)
	clone.Tags = append([]string(nil), m.Tags...)
	clone.Parameters = append([]OpenAPIParameter(nil), m.Parameters...)
	clone.Security = append([]OpenAPISecurityRequirement(nil), m.Security...)
	if m.Responses != nil {
		clone.Responses = make(map[int]OpenAPIResponse, len(m.Responses))
		for status, response := range m.Responses {
			clone.Responses[status] = response
		}
	}
	return &clone
}

func (m *RouteMetadata) withAuthRequired() *RouteMetadata {
	clone := m.clone()
	clone.AuthRequired = true
	return clone
}
