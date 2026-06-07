# Email Verification

Email verification is enabled by default. Limen stores verification tokens and
marks users as verified when a valid token is submitted.

## Configure Delivery

Provide a callback that sends the verification token through your email system:

```go
Email: limen.NewDefaultEmailConfig(
	limen.WithEmailVerification(
		limen.WithSendEmailVerificationMail(func(email string, token string) {
			sendVerificationEmail(email, token)
		}),
	),
)
```

The default token expiration is 24 hours.

```go
Email: limen.NewDefaultEmailConfig(
	limen.WithEmailVerification(
		limen.WithEmailVerificationExpiration(2*time.Hour),
	),
)
```

## Disable Verification

```go
Email: limen.NewDefaultEmailConfig(
	limen.WithEmailVerification(
		limen.WithDisableEmailVerification(),
	),
)
```

## Custom Tokens

Override token generation when you need a signed token, short numeric code, or
another delivery format:

```go
Email: limen.NewDefaultEmailConfig(
	limen.WithEmailVerification(
		limen.WithEmailVerificationTokenGenerator(func(user *limen.User) (string, error) {
			return buildSignedVerificationToken(user)
		}),
	),
)
```

## Routes

When email verification is enabled, Limen registers:

```text
POST /api/auth/verify-email
POST /api/auth/email-verifications
```

`/email-verifications` requires an authenticated session and creates a new token
for the current user. `/verify-email` accepts the token and marks the email as
verified.
