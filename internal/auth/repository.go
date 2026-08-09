package auth

import (
	"context"
	"net/netip"
	"time"

	"github.com/google/uuid"
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

// AuthRepository is the storage boundary for the auth layer. It is defined in
// terms of domain types only, so the service depends on this interface rather
// than on any particular database. Swapping databases means writing a new
// implementation; callers do not change.
type AuthRepository interface {
	CreateSession(ctx context.Context, params CreateSessionParams) (Session, error)
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
