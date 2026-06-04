package limen

import "testing"

func TestCoreSchemaOperationalIndexes(t *testing.T) {
	t.Parallel()

	schemas, err := discoverSchemas(NewDefaultSchemaConfig(), nil)
	if err != nil {
		t.Fatalf("discoverSchemas: %v", err)
	}

	assertSchemaIndex(t, schemas[CoreSchemaSessions], "idx_sessions_token", true, SessionSchemaTokenField)
	assertSchemaIndex(t, schemas[CoreSchemaSessions], "idx_sessions_expires_at", false, SessionSchemaExpiresAtField)
	assertSchemaIndex(t, schemas[CoreSchemaSessions], "idx_sessions_user_id_expires_at", false, SessionSchemaUserIDField, SessionSchemaExpiresAtField)
	assertSchemaIndex(t, schemas[CoreSchemaVerifications], "idx_verifications_expires_at", false, VerificationSchemaExpiresAtField)
	assertSchemaIndex(t, schemas[CoreSchemaRateLimits], "idx_rate_limits_last_request_at", false, RateLimitSchemaLastRequestAtField)
}

func assertSchemaIndex(t *testing.T, schema SchemaDefinition, name string, unique bool, columns ...SchemaField) {
	t.Helper()

	for _, index := range schema.Indexes {
		if index.Name != name {
			continue
		}
		if index.Unique != unique {
			t.Fatalf("%s unique = %v, want %v", name, index.Unique, unique)
		}
		if len(index.Columns) != len(columns) {
			t.Fatalf("%s columns = %#v, want %#v", name, index.Columns, columns)
		}
		for i := range columns {
			if index.Columns[i] != columns[i] {
				t.Fatalf("%s columns = %#v, want %#v", name, index.Columns, columns)
			}
		}
		return
	}
	t.Fatalf("missing index %s on %s", name, schema.GetTableName())
}
