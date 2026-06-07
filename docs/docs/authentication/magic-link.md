# Magic Link

The magic-link plugin adds passwordless email sign-in. Users request a link,
receive a token through your delivery callback, and verify it to create a
session.

## Install

```bash
go get github.com/ragokan/limen/plugins/magic-link
```

## Enable

```go
import magiclink "github.com/ragokan/limen/plugins/magic-link"

auth, err := limen.New(&limen.Config{
	Database: adapter,
	Secret:   []byte(os.Getenv("LIMEN_SECRET")),
	Plugins: []limen.Plugin{
		magiclink.New(
			magiclink.WithSendMagicLink(func(message magiclink.MagicLinkMessage) {
				sendEmail(message.Email, message.URL)
			}),
		),
	},
})
```

Defaults:

- token expiration: 15 minutes
- auto-create users: enabled
- max uses: 1
- mark email verified on success: enabled

## Routes

With `WithHTTPBasePath("/api/auth")`, routes are:

```text
POST /api/auth/magic-link/signin
GET  /api/auth/magic-link/verify
```

## Request A Link

```http
POST /api/auth/magic-link/signin
Content-Type: application/json

{
  "email": "jane@example.com",
  "redirect_uri": "https://app.example.com/auth/callback"
}
```

Deliver the generated URL from your `WithSendMagicLink` callback.

## Verification

The user follows the magic link and Limen verifies the token. On success, Limen
creates a session and redirects according to the request options.

## Common Options

```go
magiclink.New(
	magiclink.WithTokenExpiration(10*time.Minute),
	magiclink.WithAutoCreateUser(false),
	magiclink.WithMaxUses(1),
	magiclink.WithMarkEmailVerified(true),
)
```

Use `WithMapMetaToUser` to persist request metadata to newly auto-created
users.
