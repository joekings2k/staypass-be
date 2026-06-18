package dbtest

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joekings2k/staypass-be/internal/platform/config"
	"github.com/joekings2k/staypass-be/internal/platform/database"
	dbgen "github.com/joekings2k/staypass-be/internal/platform/db/gen"
	"github.com/stretchr/testify/require"
)

var testpool *pgxpool.Pool

func TestMain(m *testing.M) {
	config, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}
	testDBURL := config.TestDatabaseURL

	if !strings.Contains(testDBURL, "_test") {
		log.Fatalf("refusing to run tests against %q (database name must contain _test)", testDBURL)
	}
	ctx := context.Background()

	testpool, err = database.New(ctx, testDBURL)
	if err != nil {
		log.Fatalf("failed to connect to test database: %v", err)
	}

	code := m.Run()
	testpool.Close()
	os.Exit(code)

}

func WithTx(t *testing.T) *dbgen.Queries {
	t.Helper()
	tx, err := testpool.Begin(context.Background())
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := tx.Rollback(context.Background()); err != nil {
			log.Printf("failed to rollback transaction: %v", err)
		}
	})

	return dbgen.New(tx)
}
