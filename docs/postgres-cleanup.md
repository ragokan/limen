# PostgreSQL Cleanup And TTL

PostgreSQL does not expire rows automatically. Limen handles TTL behavior in the
application layer.

## Cleanup Paths

`CleanupExpired(ctx)` deletes:

- expired `sessions`
- expired `verifications`
- expired database-backed `rate_limits` when all configured rate-limit windows
  are static

Cleanup runs once after `limen.New` by default:

```go
auth, err := limen.New(&limen.Config{
	Database: db,
	Secret:   secret,
	Cleanup:  limen.NewDefaultCleanupConfig(limen.WithCleanupOnInit(true)),
})
```

Disable init cleanup when another process owns cleanup scheduling:

```go
Cleanup: limen.NewDefaultCleanupConfig(limen.WithCleanupOnInit(false))
```

Run manual cleanup from your scheduler:

```go
if err := auth.CleanupExpired(ctx); err != nil {
	return err
}
```

Expired sessions are also removed lazily when they are accessed through session
validation. `ListSessions(ctx, userID)` returns only active database sessions.

## Indexes

Core schemas include indexes for the cleanup and session-listing paths:

- `idx_sessions_token`
- `idx_sessions_user_id`
- `idx_sessions_expires_at`
- `idx_sessions_user_id_expires_at`
- `idx_verifications_value`
- `idx_verifications_subject`
- `idx_verifications_expires_at`
- `idx_rate_limits_key`
- `idx_rate_limits_last_request_at`

Regenerate migrations after upgrading if your existing database was created
before these indexes existed.
