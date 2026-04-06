package db_test

import (
	"context"
	"testing"

	"github.com/rhajizada/llamero/internal/db"
)

func TestConnectRejectsEmptyURL(t *testing.T) {
	t.Parallel()

	if _, err := db.Connect(context.Background(), ""); err == nil {
		t.Fatal("expected empty database url to fail")
	}
}

func TestMigrateRejectsEmptyURL(t *testing.T) {
	t.Parallel()

	if err := db.Migrate(context.Background(), "", "."); err == nil {
		t.Fatal("expected empty database url to fail")
	}
}
