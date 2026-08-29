package onboarding

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joekings2k/staypass-be/internal/membership"
	"github.com/joekings2k/staypass-be/internal/platform/config"
	"github.com/joekings2k/staypass-be/internal/platform/database"
	dbgen "github.com/joekings2k/staypass-be/internal/platform/db/gen"
	"github.com/joekings2k/staypass-be/internal/platform/utils"
	"github.com/joekings2k/staypass-be/internal/user"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// test doubles
// ---------------------------------------------------------------------------

// fakeHasher stands in for bcrypt. Real bcrypt is salted, so its output changes
// every call and cannot be asserted on; it also cannot be made to fail on
// demand. Hashing itself is covered by the hash package's own tests.
type fakeHasher struct {
	out string
	err error
}

func (h *fakeHasher) Hash(password string) (string, error) {
	if h.err != nil {
		return "", h.err
	}
	return h.out, nil
}

func (h *fakeHasher) Compare(hashedPassword, password string) error { return nil }

// spyRunner never invokes the callback. It stands in for "the transaction could
// not be opened" and records whether Register tried to open one at all.
type spyRunner struct {
	err    error
	called bool
}

func (r *spyRunner) Run(ctx context.Context, fn func(q *dbgen.Queries) error) error {
	r.called = true
	return r.err
}

// txRunnerOn binds an already-open transaction to the TransactionRunner
// interface. The test owns the transaction and rolls it back, so nothing the
// service writes survives.
type txRunnerOn struct{ q *dbgen.Queries }

func (r txRunnerOn) Run(ctx context.Context, fn func(q *dbgen.Queries) error) error { return fn(r.q) }

// ---------------------------------------------------------------------------
// test database
// ---------------------------------------------------------------------------

var (
	testPool *pgxpool.Pool
	// why the pool is nil, so a broken test database is reported rather than
	// silently skipped
	testPoolErr error
)

func TestMain(m *testing.M) {
	code := func() int {
		cfg, err := config.Load()
		if err != nil {
			testPoolErr = fmt.Errorf("load config: %w", err)
			return m.Run()
		}
		if !strings.Contains(cfg.TestDatabaseURL, "_test") {
			testPoolErr = fmt.Errorf("refusing to run against %q (database name must contain _test)", cfg.TestDatabaseURL)
			return m.Run()
		}

		pool, err := database.New(context.Background(), cfg.TestDatabaseURL)
		if err != nil {
			testPoolErr = fmt.Errorf("connect to test database: %w", err)
			return m.Run()
		}
		defer pool.Close()

		testPool = pool
		return m.Run()
	}()
	os.Exit(code)
}

func requireDB(t *testing.T) {
	t.Helper()
	if testPool == nil {
		t.Skipf("no test database: %v", testPoolErr)
	}
}

// randomParams returns a valid RegisterParams with unique phone/email/slug, so
// tests that commit can be rerun without colliding on unique constraints.
func randomParams(t *testing.T) RegisterParams {
	t.Helper()

	estateAndSlug, err := utils.RandomEstateAndSlug()
	require.NoError(t, err)
	email, err := utils.RandomEmail()
	require.NoError(t, err)
	phone, err := utils.RandomPhoneNumber(11)
	require.NoError(t, err)

	return RegisterParams{
		Phone:      phone,
		FullName:   "Ada Lovelace",
		Password:   "supersecret",
		Email:      email,
		EstateName: estateAndSlug.Name,
		Slug:       estateAndSlug.Slug,
		Address:    "1 Marina Road",
		City:       "Lagos",
		State:      "Lagos",
	}
}

// ---------------------------------------------------------------------------
// tests
// ---------------------------------------------------------------------------

// The estate, the user and the membership are each already covered by the
// dbtest package. What only onboarding can get wrong is the wiring: the
// membership must point at the estate and user this call just created, with the
// estate_admin role.
func TestRegister_CreatesEstateUserAndMembership(t *testing.T) {
	requireDB(t)
	ctx := context.Background()

	tx, err := testPool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	q := dbgen.New(tx)

	params := randomParams(t)
	svc := NewOnboardingService(txRunnerOn{q: q}, &fakeHasher{out: "stored-hash"})

	gotEstate, gotUser, gotMembership, err := svc.Register(ctx, params)
	require.NoError(t, err)

	require.Equal(t, params.EstateName, gotEstate.Name)
	require.Equal(t, params.Slug, gotEstate.Slug)

	require.Equal(t, params.FullName, gotUser.FullName)
	require.NotNil(t, gotUser.Email)
	require.Equal(t, params.Email, *gotUser.Email)
	require.False(t, gotUser.IsSuperAdmin, "the first estate admin is not a super admin")

	require.Equal(t, gotEstate.ID, gotMembership.EstateID)
	require.Equal(t, gotUser.ID, gotMembership.UserID)
	require.Equal(t, membership.MembershipRoleEstateAdmin, gotMembership.Role)
	require.Equal(t, membership.MembershipStatusActive, gotMembership.Status)

	// the hash is what was stored, never the plaintext
	stored, err := q.GetUserByEmail(ctx, &params.Email)
	require.NoError(t, err)
	require.NotNil(t, stored.PasswordHash)
	require.Equal(t, "stored-hash", *stored.PasswordHash)
}

// This is the reason the service takes a TransactionRunner instead of calling
// three repositories in a row. Without it, every registration with an
// already-taken email would leave an orphaned estate behind.
//
// It cannot use txRunnerOn: the unique violation aborts the surrounding
// Postgres transaction, so that same transaction cannot be queried afterwards
// to prove the estate is gone. It commits for real and cleans up after itself.
func TestRegister_RollsBackEstateWhenUserCreationFails(t *testing.T) {
	requireDB(t)
	ctx := context.Background()

	// seed a committed user that already owns an email address
	seed := randomParams(t)
	seedHash := "seeded"
	seeded, err := user.NewPgUserRepository(dbgen.New(testPool)).CreateUser(ctx, user.CreateUserParams{
		Phone:        seed.Phone,
		Email:        &seed.Email,
		FullName:     "Existing User",
		PasswordHash: &seedHash,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(ctx, `DELETE FROM users WHERE id = $1`, seeded.ID)
	})

	// register a brand new estate, but reuse that email
	params := randomParams(t)
	params.Email = seed.Email
	t.Cleanup(func() {
		// only reachable if the rollback is broken, but do not leak the row
		_, _ = testPool.Exec(ctx, `DELETE FROM estates WHERE slug = $1`, params.Slug)
	})

	svc := NewOnboardingService(database.NewTransactionRunner(testPool), &fakeHasher{out: "stored-hash"})

	gotEstate, gotUser, gotMembership, err := svc.Register(ctx, params)
	require.ErrorIs(t, err, user.ErrEmailTaken)
	require.Nil(t, gotEstate)
	require.Nil(t, gotUser)
	require.Nil(t, gotMembership)

	// the estate created earlier in the same transaction must be gone
	_, err = dbgen.New(testPool).GetEstateBySlug(ctx, params.Slug)
	require.ErrorIs(t, err, pgx.ErrNoRows, "estate was committed despite the user failing")
}

// Hashing happens before the transaction opens, so a hash failure must not cost
// a pooled connection. The assertion that matters is runner.called == false.
func TestRegister_HashFailureShortCircuits(t *testing.T) {
	hashErr := errors.New("bcrypt exploded")
	runner := &spyRunner{}

	svc := NewOnboardingService(runner, &fakeHasher{err: hashErr})

	gotEstate, gotUser, gotMembership, err := svc.Register(context.Background(), RegisterParams{
		Password: "supersecret",
	})

	require.ErrorIs(t, err, hashErr)
	require.Nil(t, gotEstate)
	require.Nil(t, gotUser)
	require.Nil(t, gotMembership)
	require.False(t, runner.called, "must not open a transaction when hashing fails")
}
