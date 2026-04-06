package roles_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rhajizada/llamero/internal/roles"
)

func TestLoadAndResolve(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "roles.yaml")
	raw := strings.TrimSpace(`
default_role: viewer
roles:
  - name: viewer
    scopes: [models:read, models:read]
  - name: admin
    scopes: [models:write, models:read]
`)
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write roles file: %v", err)
	}

	store, err := roles.Load(path, map[string][]string{"admin": {"ops", "admins"}})
	if err != nil {
		t.Fatalf("load roles: %v", err)
	}

	role, scopes, ok := store.Resolve([]string{"", "ops"})
	if !ok {
		t.Fatal("expected role resolution to succeed")
	}
	if role != "admin" {
		t.Fatalf("unexpected role: %s", role)
	}
	if !reflect.DeepEqual(scopes, []string{"models:write", "models:read"}) {
		t.Fatalf("unexpected scopes: %#v", scopes)
	}

	defaultRole, defaultScopes := store.Default()
	if defaultRole != "viewer" {
		t.Fatalf("unexpected default role: %s", defaultRole)
	}
	if !reflect.DeepEqual(defaultScopes, []string{"models:read"}) {
		t.Fatalf("unexpected default scopes: %#v", defaultScopes)
	}
}

func TestLoadRejectsInvalidRoleMappings(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "roles.yaml")
	raw := strings.TrimSpace(`
default_role: viewer
roles:
  - name: viewer
    scopes: [models:read]
`)
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write roles file: %v", err)
	}

	_, err := roles.Load(path, map[string][]string{"admin": {"ops"}})
	if err == nil || !strings.Contains(err.Error(), "undefined role") {
		t.Fatalf("expected undefined role error, got %v", err)
	}
}
