# Credential Password

The credential-password plugin adds email/password sign up, email or username
sign in, password reset helpers, and password change routes.

## Install

```bash
go get github.com/ragokan/limen/plugins/credential-password
```

## Enable The Plugin

```go
import credentialpassword "github.com/ragokan/limen/plugins/credential-password"

auth, err := limen.New(&limen.Config{
	Database: adapter,
	Secret:   []byte(os.Getenv("LIMEN_SECRET")),
	Plugins: []limen.Plugin{
		credentialpassword.New(),
	},
})
```

## Password Rules

Defaults:

- minimum length: 4
- uppercase letter required
- number required
- symbol not required

`Password1` is a valid example password. Tune the rules when constructing the
plugin:

```go
credentialpassword.New(
	credentialpassword.WithPasswordMinLength(12),
	credentialpassword.WithPasswordRequireSymbols(true),
)
```

## Username Support

By default, users sign in with email. Enable username support if you want
`credential` to accept either email or username:

```go
credentialpassword.New(
	credentialpassword.WithUsernameSupport(true),
)
```

Require usernames during sign up:

```go
credentialpassword.New(
	credentialpassword.WithRequireUsernameOnSignUp(true),
)
```

Username support adds a `username` column to the users table. Refresh
`.limen/schemas.json` and regenerate migrations after enabling it.

## Routes

With `WithHTTPBasePath("/api/auth")`, the plugin registers:

```text
POST /api/auth/signup/credential
POST /api/auth/signin/credential
POST /api/auth/passwords/request-reset
POST /api/auth/passwords/reset
POST /api/auth/passwords/change
PUT  /api/auth/passwords
POST /api/auth/usernames/check
```

Password change and password set routes require an authenticated session.

## Sign Up

```http
POST /api/auth/signup/credential
Content-Type: application/json

{
  "email": "jane@example.com",
  "password": "Password1"
}
```

## Sign In

```http
POST /api/auth/signin/credential
Content-Type: application/json

{
  "credential": "jane@example.com",
  "password": "Password1",
  "remember_me": true
}
```

## Server-Side API

Use the plugin API when you need to call credential auth from Go code:

```go
api := credentialpassword.Use(auth)

result, err := api.SignInWithCredentialAndPassword(
	ctx,
	"jane@example.com",
	"Password1",
)
```

For HTTP-first apps, prefer the routes above and use `auth.GetSession(r)` in
your protected application handlers.
