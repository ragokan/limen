package organization

import (
	"time"

	"github.com/ragokan/limen"
)

type organizationSchema struct{ limen.BaseSchema }
type membershipSchema struct{ limen.BaseSchema }
type invitationSchema struct{ limen.BaseSchema }

func newOrganizationSchema() *organizationSchema { return &organizationSchema{} }
func newMembershipSchema() *membershipSchema     { return &membershipSchema{} }
func newInvitationSchema() *invitationSchema     { return &invitationSchema{} }

func (s *organizationSchema) GetSoftDeleteField() string { return "" }
func (s *membershipSchema) GetSoftDeleteField() string   { return "" }
func (s *invitationSchema) GetSoftDeleteField() string   { return "" }

func (s *organizationSchema) GetNameField() string { return s.GetField(OrganizationNameField) }
func (s *organizationSchema) GetSlugField() string { return s.GetField(OrganizationSlugField) }
func (s *organizationSchema) GetCreatedAtField() string {
	return s.GetField(OrganizationCreatedAtField)
}
func (s *organizationSchema) GetUpdatedAtField() string {
	return s.GetField(OrganizationUpdatedAtField)
}

func (s *membershipSchema) GetOrganizationIDField() string {
	return s.GetField(MembershipOrganizationIDField)
}
func (s *membershipSchema) GetUserIDField() string { return s.GetField(MembershipUserIDField) }
func (s *membershipSchema) GetRoleField() string   { return s.GetField(MembershipRoleField) }
func (s *membershipSchema) GetCreatedAtField() string {
	return s.GetField(MembershipCreatedAtField)
}
func (s *membershipSchema) GetUpdatedAtField() string {
	return s.GetField(MembershipUpdatedAtField)
}

func (s *invitationSchema) GetOrganizationIDField() string {
	return s.GetField(InvitationOrganizationIDField)
}
func (s *invitationSchema) GetEmailField() string { return s.GetField(InvitationEmailField) }
func (s *invitationSchema) GetRoleField() string  { return s.GetField(InvitationRoleField) }
func (s *invitationSchema) GetTokenField() string { return s.GetField(InvitationTokenField) }
func (s *invitationSchema) GetExpiresAtField() string {
	return s.GetField(InvitationExpiresAtField)
}
func (s *invitationSchema) GetAcceptedAtField() string {
	return s.GetField(InvitationAcceptedAtField)
}
func (s *invitationSchema) GetCreatedAtField() string {
	return s.GetField(InvitationCreatedAtField)
}
func (s *invitationSchema) GetUpdatedAtField() string {
	return s.GetField(InvitationUpdatedAtField)
}

func (s *organizationSchema) ToStorage(data limen.Model) map[string]any {
	org := data.(*Organization)
	return map[string]any{
		s.GetNameField():      org.Name,
		s.GetSlugField():      org.Slug,
		s.GetCreatedAtField(): org.CreatedAt,
		s.GetUpdatedAtField(): org.UpdatedAt,
	}
}

func (s *organizationSchema) FromStorage(data map[string]any) limen.Model {
	return &Organization{
		ID:        data[s.GetIDField()],
		Name:      stringValue(data[s.GetNameField()]),
		Slug:      stringValue(data[s.GetSlugField()]),
		CreatedAt: timeValue(data[s.GetCreatedAtField()]),
		UpdatedAt: timeValue(data[s.GetUpdatedAtField()]),
		raw:       data,
	}
}

func (s *membershipSchema) ToStorage(data limen.Model) map[string]any {
	membership := data.(*Membership)
	return map[string]any{
		s.GetOrganizationIDField(): membership.OrganizationID,
		s.GetUserIDField():         membership.UserID,
		s.GetRoleField():           string(membership.Role),
		s.GetCreatedAtField():      membership.CreatedAt,
		s.GetUpdatedAtField():      membership.UpdatedAt,
	}
}

func (s *membershipSchema) FromStorage(data map[string]any) limen.Model {
	return &Membership{
		ID:             data[s.GetIDField()],
		OrganizationID: data[s.GetOrganizationIDField()],
		UserID:         data[s.GetUserIDField()],
		Role:           Role(stringValue(data[s.GetRoleField()])),
		CreatedAt:      timeValue(data[s.GetCreatedAtField()]),
		UpdatedAt:      timeValue(data[s.GetUpdatedAtField()]),
		raw:            data,
	}
}

func (s *invitationSchema) ToStorage(data limen.Model) map[string]any {
	invitation := data.(*Invitation)
	return map[string]any{
		s.GetOrganizationIDField(): invitation.OrganizationID,
		s.GetEmailField():          invitation.Email,
		s.GetRoleField():           string(invitation.Role),
		s.GetTokenField():          invitation.tokenHash,
		s.GetExpiresAtField():      invitation.ExpiresAt,
		s.GetAcceptedAtField():     invitation.AcceptedAt,
		s.GetCreatedAtField():      invitation.CreatedAt,
		s.GetUpdatedAtField():      invitation.UpdatedAt,
	}
}

func (s *invitationSchema) FromStorage(data map[string]any) limen.Model {
	return &Invitation{
		ID:             data[s.GetIDField()],
		OrganizationID: data[s.GetOrganizationIDField()],
		Email:          stringValue(data[s.GetEmailField()]),
		Role:           Role(stringValue(data[s.GetRoleField()])),
		tokenHash:      stringValue(data[s.GetTokenField()]),
		ExpiresAt:      timeValue(data[s.GetExpiresAtField()]),
		AcceptedAt:     timePtr(data[s.GetAcceptedAtField()]),
		CreatedAt:      timeValue(data[s.GetCreatedAtField()]),
		UpdatedAt:      timeValue(data[s.GetUpdatedAtField()]),
		raw:            data,
	}
}

func buildOrganizationTableDef(schemaConfig *limen.SchemaConfig, schema *organizationSchema) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForTable(
		limen.SchemaName(OrganizationsTableName),
		OrganizationsTableName,
		schema,
		limen.WithSchemaIDField(schemaConfig),
		limen.WithSchemaField(string(OrganizationNameField), limen.ColumnTypeString),
		limen.WithSchemaField(string(OrganizationSlugField), limen.ColumnTypeString),
		limen.WithSchemaField(string(OrganizationCreatedAtField), limen.ColumnTypeTime, limen.WithDefaultValue(string(limen.DatabaseDefaultValueNow))),
		limen.WithSchemaField(string(OrganizationUpdatedAtField), limen.ColumnTypeTime, limen.WithDefaultValue(string(limen.DatabaseDefaultValueNow))),
		limen.WithSchemaUniqueIndex("idx_organizations_slug", []limen.SchemaField{OrganizationSlugField}),
	)
}

func buildMembershipTableDef(schemaConfig *limen.SchemaConfig, schema *membershipSchema) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForTable(
		limen.SchemaName(MembershipsTableName),
		MembershipsTableName,
		schema,
		limen.WithSchemaIDField(schemaConfig),
		limen.WithSchemaField(string(MembershipOrganizationIDField), schemaConfig.GetIDColumnType()),
		limen.WithSchemaField(string(MembershipUserIDField), schemaConfig.GetIDColumnType()),
		limen.WithSchemaField(string(MembershipRoleField), limen.ColumnTypeString),
		limen.WithSchemaField(string(MembershipCreatedAtField), limen.ColumnTypeTime, limen.WithDefaultValue(string(limen.DatabaseDefaultValueNow))),
		limen.WithSchemaField(string(MembershipUpdatedAtField), limen.ColumnTypeTime, limen.WithDefaultValue(string(limen.DatabaseDefaultValueNow))),
		limen.WithSchemaUniqueIndex("idx_organization_memberships_org_user", []limen.SchemaField{MembershipOrganizationIDField, MembershipUserIDField}),
		limen.WithSchemaIndex("idx_organization_memberships_user_id", []limen.SchemaField{MembershipUserIDField}),
		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_organization_memberships_organizations_organization_id",
			Column:           MembershipOrganizationIDField,
			ReferencedSchema: limen.SchemaName(OrganizationsTableName),
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionCascade,
			OnUpdate:         limen.FKActionCascade,
		}),
		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_organization_memberships_users_user_id",
			Column:           MembershipUserIDField,
			ReferencedSchema: limen.CoreSchemaUsers,
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionCascade,
			OnUpdate:         limen.FKActionCascade,
		}),
	)
}

func buildInvitationTableDef(schemaConfig *limen.SchemaConfig, schema *invitationSchema) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForTable(
		limen.SchemaName(InvitationsTableName),
		InvitationsTableName,
		schema,
		limen.WithSchemaIDField(schemaConfig),
		limen.WithSchemaField(string(InvitationOrganizationIDField), schemaConfig.GetIDColumnType()),
		limen.WithSchemaField(string(InvitationEmailField), limen.ColumnTypeString),
		limen.WithSchemaField(string(InvitationRoleField), limen.ColumnTypeString),
		limen.WithSchemaField(string(InvitationTokenField), limen.ColumnTypeString),
		limen.WithSchemaField(string(InvitationExpiresAtField), limen.ColumnTypeTime),
		limen.WithSchemaField(string(InvitationAcceptedAtField), limen.ColumnTypeTime, limen.WithNullable(true)),
		limen.WithSchemaField(string(InvitationCreatedAtField), limen.ColumnTypeTime, limen.WithDefaultValue(string(limen.DatabaseDefaultValueNow))),
		limen.WithSchemaField(string(InvitationUpdatedAtField), limen.ColumnTypeTime, limen.WithDefaultValue(string(limen.DatabaseDefaultValueNow))),
		limen.WithSchemaUniqueIndex("idx_organization_invitations_token", []limen.SchemaField{InvitationTokenField}),
		limen.WithSchemaIndex("idx_organization_invitations_org_email", []limen.SchemaField{InvitationOrganizationIDField, InvitationEmailField}),
		limen.WithSchemaIndex("idx_organization_invitations_expires_at", []limen.SchemaField{InvitationExpiresAtField}),
		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_organization_invitations_organizations_organization_id",
			Column:           InvitationOrganizationIDField,
			ReferencedSchema: limen.SchemaName(OrganizationsTableName),
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionCascade,
			OnUpdate:         limen.FKActionCascade,
		}),
	)
}

func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func timeValue(value any) time.Time {
	if t, ok := value.(time.Time); ok {
		return t
	}
	return time.Time{}
}

func timePtr(value any) *time.Time {
	if t, ok := value.(time.Time); ok {
		return &t
	}
	if t, ok := value.(*time.Time); ok {
		return t
	}
	return nil
}
