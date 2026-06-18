-- name: CreateSession :one
-- Create a new session for a user. The refresh token is hashed before storage.
INSERT INTO sessions (user_id, refresh_token_hash, user_agent, ip, expires_at)
VALUES (@user_id, @refresh_token_hash, @user_agent, @ip, @expires_at)
RETURNING *;


-- name: GetSessionById :one
-- Get a session by its ID. Exclude revoked sessions.
SELECT * FROM sessions
WHERE id = $1 AND revoked_at IS NULL;


-- name: GetActiveSessionByTokenHash :one
-- Get an active session by the hash of its refresh token. Exclude revoked sessions.
SELECT * FROM sessions
WHERE refresh_token_hash = $1 
AND revoked_at IS NULL 
AND expires_at > now();   

-- name: RevokeSession :exec
-- Revoke a session by its ID.
UPDATE sessions
SET revoked_at = now()
WHERE id = @id 
AND revoked_at IS NULL;

-- name: RevokeAllSessionsForUser :exec
-- Revoke all sessions for a user.  
UPDATE sessions
SET revoked_at = now()
WHERE user_id = @user_id
AND revoked_at IS NULL;

-- name: DeleteExpiredSessions :exec
-- Delete expired sessions , clean up old sessions that are no longer valid. cron job can call this periodically.
DELETE FROM sessions
WHERE expires_at < now();


-- ---- account claim tokens ----


-- name: CreateAccountClaimToken :one 
-- Create a new account claim token for a user. The token is hashed before storage.
INSERT INTO account_claim_tokens (user_id, token_hash, expires_at)
VALUES (@user_id, @token_hash, @expires_at)
RETURNING *;

-- name: GetActiveClaimToken :one 
-- Get an active claim token by its hash. Exclude used tokens.
SELECT * FROM account_claim_tokens
WHERE token_hash = $1
AND used_at IS NULL
AND expires_at > now();

-- name: MarkClaimTokenUsed :exec
-- Mark a claim token as used by its ID.
UPDATE account_claim_tokens
SET used_at = now()
WHERE id = @id
AND used_at IS NULL;
