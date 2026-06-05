package limen

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

const (
	TrustedProxyHeaderForwarded     = "Forwarded"
	TrustedProxyHeaderXForwardedFor = "X-Forwarded-For"
	TrustedProxyHeaderXRealIP       = "X-Real-IP"
)

type TrustedProxyIPExtractorOption func(*trustedProxyIPExtractorConfig)

type trustedProxyIPExtractorConfig struct {
	trustedProxies []netip.Prefix
	headers        []string
	ipv6PrefixBits int
	errs           []error
}

func NewTrustedProxyIPExtractor(opts ...TrustedProxyIPExtractorOption) (RequestExtractorFn, error) {
	config := &trustedProxyIPExtractorConfig{
		headers: nil,
	}
	for _, opt := range opts {
		opt(config)
	}
	if config.ipv6PrefixBits < 0 || config.ipv6PrefixBits > 128 {
		return nil, fmt.Errorf("limen: IPv6 prefix bits must be between 0 and 128")
	}
	if len(config.errs) > 0 {
		return nil, errors.Join(config.errs...)
	}

	extractor := &trustedProxyIPExtractor{
		trustedProxies: config.trustedProxies,
		headers:        config.headers,
		ipv6PrefixBits: config.ipv6PrefixBits,
	}
	return extractor.extract, nil
}

func WithTrustedProxyCIDRs(cidrs ...string) TrustedProxyIPExtractorOption {
	return func(c *trustedProxyIPExtractorConfig) {
		for _, cidr := range cidrs {
			prefix, ok := parseTrustedProxyPrefix(cidr)
			if ok {
				c.trustedProxies = append(c.trustedProxies, prefix)
			} else {
				c.errs = append(c.errs, fmt.Errorf("limen: invalid trusted proxy CIDR or IP %q", cidr))
			}
		}
	}
}

func WithTrustedProxyHeaders(headers ...string) TrustedProxyIPExtractorOption {
	return func(c *trustedProxyIPExtractorConfig) {
		c.headers = append([]string(nil), headers...)
	}
}

func WithTrustedProxyIPv6Prefix(bits int) TrustedProxyIPExtractorOption {
	return func(c *trustedProxyIPExtractorConfig) {
		c.ipv6PrefixBits = bits
	}
}

type trustedProxyIPExtractor struct {
	trustedProxies []netip.Prefix
	headers        []string
	ipv6PrefixBits int
}

func (e *trustedProxyIPExtractor) extract(req *http.Request) string {
	if req == nil {
		return ""
	}

	remote, ok := parseIPFromAddr(req.RemoteAddr)
	if !ok {
		return strings.TrimSpace(req.RemoteAddr)
	}
	if !e.isTrusted(remote) {
		return formatClientIP(remote, e.ipv6PrefixBits)
	}

	for _, header := range e.headers {
		chain := forwardedHeaderChain(req, header)
		if len(chain) == 0 {
			continue
		}
		chain = append(chain, remote)
		for i := len(chain) - 1; i >= 0; i-- {
			if !e.isTrusted(chain[i]) {
				return formatClientIP(chain[i], e.ipv6PrefixBits)
			}
		}
		return formatClientIP(chain[0], e.ipv6PrefixBits)
	}

	return formatClientIP(remote, e.ipv6PrefixBits)
}

func (e *trustedProxyIPExtractor) isTrusted(addr netip.Addr) bool {
	for _, prefix := range e.trustedProxies {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func forwardedHeaderChain(req *http.Request, header string) []netip.Addr {
	switch http.CanonicalHeaderKey(header) {
	case TrustedProxyHeaderForwarded:
		return parseForwardedHeader(req.Header.Values(TrustedProxyHeaderForwarded))
	case TrustedProxyHeaderXForwardedFor:
		return parseCommaSeparatedIPs(req.Header.Values(TrustedProxyHeaderXForwardedFor))
	case TrustedProxyHeaderXRealIP:
		return parseCommaSeparatedIPs(req.Header.Values(TrustedProxyHeaderXRealIP))
	default:
		return parseCommaSeparatedIPs(req.Header.Values(header))
	}
}

func parseForwardedHeader(values []string) []netip.Addr {
	var out []netip.Addr
	for _, value := range values {
		for _, entry := range strings.Split(value, ",") {
			for _, part := range strings.Split(entry, ";") {
				key, rawValue, ok := strings.Cut(strings.TrimSpace(part), "=")
				if !ok || !strings.EqualFold(key, "for") {
					continue
				}
				if addr, ok := parseIPFromAddr(rawValue); ok {
					out = append(out, addr)
				}
				break
			}
		}
	}
	return out
}

func parseCommaSeparatedIPs(values []string) []netip.Addr {
	var out []netip.Addr
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if addr, ok := parseIPFromAddr(part); ok {
				out = append(out, addr)
			}
		}
	}
	return out
}

func parseTrustedProxyPrefix(value string) (netip.Prefix, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return netip.Prefix{}, false
	}
	if prefix, err := netip.ParsePrefix(value); err == nil {
		return prefix.Masked(), true
	}
	addr, ok := parseIPFromAddr(value)
	if !ok {
		return netip.Prefix{}, false
	}
	bits := 32
	if addr.Is6() && !addr.Is4In6() {
		bits = 128
	}
	return netip.PrefixFrom(addr, bits), true
}

func parseIPFromAddr(value string) (netip.Addr, bool) {
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if value == "" {
		return netip.Addr{}, false
	}

	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	value = strings.TrimPrefix(strings.TrimSuffix(value, "]"), "[")

	addr, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Addr{}, false
	}
	if addr.Is4In6() {
		addr = addr.Unmap()
	}
	return addr, true
}

func formatClientIP(addr netip.Addr, ipv6PrefixBits int) string {
	if ipv6PrefixBits > 0 && addr.Is6() && !addr.Is4In6() {
		return netip.PrefixFrom(addr, ipv6PrefixBits).Masked().String()
	}
	return addr.String()
}
