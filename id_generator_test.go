package limen

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestUUIDv7IDGenerator(t *testing.T) {
	generator := NewUUIDv7IDGenerator()

	if generator.GetColumnType() != ColumnTypeUUID {
		t.Fatalf("expected UUID column type, got %s", generator.GetColumnType())
	}

	value, err := generator.Generate(context.Background())
	if err != nil {
		t.Fatalf("generate UUIDv7 ID: %v", err)
	}

	id, ok := value.(string)
	if !ok {
		t.Fatalf("expected string ID, got %T", value)
	}

	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("parse generated ID: %v", err)
	}
	if parsed.Version() != uuid.Version(7) {
		t.Fatalf("expected UUIDv7, got version %d", parsed.Version())
	}
}

func TestWithSchemaUUIDv7IDs(t *testing.T) {
	schemaConfig := NewDefaultSchemaConfig(WithSchemaUUIDv7IDs())

	if schemaConfig.GetIDColumnType() != ColumnTypeUUID {
		t.Fatalf("expected UUID ID columns, got %s", schemaConfig.GetIDColumnType())
	}

	userSchema := schemaConfig.User.Introspect(schemaConfig).(*SchemaDefinition)
	idColumn := userSchema.Columns[0]
	if idColumn.LogicalField != SchemaIDField || idColumn.Type != ColumnTypeUUID {
		t.Fatalf("expected first user column to be UUID id, got %+v", idColumn)
	}

	config := &Config{Schema: schemaConfig}
	data, err := config.serializeSchemasToJSON(SchemaDefinitionMap{
		CoreSchemaUsers: *userSchema,
	})
	if err != nil {
		t.Fatalf("serialize schemas: %v", err)
	}

	var file struct {
		UseAutoIncrementID bool `json:"useAutoIncrementID"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("decode serialized schemas: %v", err)
	}
	if file.UseAutoIncrementID {
		t.Fatal("expected UUIDv7 schema config to disable auto-increment IDs")
	}
}
