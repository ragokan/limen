package apikey

import (
	"encoding/json"
	"time"

	"github.com/ragokan/limen"
)

type apiKeySchema struct {
	limen.BaseSchema
}

func newAPIKeySchema() *apiKeySchema {
	return &apiKeySchema{BaseSchema: limen.BaseSchema{}}
}

func (s *apiKeySchema) GetSoftDeleteField() string { return "" }

func (s *apiKeySchema) GetUserIDField() string     { return s.GetField(SchemaUserIDField) }
func (s *apiKeySchema) GetNameField() string       { return s.GetField(SchemaNameField) }
func (s *apiKeySchema) GetPrefixField() string     { return s.GetField(SchemaPrefixField) }
func (s *apiKeySchema) GetKeyHashField() string    { return s.GetField(SchemaKeyHashField) }
func (s *apiKeySchema) GetScopesField() string     { return s.GetField(SchemaScopesField) }
func (s *apiKeySchema) GetExpiresAtField() string  { return s.GetField(SchemaExpiresAtField) }
func (s *apiKeySchema) GetLastUsedAtField() string { return s.GetField(SchemaLastUsedAtField) }
func (s *apiKeySchema) GetRevokedAtField() string  { return s.GetField(SchemaRevokedAtField) }
func (s *apiKeySchema) GetCreatedAtField() string  { return s.GetField(SchemaCreatedAtField) }
func (s *apiKeySchema) GetUpdatedAtField() string  { return s.GetField(SchemaUpdatedAtField) }

func (s *apiKeySchema) ToStorage(data limen.Model) map[string]any {
	key := data.(*APIKey)
	return map[string]any{
		s.GetUserIDField():     key.UserID,
		s.GetNameField():       key.Name,
		s.GetPrefixField():     key.Prefix,
		s.GetKeyHashField():    key.keyHash,
		s.GetScopesField():     marshalScopes(key.Scopes),
		s.GetExpiresAtField():  key.ExpiresAt,
		s.GetLastUsedAtField(): key.LastUsedAt,
		s.GetRevokedAtField():  key.RevokedAt,
		s.GetCreatedAtField():  key.CreatedAt,
		s.GetUpdatedAtField():  key.UpdatedAt,
	}
}

func (s *apiKeySchema) FromStorage(data map[string]any) limen.Model {
	return &APIKey{
		ID:         data[s.GetIDField()],
		UserID:     data[s.GetUserIDField()],
		Name:       stringValue(data[s.GetNameField()]),
		Prefix:     stringValue(data[s.GetPrefixField()]),
		keyHash:    stringValue(data[s.GetKeyHashField()]),
		Scopes:     unmarshalScopes(stringValue(data[s.GetScopesField()])),
		ExpiresAt:  timePtr(data[s.GetExpiresAtField()]),
		LastUsedAt: timePtr(data[s.GetLastUsedAtField()]),
		RevokedAt:  timePtr(data[s.GetRevokedAtField()]),
		CreatedAt:  timeValue(data[s.GetCreatedAtField()]),
		UpdatedAt:  timeValue(data[s.GetUpdatedAtField()]),
		raw:        data,
	}
}

func (s *apiKeySchema) Serialize(data limen.Model) map[string]any {
	key := data.(*APIKey)
	out := map[string]any{
		s.GetIDField():         key.ID,
		s.GetUserIDField():     key.UserID,
		s.GetNameField():       key.Name,
		s.GetPrefixField():     key.Prefix,
		s.GetScopesField():     key.Scopes,
		s.GetExpiresAtField():  key.ExpiresAt,
		s.GetLastUsedAtField(): key.LastUsedAt,
		s.GetRevokedAtField():  key.RevokedAt,
		s.GetCreatedAtField():  key.CreatedAt,
		s.GetUpdatedAtField():  key.UpdatedAt,
	}
	return out
}

func buildAPIKeyTableDef(schemaConfig *limen.SchemaConfig, schema *apiKeySchema) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForTable(
		limen.SchemaName(SchemaTableName),
		SchemaTableName,
		schema,
		limen.WithSchemaIDField(schemaConfig),
		limen.WithSchemaField(string(SchemaUserIDField), schemaConfig.GetIDColumnType()),
		limen.WithSchemaField(string(SchemaNameField), limen.ColumnTypeString),
		limen.WithSchemaField(string(SchemaPrefixField), limen.ColumnTypeString),
		limen.WithSchemaField(string(SchemaKeyHashField), limen.ColumnTypeString),
		limen.WithSchemaField(string(SchemaScopesField), limen.ColumnTypeText, limen.WithNullable(true)),
		limen.WithSchemaField(string(SchemaExpiresAtField), limen.ColumnTypeTime, limen.WithNullable(true)),
		limen.WithSchemaField(string(SchemaLastUsedAtField), limen.ColumnTypeTime, limen.WithNullable(true)),
		limen.WithSchemaField(string(SchemaRevokedAtField), limen.ColumnTypeTime, limen.WithNullable(true)),
		limen.WithSchemaField(string(SchemaCreatedAtField), limen.ColumnTypeTime, limen.WithDefaultValue(string(limen.DatabaseDefaultValueNow))),
		limen.WithSchemaField(string(SchemaUpdatedAtField), limen.ColumnTypeTime, limen.WithDefaultValue(string(limen.DatabaseDefaultValueNow))),
		limen.WithSchemaIndex("idx_api_keys_user_id", []limen.SchemaField{SchemaUserIDField}),
		limen.WithSchemaUniqueIndex("idx_api_keys_prefix", []limen.SchemaField{SchemaPrefixField}),
		limen.WithSchemaIndex("idx_api_keys_expires_at", []limen.SchemaField{SchemaExpiresAtField}),
		limen.WithSchemaIndex("idx_api_keys_revoked_at", []limen.SchemaField{SchemaRevokedAtField}),
		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_api_keys_users_user_id",
			Column:           SchemaUserIDField,
			ReferencedSchema: limen.CoreSchemaUsers,
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionCascade,
			OnUpdate:         limen.FKActionCascade,
		}),
	)
}

func marshalScopes(scopes []string) string {
	if len(scopes) == 0 {
		return ""
	}
	data, err := json.Marshal(scopes)
	if err != nil {
		return ""
	}
	return string(data)
}

func unmarshalScopes(value string) []string {
	if value == "" {
		return nil
	}
	var scopes []string
	if err := json.Unmarshal([]byte(value), &scopes); err != nil {
		return nil
	}
	return scopes
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case *string:
		if typed == nil {
			return ""
		}
		return *typed
	default:
		return ""
	}
}

func timeValue(value any) time.Time {
	if t := timePtr(value); t != nil {
		return *t
	}
	return time.Time{}
}

func timePtr(value any) *time.Time {
	switch typed := value.(type) {
	case time.Time:
		return &typed
	case *time.Time:
		return typed
	default:
		return nil
	}
}
