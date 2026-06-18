package dbtest

import (
	"context"
	"log"

	"testing"

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
