package limen

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultSessionIPAddressExtractor_UsesRemoteAddr(t *testing.T) {
	t.Parallel()

	config := NewDefaultSessionConfig()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	req.RemoteAddr = "203.0.113.20:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.20")
	req.Header.Set("X-Real-IP", "198.51.100.21")

	assert.Equal(t, "203.0.113.20", config.IPAddressExtractor(req))
}

func TestDefaultSessionIPAddressExtractor_AllowsRemoteAddrWithoutPort(t *testing.T) {
	t.Parallel()

	config := NewDefaultSessionConfig()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	req.RemoteAddr = "203.0.113.20"

	assert.Equal(t, "203.0.113.20", config.IPAddressExtractor(req))
}
