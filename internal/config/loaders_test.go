package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rhajizada/llamero/internal/config"
)

func TestLoaders(t *testing.T) {
	dir := t.TempDir()
	privPath := filepath.Join(dir, "private.pem")
	if err := os.WriteFile(privPath, []byte("key"), 0o600); err != nil {
		t.Fatalf("write test key: %v", err)
	}

	t.Setenv("LLAMERO_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("LLAMERO_OAUTH_CLIENT_SECRET", "client-secret")
	t.Setenv("LLAMERO_OAUTH_AUTHORIZE_URL", "https://issuer.example.com/authorize")
	t.Setenv("LLAMERO_OAUTH_TOKEN_URL", "https://issuer.example.com/token")
	t.Setenv("LLAMERO_OAUTH_USERINFO_URL", "https://issuer.example.com/userinfo")
	t.Setenv("LLAMERO_OAUTH_REDIRECT_URL", "https://llamero.example.com/callback")
	t.Setenv("LLAMERO_JWT_PRIVATE_KEY_PATH", privPath)
	t.Setenv("LLAMERO_JWT_TTL", "2h")
	t.Setenv("LLAMERO_ROLE_GROUPS", "admin=admins,ops")
	t.Setenv("LLAMERO_REDIS_ADDR", "localhost:6379")
	t.Setenv("LLAMERO_POSTGRES_HOST", "localhost")
	t.Setenv("LLAMERO_POSTGRES_USER", "postgres")
	t.Setenv("LLAMERO_POSTGRES_PASSWORD", "secret")
	t.Setenv("LLAMERO_POSTGRES_DBNAME", "llamero")

	serverCfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("LoadServer returned error: %v", err)
	}
	if serverCfg.JWT.TTL != 2*time.Hour || serverCfg.Roles.Groups["admin"][0] != "admins" {
		t.Fatalf("unexpected server config: %#v", serverCfg)
	}

	workerCfg, err := config.LoadWorker()
	if err != nil {
		t.Fatalf("LoadWorker returned error: %v", err)
	}
	if workerCfg.Store.Addr != "localhost:6379" || workerCfg.Database.Postgres.DBName != "llamero" {
		t.Fatalf("unexpected worker config: %#v", workerCfg)
	}

	schedulerCfg, err := config.LoadScheduler()
	if err != nil {
		t.Fatalf("LoadScheduler returned error: %v", err)
	}
	if schedulerCfg.Store.Addr != "localhost:6379" {
		t.Fatalf("unexpected scheduler config: %#v", schedulerCfg)
	}
}
