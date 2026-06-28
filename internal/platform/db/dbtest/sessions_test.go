package dbtest

import (
	"context"
	"testing"
	"time"

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
func TestCreateSession(t *testing.T) {
	q := WithTx(t)
	createRandomSession(t, q)
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
