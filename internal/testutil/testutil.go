package testutil

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/db"
	"github.com/rhajizada/llamero/internal/redisstore"
	"github.com/rhajizada/llamero/internal/roles"
)

const (
	postgresReadyLogOccurrences = 2
	redisPort                   = "6379/tcp"
)

func MustWriteEd25519JWTConfig(t *testing.T) config.JWTConfig {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}

	dir := t.TempDir()
	privPath := filepath.Join(dir, "private.pem")
	pubPath := filepath.Join(dir, "public.pem")
	if writeErr := os.WriteFile(
		privPath,
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}),
		0o600,
	); writeErr != nil {
		t.Fatalf("write private key: %v", writeErr)
	}
	if writeErr := os.WriteFile(
		pubPath,
		pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}),
		0o600,
	); writeErr != nil {
		t.Fatalf("write public key: %v", writeErr)
	}

	return config.JWTConfig{
		Issuer:         "llamero-test",
		Audience:       "clients",
		PrivateKeyPath: privPath,
		PublicKeyPath:  pubPath,
		SigningMethod:  "EdDSA",
		TTL:            time.Hour,
	}
}

func MustNewTokenIssuer(t *testing.T) *auth.TokenIssuer {
	t.Helper()

	issuer, err := auth.NewTokenIssuer(MustWriteEd25519JWTConfig(t))
	if err != nil {
		t.Fatalf("new token issuer: %v", err)
	}
	return issuer
}

func MustLoadRoles(t *testing.T, raw string, groups map[string][]string) *roles.Store {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "roles.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(raw)), 0o600); err != nil {
		t.Fatalf("write roles file: %v", err)
	}
	store, err := roles.Load(path, groups)
	if err != nil {
		t.Fatalf("load roles: %v", err)
	}
	return store
}

func MigrationsDir(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve testutil path: runtime caller unavailable")
	}

	return filepath.Join(filepath.Dir(file), "..", "..", "data", "sql", "migrations")
}

func MustStartPostgres(t *testing.T) (context.Context, string) {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx := context.Background()
	container, err := tcpostgres.Run(
		ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("llamero"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(postgresReadyLogOccurrences).
				WithStartupTimeout(time.Minute),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	t.Cleanup(func() {
		if terminateErr := container.Terminate(context.Background()); terminateErr != nil {
			t.Errorf("terminate postgres container: %v", terminateErr)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("build postgres connection string: %v", err)
	}

	return ctx, dsn
}

func MustOpenMigratedPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx, dsn := MustStartPostgres(t)
	if err := db.Migrate(ctx, dsn, MigrationsDir(t)); err != nil {
		t.Fatalf("migrate postgres: %v", err)
	}

	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}

func MustStartRedis(t *testing.T) (context.Context, string) {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{redisPort},
			WaitingFor:   wait.ForListeningPort(redisPort).WithStartupTimeout(time.Minute),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start redis container: %v", err)
	}

	t.Cleanup(func() {
		if terminateErr := container.Terminate(context.Background()); terminateErr != nil {
			t.Errorf("terminate redis container: %v", terminateErr)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("resolve redis host: %v", err)
	}
	port, err := container.MappedPort(ctx, redisPort)
	if err != nil {
		t.Fatalf("resolve redis port: %v", err)
	}

	return ctx, net.JoinHostPort(host, port.Port())
}

func MustOpenRedisStore(t *testing.T) *redisstore.Store {
	t.Helper()

	_, addr := MustStartRedis(t)
	store, err := redisstore.New(&config.RedisConfig{Addr: addr})
	if err != nil {
		t.Fatalf("connect redis store: %v", err)
	}

	return store
}
