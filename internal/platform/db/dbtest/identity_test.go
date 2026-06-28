package dbtest

import (
	"context"
	"log"

	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	dbgen "github.com/joekings2k/staypass-be/internal/platform/db/gen"
	"github.com/joekings2k/staypass-be/internal/platform/utils"
	"github.com/stretchr/testify/require"
)

func createRandomUser(t *testing.T, q *dbgen.Queries) dbgen.User {
	t.Helper()
	randPhoneNumber, err := utils.RandomPhoneNumber(8)
	require.NoError(t, err)
	randEmail, err := utils.RandomEmail()
	require.NoError(t, err)
	randPasswordHash := utils.RandomString(64)
	args := dbgen.CreateUserParams{
		Phone:        randPhoneNumber,
		Email:        &randEmail,
		FullName:     "Test User",
		PasswordHash: &randPasswordHash,
		IsSuperAdmin: false,
	}
	user, err := q.CreateUser(context.Background(), args)
	require.NoError(t, err)
	require.NotEmpty(t, user)
	require.Equal(t, args.Phone, user.Phone)
	require.Equal(t, args.Email, user.Email)
	require.Equal(t, args.FullName, user.FullName)
	require.Equal(t, args.PasswordHash, user.PasswordHash)
	require.Equal(t, args.IsSuperAdmin, user.IsSuperAdmin)
	require.NotZero(t, user.ID)
	require.NotZero(t, user.CreatedAt.Time)
	require.NotZero(t, user.UpdatedAt.Time)
	require.False(t, user.DeletedAt.Valid)
	return user
}

func createRandomMembership(t *testing.T, q *dbgen.Queries, r dbgen.MembershipRole, userID, estateID pgtype.UUID) dbgen.Membership {
	ctx := context.Background()

	args := dbgen.CreateMembershipParams{
		EstateID: estateID,
		UserID:   userID,
		Role:     r,
	}
	membership, err := q.CreateMembership(ctx, args)
	require.NoError(t, err)
	require.NotEmpty(t, membership)
	require.Equal(t, args.EstateID, membership.EstateID)
	require.Equal(t, args.UserID, membership.UserID)
	require.Equal(t, args.Role, membership.Role)
	require.NotZero(t, membership.ID)
	require.NotZero(t, membership.CreatedAt.Time)
	require.NotZero(t, membership.UpdatedAt.Time)
	return membership
}

func TestCreateUser(t *testing.T) {
	q := WithTx(t)
	createRandomUser(t, q)
}

func TestGetUserByEmail(t *testing.T) {
	ctx := context.Background()
	q := WithTx(t)
	user1 := createRandomUser(t, q)
	log.Printf("user1: %+v", user1)

	resUser, err := q.GetUserByEmail(ctx, user1.Email)
	log.Printf("resUser: %+v", resUser)
	require.NoError(t, err)
	require.Equal(t, user1.ID, resUser.ID)
	require.Equal(t, user1.Phone, resUser.Phone)
	require.Equal(t, user1.Email, resUser.Email)
	require.Equal(t, user1.FullName, resUser.FullName)
	require.Equal(t, user1.PasswordHash, resUser.PasswordHash)
	require.Equal(t, user1.IsSuperAdmin, resUser.IsSuperAdmin)
	require.WithinDuration(t, user1.CreatedAt.Time, resUser.CreatedAt.Time, 0)
	require.WithinDuration(t, user1.UpdatedAt.Time, resUser.UpdatedAt.Time, 0)
	require.Equal(t, user1.DeletedAt.Time, resUser.DeletedAt.Time)
}

func TestGetUserByID(t *testing.T) {
	ctx := context.Background()
	q := WithTx(t)
	user1 := createRandomUser(t, q)
	resUser, err := q.GetUserByID(ctx, user1.ID)
	require.NoError(t, err)
	require.Equal(t, user1.ID, resUser.ID)
	require.Equal(t, user1.Phone, resUser.Phone)
	require.Equal(t, user1.Email, resUser.Email)
	require.Equal(t, user1.FullName, resUser.FullName)
	require.Equal(t, user1.PasswordHash, resUser.PasswordHash)
	require.Equal(t, user1.IsSuperAdmin, resUser.IsSuperAdmin)
	require.WithinDuration(t, user1.CreatedAt.Time, resUser.CreatedAt.Time, 0)
	require.WithinDuration(t, user1.UpdatedAt.Time, resUser.UpdatedAt.Time, 0)
	require.Equal(t, user1.DeletedAt.Time, resUser.DeletedAt.Time)
}

func TestGetUserByPhone(t *testing.T) {
	ctx := context.Background()
	q := WithTx(t)
	user1 := createRandomUser(t, q)
	resUser, err := q.GetUserByPhone(ctx, user1.Phone)
	require.NoError(t, err)
	require.Equal(t, user1.ID, resUser.ID)
	require.Equal(t, user1.Phone, resUser.Phone)
	require.Equal(t, user1.Email, resUser.Email)
	require.Equal(t, user1.FullName, resUser.FullName)
	require.Equal(t, user1.PasswordHash, resUser.PasswordHash)
	require.Equal(t, user1.IsSuperAdmin, resUser.IsSuperAdmin)
	require.WithinDuration(t, user1.CreatedAt.Time, resUser.CreatedAt.Time, 0)
	require.WithinDuration(t, user1.UpdatedAt.Time, resUser.UpdatedAt.Time, 0)
	require.Equal(t, user1.DeletedAt.Time, resUser.DeletedAt.Time)
}

func TestListEstateMembershipForUser(t *testing.T) {
	ctx := context.Background()
	q := WithTx(t)
	estate := createRandomEstate(t, q)
	user := createRandomUser(t, q)

	// A user may hold several roles in one estate, but (estate_id, user_id, role)
	// is UNIQUE — so seed distinct roles, not duplicates.
	roles := []dbgen.MembershipRole{
		dbgen.MembershipRoleEstateAdmin,
		dbgen.MembershipRoleLandlord,
		dbgen.MembershipRoleResident,
	}
	for _, r := range roles {
		createRandomMembership(t, q, r, user.ID, estate.ID)
	}

	memberships, err := q.ListEstateMembershipForUser(ctx, dbgen.ListEstateMembershipForUserParams{
		UserID:   user.ID,
		EstateID: estate.ID,
	})
	require.NoError(t, err)
	require.Len(t, memberships, len(roles))
	for _, m := range memberships {
		require.Equal(t, user.ID, m.UserID)
		require.Equal(t, estate.ID, m.EstateID)
		require.Equal(t, dbgen.MembershipStatusActive, m.Status)
	}
}

func TestCreateMembership(t *testing.T) {
	q := WithTx(t)
	user := createRandomUser(t, q)
	estate := createRandomEstate(t, q)
	createRandomMembership(t, q, dbgen.MembershipRoleLandlord, user.ID, estate.ID)
}

func TestSuspendMembershipById(t *testing.T) {
	q := WithTx(t)
	ctx := context.Background()
	user := createRandomUser(t, q)
	estate := createRandomEstate(t, q)
	membership := createRandomMembership(t, q, dbgen.MembershipRoleLandlord, user.ID, estate.ID)

	suspenededMemberShip, err := q.SuspendMembershipByID(ctx, membership.ID)
	require.NoError(t, err)
	require.Equal(t, user.ID, suspenededMemberShip.UserID)
	require.Equal(t, estate.ID, suspenededMemberShip.EstateID)
	require.Equal(t, suspenededMemberShip.Status, dbgen.MembershipStatusSuspended)

}
