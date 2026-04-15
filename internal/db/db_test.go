package db_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/db"
	"github.com/rhajizada/llamero/internal/testutil"
)

func TestConnect(t *testing.T) {
	ctx, dsn := testutil.MustStartPostgres(t)

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "connects with valid url", url: dsn},
		{name: "rejects empty url", url: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := db.Connect(ctx, tc.url)
			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			t.Cleanup(pool.Close)
			assert.NoError(t, pool.Ping(ctx))
		})
	}
}

func TestMigrate(t *testing.T) {
	ctx, dsn := testutil.MustStartPostgres(t)
	migrationsDir := testutil.MigrationsDir(t)

	tests := []struct {
		name        string
		url         string
		dir         string
		wantErr     bool
		verifyTable string
	}{
		{name: "applies migrations", url: dsn, dir: migrationsDir, verifyTable: "users"},
		{name: "rejects empty url", url: "", dir: migrationsDir, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := db.Migrate(ctx, tc.url, tc.dir)
			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			pool, openErr := db.Connect(ctx, tc.url)
			require.NoError(t, openErr)
			t.Cleanup(pool.Close)

			var tableName *string
			queryErr := pool.QueryRow(ctx, "SELECT to_regclass('public.' || $1)::text", tc.verifyTable).Scan(&tableName)
			require.NoError(t, queryErr)
			require.NotNil(t, tableName)
			assert.Equal(t, tc.verifyTable, *tableName)
		})
	}
}
