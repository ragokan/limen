package limen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrustedProxyIPExtractorIgnoresForwardedHeadersFromUntrustedRemote(t *testing.T) {
	t.Parallel()

	extractor, err := NewTrustedProxyIPExtractor(WithTrustedProxyCIDRs("10.0.0.0/8"))
	require.NoError(t, err)

	req := newTrustedProxyRequest("203.0.113.10:1234")
	req.Header.Set("X-Forwarded-For", "198.51.100.20")
	req.Header.Set("X-Real-IP", "198.51.100.21")

	assert.Equal(t, "203.0.113.10", extractor(req))
}

func TestTrustedProxyIPExtractorUsesXForwardedForFromTrustedRemote(t *testing.T) {
	t.Parallel()

	extractor, err := NewTrustedProxyIPExtractor(
		WithTrustedProxyCIDRs("10.0.0.0/8"),
		WithTrustedProxyHeaders(TrustedProxyHeaderXForwardedFor),
	)
	require.NoError(t, err)

	req := newTrustedProxyRequest("10.0.0.5:1234")
	req.Header.Set("X-Forwarded-For", "198.51.100.20, 10.0.0.4")

	assert.Equal(t, "198.51.100.20", extractor(req))
}

func TestTrustedProxyIPExtractorUsesForwardedHeader(t *testing.T) {
	t.Parallel()

	extractor, err := NewTrustedProxyIPExtractor(
		WithTrustedProxyCIDRs("127.0.0.1", "10.0.0.0/8"),
		WithTrustedProxyHeaders(TrustedProxyHeaderForwarded),
	)
	require.NoError(t, err)

	req := newTrustedProxyRequest("127.0.0.1:1234")
	req.Header.Set("Forwarded", `for=192.0.2.60;proto=https, for=10.0.0.4`)

	assert.Equal(t, "192.0.2.60", extractor(req))
}

func TestTrustedProxyIPExtractorUsesForwardedForAfterOtherParameters(t *testing.T) {
	t.Parallel()

	extractor, err := NewTrustedProxyIPExtractor(
		WithTrustedProxyCIDRs("127.0.0.1"),
		WithTrustedProxyHeaders(TrustedProxyHeaderForwarded),
	)
	require.NoError(t, err)

	req := newTrustedProxyRequest("127.0.0.1:1234")
	req.Header.Set("Forwarded", `proto=https;host=example.test;for=192.0.2.61`)

	assert.Equal(t, "192.0.2.61", extractor(req))
}

func TestTrustedProxyIPExtractorFallsBackToXRealIP(t *testing.T) {
	t.Parallel()

	extractor, err := NewTrustedProxyIPExtractor(
		WithTrustedProxyCIDRs("10.0.0.0/8"),
		WithTrustedProxyHeaders(TrustedProxyHeaderXRealIP),
	)
	require.NoError(t, err)

	req := newTrustedProxyRequest("10.0.0.5:1234")
	req.Header.Set("X-Real-IP", "198.51.100.30")

	assert.Equal(t, "198.51.100.30", extractor(req))
}

func TestTrustedProxyIPExtractorFallsBackToRemoteForInvalidForwardedValue(t *testing.T) {
	t.Parallel()

	extractor, err := NewTrustedProxyIPExtractor(
		WithTrustedProxyCIDRs("10.0.0.0/8"),
		WithTrustedProxyHeaders(TrustedProxyHeaderXForwardedFor),
	)
	require.NoError(t, err)

	req := newTrustedProxyRequest("10.0.0.5:1234")
	req.Header.Set("X-Forwarded-For", "not-an-ip")

	assert.Equal(t, "10.0.0.5", extractor(req))
}

func TestTrustedProxyIPExtractorNormalizesIPv6Prefix(t *testing.T) {
	t.Parallel()

	extractor, err := NewTrustedProxyIPExtractor(
		WithTrustedProxyCIDRs("10.0.0.0/8"),
		WithTrustedProxyHeaders(TrustedProxyHeaderXForwardedFor),
		WithTrustedProxyIPv6Prefix(64),
	)
	require.NoError(t, err)

	req := newTrustedProxyRequest("10.0.0.5:1234")
	req.Header.Set("X-Forwarded-For", "2001:db8:abcd:1234:1111:2222:3333:4444")

	assert.Equal(t, "2001:db8:abcd:1234::/64", extractor(req))
}

func TestTrustedProxyIPExtractorRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	_, err := NewTrustedProxyIPExtractor(WithTrustedProxyCIDRs("not-a-cidr"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid trusted proxy")

	_, err = NewTrustedProxyIPExtractor(WithTrustedProxyIPv6Prefix(129))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IPv6 prefix")
}

func TestTrustedProxyIPExtractorCanBeSharedByRateLimitAndSessions(t *testing.T) {
	t.Parallel()

	extractor, err := NewTrustedProxyIPExtractor(
		WithTrustedProxyCIDRs("10.0.0.0/8"),
		WithTrustedProxyHeaders(TrustedProxyHeaderXForwardedFor),
	)
	require.NoError(t, err)

	rateLimitConfig := NewDefaultRateLimiterConfig(WithRateLimiterKeyGenerator(extractor))
	sessionConfig := NewDefaultSessionConfig(WithSessionIPAddressExtractor(extractor))
	req := newTrustedProxyRequest("10.0.0.5:1234")
	req.Header.Set("X-Forwarded-For", "198.51.100.40")

	assert.Equal(t, "198.51.100.40", rateLimitConfig.KeyGenerator(req))
	assert.Equal(t, "198.51.100.40", sessionConfig.IPAddressExtractor(req))
}

func TestTrustedProxyIPExtractorDefaultsToNoForwardedHeaders(t *testing.T) {
	t.Parallel()

	extractor, err := NewTrustedProxyIPExtractor(WithTrustedProxyCIDRs("10.0.0.0/8"))
	require.NoError(t, err)

	req := newTrustedProxyRequest("10.0.0.5:1234")
	req.Header.Set("X-Forwarded-For", "198.51.100.50")

	assert.Equal(t, "10.0.0.5", extractor(req))
}

func TestDefaultRemoteAddrExtractorNormalizesBracketedIPv6(t *testing.T) {
	t.Parallel()

	config := NewDefaultSessionConfig()
	req := newTrustedProxyRequest("[2001:db8::1]:1234")

	assert.Equal(t, "2001:db8::1", config.IPAddressExtractor(req))
}

func newTrustedProxyRequest(remoteAddr string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req.RemoteAddr = remoteAddr
	return req
}
