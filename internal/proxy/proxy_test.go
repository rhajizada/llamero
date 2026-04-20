package proxy_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/proxy"
)

func TestForwardLLM(t *testing.T) {
	t.Parallel()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/base/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("expected auth header to be stripped, got %q", got)
		}
		if got := r.Header.Get("X-Forwarded-Host"); got != "api.example.com" {
			t.Fatalf("unexpected forwarded host: %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"model":"llama3"}` {
			t.Fatalf("unexpected body: %s", string(body))
		}
		w.Header().Set("X-Backend", "ok")
		_, _ = w.Write([]byte("chat"))
	}))
	t.Cleanup(backend.Close)

	client := proxy.New(nil)
	req := httptest.NewRequest(
		http.MethodPost,
		"https://api.example.com/api/chat/completions?stream=true",
		bytes.NewReader([]byte(`{"model":"llama3"}`)),
	)
	req.Host = "api.example.com"
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	req.RemoteAddr = "10.0.0.5:1234"

	resp, err := client.ForwardLLM(req, backend.URL+"/base", []byte(`{"model":"llama3"}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, "ok", resp.Header.Get("X-Backend"))
}

func TestForwardGET(t *testing.T) {
	t.Parallel()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/base/api/tags" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		_, _ = w.Write([]byte("tags"))
	}))
	t.Cleanup(backend.Close)

	client := proxy.New(nil)

	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/api/backends/id/tags", nil)
	req.Host = "api.example.com"
	resp, err := client.ForwardGET(req, backend.URL+"/base", "/api/tags")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "tags", string(body))
}

func TestWriteResponse(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header: http.Header{
			"X-Response": {"ok"},
			"Connection": {"keep-alive"},
		},
		Body: io.NopCloser(strings.NewReader("hello")),
	}
	require.NoError(t, proxy.WriteResponse(rec, resp))
	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Equal(t, "ok", rec.Header().Get("X-Response"))
	assert.Empty(t, rec.Header().Get("Connection"))
	assert.Equal(t, "hello", rec.Body.String())
}

func TestForwardErrorsAndPathVariants(t *testing.T) {
	t.Parallel()

	client := proxy.New(nil)

	for _, tc := range []struct {
		name        string
		method      string
		requestPath string
		baseAddress string
		targetPath  string
		call        func(*http.Request) (*http.Response, error)
	}{
		{
			name:        "invalid base address",
			method:      http.MethodGet,
			requestPath: "https://api.example.com/api/tags",
			baseAddress: "://bad",
			targetPath:  "/api/tags",
			call: func(r *http.Request) (*http.Response, error) {
				return client.ForwardGET(r, "://bad", "/api/tags")
			},
		},
		{
			name:        "unsupported scheme",
			method:      http.MethodGet,
			requestPath: "https://api.example.com/api/tags",
			baseAddress: "ftp://backend.example.com",
			targetPath:  "/api/tags",
			call: func(r *http.Request) (*http.Response, error) {
				return client.ForwardGET(r, "ftp://backend.example.com", "/api/tags")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tc.method, tc.requestPath, nil)
			_, err := tc.call(req)
			assert.Error(t, err)
		})
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Forwarded-Port-Seen", r.Header.Get("X-Forwarded-Port"))
		_, _ = w.Write([]byte(r.URL.Path))
	}))
	t.Cleanup(backend.Close)

	for _, tc := range []struct {
		name     string
		path     string
		expected string
		port     string
	}{
		{name: "empty path maps to v1", path: "", expected: "/base/v1", port: "80"},
		{name: "v1 path unchanged", path: "/v1/models", expected: "/base/v1/models", port: "8443"},
		{name: "custom path unchanged", path: "/custom", expected: "/base/custom", port: "9000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := "https://api.example.com:8443" + tc.path
			if tc.name == "empty path maps to v1" {
				targetURL = "http://api.example.com"
			}
			req := httptest.NewRequest(http.MethodPost, targetURL, bytes.NewReader([]byte(`{"model":"llama3"}`)))
			if tc.port == "9000" {
				req.Header.Set("X-Forwarded-Port", "9000")
			}
			resp, err := client.ForwardLLM(req, backend.URL+"/base", []byte(`{"model":"llama3"}`))
			require.NoError(t, err)
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			assert.Equal(t, tc.expected, string(body))
			assert.Equal(t, tc.port, resp.Header.Get("X-Forwarded-Port-Seen"))
		})
	}
}
