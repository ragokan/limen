package main

import (
	"strings"
	"testing"
	"time"

	"github.com/thecodearcher/limen"
)

func TestMigrationVersionIncrementsTimestampPerMigration(t *testing.T) {
	base := time.Date(2026, 6, 7, 3, 44, 8, 0, time.UTC)

	got := []string{
		migrationVersion(base, 0, "users"),
		migrationVersion(base, 1, "accounts"),
		migrationVersion(base, 2, "sessions"),
	}

	want := []string{
		"20260607034408_users",
		"20260607034409_accounts",
		"20260607034410_sessions",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("version %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestOrderSchemasByDependenciesPutsReferencedTablesFirst(t *testing.T) {
	schemas := limen.SchemaDefinitionMap{
		"accounts": schemaDef("accounts", fk("app_users")),
		"sessions": schemaDef("sessions", fk("app_users")),
		"users":    schemaDef("app_users"),
	}

	ordered, err := orderSchemasByDependencies(schemas)
	if err != nil {
		t.Fatalf("orderSchemasByDependencies returned error: %v", err)
	}

	assertBefore(t, ordered, "users", "accounts")
	assertBefore(t, ordered, "users", "sessions")
}

func TestOrderSchemasByDependenciesSortsIndependentSchemas(t *testing.T) {
	schemas := limen.SchemaDefinitionMap{
		"users":         schemaDef("users"),
		"rate_limits":   schemaDef("rate_limits"),
		"verifications": schemaDef("verifications"),
	}

	ordered, err := orderSchemasByDependencies(schemas)
	if err != nil {
		t.Fatalf("orderSchemasByDependencies returned error: %v", err)
	}

	want := []limen.SchemaName{"rate_limits", "users", "verifications"}
	for i := range want {
		if ordered[i] != want[i] {
			t.Fatalf("ordered[%d] = %q, want %q; full order: %v", i, ordered[i], want[i], ordered)
		}
	}
}

func TestOrderSchemasByDependenciesDetectsCycles(t *testing.T) {
	schemas := limen.SchemaDefinitionMap{
		"a": schemaDef("a", fk("b")),
		"b": schemaDef("b", fk("a")),
	}

	_, err := orderSchemasByDependencies(schemas)
	if err == nil {
		t.Fatal("expected cycle error")
	}
	if !strings.Contains(err.Error(), "dependency cycle") {
		t.Fatalf("expected dependency cycle error, got %v", err)
	}
}

func schemaDef(tableName string, foreignKeys ...limen.ForeignKeyDefinition) limen.SchemaDefinition {
	return limen.SchemaDefinition{
		TableName: limen.SchemaTableName(tableName),
		Columns: []limen.ColumnDefinition{
			{
				Name:         "id",
				LogicalField: limen.SchemaIDField,
				Type:         limen.ColumnTypeInt64,
				IsPrimaryKey: true,
			},
		},
		ForeignKeys: foreignKeys,
	}
}

func fk(referencedSchema limen.SchemaName) limen.ForeignKeyDefinition {
	return limen.ForeignKeyDefinition{
		Name:             "fk_test",
		Column:           "user_id",
		ReferencedSchema: referencedSchema,
		ReferencedField:  limen.SchemaIDField,
	}
}

func assertBefore(t *testing.T, ordered []limen.SchemaName, before, after limen.SchemaName) {
	t.Helper()

	beforeIndex := schemaIndex(ordered, before)
	afterIndex := schemaIndex(ordered, after)
	if beforeIndex == -1 {
		t.Fatalf("%q not found in ordered schemas %v", before, ordered)
	}
	if afterIndex == -1 {
		t.Fatalf("%q not found in ordered schemas %v", after, ordered)
	}
	if beforeIndex > afterIndex {
		t.Fatalf("expected %q before %q, got %v", before, after, ordered)
	}
}

func schemaIndex(ordered []limen.SchemaName, target limen.SchemaName) int {
	for i, name := range ordered {
		if name == target {
			return i
		}
	}
	return -1
}
