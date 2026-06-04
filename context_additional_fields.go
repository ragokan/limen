package limen

import (
	"context"
	"net/http"
)

type contextKeyAdditionalFields struct{}

type AdditionalFieldsFunc func(ctx *AdditionalFieldsContext) (map[string]any, error)

type AdditionalFieldsContext struct {
	request  *http.Request
	response http.ResponseWriter
}

func newAdditionalFieldsContext(request *http.Request, response http.ResponseWriter) *AdditionalFieldsContext {
	ctx := &AdditionalFieldsContext{
		request:  request,
		response: response,
	}

	return ctx
}

func (ctx *AdditionalFieldsContext) GetBody() map[string]any {
	if ctx == nil {
		return nil
	}
	return GetJSONBody(ctx.request)
}

func (ctx *AdditionalFieldsContext) GetBodyValue(key string) any {
	return ctx.GetBody()[key]
}

func (ctx *AdditionalFieldsContext) GetHeader(key string) string {
	return ctx.request.Header.Get(key)
}

func (ctx *AdditionalFieldsContext) GetHeaders() http.Header {
	return ctx.request.Header
}

func (ctx *AdditionalFieldsContext) IsEmpty(key string) bool {
	value := ctx.GetBodyValue(key)
	return value == nil || value == ""
}

func withAdditionalFieldsContext(ctx context.Context, r *http.Request, w http.ResponseWriter) context.Context {
	return context.WithValue(ctx, contextKeyAdditionalFields{}, newAdditionalFieldsContext(r, w))
}

func updateAdditionalFieldsRequest(r *http.Request) {
	if r == nil {
		return
	}
	if afCtx, ok := r.Context().Value(contextKeyAdditionalFields{}).(*AdditionalFieldsContext); ok {
		afCtx.request = r
	}
}

// getAdditionalFieldsContext retrieves the AdditionalFieldsContext from the req context.
// Returns an empty context (with nil request/response) if not in HTTP context (e.g., background jobs, CLI).
func getAdditionalFieldsContext(ctx context.Context) *AdditionalFieldsContext {
	if afCtx, ok := ctx.Value(contextKeyAdditionalFields{}).(*AdditionalFieldsContext); ok {
		return afCtx
	}

	return newAdditionalFieldsContext(nil, nil)
}
