package main

import (
	"strings"
	"testing"

	"github.com/ragokan/limen"
)

func TestGenerateCreateTableIncludesOperationalIndexes(t *testing.T) {
	t.Parallel()

	schema := limen.NewDefaultSchemaConfig().Session.Introspect(limen.NewDefaultSchemaConfig()).(*limen.SchemaDefinition)
	generator, err := newSQLMigrationGenerator(NewPostgresDriver(), &cliConfig{UseAutoIncrementID: true})
	if err != nil {
		t.Fatalf("newSQLMigrationGenerator: %v", err)
	}

	sql, err := generator.generateCreateTable(schema)
	if err != nil {
		t.Fatalf("generateCreateTable: %v", err)
	}

	for _, want := range []string{
		"CREATE UNIQUE INDEX idx_sessions_token ON sessions (token);",
		"CREATE INDEX idx_sessions_user_id ON sessions (user_id);",
		"CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);",
		"CREATE INDEX idx_sessions_user_id_expires_at ON sessions (user_id, expires_at);",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("generated SQL missing %q:\n%s", want, sql)
		}
	}
}

func TestPostgresIntrospectIndexesQueryUsesCurrentSchema(t *testing.T) {
	t.Parallel()

	driver := &postgresDriver{currentSchema: "auth"}
	query, args := driver.IntrospectIndexesQuery("sessions")

	if !strings.Contains(query, "ns.nspname = $2") {
		t.Fatalf("query must constrain schema: %s", query)
	}
	if len(args) != 2 || args[0] != "sessions" || args[1] != "auth" {
		t.Fatalf("args = %#v", args)
	}
}
