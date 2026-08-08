package auth

import (
	"context"
	"errors"
	"time"

	"github.com/heywinit/prowl/server/internal/database"
	"github.com/heywinit/prowl/server/internal/securetoken"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errNotFound = errors.New("not found")
var errAccountExists = errors.New("account exists")
var errIdentityOwned = errors.New("identity already belongs to another account")

type Store struct {
	pool    *pgxpool.Pool
	queries *database.Queries
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool, queries: database.New(pool)} }

func userFromValues(id int64, email string, verified pgtype.Timestamptz) User {
	return User{ID: id, Email: email, EmailVerifiedAt: database.TimePtr(verified)}
}

func (s *Store) createPasswordUser(ctx context.Context, email, hash string) (User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)
	row, err := q.CreateUser(ctx, database.CreateUserParams{Email: email, EmailNormalized: normalizeEmail(email)})
	if err != nil {
		return User{}, err
	}
	if err = q.InsertPasswordCredential(ctx, database.InsertPasswordCredentialParams{UserID: row.UserID, PasswordHash: hash}); err != nil {
		return User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return User{}, err
	}

	return userFromValues(row.UserID, row.Email, row.EmailVerifiedAt), nil
}

func (s *Store) userByEmail(ctx context.Context, email string) (User, string, error) {
	row, err := s.queries.GetUserByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, "", errNotFound
	}
	if err != nil {
		return User{}, "", err
	}

	return userFromValues(row.UserID, row.Email, row.EmailVerifiedAt), row.PasswordHash, nil
}

func (s *Store) userBySession(ctx context.Context, token string) (User, []string, error) {
	digest := securetoken.Digest(token)
	row, err := s.queries.GetUserBySession(ctx, digest)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, nil, errNotFound
	}
	if err != nil {
		return User{}, nil, err
	}

	providers, err := s.queries.ListAuthProviders(ctx, row.UserID)
	return userFromValues(row.UserID, row.Email, row.EmailVerifiedAt), providers, err
}

func (s *Store) createSession(ctx context.Context, userID int64) (string, error) {
	token, digest, err := securetoken.New()
	if err != nil {
		return "", err
	}
	err = s.queries.CreateSession(ctx, database.CreateSessionParams{UserID: userID, TokenHash: digest, ExpiresAt: database.Timestamp(time.Now().Add(30 * 24 * time.Hour))})

	return token, err
}

func (s *Store) deleteSession(ctx context.Context, token string) error {
	return s.queries.DeleteSession(ctx, securetoken.Digest(token))
}

func (s *Store) createAuthToken(ctx context.Context, userID int64, purpose string, ttl time.Duration) (string, error) {
	token, digest, err := securetoken.New()
	if err != nil {
		return "", err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)
	if err = q.InvalidateAuthTokens(ctx, database.InvalidateAuthTokensParams{UserID: userID, Purpose: purpose}); err != nil {
		return "", err
	}
	if err = q.CreateAuthToken(ctx, database.CreateAuthTokenParams{UserID: userID, Purpose: purpose, TokenHash: digest, ExpiresAt: database.Timestamp(time.Now().Add(ttl))}); err != nil {
		return "", err
	}

	return token, tx.Commit(ctx)
}

func (s *Store) consumeAuthToken(ctx context.Context, token, purpose string) (User, error) {
	digest := securetoken.Digest(token)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)
	q := s.queries.WithTx(tx)
	userID, err := q.ConsumeAuthToken(ctx, database.ConsumeAuthTokenParams{TokenHash: digest, Purpose: purpose})
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, errNotFound
	}
	if err != nil {
		return User{}, err
	}
	if purpose == "verify_email" {
		if err = q.VerifyUserEmail(ctx, userID); err != nil {
			return User{}, err
		}
	}
	row, err := q.GetUserByID(ctx, userID)
	if err != nil {
		return User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return userFromValues(row.UserID, row.Email, row.EmailVerifiedAt), nil
}

func (s *Store) resetPassword(ctx context.Context, token, hash string) (User, error) {
	digest := securetoken.Digest(token)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)
	q := s.queries.WithTx(tx)
	userID, err := q.ConsumeAuthToken(ctx, database.ConsumeAuthTokenParams{TokenHash: digest, Purpose: "reset_password"})
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, errNotFound
	}
	if err != nil {
		return User{}, err
	}
	row, err := q.GetUserByID(ctx, userID)
	if err != nil {
		return User{}, err
	}
	if err = q.UpsertPasswordCredential(ctx, database.UpsertPasswordCredentialParams{UserID: userID, PasswordHash: hash}); err != nil {
		return User{}, err
	}
	if err = q.DeleteUserSessions(ctx, userID); err != nil {
		return User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return userFromValues(row.UserID, row.Email, row.EmailVerifiedAt), nil
}

func (s *Store) saveOAuthAttempt(ctx context.Context, state string, a oauthAttempt) error {
	digest := securetoken.Digest(state)
	return s.queries.CreateOAuthAttempt(ctx, database.CreateOAuthAttemptParams{StateHash: digest, Provider: a.Provider, CodeVerifier: database.OptionalText(a.CodeVerifier), Nonce: database.OptionalText(a.Nonce), Intent: a.Intent, UserID: database.OptionalInt64(a.UserID), ReturnTo: a.ReturnTo, ExpiresAt: database.Timestamp(time.Now().Add(10 * time.Minute))})
}

func (s *Store) consumeOAuthAttempt(ctx context.Context, state string) (oauthAttempt, error) {
	digest := securetoken.Digest(state)
	row, err := s.queries.ConsumeOAuthAttempt(ctx, digest)
	if errors.Is(err, pgx.ErrNoRows) {
		return oauthAttempt{}, errNotFound
	}
	if err != nil {
		return oauthAttempt{}, err
	}
	a := oauthAttempt{Provider: row.Provider, CodeVerifier: row.CodeVerifier.String, Nonce: row.Nonce.String, Intent: row.Intent, ReturnTo: row.ReturnTo}
	if row.UserID.Valid {
		value := row.UserID.Int64
		a.UserID = &value
	}
	return a, nil
}

func (s *Store) completeOAuth(ctx context.Context, provider string, identity providerIdentity, a oauthAttempt) (User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)
	q := s.queries.WithTx(tx)
	existing, err := q.GetUserByOAuthIdentity(ctx, database.GetUserByOAuthIdentityParams{Provider: provider, ProviderSubject: identity.Subject})
	if err == nil {
		user := userFromValues(existing.UserID, existing.Email, existing.EmailVerifiedAt)
		if a.Intent == "link" && (a.UserID == nil || *a.UserID != user.ID) {
			return User{}, errIdentityOwned
		}
		if err = q.TouchOAuthIdentity(ctx, database.TouchOAuthIdentityParams{Provider: provider, ProviderSubject: identity.Subject, ProviderEmail: database.OptionalText(identity.Email)}); err != nil {
			return User{}, err
		}
		return user, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return User{}, err
	}
	if a.Intent == "link" {
		if a.UserID == nil {
			return User{}, errNotFound
		}
		row, err := q.GetUserByID(ctx, *a.UserID)
		if err != nil {
			return User{}, err
		}
		if err = q.InsertOAuthIdentity(ctx, database.InsertOAuthIdentityParams{UserID: *a.UserID, Provider: provider, ProviderSubject: identity.Subject, ProviderEmail: database.OptionalText(identity.Email)}); err != nil {
			return User{}, err
		}
		if err = tx.Commit(ctx); err != nil {
			return User{}, err
		}
		return userFromValues(row.UserID, row.Email, row.EmailVerifiedAt), nil
	}
	if !identity.EmailVerified || identity.Email == "" {
		return User{}, errors.New("provider did not return a verified email")
	}
	if _, err = q.FindUserIDByEmail(ctx, normalizeEmail(identity.Email)); err == nil {
		return User{}, errAccountExists
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return User{}, err
	}
	row, err := q.CreateOAuthUser(ctx, database.CreateOAuthUserParams{Email: identity.Email, EmailNormalized: normalizeEmail(identity.Email)})
	if err != nil {
		return User{}, err
	}
	if err = q.InsertOAuthIdentity(ctx, database.InsertOAuthIdentityParams{UserID: row.UserID, Provider: provider, ProviderSubject: identity.Subject, ProviderEmail: database.OptionalText(identity.Email)}); err != nil {
		return User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return userFromValues(row.UserID, row.Email, row.EmailVerifiedAt), nil
}
