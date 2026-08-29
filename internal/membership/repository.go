package membership

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	estateType "github.com/joekings2k/staypass-be/internal/estate"
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
type MembershipRepository interface {
	CreateMembership(context.Context, CreateMembershipParams) (*Membership, error)
	ListEstateMembersipForUser(ctx context.Context, userID uuid.UUID) ([]*Membership, error)
	ListUserEstateMemberships(ctx context.Context, userID uuid.UUID) ([]UserEstatesAndRoles, error) // this list all the memberships for a user across all estates with his estates

}
type pgMembershipRepository struct {
	db *dbgen.Queries
}

type ListUserEstateMembershipsrows struct {
	membership Membership
	Estate     estateType.Estate
}

func NewMembershipRepository(db *dbgen.Queries) MembershipRepository {
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
func (r *pgMembershipRepository) ListEstateMembersipForUser(ctx context.Context, userID uuid.UUID) ([]*Membership, error) {
	memberships, err := r.db.ListUserMembershipsForUser(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return nil, err
	}
	result := make([]*Membership, 0, len(memberships))
	for _, m := range memberships {
		result = append(result, ToDomainMembership(m))
	}
	return result, nil

}
func (r *pgMembershipRepository) ListUserEstateMemberships(ctx context.Context, userID uuid.UUID) ([]UserEstatesAndRoles, error) {

	rows, err := r.db.ListUserEstateMemberships(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return nil, err
	}
	userEstateAndRoles := groupByEstate(rows)
	return userEstateAndRoles, nil

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

type UserEstatesAndRoles struct {
	EstateID   uuid.UUID
	EstateName string
	EstateSlug string
	Roles      []MembershipRole
}

func groupByEstate(rows []dbgen.ListUserEstateMembershipsRow) []UserEstatesAndRoles {
	grouped := make(map[uuid.UUID]int, len(rows))
	result := make([]UserEstatesAndRoles, 0, len(rows))
	for _, row := range rows {
		estateID := uuid.UUID(row.Estate.ID.Bytes)
		if i, ok := grouped[estateID]; ok {
			result[i].Roles = append(result[i].Roles, MembershipRole(row.Membership.Role))
			continue
		}
		grouped[estateID] = len(result)
		result = append(result, UserEstatesAndRoles{
			EstateID:   estateID,
			EstateName: row.Estate.Name,
			EstateSlug: row.Estate.Slug,
			Roles:      []MembershipRole{MembershipRole(row.Membership.Role)},
		})
	}
	return result
}

// grouped := make(map[uuid.UUID]*UserEstatesAndRoles)
// 	for _, row := range rows {
// 		estateID := row.Membership.EstateID.Bytes
// 		if exisiting, ok := grouped[estateID]; ok {
// 			exisiting.Roles = append(exisiting.Roles, MembershipRole(row.Membership.Role))
// 			continue
// 		}
// 		grouped[estateID] = &UserEstatesAndRoles{
// 			EstateName: row.Estate.Name,
// 			Roles: []MembershipRole{
// 				MembershipRole(row.Membership.Role),
// 			},
// 		}
// 	}
// 	result := make([]UserEstatesAndRoles, 0, len(grouped))
// 	for _, estate := range grouped {
// 		result = append(result, *estate)
// 	}
// 	return result
