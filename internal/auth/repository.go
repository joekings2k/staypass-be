package auth

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	dbgen "github.com/joekings2k/staypass-be/internal/platform/db/gen"
)

// Session is the DB-agnostic representation of a session used throughout the
// auth layer. It deliberately avoids pgx/sqlc types so the rest of the package
// (and any future non-Postgres implementation) never depends on the database.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	RevokedAt *time.Time // nil means the session has not been revoked
	CreatedAt time.Time
}

// CreateSessionParams is the DB-agnostic input for creating a session.
type CreateSessionParams struct {
	UserID           uuid.UUID
	RefreshTokenHash string
	UserAgent        *string
	IP               *netip.Addr
	ExpiresAt        time.Time
}

type AccountClaimToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	UsedAt    time.Time
	ExpiredAt time.Time
}

// AuthRepository is the storage boundary for the auth layer. It is defined in
// terms of domain types only, so the service depends on this interface rather
// than on any particular database. Swapping databases means writing a new
// implementation; callers do not change.
type AuthRepository interface {
	CreateSession(ctx context.Context, params CreateSessionParams) (Session, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (Session, error)
	GetActiveSessionUserByTokenHash(ctx context.Context, tokenHash string, userID uuid.UUID) (Session, error)
	RevokeSession(ctx context.Context, sessionID uuid.UUID) error
	RevokeAllSessionsForUser(ctx context.Context, userID uuid.UUID) error
	ConsumeAccountClaimTokne(ctx context.Context, tokenHash string) (AccountClaimToken, error)
}

type pgAuthRepository struct {
	queries *dbgen.Queries
}

func NewPgAuthRepository(db *dbgen.Queries) AuthRepository {
	return &pgAuthRepository{
		queries: db,
	}
}

func (r *pgAuthRepository) CreateSession(ctx context.Context, params CreateSessionParams) (Session, error) {
	dbSession, err := r.queries.CreateSession(ctx, toDBCreateSessionParams(params))
	if err != nil {
		return Session{}, err
	}
	return toDomainSession(dbSession), nil
}
func (r *pgAuthRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (Session, error) {
	dbSession, err := r.queries.GetActiveSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrSessionNotFound
		}
		return Session{}, err
	}
	return toDomainSession(dbSession), nil
}

func (r *pgAuthRepository) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	return r.queries.RevokeSession(ctx, pgtype.UUID{Bytes: sessionID, Valid: true})
}

func (r *pgAuthRepository) RevokeAllSessionsForUser(ctx context.Context, userID uuid.UUID) error {
	return r.queries.RevokeAllSessionsForUser(ctx, pgtype.UUID{Bytes: userID, Valid: true})
}

func (r *pgAuthRepository) GetActiveSessionUserByTokenHash(ctx context.Context, tokenHash string, userID uuid.UUID) (Session, error) {
	arg := dbgen.GetActiveSessionUserByTokenHashParams{
		RefreshTokenHash: tokenHash,
		UserID:           pgtype.UUID{Bytes: userID, Valid: true},
	}
	dbSession, err := r.queries.GetActiveSessionUserByTokenHash(ctx, arg)
	if err != nil {
		return Session{}, err
	}
	return toDomainSession(dbSession), nil
}

func (r *pgAuthRepository) ConsumeAccountClaimTokne(ctx context.Context, tokenHash string) (AccountClaimToken, error) {
	accountCliamTokenResult, err := r.queries.ConsumeAccountClaimToken(ctx, tokenHash)
	if err != nil {
		return AccountClaimToken{}, err
	}
	return toDomainAccountClaim(accountCliamTokenResult), nil
}

// --- mapping helpers: the only place pgx/sqlc types touch the auth package ---

func toDBCreateSessionParams(p CreateSessionParams) dbgen.CreateSessionParams {
	return dbgen.CreateSessionParams{
		UserID:           pgtype.UUID{Bytes: p.UserID, Valid: true},
		RefreshTokenHash: p.RefreshTokenHash,
		UserAgent:        p.UserAgent,
		Ip:               p.IP,
		ExpiresAt:        pgtype.Timestamptz{Time: p.ExpiresAt, Valid: true},
	}
}

func toDomainSession(s dbgen.Session) Session {
	session := Session{
		ID:        uuid.UUID(s.ID.Bytes),
		UserID:    uuid.UUID(s.UserID.Bytes),
		ExpiresAt: s.ExpiresAt.Time,
		CreatedAt: s.CreatedAt.Time,
	}
	if s.RevokedAt.Valid {
		revokedAt := s.RevokedAt.Time
		session.RevokedAt = &revokedAt
	}
	return session
}

func toDomainAccountClaim(claim dbgen.AccountClaimToken) AccountClaimToken {
	accountClaimToken := AccountClaimToken{
		ID:        uuid.UUID(claim.ID.Bytes),
		UserID:    uuid.UUID(claim.UserID.Bytes),
		TokenHash: claim.TokenHash,
		UsedAt:    claim.UsedAt.Time,
		ExpiredAt: claim.ExpiresAt.Time,
	}
	return accountClaimToken
}
