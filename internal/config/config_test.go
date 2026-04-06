package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rhajizada/llamero/internal/config"
)

func TestParseRoleGroups(t *testing.T) {
	t.Parallel()

	got, err := config.ParseRoleGroups("admin=ops, admins ; viewer = readers, readers")
	if err != nil {
		t.Fatalf("parseRoleGroups returned error: %v", err)
	}

	want := map[string][]string{
		"admin":  {"ops", "admins"},
		"viewer": {"readers"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected groups: got %#v want %#v", got, want)
	}
}

func TestParseRoleGroupsRejectsInvalidEntry(t *testing.T) {
	t.Parallel()

	_, err := config.ParseRoleGroups("admin")
	if err == nil || !strings.Contains(err.Error(), "invalid role mapping entry") {
		t.Fatalf("expected invalid mapping error, got %v", err)
	}
}

func TestPostgresConfigDSN(t *testing.T) {
	t.Parallel()

	dsn := (config.PostgresConfig{
		Host:     "db",
		Port:     5432,
		User:     "user",
		Password: "secret",
		DBName:   "llamero",
		SSLMode:  "disable",
	}).DSN()

	const want = "postgres://user:secret@db:5432/llamero?sslmode=disable"
	if dsn != want {
		t.Fatalf("unexpected DSN: got %q want %q", dsn, want)
	}
}

func TestLoadBackendDefinitions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "backends.yaml")
	raw := strings.TrimSpace(`
backends:
  - id: primary
    address: http://backend-1:11434
    tags: [gpu, us-east]
    weight: 2
  - id: secondary
    address: https://backend-2.example.com
`)
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write backends file: %v", err)
	}

	backends, err := config.LoadBackendDefinitions(path)
	if err != nil {
		t.Fatalf("LoadBackendDefinitions returned error: %v", err)
	}

	if len(backends) != 2 {
		t.Fatalf("unexpected backend count: %d", len(backends))
	}
	if backends[0].ID != "primary" || backends[0].Weight != 2 {
		t.Fatalf("unexpected first backend: %#v", backends[0])
	}
}
