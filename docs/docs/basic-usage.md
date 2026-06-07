# Basic Usage

After installation, most application code interacts with Limen through HTTP
routes and `auth.GetSession(r)`.

The examples below assume:

- `BaseURL` is `http://localhost:8080`
- `WithHTTPBasePath("/api/auth")` is configured
- The credential-password plugin is enabled

## Sign Up

Create a user with email and password:

```http
POST /api/auth/signup/credential
Content-Type: application/json

{
  "email": "jane@example.com",
  "password": "Password1"
}
```

If username support is enabled for the credential-password plugin, include
`username` in the request body.

## Sign In

Sign in with an email address or username:

```http
POST /api/auth/signin/credential
Content-Type: application/json

{
  "credential": "jane@example.com",
  "password": "Password1",
  "remember_me": true
}
```

On success, Limen creates a session and returns the authenticated user and
session details. Cookie behavior depends on your HTTP and session config.

## Read The Current Session

Use the built-in route from a browser or HTTP client:

```http
GET /api/auth/me
```

Use `auth.GetSession(r)` inside your own Go handlers:

```go
mux.HandleFunc("GET /api/profile", func(w http.ResponseWriter, r *http.Request) {
	session, err := auth.GetSession(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"user":    session.User,
		"session": session.Session,
	})
})
```

## Sign Out

End the current session:

```http
POST /api/auth/signout
```

## Manage Sessions

List active sessions for the current user:

```http
GET /api/auth/sessions
```

Revoke all sessions for the current user:

```http
POST /api/auth/revoke-sessions
```

## Add OAuth

Install the OAuth core plugin and the provider module you need:

```bash
go get github.com/ragokan/limen/plugins/oauth
go get github.com/ragokan/limen/plugins/oauth-google
```

Configure the provider in `Plugins`, then start the provider flow with:

```http
GET /api/auth/oauth/google/authorize?callback_url=https://app.example.com/callback
```

See [OAuth Providers](../oauth-providers.md) for provider modules, scopes, and
verified-email behavior.

## Working Examples

The repository includes standalone examples:

```bash
DATABASE_URL="postgres://..." go run ./examples/basic
DATABASE_URL="postgres://..." go run ./examples/gin
DATABASE_URL="postgres://..." go run ./examples/adapters/sql
DATABASE_URL="postgres://..." go run ./examples/adapters/gorm
```

Next: [CLI](concepts/cli.md).
