-- +goose Up
-- Identity & tenancy. Roles live on memberships, never on users.

CREATE TYPE membership_role AS ENUM (
  'super_admin',
  'estate_admin',
  'gateman',
  'landlord',
  'resident'
);

CREATE TYPE membership_status AS ENUM ('active', 'suspended', 'ended');

CREATE TABLE users (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  phone          text NOT NULL UNIQUE,          -- E.164, primary identifier (phone-first)
  email          text UNIQUE,                   -- optional
  full_name      text NOT NULL,
  password_hash  text,                          -- null until a provisioned account is claimed
  is_super_admin boolean NOT NULL DEFAULT false,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  deleted_at     timestamptz
);

CREATE TRIGGER trg_users_updated BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE estates (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name        text NOT NULL,
  slug        text NOT NULL UNIQUE,             -- url-safe, future per-estate routing
  address     text,
  city        text,
  state       text,
  timezone    text NOT NULL DEFAULT 'Africa/Lagos',
  is_active   boolean NOT NULL DEFAULT true,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_estates_updated BEFORE UPDATE ON estates
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- one row per (user, estate, role). a landlord who also lives in the
-- estate has two rows.
CREATE TABLE memberships (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  estate_id   uuid NOT NULL REFERENCES estates(id),
  user_id     uuid NOT NULL REFERENCES users(id),
  role        membership_role NOT NULL,
  status      membership_status NOT NULL DEFAULT 'active',
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  ended_at    timestamptz,
  UNIQUE (estate_id, user_id, role)
);

CREATE INDEX idx_memberships_estate ON memberships (estate_id);
CREATE INDEX idx_memberships_user   ON memberships (user_id);

CREATE TRIGGER trg_memberships_updated BEFORE UPDATE ON memberships
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS estates;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS membership_status;
DROP TYPE IF EXISTS membership_role;