package organization

import "time"

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

type Organization struct {
	ID        any       `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	raw       map[string]any
}

func (o Organization) Raw() map[string]any { return o.raw }

type Membership struct {
	ID             any       `json:"id"`
	OrganizationID any       `json:"organization_id"`
	UserID         any       `json:"user_id"`
	Role           Role      `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	raw            map[string]any
}

func (m Membership) Raw() map[string]any { return m.raw }

type Invitation struct {
	ID             any        `json:"id"`
	OrganizationID any        `json:"organization_id"`
	Email          string     `json:"email"`
	Role           Role       `json:"role"`
	Token          string     `json:"token,omitempty"`
	ExpiresAt      time.Time  `json:"expires_at"`
	AcceptedAt     *time.Time `json:"accepted_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	tokenHash      string
	raw            map[string]any
}

func (i Invitation) Raw() map[string]any { return i.raw }
