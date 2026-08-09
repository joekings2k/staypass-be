package membership

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	dbgen "github.com/joekings2k/staypass-be/internal/platform/db/gen"
)

type MembershipRole string
type MembershipStatus string

const (
	MembershipRoleSuperAdmin  MembershipRole = "super_admin"
	MembershipRoleEstateAdmin MembershipRole = "estate_admin"
	MembershipRoleGateman     MembershipRole = "gateman"
	MembershipRoleLandlord    MembershipRole = "landlord"
	MembershipRoleResident    MembershipRole = "resident"
)

const (
	MembershipStatusActive    MembershipStatus = "active"
	MembershipStatusSuspended MembershipStatus = "suspended"
	MembershipStatusEnded     MembershipStatus = "ended"
)

type Membership struct {
	ID        uuid.UUID
	EstateID  uuid.UUID
	UserID    uuid.UUID
	Role      MembershipRole
	Status    MembershipStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	EndedAt   time.Time
}

type CreateMembershipParams struct {
	EstateID uuid.UUID
	UserID   uuid.UUID
	Role     MembershipRole
}
type MemebershipRepository interface {
	CreateMembership(context.Context, CreateMembershipParams) (*Membership, error)
}
type pgMembershipRepository struct {
	db *dbgen.Queries
}

func NewMembershipRepository(db *dbgen.Queries) MemebershipRepository {
	return &pgMembershipRepository{
		db: db,
	}

}

func (r *pgMembershipRepository) CreateMembership(ctx context.Context, args CreateMembershipParams) (*Membership, error) {
	membership, err := r.db.CreateMembership(ctx, toDBCreateMembershipParmas(args))
	if err != nil {
		return nil, err
	}

	return ToDomainMembership(membership), nil

}

func ToDomainMembership(arg dbgen.Membership) *Membership {
	member := &Membership{
		ID:        arg.ID.Bytes,
		EstateID:  arg.EstateID.Bytes,
		UserID:    arg.UserID.Bytes,
		Role:      MembershipRole(arg.Role),
		Status:    MembershipStatus(arg.Status),
		CreatedAt: arg.CreatedAt.Time,
		UpdatedAt: arg.CreatedAt.Time,
	}
	if arg.EndedAt.Valid {
		member.EndedAt = arg.EndedAt.Time
	}
	return member

}
func toDBCreateMembershipParmas(args CreateMembershipParams) dbgen.CreateMembershipParams {
	return dbgen.CreateMembershipParams{
		EstateID: pgtype.UUID{Bytes: args.EstateID, Valid: true},
		UserID:   pgtype.UUID{Bytes: args.UserID, Valid: true},
		Role:     dbgen.MembershipRole(args.Role),
	}
}
