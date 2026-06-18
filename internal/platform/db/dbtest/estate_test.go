package dbtest

import (
	"context"
	"testing"

	dbgen "github.com/joekings2k/staypass-be/internal/platform/db/gen"
	"github.com/joekings2k/staypass-be/internal/platform/utils"
	"github.com/stretchr/testify/require"
)

func createRandomEstate(t *testing.T, q *dbgen.Queries) dbgen.Estate {
	t.Helper()
	random, err := utils.RandomEstateAndSlug()
	require.NoError(t, err)
	args := dbgen.CreateEstateParams{
		Name:     random.Name,
		Slug:     random.Slug,
		Timezone: "Africa/Lagos",
	}
	estate, err := q.CreateEstate(context.Background(), args)
	require.NoError(t, err)
	require.NotEmpty(t, estate)
	require.Equal(t, args.Name, estate.Name)
	require.Equal(t, args.Slug, estate.Slug)
	require.Equal(t, args.Timezone, estate.Timezone)
	require.NotZero(t, estate.ID)
	require.NotZero(t, estate.CreatedAt.Time)
	require.NotZero(t, estate.UpdatedAt.Time)
	require.True(t, estate.IsActive)
	return estate
}

func TestCreateEstate(t *testing.T) {
	q := WithTx(t)
	createRandomEstate(t, q)
}

func TestGetEstateById(t *testing.T) {
	ctx := context.Background()
	q := WithTx(t)
	estate1 := createRandomEstate(t, q)

	resEstate, err := q.GetEstateByID(ctx, estate1.ID)
	require.NoError(t, err)
	require.Equal(t, estate1.Name, resEstate.Name)
	require.Equal(t, estate1.Slug, resEstate.Slug)
	require.NotZero(t, estate1.ID)
	require.NotZero(t, estate1.CreatedAt.Time)
	require.NotZero(t, estate1.UpdatedAt.Time)
}

func TestGetEstateBySlug(t *testing.T) {
	ctx := context.Background()
	q := WithTx(t)
	estate1 := createRandomEstate(t, q)

	resEstate, err := q.GetEstateBySlug(ctx, estate1.Slug)
	require.NoError(t, err)
	require.Equal(t, estate1.Name, resEstate.Name)
	require.Equal(t, estate1.Slug, resEstate.Slug)
	require.NotZero(t, resEstate.ID)
	require.NotZero(t, resEstate.CreatedAt.Time)
	require.NotZero(t, resEstate.UpdatedAt.Time)

}

func TestListEstatesPaginated(t *testing.T) {
	ctx := context.Background()
	q := WithTx(t)

	// Seed more rows than a single page so we can prove the DB only
	// returns one page at a time.
	for range 10 {
		createRandomEstate(t, q)
	}

	const pageSize = 5

	page1, err := q.ListEstates(ctx, dbgen.ListEstatesParams{
		PageSize:   pageSize,
		PageOffset: 0,
	})
	require.NoError(t, err)
	require.Len(t, page1, pageSize)

	page2, err := q.ListEstates(ctx, dbgen.ListEstatesParams{
		PageSize:   pageSize,
		PageOffset: pageSize,
	})
	require.NoError(t, err)
	require.Len(t, page2, pageSize)

	// Pages must not overlap.
	require.NotEqual(t, page1[0].ID, page2[0].ID)

	// Ordering by name is stable across the page boundary.
	require.LessOrEqual(t, page1[len(page1)-1].Name, page2[0].Name)
}
