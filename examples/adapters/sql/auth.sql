-- Example PostgreSQL schema for Limen's database/sql adapter.
--
-- This file mirrors the default schema definitions and common plugin schemas
-- in the repository. Keep only the optional plugin sections you enable in
-- your Limen configuration.

BEGIN;

-- Core auth schema ----------------------------------------------------------

CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL,
  email VARCHAR(255) NOT NULL,
  password VARCHAR(255),
  email_verified_at TIMESTAMPTZ,
  first_name VARCHAR(255),
  last_name VARCHAR(255),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);

CREATE TABLE IF NOT EXISTS accounts (
  id BIGSERIAL,
  user_id BIGINT NOT NULL,
  provider VARCHAR(255) NOT NULL,
  provider_account_id VARCHAR(255),
  access_token TEXT NOT NULL,
  refresh_token TEXT,
  access_token_expires_at TIMESTAMPTZ,
  scope VARCHAR(255) NOT NULL,
  id_token TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (id),
  CONSTRAINT fk_accounts_users_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE RESTRICT
    ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_accounts_user_id_provider ON accounts (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_provider_provider_account_id
  ON accounts (provider, provider_account_id);

CREATE TABLE IF NOT EXISTS sessions (
  id BIGSERIAL,
  token VARCHAR(255) NOT NULL,
  user_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMPTZ NOT NULL,
  last_access TIMESTAMPTZ NOT NULL,
  metadata JSONB,
  PRIMARY KEY (id),
  CONSTRAINT fk_sessions_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE RESTRICT
    ON UPDATE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_token ON sessions (token);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions (expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id_expires_at
  ON sessions (user_id, expires_at);

CREATE TABLE IF NOT EXISTS verifications (
  id BIGSERIAL,
  subject VARCHAR(255) NOT NULL,
  value VARCHAR(255) NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_verifications_value ON verifications (value);
CREATE INDEX IF NOT EXISTS idx_verifications_subject ON verifications (subject);
CREATE INDEX IF NOT EXISTS idx_verifications_expires_at ON verifications (expires_at);

CREATE TABLE IF NOT EXISTS rate_limits (
  id BIGSERIAL,
  key VARCHAR(255) NOT NULL,
  count INTEGER NOT NULL,
  last_request_at BIGINT NOT NULL,
  PRIMARY KEY (id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_rate_limits_key ON rate_limits (key);
CREATE INDEX IF NOT EXISTS idx_rate_limits_last_request_at
  ON rate_limits (last_request_at);

-- credential-password plugin ------------------------------------------------
-- Required only when username support is enabled.

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS username VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);

-- two-factor plugin ---------------------------------------------------------

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS two_factor_enabled BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS two_factors (
  id BIGSERIAL,
  user_id BIGINT NOT NULL,
  secret VARCHAR(255),
  backup_codes TEXT,
  PRIMARY KEY (id),
  CONSTRAINT fk_two_factors_users_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE RESTRICT
    ON UPDATE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_two_factors_user_id ON two_factors (user_id);

-- session-jwt plugin --------------------------------------------------------

CREATE TABLE IF NOT EXISTS jwt_refresh_tokens (
  id BIGSERIAL,
  token VARCHAR(255) NOT NULL,
  user_id BIGINT NOT NULL,
  jwt_id VARCHAR(255) NOT NULL,
  family VARCHAR(255) NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT fk_jwt_refresh_tokens_users_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE CASCADE
    ON UPDATE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_jwt_refresh_tokens_token
  ON jwt_refresh_tokens (token);
CREATE INDEX IF NOT EXISTS idx_jwt_refresh_tokens_user_id
  ON jwt_refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_jwt_refresh_tokens_family
  ON jwt_refresh_tokens (family);

CREATE TABLE IF NOT EXISTS jwt_blacklist (
  jti VARCHAR(255),
  expires_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (jti)
);

CREATE INDEX IF NOT EXISTS idx_jwt_blacklist_expires_at
  ON jwt_blacklist (expires_at);

-- api-key plugin ------------------------------------------------------------

CREATE TABLE IF NOT EXISTS api_keys (
  id BIGSERIAL,
  user_id BIGINT NOT NULL,
  name VARCHAR(255) NOT NULL,
  prefix VARCHAR(255) NOT NULL,
  key_hash VARCHAR(255) NOT NULL,
  scopes TEXT,
  expires_at TIMESTAMPTZ,
  last_used_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT fk_api_keys_users_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE CASCADE
    ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys (prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys (expires_at);
CREATE INDEX IF NOT EXISTS idx_api_keys_revoked_at ON api_keys (revoked_at);

-- organization plugin -------------------------------------------------------

CREATE TABLE IF NOT EXISTS organizations (
  id BIGSERIAL,
  name VARCHAR(255) NOT NULL,
  slug VARCHAR(255) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_organizations_slug ON organizations (slug);

CREATE TABLE IF NOT EXISTS organization_memberships (
  id BIGSERIAL,
  organization_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  role VARCHAR(255) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT fk_organization_memberships_organizations_organization_id
    FOREIGN KEY (organization_id)
    REFERENCES organizations (id)
    ON DELETE CASCADE
    ON UPDATE CASCADE,
  CONSTRAINT fk_organization_memberships_users_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE CASCADE
    ON UPDATE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_organization_memberships_org_user
  ON organization_memberships (organization_id, user_id);
CREATE INDEX IF NOT EXISTS idx_organization_memberships_user_id
  ON organization_memberships (user_id);

CREATE TABLE IF NOT EXISTS organization_invitations (
  id BIGSERIAL,
  organization_id BIGINT NOT NULL,
  email VARCHAR(255) NOT NULL,
  role VARCHAR(255) NOT NULL,
  token VARCHAR(255) NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  accepted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT fk_organization_invitations_organizations_organization_id
    FOREIGN KEY (organization_id)
    REFERENCES organizations (id)
    ON DELETE CASCADE
    ON UPDATE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_organization_invitations_token
  ON organization_invitations (token);
CREATE INDEX IF NOT EXISTS idx_organization_invitations_org_email
  ON organization_invitations (organization_id, email);
CREATE INDEX IF NOT EXISTS idx_organization_invitations_expires_at
  ON organization_invitations (expires_at);

COMMIT;
