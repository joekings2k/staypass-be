-- +goose Up
-- Shared foundation: uuid generation + the updated_at trigger function
-- that nearly every table reuses.
 
CREATE EXTENSION IF NOT EXISTS pgcrypto;
 
-- +goose StatementBegin
CREATE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
 
-- +goose Down
DROP FUNCTION IF EXISTS set_updated_at();
-- pgcrypto left in place intentionally; dropping it is rarely what you want.
 