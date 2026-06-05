# Organizations And RBAC

The organization plugin adds organizations, memberships, and simple role-based
authorization.

## Install

```bash
go get github.com/ragokan/limen/plugins/organization
```

```go
import organization "github.com/ragokan/limen/plugins/organization"

auth, err := limen.New(&limen.Config{
	Database: adapter,
	Secret:   secret,
	Plugins: []limen.Plugin{
		organization.New(),
	},
})
```

## Roles

Built-in roles:

- `owner`
- `admin`
- `member`

Owners satisfy all role checks.

The bundled HTTP management routes for members and invitations require an
`owner` membership. The programmatic API can still be used from application
code when you want custom permission rules.

## Routes

The plugin mounts under `/organizations` relative to Limen's HTTP base path.

```text
POST   /auth/organizations
GET    /auth/organizations
POST   /auth/organizations/:id/members
DELETE /auth/organizations/:id/members/:user_id
POST   /auth/organizations/:id/invitations
GET    /auth/organizations/:id/invitations
POST   /auth/organizations/invitations/accept
```

Create request:

```json
{
  "name": "Acme",
  "slug": "acme"
}
```

Creating an organization also creates an `owner` membership for the current
session user.

Generic member and invitation flows cannot create `owner` memberships, and
owners cannot be removed by the generic member-removal API.

## Programmatic API

```go
orgs := organization.Use(auth)

org, err := orgs.CreateOrganization(ctx, userID, "Acme", "acme")
ok, err := orgs.HasRole(ctx, org.ID, userID, organization.RoleAdmin)
```

Use middleware on your own routes:

```go
middleware := orgs.MiddlewareRequireOrganizationRole("organization_id", organization.RoleAdmin)
mux.Handle("/organizations/{organization_id}/settings", middleware(http.HandlerFunc(handleSettings)))
```

The route parameter name must match the `paramName` argument.

## Invitations

Invitation creation returns a token once. Limen stores only an HMAC-SHA256 hash
of the token, and invitation list responses redact the plaintext token. Send the
token through your own email delivery flow, then have the invited user accept it
while signed in. Acceptance requires the signed-in user's email to match the
invitation email.
