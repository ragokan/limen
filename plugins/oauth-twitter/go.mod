module github.com/thecodearcher/limen/plugins/oauth-twitter

go 1.25.0

require (
	github.com/thecodearcher/limen/plugins/oauth v0.1.0
	golang.org/x/oauth2 v0.36.0
)

require (
	github.com/coreos/go-oidc/v3 v3.18.0 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/thecodearcher/limen v0.1.1 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)

replace github.com/thecodearcher/limen => ../..

replace github.com/thecodearcher/limen/plugins/oauth => ../oauth
