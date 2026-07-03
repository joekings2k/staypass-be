package dbtest

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	dbgen "github.com/joekings2k/staypass-be/internal/platform/db/gen"
	"github.com/joekings2k/staypass-be/internal/platform/utils"
	"github.com/stretchr/testify/require"
)

func createRandomSession(t *testing.T, q *dbgen.Queries) dbgen.Session {
	t.Helper()
	user := createRandomUser(t, q)
	hash := utils.RandomString(10)
	expirationTime := pgtype.Timestamp{Time: time.Now().Add(time.Second * 5), Valid: true}

	args := dbgen.CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: hash,
		ExpiresAt:        pgtype.Timestamptz(expirationTime),
	}

	session, err := q.CreateSession(context.Background(), args)
	require.NoError(t, err)
	require.NotEmpty(t, session)
	require.NotEmpty(t, session.UserID)
	require.NotEmpty(t, session.RefreshTokenHash)
	require.Equal(t, args.UserID, session.UserID)
	require.Equal(t, args.RefreshTokenHash, session.RefreshTokenHash)
	return session

}
func createRandomSessionWithUserId(t *testing.T, q *dbgen.Queries, userId pgtype.UUID) dbgen.Session {
	t.Helper()

	hash := utils.RandomString(10)
	expirationTime := pgtype.Timestamp{Time: time.Now().Add(time.Second * 5), Valid: true}

	args := dbgen.CreateSessionParams{
		UserID:           userId,
		RefreshTokenHash: hash,
		ExpiresAt:        pgtype.Timestamptz(expirationTime),
	}

	session, err := q.CreateSession(context.Background(), args)
	require.NoError(t, err)
	require.NotEmpty(t, session)
	require.NotEmpty(t, session.UserID)
	require.NotEmpty(t, session.RefreshTokenHash)
	require.Equal(t, args.UserID, session.UserID)
	require.Equal(t, args.RefreshTokenHash, session.RefreshTokenHash)
	return session

}
func createRandomSessionWithExpiry(t *testing.T, q *dbgen.Queries, duration time.Time) dbgen.Session {
	t.Helper()
	user := createRandomUser(t, q)
	hash := utils.RandomString(10)

	args := dbgen.CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: hash,
		ExpiresAt:        pgtype.Timestamptz{Time: duration, Valid: true},
	}
	session, err := q.CreateSession(context.Background(), args)
	require.NoError(t, err)
	require.WithinDuration(t, args.ExpiresAt.Time, session.ExpiresAt.Time, time.Second)
	return session

}
func createMultipleSessionsForUser(t *testing.T, q *dbgen.Queries, sessionNum int32) []dbgen.User {
	t.Helper()

	users := make([]dbgen.User, 0, 2)

	for i := 0; i < 2; i++ {
		user := createRandomUser(t, q)
		users = append(users, user)
	}

	for i := 0; i < int(sessionNum); i++ {
		index, _ := utils.RandomNumber(1)
		createRandomSessionWithUserId(t, q, users[index].ID)
	}
	return users
}
func TestCreateSession(t *testing.T) {
	q := WithTx(t)
	createRandomSession(t, q)
}

func TestGetSessionByUserId(t *testing.T) {
	q := WithTx(t)
	ctx := context.Background()
	users := createMultipleSessionsForUser(t, q, 9)

	resSessions, err := q.GetSessionByUserId(ctx, users[1].ID)
	require.NoError(t, err)

	for _, session := range resSessions {
		require.Equal(t, session.UserID, users[1].ID)
	}

}

func TestGetSessionById(t *testing.T) {
	q := WithTx(t)
	ctx := context.Background()
	session := createRandomSession(t, q)

	resSession, err := q.GetSessionById(ctx, session.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resSession.ID)
	require.NotEmpty(t, resSession.RefreshTokenHash)
	require.NotEmpty(t, resSession.UserID)
	require.Equal(t, session.ID, resSession.ID)
	require.Equal(t, session.RefreshTokenHash, resSession.RefreshTokenHash)
	require.Equal(t, session.UserID, resSession.UserID)
}

func TestGetActiveSessionByTokenHash(t *testing.T) {
	q := WithTx(t)
	ctx := context.Background()
	session := createRandomSession(t, q)

	resSession, err := q.GetActiveSessionByTokenHash(ctx, session.RefreshTokenHash)
	require.NoError(t, err)
	require.NotEmpty(t, resSession.ID)
	require.NotEmpty(t, resSession.UserID)
	require.NotEmpty(t, resSession.RefreshTokenHash)
	require.WithinDuration(t, session.ExpiresAt.Time, resSession.ExpiresAt.Time, time.Second)
	require.Equal(t, session.ID, resSession.ID)
	require.Equal(t, session.RefreshTokenHash, resSession.RefreshTokenHash)
	require.Equal(t, session.UserID, resSession.UserID)
}

func TestGetActiveSessionByTokenHashNotFound(t *testing.T) {
	q := WithTx(t)
	ctx := context.Background()

	resSession, err := q.GetActiveSessionByTokenHash(ctx, utils.RandomString(32))
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	require.Empty(t, resSession.ID)
}
func TestGetActiveSessionByTokenHashExpired(t *testing.T) {
	q := WithTx(t)

	ctx := context.Background()
	session := createRandomSessionWithExpiry(t, q, time.Now().Add(-time.Hour))
	_, err := q.GetActiveSessionByTokenHash(ctx, session.RefreshTokenHash)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestRevokeSession(t *testing.T) {
	q := WithTx(t)
	ctx := context.Background()
	session := createRandomSession(t, q)

	err := q.RevokeSession(ctx, session.ID)
	require.NoError(t, err)

	// GetSessionById excludes revoked sessions (WHERE revoked_at IS NULL),
	// so once revoked it must return no rows. That absence proves the revoke.
	_, err = q.GetSessionById(ctx, session.ID)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestRevokeAllSessionsForUser(t *testing.T) {

	q := WithTx(t)
	ctx := context.Background()

	users := createMultipleSessionsForUser(t, q, 6)

	err := q.RevokeAllSessionsForUser(ctx, users[1].ID)
	require.NoError(t, err)

	sessions, err := q.GetSessionByUserId(ctx, users[1].ID)
	require.NoError(t, err)
	require.Empty(t, sessions)

}

func createRandomClaimToken(t *testing.T, q *dbgen.Queries, userId pgtype.UUID) dbgen.AccountClaimToken {
	t.Helper()
	tokenHash := utils.RandomString(12)
	expiresAt := time.Now().Add(time.Hour)

	args := dbgen.CreateAccountClaimTokenParams{
		UserID:    userId,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}

	claimToken, err := q.CreateAccountClaimToken(context.Background(), args)
	require.NoError(t, err)
	require.Equal(t, args.UserID, claimToken.UserID)
	require.Equal(t, args.TokenHash, claimToken.TokenHash)
	require.WithinDuration(t, args.ExpiresAt.Time, claimToken.ExpiresAt.Time, time.Second)
	return claimToken
}

func TestCreateAccountClaimToken(t *testing.T) {
	q := WithTx(t)
	user := createRandomUser(t, q)
	createRandomClaimToken(t, q, user.ID)

}

func TestGetActiveClaimToken(t *testing.T) {
	q := WithTx(t)
	ctx := context.Background()
	user := createRandomUser(t, q)
	claimtoken := createRandomClaimToken(t, q, user.ID)

	activeClaimToken, err := q.GetActiveClaimToken(ctx, claimtoken.TokenHash)
	require.NoError(t, err)
	require.Equal(t, claimtoken.UserID, activeClaimToken.UserID)
	require.Equal(t, claimtoken.TokenHash, activeClaimToken.TokenHash)
	require.False(t, activeClaimToken.UsedAt.Valid)
	require.True(t, activeClaimToken.ExpiresAt.Valid)
}
