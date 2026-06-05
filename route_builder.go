package limen

import (
	"net/http"
	"slices"
)

// RouteBuilder provides a clean API for plugins to register routes.
type RouteBuilder struct {
	group *routerGroup
	core  *LimenHTTPCore
}

// isRouteDisabled checks if a route ID is in the disabled list
func (b *RouteBuilder) isRouteDisabled(routeID RouteID, path string) bool {
	if b.core.config == nil || len(b.core.config.disabledPaths) == 0 {
		return false
	}

	routeDisabled := slices.Contains(b.core.config.disabledPaths, string(routeID))
	pathDisabled := slices.Contains(b.core.config.disabledPaths, path)
	return routeDisabled || pathDisabled
}

// AddRoute adds a route to the router
func (b *RouteBuilder) AddRoute(method HTTPMethod, path string, routeID RouteID, handler http.HandlerFunc, metadata *RouteMetadata, middleware ...Middleware) {
	if b.isRouteDisabled(routeID, path) {
		return
	}

	b.group.AddRoute(method, path, handler, routeID, metadata, middleware...)
}

// POST registers a POST route
func (b *RouteBuilder) POST(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	b.POSTWithMetadata(path, routeID, handler, nil, middleware...)
}

func (b *RouteBuilder) POSTWithMetadata(path string, routeID RouteID, handler http.HandlerFunc, metadata *RouteMetadata, middleware ...Middleware) {
	b.AddRoute(MethodPOST, path, routeID, handler, metadata, middleware...)
}

// GET registers a GET route
func (b *RouteBuilder) GET(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	b.GETWithMetadata(path, routeID, handler, nil, middleware...)
}

func (b *RouteBuilder) GETWithMetadata(path string, routeID RouteID, handler http.HandlerFunc, metadata *RouteMetadata, middleware ...Middleware) {
	b.AddRoute(MethodGET, path, routeID, handler, metadata, middleware...)
}

// PUT registers a PUT route
func (b *RouteBuilder) PUT(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	b.PUTWithMetadata(path, routeID, handler, nil, middleware...)
}

func (b *RouteBuilder) PUTWithMetadata(path string, routeID RouteID, handler http.HandlerFunc, metadata *RouteMetadata, middleware ...Middleware) {
	b.AddRoute(MethodPUT, path, routeID, handler, metadata, middleware...)
}

// DELETE registers a DELETE route
func (b *RouteBuilder) DELETE(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	b.DELETEWithMetadata(path, routeID, handler, nil, middleware...)
}

func (b *RouteBuilder) DELETEWithMetadata(path string, routeID RouteID, handler http.HandlerFunc, metadata *RouteMetadata, middleware ...Middleware) {
	b.AddRoute(MethodDELETE, path, routeID, handler, metadata, middleware...)
}

// PATCH registers a PATCH route
func (b *RouteBuilder) PATCH(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	b.PATCHWithMetadata(path, routeID, handler, nil, middleware...)
}

func (b *RouteBuilder) PATCHWithMetadata(path string, routeID RouteID, handler http.HandlerFunc, metadata *RouteMetadata, middleware ...Middleware) {
	b.AddRoute(MethodPATCH, path, routeID, handler, metadata, middleware...)
}

// ProtectedPOST registers a POST route with session requirement
func (b *RouteBuilder) ProtectedPOST(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	b.ProtectedPOSTWithMetadata(path, routeID, handler, nil, middleware...)
}

func (b *RouteBuilder) ProtectedPOSTWithMetadata(path string, routeID RouteID, handler http.HandlerFunc, metadata *RouteMetadata, middleware ...Middleware) {
	allMiddleware := append([]Middleware{b.core.MiddlewareRequireSession()}, middleware...)
	b.POSTWithMetadata(path, routeID, handler, metadata.withAuthRequired(), allMiddleware...)
}

// ProtectedGET registers a GET route with session requirement
func (b *RouteBuilder) ProtectedGET(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	b.ProtectedGETWithMetadata(path, routeID, handler, nil, middleware...)
}

func (b *RouteBuilder) ProtectedGETWithMetadata(path string, routeID RouteID, handler http.HandlerFunc, metadata *RouteMetadata, middleware ...Middleware) {
	allMiddleware := append([]Middleware{b.core.MiddlewareRequireSession()}, middleware...)
	b.GETWithMetadata(path, routeID, handler, metadata.withAuthRequired(), allMiddleware...)
}

// ProtectedPUT registers a PUT route with session requirement
func (b *RouteBuilder) ProtectedPUT(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	b.ProtectedPUTWithMetadata(path, routeID, handler, nil, middleware...)
}

func (b *RouteBuilder) ProtectedPUTWithMetadata(path string, routeID RouteID, handler http.HandlerFunc, metadata *RouteMetadata, middleware ...Middleware) {
	allMiddleware := append([]Middleware{b.core.MiddlewareRequireSession()}, middleware...)
	b.PUTWithMetadata(path, routeID, handler, metadata.withAuthRequired(), allMiddleware...)
}

// ProtectedDELETE registers a DELETE route with session requirement
func (b *RouteBuilder) ProtectedDELETE(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	b.ProtectedDELETEWithMetadata(path, routeID, handler, nil, middleware...)
}

func (b *RouteBuilder) ProtectedDELETEWithMetadata(path string, routeID RouteID, handler http.HandlerFunc, metadata *RouteMetadata, middleware ...Middleware) {
	allMiddleware := append([]Middleware{b.core.MiddlewareRequireSession()}, middleware...)
	b.DELETEWithMetadata(path, routeID, handler, metadata.withAuthRequired(), allMiddleware...)
}

// ProtectedPATCH registers a PATCH route with session requirement
func (b *RouteBuilder) ProtectedPATCH(path string, routeID RouteID, handler http.HandlerFunc, middleware ...Middleware) {
	b.ProtectedPATCHWithMetadata(path, routeID, handler, nil, middleware...)
}

func (b *RouteBuilder) ProtectedPATCHWithMetadata(path string, routeID RouteID, handler http.HandlerFunc, metadata *RouteMetadata, middleware ...Middleware) {
	allMiddleware := append([]Middleware{b.core.MiddlewareRequireSession()}, middleware...)
	b.PATCHWithMetadata(path, routeID, handler, metadata.withAuthRequired(), allMiddleware...)
}
