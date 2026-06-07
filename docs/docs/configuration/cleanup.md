# Cleanup

Limen can remove expired auth data from the database. Cleanup covers:

- expired sessions
- expired verification tokens
- expired database-backed rate-limit rows when all rate-limit windows are static

Cleanup runs during `limen.New` by default.

## Configure Startup Cleanup

```go
auth, err := limen.New(&limen.Config{
	Database: adapter,
	Secret:   []byte(os.Getenv("LIMEN_SECRET")),
	Cleanup:  limen.NewDefaultCleanupConfig(limen.WithCleanupOnInit(true)),
})
```

Disable startup cleanup if you prefer to run cleanup from a scheduled job:

```go
Cleanup: limen.NewDefaultCleanupConfig(
	limen.WithCleanupOnInit(false),
)
```

## Run Cleanup Manually

```go
if err := auth.CleanupExpired(ctx); err != nil {
	log.Printf("limen cleanup failed: %v", err)
}
```

A common production setup is to disable startup cleanup and run
`CleanupExpired` from one cron job or background worker.

## Rate Limit Cleanup

Database-backed rate-limit cleanup only runs when Limen can calculate a static
maximum window. If any rate-limit rule uses a dynamic limit provider, Limen
skips rate-limit cleanup because it cannot know a safe expiration window.
