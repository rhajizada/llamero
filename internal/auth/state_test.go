package auth_test

import (
	"testing"
	"time"

	"github.com/rhajizada/llamero/internal/auth"
)

func TestStateStoreIssueAndConsume(t *testing.T) {
	t.Parallel()

	store := auth.NewStateStore(time.Minute)
	token := store.Issue()
	if token == "" {
		t.Fatal("expected issued token")
	}
	if !store.Consume(token) {
		t.Fatal("expected first consume to succeed")
	}
	if store.Consume(token) {
		t.Fatal("expected second consume to fail")
	}
}

func TestStateStoreConsumeRejectsExpiredAndEmptyTokens(t *testing.T) {
	t.Parallel()

	store := auth.NewStateStore(5 * time.Millisecond)
	if store.Consume("") {
		t.Fatal("expected empty token to be rejected")
	}

	token := store.Issue()
	time.Sleep(20 * time.Millisecond)
	if store.Consume(token) {
		t.Fatal("expected expired token to be rejected")
	}
	if store.Consume(token) {
		t.Fatal("expected expired token to stay invalid on repeated consume")
	}
}
