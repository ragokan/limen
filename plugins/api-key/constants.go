package apikey

import "github.com/ragokan/limen"

const (
	SchemaTableName limen.SchemaTableName = "api_keys"

	SchemaUserIDField     limen.SchemaField = "user_id"
	SchemaNameField       limen.SchemaField = "name"
	SchemaPrefixField     limen.SchemaField = "prefix"
	SchemaKeyHashField    limen.SchemaField = "key_hash"
	SchemaScopesField     limen.SchemaField = "scopes"
	SchemaExpiresAtField  limen.SchemaField = "expires_at"
	SchemaLastUsedAtField limen.SchemaField = "last_used_at"
	SchemaRevokedAtField  limen.SchemaField = "revoked_at"
	SchemaCreatedAtField  limen.SchemaField = "created_at"
	SchemaUpdatedAtField  limen.SchemaField = "updated_at"
)

const (
	defaultBasePath         = "/api-keys"
	defaultHeaderName       = "X-Limen-API-Key"
	defaultAuthorization    = "ApiKey "
	defaultKeyPrefix        = "limen_sk_"
	defaultStoredPrefixSize = 20
)
