package httpjson_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rhajizada/llamero/internal/httpjson"
)

func TestWriteAndWriteError(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	httpjson.Write(rec, http.StatusAccepted, map[string]string{"status": "ok"})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content type: %q", got)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	httpjson.WriteError(rec, http.StatusBadRequest, "bad request")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected error status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"error":"bad request"`) {
		t.Fatalf("unexpected error body: %s", rec.Body.String())
	}
}
