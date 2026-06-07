# OAuth

The OAuth plugin adds provider-based sign-in, account linking, token storage,
and provider token refresh routes.

Install the OAuth core plugin and one or more provider modules:

```bash
go get github.com/ragokan/limen/plugins/oauth
go get github.com/ragokan/limen/plugins/oauth-google
```

## Configure Providers

```go
import (
	"github.com/ragokan/limen/plugins/oauth"
	oauthgoogle "github.com/ragokan/limen/plugins/oauth-google"
)

auth, err := limen.New(&limen.Config{
	BaseURL:  "http://localhost:8080",
	Database: adapter,
	Secret:   []byte(os.Getenv("LIMEN_SECRET")),
	HTTP: limen.NewDefaultHTTPConfig(
		limen.WithHTTPBasePath("/api/auth"),
		limen.WithHTTPTrustedOrigins([]string{"https://app.example.com"}),
	),
	Plugins: []limen.Plugin{
		oauth.New(
			oauth.WithProviders(
				oauthgoogle.New(
					oauthgoogle.WithClientID(os.Getenv("GOOGLE_CLIENT_ID")),
					oauthgoogle.WithClientSecret(os.Getenv("GOOGLE_CLIENT_SECRET")),
				),
			),
		),
	},
})
```

The OAuth plugin uses `Config.Secret` by default for state tokens and token
encryption. Use `oauth.WithSecret` only when you need a plugin-specific 32-byte
secret.

## Routes

With `WithHTTPBasePath("/api/auth")`, OAuth routes are mounted under
`/api/auth/oauth`.

```text
GET    /api/auth/oauth/:provider/authorize
GET    /api/auth/oauth/:provider/callback
POST   /api/auth/oauth/:provider/callback
GET    /api/auth/oauth/:provider/link
GET    /api/auth/oauth/accounts
DELETE /api/auth/oauth/:provider/unlink
GET    /api/auth/oauth/:provider/tokens
POST   /api/auth/oauth/:provider/tokens/refresh
```

Link, unlink, list accounts, token read, and token refresh routes require an
authenticated session.

## Start Sign-In

Redirect the user to:

```text
GET /api/auth/oauth/google/authorize?callback_url=https://app.example.com/callback
```

Limen validates redirect targets against trusted origins, redirects to the
provider, handles the callback, and creates a session on success.

## Sign-Up Policy

By default, OAuth can create a user if no matching account exists. Require
explicit sign-up instead:

```go
oauth.New(
	oauth.WithRequireExplicitSignUp(),
	oauth.WithProviders(...),
)
```

## State And Token Storage

By default, OAuth state is stateless and cookie-backed. Store state in the
database when you need server-side state tracking:

```go
oauth.New(
	oauth.WithDatabaseState(),
	oauth.WithProviders(...),
)
```

OAuth tokens are encrypted by default. Disable this only when another storage
layer already encrypts tokens:

```go
oauth.New(
	oauth.WithDisableTokensEncryption(),
	oauth.WithProviders(...),
)
```

See [OAuth Providers](../../oauth-providers.md) for supported provider modules,
scopes, and verified-email behavior.
