# Protocol Roadmap

These protocol surfaces are intentionally later than OpenAPI, admin, API keys,
and organizations.

## Passkeys / WebAuthn

Passkeys should be the next user-facing authentication protocol after the core
platform APIs stabilize. The implementation should use a proven WebAuthn library
for ceremony validation and should include:

- credential table
- registration challenge storage
- authentication challenge storage
- origin/RP ID configuration
- account recovery guidance

## Enterprise SSO

Enterprise SSO should build on organizations. Required pieces:

- organization-owned SSO connection table
- OIDC and SAML connection configuration
- domain verification
- enforced SSO policy per organization
- JIT membership provisioning controls

OIDC should come before SAML because it fits the existing OAuth/OIDC provider
code paths more naturally.

## SCIM

SCIM should come after organizations and enterprise SSO. Required pieces:

- organization-scoped bearer/API-key auth
- user and group provisioning endpoints
- deprovisioning behavior
- mapping between SCIM users/groups and Limen users/memberships

SCIM should not be implemented before organization membership semantics are
stable.
