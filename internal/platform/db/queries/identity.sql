-- name: CreateUser :one 
-- password hash is optional, null until a provisioned account is claimed
INSERT INTO users (phone, email, full_name, password_hash, is_super_admin)
VALUES (@phone, @email, @full_name, @password_hash, @is_super_admin)
RETURNING *;

-- name: GetUserByPhone :one
-- Get a user by their phone number. Exclude soft-deleted users.
SELECT * FROM users 
WHERE phone = $1 AND deleted_at IS NULL;


-- name: GetUserByID :one
-- Get a user by their ID. Exclude soft-deleted users.
SELECT * FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
-- Get a user by their email. Exclude soft-deleted users.
SELECT * FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: SetUserPasswordHash :one
-- Update a user's password hash. Returns the updated user.
UPDATE users
SET password_hash = @password_hash
WHERE id = @id AND deleted_at IS NULL
RETURNING *;

-- name: ListEstateMembershipForUser :many
-- List all memberships for a given user in a specific estate.
SELECT * FROM memberships
WHERE user_id = @user_id
AND estate_id = @estate_id
AND status = 'active'
ORDER BY created_at DESC;

-- name: ListUserMembershipsForUser :many
-- List all memberships for a given user.
SELECT * FROM memberships
WHERE user_id = @user_id
AND status = 'active'
ORDER BY created_at DESC;


-- name: CreateMembership :one
-- Create a new membership for a user
INSERT INTO memberships (estate_id, user_id, role)
VALUES (@estate_id, @user_id, @role)
RETURNING *;

-- name: SuspendMembershipByID :one
-- Suspend a single membership for a user using the membership id
UPDATE memberships
SET status = 'suspended'
WHERE id = $1
AND status = 'active'
RETURNING *; 



-- name: SuspendMembershipForUser :many
-- Suspend a single membership (reversible). Only acts on active memberships.
UPDATE memberships
SET status = 'suspended'
WHERE user_id = @user_id
AND status = 'active'
RETURNING *;
