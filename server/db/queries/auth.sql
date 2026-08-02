-- name: CreateUser :one
INSERT INTO users (email, email_normalized)
VALUES ($1, $2)
RETURNING user_id, email, email_verified_at;

-- name: CreateOAuthUser :one
INSERT INTO users (email, email_normalized, email_verified_at)
VALUES ($1, $2, now())
RETURNING user_id, email, email_verified_at;

-- name: GetUserByID :one
SELECT user_id, email, email_verified_at
FROM users WHERE user_id = $1;

-- name: GetUserByEmail :one
SELECT u.user_id, u.email, u.email_verified_at,
       COALESCE(p.password_hash, '')::text AS password_hash
FROM users u
LEFT JOIN password_credentials p USING (user_id)
WHERE u.email_normalized = $1;

-- name: FindUserIDByEmail :one
SELECT user_id FROM users WHERE email_normalized = $1;

-- name: InsertPasswordCredential :exec
INSERT INTO password_credentials (user_id, password_hash) VALUES ($1, $2);

-- name: UpsertPasswordCredential :exec
INSERT INTO password_credentials (user_id, password_hash) VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE SET password_hash = excluded.password_hash, updated_at = now();

-- name: VerifyUserEmail :exec
UPDATE users SET email_verified_at = COALESCE(email_verified_at, now()), updated_at = now()
WHERE user_id = $1;

-- name: CreateSession :exec
INSERT INTO auth_sessions (user_id, token_hash, expires_at) VALUES ($1, $2, $3);

-- name: DeleteSession :exec
DELETE FROM auth_sessions WHERE token_hash = $1;

-- name: DeleteUserSessions :exec
DELETE FROM auth_sessions WHERE user_id = $1;

-- name: GetUserBySession :one
SELECT u.user_id, u.email, u.email_verified_at
FROM auth_sessions s JOIN users u USING (user_id)
WHERE s.token_hash = $1 AND s.expires_at > now();

-- name: ListAuthProviders :many
SELECT 'password'::text AS provider FROM password_credentials WHERE password_credentials.user_id = $1
UNION ALL
SELECT provider FROM oauth_identities WHERE oauth_identities.user_id = $1
ORDER BY provider;

-- name: InvalidateAuthTokens :exec
UPDATE auth_tokens SET consumed_at = now()
WHERE user_id = $1 AND purpose = $2 AND consumed_at IS NULL;

-- name: CreateAuthToken :exec
INSERT INTO auth_tokens (user_id, purpose, token_hash, expires_at)
VALUES ($1, $2, $3, $4);

-- name: ConsumeAuthToken :one
UPDATE auth_tokens SET consumed_at = now()
WHERE token_hash = $1 AND purpose = $2 AND consumed_at IS NULL AND expires_at > now()
RETURNING user_id;

-- name: CreateOAuthAttempt :exec
INSERT INTO oauth_attempts
    (state_hash, provider, code_verifier, nonce, intent, user_id, return_to, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ConsumeOAuthAttempt :one
DELETE FROM oauth_attempts
WHERE state_hash = $1 AND expires_at > now()
RETURNING provider, code_verifier, nonce, intent, user_id, return_to;

-- name: GetUserByOAuthIdentity :one
SELECT u.user_id, u.email, u.email_verified_at
FROM oauth_identities i JOIN users u USING (user_id)
WHERE i.provider = $1 AND i.provider_subject = $2;

-- name: TouchOAuthIdentity :exec
UPDATE oauth_identities SET provider_email = $3, updated_at = now()
WHERE provider = $1 AND provider_subject = $2;

-- name: InsertOAuthIdentity :exec
INSERT INTO oauth_identities (user_id, provider, provider_subject, provider_email)
VALUES ($1, $2, $3, $4);
