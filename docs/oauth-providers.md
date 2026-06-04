# OAuth Providers

Limen separates the OAuth core from provider modules. Install only the providers
your application uses.

Provider behavior below is checked against provider documentation as of
2026-06-04.

## Provider Matrix

| Provider | Module | Env vars | Default scopes | Email verification | Notes |
| --- | --- | --- | --- | --- | --- |
| Google | `github.com/ragokan/limen/plugins/oauth-google` | `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` | `openid`, `email`, `profile` | From verified ID-token `email_verified` claim | OIDC nonce validation enabled |
| Apple | `github.com/ragokan/limen/plugins/oauth-apple` | `APPLE_CLIENT_ID`, `APPLE_CLIENT_SECRET` | `name`, `email` | From verified ID-token `email_verified` claim | Uses `response_mode=form_post`; name is only sent on first authorization |
| Facebook | `github.com/ragokan/limen/plugins/oauth-facebook` | `FACEBOOK_CLIENT_ID`, `FACEBOOK_CLIENT_SECRET` | `email`, `public_profile` | Not trusted by default | Graph API email is not treated as verified |
| LinkedIn | `github.com/ragokan/limen/plugins/oauth-linkedin` | `LINKEDIN_CLIENT_ID`, `LINKEDIN_CLIENT_SECRET` | `openid`, `profile`, `email` | From verified ID-token `email_verified` claim | Uses issuer `https://www.linkedin.com/oauth`; PKCE disabled for the web auth-code flow |
| Spotify | `github.com/ragokan/limen/plugins/oauth-spotify` | `SPOTIFY_CLIENT_ID`, `SPOTIFY_CLIENT_SECRET` | `user-read-email` | Not trusted | Spotify documents profile email as unverified |
| Discord | `github.com/ragokan/limen/plugins/oauth-discord` | `DISCORD_CLIENT_ID`, `DISCORD_CLIENT_SECRET` | `identify`, `email` | From Discord `verified` field | Uses `/api/users/@me` |
| GitHub | `github.com/ragokan/limen/plugins/oauth-github` | `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET` | `read:user`, `user:email` | From verified primary email API | Fetches primary email when profile email is absent |
| Microsoft | `github.com/ragokan/limen/plugins/oauth-microsoft` | `MICROSOFT_CLIENT_ID`, `MICROSOFT_CLIENT_SECRET` | `openid`, `profile`, `email`, `User.Read` | From ID-token/user profile claims | Tenant configurable |
| Twitch | `github.com/ragokan/limen/plugins/oauth-twitch` | `TWITCH_CLIENT_ID`, `TWITCH_CLIENT_SECRET` | `openid`, `user:read:email` | From ID-token `email_verified` claim | OIDC nonce validation enabled |
| Twitter/X | `github.com/ragokan/limen/plugins/oauth-twitter` | `TWITTER_CLIENT_ID`, `TWITTER_CLIENT_SECRET` | `users.read`, `users.email`, `tweet.read`, `offline.access` | Treated as verified when `confirmed_email` is returned | Requires email access enabled in the X app |
| ConsentKeys | `github.com/ragokan/limen/plugins/oauth-consentkeys` | `CONSENTKEYS_CLIENT_ID`, `CONSENTKEYS_CLIENT_SECRET` | `openid`, `profile`, `email` | From userinfo `email_verified` claim | Uses OIDC discovery |
| Generic | `github.com/ragokan/limen/plugins/oauth-generic` | Application-defined | `openid`, `email`, `profile` | Application-defined mapper | Use for OIDC/OAuth providers not listed above |

Instagram is intentionally not shipped as a first-party sign-in provider because
the current Instagram APIs do not provide a trusted user email suitable for
Limen's email-based account model. Use the generic provider only if your
application has an additional trusted email source.

## Provider Setup Notes

All redirect URLs should use your configured `BaseURL` and HTTP base path. With
the default HTTP path, provider callbacks are:

```text
https://auth.example.com/auth/oauth/{provider}/callback
```

Use explicit redirect URLs when your public callback differs from `BaseURL`, for
example behind a gateway:

```go
oauthgoogle.New(
	oauthgoogle.WithRedirectURL("https://auth.example.com/auth/oauth/google/callback"),
)
```

Provider constructors read their documented environment variables by default, so
typical wiring only needs to register the providers:

```go
auth, err := limen.New(&limen.Config{
	BaseURL:  "https://auth.example.com",
	Database: db,
	Secret:   []byte(os.Getenv("LIMEN_SECRET")),
	Plugins: []limen.Plugin{
		oauth.New(
			oauth.WithProviders(
				oauthgoogle.New(),
				oauthapple.New(),
				oauthfacebook.New(),
				oauthlinkedin.New(),
			),
		),
	},
})
```

If you mount Limen under a custom path, make the HTTP base path and registered
provider callbacks match:

```go
HTTP: limen.NewDefaultHTTPConfig(
	limen.WithHTTPBasePath("/api/auth"),
)
```

```text
https://auth.example.com/api/auth/oauth/google/callback
```

### Google

Register the exact callback URL in Google Cloud Console. Keep the default
`openid`, `email`, and `profile` scopes unless your application has a specific
reason to narrow them. Limen validates the ID token with issuer
`https://accounts.google.com`, verifies the nonce, and reads `email_verified`
from the token claims.

### Apple

Use a Services ID as `APPLE_CLIENT_ID` for web sign-in and provide the Apple
client-secret JWT as `APPLE_CLIENT_SECRET`. Apple sends callbacks with
`response_mode=form_post`; Limen stores the POST body briefly and resumes the
normal callback flow. Apple only sends the user's name on the first
authorization, so persist it when available.

### Facebook

Register the Facebook Login callback in Meta for Developers and request
`email`, `public_profile`. The Graph API profile response can include an email,
but it does not normally include a reliable email-verification claim. Limen
therefore treats Facebook profile emails as unverified for implicit account
linking.

### Instagram

Instagram is not exposed as a bundled sign-in provider. Current Instagram APIs
are profile/media APIs rather than a dependable email identity provider. Do not
use Instagram as the only login path for Limen's email-based account model
unless your application collects and verifies email separately.

### LinkedIn

Use the Sign In with LinkedIn using OpenID Connect product and request
`openid`, `profile`, and `email`. Limen validates ID tokens with issuer
`https://www.linkedin.com/oauth`, matching LinkedIn's OIDC discovery path, and
uses the optional `email_verified` claim when LinkedIn returns it.

## Verified Email Rules

Limen treats email ownership conservatively:

- A new OAuth user can be created with an unverified provider email. The local
  user remains unverified.
- An already-linked provider account can sign in even when the provider does not
  expose a trusted email-verification signal.
- Implicit linking by matching email requires both a verified provider email and
  a verified local email.
- Explicit linking by a currently authenticated user can link an unverified
  provider email when the provider account is not already linked elsewhere.

## OIDC Safety

OIDC providers verify ID tokens against the provider issuer and client ID before
mapping profile claims. Providers that support nonce bind the authorization
request to the ID token and reject nonce mismatch. The OAuth state also records
the expected provider, so a callback for one provider cannot be replayed through
another provider route.

## Provider Documentation

- Google OpenID Connect:
  <https://developers.google.com/identity/openid-connect/openid-connect>
- Sign in with Apple:
  <https://developer.apple.com/documentation/signinwithapple>
- Facebook Login permissions:
  <https://developers.facebook.com/docs/facebook-login/permissions>
- Instagram API with Instagram Login:
  <https://developers.facebook.com/docs/instagram-platform/instagram-api-with-instagram-login>
- Sign In with LinkedIn using OpenID Connect:
  <https://learn.microsoft.com/linkedin/consumer/integrations/self-serve/sign-in-with-linkedin-v2>
