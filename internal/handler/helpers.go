package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func getString(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}

func collectStrings(value any) []string {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		var out []string
		for _, item := range v {
			if s := getString(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if v == "" {
			return nil
		}
		if strings.Contains(v, ",") {
			parts := strings.Split(v, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			return dedupeStrings(parts)
		}
		return []string{strings.TrimSpace(v)}
	default:
		return nil
	}
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	var out []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func nullableString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
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
