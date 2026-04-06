package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func WriteResponse(w http.ResponseWriter, resp *http.Response) error {
	copyHeaders(w.Header(), resp.Header)
	stripHopHeaders(w.Header())
	w.WriteHeader(resp.StatusCode)
	_, err := io.Copy(w, resp.Body)
	return err
}

func buildBackendURL(baseAddress, backendPath, rawQuery string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseAddress))
	if err != nil {
		return "", fmt.Errorf("parse backend address: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("backend address must use http or https: %q", baseAddress)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("backend address missing host: %q", baseAddress)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("backend address must not include credentials: %q", baseAddress)
	}
	if parsed.Opaque != "" {
		return "", fmt.Errorf("backend address must be hierarchical: %q", baseAddress)
	}
	if parsed.Fragment != "" {
		return "", fmt.Errorf("backend address must not include fragment: %q", baseAddress)
	}

	target := *parsed
	target.RawQuery = ""
	target.Fragment = ""

	backendPath = strings.TrimSpace(backendPath)
	if backendPath != "" {
		if !strings.HasPrefix(backendPath, "/") {
			backendPath = "/" + backendPath
		}
		basePath := strings.TrimRight(target.Path, "/")
		if basePath == "" {
			target.Path = backendPath
		} else {
			target.Path = basePath + backendPath
		}
	}
	if rawQuery != "" {
		target.RawQuery = rawQuery
	}

	return target.String(), nil
}

func normalizeLLMPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/v1"
	}
	if after, ok := strings.CutPrefix(path, "/api/"); ok {
		return "/v1/" + after
	}
	if strings.HasPrefix(path, "/v1/") || path == "/v1" {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		return "/v1/" + path
	}
	return path
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func stripHopHeaders(h http.Header) {
	hopHeaders := []string{
		"Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"TE",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	}
	for _, header := range hopHeaders {
		h.Del(header)
	}
}

func stripProxyHeaders(h http.Header) {
	proxyStripHeaders := []string{
		"Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"TE",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
		"Origin",
		"Referer",
		"Sec-Fetch-Dest",
		"Sec-Fetch-Mode",
		"Sec-Fetch-Site",
		"Sec-Fetch-User",
		"Authorization",
		"Authentication",
		"Content-Length",
	}
	for _, header := range proxyStripHeaders {
		h.Del(header)
	}
}

func applyForwardHeaders(out *http.Request, orig *http.Request) {
	clientIP := remoteIP(orig.RemoteAddr)
	if prior := orig.Header.Get("X-Forwarded-For"); prior != "" {
		if clientIP != "" {
			clientIP = prior + ", " + clientIP
		} else {
			clientIP = prior
		}
	}
	if clientIP != "" {
		out.Header.Set("X-Forwarded-For", clientIP)
	}
	if orig.Host != "" {
		out.Header.Set("X-Forwarded-Host", orig.Host)
	}
	proto := orig.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if orig.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	out.Header.Set("X-Forwarded-Proto", proto)
	if port := forwardedPort(orig); port != "" {
		out.Header.Set("X-Forwarded-Port", port)
	}
}

func remoteIP(addr string) string {
	if addr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func forwardedPort(r *http.Request) string {
	if port := r.Header.Get("X-Forwarded-Port"); port != "" {
		return port
	}
	if r.URL != nil && r.URL.Port() != "" {
		return r.URL.Port()
	}
	if r.TLS != nil {
		return "443"
	}
	return "80"
}
