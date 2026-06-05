package organization

import "github.com/ragokan/limen"

const (
	OrganizationsTableName limen.SchemaTableName = "organizations"
	MembershipsTableName   limen.SchemaTableName = "organization_memberships"
	InvitationsTableName   limen.SchemaTableName = "organization_invitations"

	OrganizationNameField      limen.SchemaField = "name"
	OrganizationSlugField      limen.SchemaField = "slug"
	OrganizationCreatedAtField limen.SchemaField = "created_at"
	OrganizationUpdatedAtField limen.SchemaField = "updated_at"

	MembershipOrganizationIDField limen.SchemaField = "organization_id"
	MembershipUserIDField         limen.SchemaField = "user_id"
	MembershipRoleField           limen.SchemaField = "role"
	MembershipCreatedAtField      limen.SchemaField = "created_at"
	MembershipUpdatedAtField      limen.SchemaField = "updated_at"

	InvitationOrganizationIDField limen.SchemaField = "organization_id"
	InvitationEmailField          limen.SchemaField = "email"
	InvitationRoleField           limen.SchemaField = "role"
	InvitationTokenField          limen.SchemaField = "token"
	InvitationExpiresAtField      limen.SchemaField = "expires_at"
	InvitationAcceptedAtField     limen.SchemaField = "accepted_at"
	InvitationCreatedAtField      limen.SchemaField = "created_at"
	InvitationUpdatedAtField      limen.SchemaField = "updated_at"
)

const defaultBasePath = "/organizations"
