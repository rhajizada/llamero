package roles_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/roles"
)

func TestRoleStore(t *testing.T) {
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
	require.NoError(t, os.WriteFile(path, []byte(raw), 0o600))

	invalidPath := filepath.Join(dir, "invalid-roles.yaml")
	invalidRaw := strings.TrimSpace(`
default_role: viewer
roles:
  - name: viewer
    scopes: [models:read]
`)
	require.NoError(t, os.WriteFile(invalidPath, []byte(invalidRaw), 0o600))

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "loads resolves and returns default role",
			run: func(t *testing.T) {
				store, err := roles.Load(path, map[string][]string{"admin": {"ops", "admins"}})
				require.NoError(t, err)

				role, scopes, ok := store.Resolve([]string{"", "ops"})
				assert.True(t, ok)
				assert.Equal(t, "admin", role)
				assert.Equal(t, []string{"models:write", "models:read"}, scopes)

				defaultRole, defaultScopes := store.Default()
				assert.Equal(t, "viewer", defaultRole)
				assert.Equal(t, []string{"models:read"}, defaultScopes)
			},
		},
		{
			name: "rejects undefined role mapping",
			run: func(t *testing.T) {
				_, err := roles.Load(invalidPath, map[string][]string{"admin": {"ops"}})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "undefined role")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}
