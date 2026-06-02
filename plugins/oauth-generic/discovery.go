package oauthgeneric

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// discoveryDocument holds the OIDC discovery document fields we use.
type discoveryDocument struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	Issuer                string `json:"issuer"`
}

// fetchDiscoveryDocument fetches and parses the OpenID Connect discovery document.
func fetchDiscoveryDocument(discoveryURL string) (*discoveryDocument, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, discoveryURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("discovery request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discovery fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery fetch: %s", resp.Status)
	}

	var doc discoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("discovery decode: %w", err)
	}
	return &doc, nil
}
