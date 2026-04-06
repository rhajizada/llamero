package handler_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/rhajizada/llamero/internal/handler"
)

func TestExtractUserContext(t *testing.T) {
	t.Parallel()

	h := &handler.Handler{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	if _, _, ok := h.ExtractUserContext(rec, req); ok {
		t.Fatal("expected missing auth context to fail")
	}
}

func TestMissingScopes(t *testing.T) {
	t.Parallel()

	if got := handler.MissingScopes(nil, []string{"a"}); got != nil {
		t.Fatalf("expected nil missing scopes, got %#v", got)
	}
	got := handler.MissingScopes([]string{"a", "c"}, []string{"a", "b"})
	if !reflect.DeepEqual(got, []string{"c"}) {
		t.Fatalf("unexpected missing scopes: %#v", got)
	}
}
