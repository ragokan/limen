# UUIDv7 IDs

By default, Limen assumes database-generated integer IDs. If your application
prefers globally unique string IDs, configure UUIDv7 IDs in the schema config.

UUIDv7 keeps the operational benefits of UUIDs while preserving approximate
creation-time ordering, which is friendlier for indexes than fully random IDs.

## Enable UUIDv7 IDs

```go
auth, err := limen.New(&limen.Config{
	Database: adapter,
	Secret:   []byte(os.Getenv("LIMEN_SECRET")),
	Schema: limen.NewDefaultSchemaConfig(
		limen.WithSchemaUUIDv7IDs(),
	),
	Plugins: []limen.Plugin{
		credentialpassword.New(),
	},
})
```

`WithSchemaUUIDv7IDs()` is shorthand for:

```go
limen.WithSchemaIDGenerator(limen.NewUUIDv7IDGenerator())
```

When enabled, Limen generates IDs in application code and schema ID columns use
the UUID column type.

## Migrations

Enable UUID IDs before generating your first migrations. Switching an existing
application from integer IDs to UUID IDs changes primary-key and foreign-key
types across Limen tables.

Recommended workflow:

1. Configure `WithSchemaUUIDv7IDs()`.
2. Enable `CLIConfig`.
3. Run the app once to refresh `.limen/schemas.json`.
4. Generate migrations with the CLI.
5. Apply migrations before serving traffic.

```bash
limen generate migrations \
  --driver postgres \
  --dsn "$DATABASE_URL" \
  --output ./migrations
```

## Custom ID Generators

Use `WithSchemaIDGenerator` when you need a different ID format:

```go
type ULIDGenerator struct{}

func (ULIDGenerator) Generate(ctx context.Context) (any, error) {
	return newULID(), nil
}

func (ULIDGenerator) GetColumnType() limen.ColumnType {
	return limen.ColumnTypeString
}

schema := limen.NewDefaultSchemaConfig(
	limen.WithSchemaIDGenerator(ULIDGenerator{}),
)
```

The generated value type must match the column type your generator reports.

## When To Use UUIDs

UUIDv7 IDs are a good fit when:

- records are created in multiple services or regions
- you do not want public URLs to expose sequential integer IDs
- you need stable IDs before inserting into the database

Integer IDs are still a simple default for single-database applications.
