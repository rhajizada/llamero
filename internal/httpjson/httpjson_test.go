package httpjson_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rhajizada/llamero/internal/httpjson"
)

func TestWriteAndWriteError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		write      func(*httptest.ResponseRecorder)
		wantCode   int
		wantBody   string
		wantHeader string
	}{
		{
			name: "writes json payload",
			write: func(rec *httptest.ResponseRecorder) {
				httpjson.Write(rec, http.StatusAccepted, map[string]string{"status": "ok"})
			},
			wantCode:   http.StatusAccepted,
			wantBody:   `"status":"ok"`,
			wantHeader: "application/json",
		},
		{
			name: "writes json error payload",
			write: func(rec *httptest.ResponseRecorder) {
				httpjson.WriteError(rec, http.StatusBadRequest, "bad request")
			},
			wantCode:   http.StatusBadRequest,
			wantBody:   `"error":"bad request"`,
			wantHeader: "application/json",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			tc.write(rec)

			assert.Equal(t, tc.wantCode, rec.Code)
			assert.Equal(t, tc.wantHeader, rec.Header().Get("Content-Type"))
			assert.Contains(t, rec.Body.String(), tc.wantBody)
		})
	}
}
