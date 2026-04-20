package auth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/rhajizada/llamero/internal/auth"
)

func TestStateStoreLifecycle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ttl  time.Duration
		run  func(*testing.T, *auth.StateStore)
	}{
		{
			name: "issues and consumes token once",
			ttl:  time.Minute,
			run: func(t *testing.T, store *auth.StateStore) {
				t.Helper()

				token := store.Issue()
				assert.NotEmpty(t, token)
				assert.True(t, store.Consume(token))
				assert.False(t, store.Consume(token))
			},
		},
		{
			name: "rejects empty token",
			ttl:  5 * time.Millisecond,
			run: func(t *testing.T, store *auth.StateStore) {
				t.Helper()

				assert.False(t, store.Consume(""))
			},
		},
		{
			name: "rejects expired token repeatedly",
			ttl:  5 * time.Millisecond,
			run: func(t *testing.T, store *auth.StateStore) {
				t.Helper()

				token := store.Issue()
				time.Sleep(20 * time.Millisecond)
				assert.False(t, store.Consume(token))
				assert.False(t, store.Consume(token))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := auth.NewStateStore(tc.ttl)
			tc.run(t, store)
		})
	}
}
