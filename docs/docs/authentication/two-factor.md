# Two-Factor

The two-factor plugin adds TOTP, OTP, and backup-code flows on top of an
existing primary sign-in method such as credential-password or OAuth.

## Install

```bash
go get github.com/ragokan/limen/plugins/two-factor
```

## Enable

```go
import twofactor "github.com/ragokan/limen/plugins/two-factor"

auth, err := limen.New(&limen.Config{
	Database: adapter,
	Secret:   []byte(os.Getenv("LIMEN_SECRET")),
	Plugins: []limen.Plugin{
		credentialpassword.New(),
		twofactor.New(),
	},
})
```

The plugin uses `Config.Secret` by default. Use `twofactor.WithSecret` when you
need a separate secret.

## Routes

With `WithHTTPBasePath("/api/auth")`, routes are mounted under
`/api/auth/two-factor`.

```text
POST /api/auth/two-factor/initiate-setup
GET  /api/auth/two-factor/totp/uri
POST /api/auth/two-factor/finalize-setup
POST /api/auth/two-factor/disable
POST /api/auth/two-factor/verify
POST /api/auth/two-factor/otp/send
GET  /api/auth/two-factor/backup-codes
PUT  /api/auth/two-factor/backup-codes
```

Setup, disable, TOTP URI, and backup-code routes require an authenticated
session. Login verification uses the challenge cookie created after primary
sign-in.

## Login Flow

When a user with two-factor enabled signs in successfully, the plugin removes
the session response and returns:

```json
{
  "two_factor_required": true
}
```

The client then submits the second factor:

```http
POST /api/auth/two-factor/verify
Content-Type: application/json

{
  "code": "123456",
  "method": "totp"
}
```

On success, Limen creates the final session.

## Session Rotation

By default, enabling, disabling, and verifying two-factor rotates sessions and
revokes other sessions on state changes. Keep that default for production
account security unless you have a specific compatibility reason to change it.
