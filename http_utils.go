package limen

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

type bodyContextKey struct{}

var errRequestBodyTooLarge = errors.New("request body too large")

// normalizePath normalizes the base path to start with a slash.
func normalizePath(basePath string) string {
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	basePath = strings.TrimSuffix(basePath, "/")
	return basePath
}

// GetBody extracts the parsed JSON body from the request context
func GetJSONBody(req *http.Request) map[string]any {
	if req == nil {
		return nil
	}

	if body, ok := req.Context().Value(bodyContextKey{}).(map[string]any); ok {
		return body
	}

	return nil
}

// shouldParseBody checks if the request body should be parsed as JSON
func shouldParseBody(req *http.Request) bool {
	if req.Method != "POST" && req.Method != "PUT" && req.Method != "PATCH" {
		return false
	}
	contentType := req.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, "application/json") && req.Body != nil
}

// parseJSONBody reads and parses the JSON body from the request
// Returns the parsed body map and the original body bytes for restoration
func parseJSONBody(req *http.Request, maxBodyBytes int64) (map[string]any, []byte, error) {
	defer req.Body.Close()
	reader := req.Body
	if maxBodyBytes > 0 {
		reader = io.NopCloser(io.LimitReader(req.Body, maxBodyBytes+1))
	}
	bodyBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, nil, err
	}
	if maxBodyBytes > 0 && int64(len(bodyBytes)) > maxBodyBytes {
		return nil, nil, errRequestBodyTooLarge
	}

	var body map[string]any
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		return nil, bodyBytes, err
	}

	return body, bodyBytes, nil
}

// parseAndStoreBody parses the JSON body if needed and stores it in context.
func parseAndStoreBody(req *http.Request, maxBodyBytes int64) (*http.Request, error) {
	if GetJSONBody(req) != nil {
		return req, nil
	}

	if !shouldParseBody(req) {
		return req, nil
	}

	if maxBodyBytes > 0 && req.ContentLength > maxBodyBytes {
		return req, errRequestBodyTooLarge
	}

	body, bodyBytes, err := parseJSONBody(req, maxBodyBytes)
	if err != nil {
		return req, err
	}

	req = req.WithContext(context.WithValue(req.Context(), bodyContextKey{}, body))

	// Restore body for handlers that need to read it
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return req, nil
}

func getCurrentRouteFromContext(ctx context.Context) *route {
	if route, ok := ctx.Value(currentRouteContextKey{}).(*route); ok {
		return route
	}
	return nil
}
