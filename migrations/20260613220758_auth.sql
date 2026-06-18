-- +goose Up
-- Auth: refresh-token sessions, and one-time claim tokens for provisioned
-- tenants to set their password via a WhatsApp link.

CREATE TABLE sessions (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       uuid NOT NULL REFERENCES users(id),
  refresh_token_hash text NOT NULL UNIQUE,      -- store the hash, never the token
  user_agent    text,
  ip            inet,
  expires_at    timestamptz NOT NULL,
  revoked_at    timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_sessions_user ON sessions (user_id);

CREATE TABLE account_claim_tokens (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid NOT NULL REFERENCES users(id),
  token_hash  text NOT NULL UNIQUE,
  expires_at  timestamptz NOT NULL,
  used_at     timestamptz,
  created_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS account_claim_tokens;
DROP TABLE IF EXISTS sessions;