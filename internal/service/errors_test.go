package service_test

import (
	"errors"
	"testing"

	"github.com/rhajizada/llamero/internal/service"
)

func TestErrorFormattingAndUnwrap(t *testing.T) {
	t.Parallel()

	base := errors.New("boom")
	err := &service.Error{Code: 502, Message: "backend failed", Err: base}
	if got := err.Error(); got != "backend failed: boom" {
		t.Fatalf("unexpected error string: %q", got)
	}
	if !errors.Is(err, base) {
		t.Fatal("expected errors.Is to match wrapped error")
	}

	err = &service.Error{Code: 404, Message: "not found"}
	if got := err.Error(); got != "not found" {
		t.Fatalf("unexpected message without wrapped error: %q", got)
	}
}
