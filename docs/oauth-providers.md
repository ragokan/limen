# OAuth Providers

Limen separates the OAuth core from provider modules. Install only the providers
your application uses.

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
