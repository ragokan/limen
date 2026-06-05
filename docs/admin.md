# Admin Plugin

The admin plugin provides a small first-party administration surface for Limen.
Administrators are explicit: configure admin emails or user IDs at startup.
Admin email matches require `EmailVerifiedAt` to be set. Prefer immutable user
IDs when bootstrapping a production owner account.

## Install

```bash
go get github.com/ragokan/limen/plugins/admin
```

```go
import admin "github.com/ragokan/limen/plugins/admin"

auth, err := limen.New(&limen.Config{
	Database: adapter,
	Secret:   secret,
	Plugins: []limen.Plugin{
		admin.New(
			admin.WithAdminEmails("owner@example.com"),
		),
	},
})
```

## Routes

The plugin mounts under `/admin` relative to Limen's HTTP base path.

```text
GET  /auth/admin/users
GET  /auth/admin/users/:id
POST /auth/admin/users/:id/revoke-sessions
```

All routes require a valid user session and configured admin access.

## Programmatic API

```go
adminAPI := admin.Use(auth)

users, err := adminAPI.ListUsers(ctx)
err = adminAPI.RevokeUserSessions(ctx, userID)
```

Admin responses exclude password hashes.

## User Status

This plugin does not yet expose ban/disable fields. Those fields should be added
only with matching authentication enforcement, so disabled users cannot continue
signing in through another provider.
