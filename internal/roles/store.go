package roles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultPath defines where the server looks for role mappings if no override is supplied.
const DefaultPath = "config/roles.yaml"

// Store provides lookup helpers for role mappings and scopes.
type Store struct {
	defaultRole string
	roleScopes  map[string][]string
	groupIndex  map[string]string
}

type document struct {
	DefaultRole string      `yaml:"default_role"`
	Roles       []RoleEntry `yaml:"roles"`
}

// RoleEntry defines a single role to scope mapping.
type RoleEntry struct {
	Name   string   `yaml:"name"`
	Scopes []string `yaml:"scopes"`
}

// Load reads the YAML document from disk and builds a Store.
func Load(path string, groupMap map[string][]string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultPath
	}

	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read roles file: %w", err)
	}

	var doc document
	if err = yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse roles file: %w", err)
	}

	if len(doc.Roles) == 0 {
		return nil, fmt.Errorf("roles file %s defines no roles", path)
	}
	if strings.TrimSpace(doc.DefaultRole) == "" {
		return nil, fmt.Errorf("roles file %s must set default_role", path)
	}

	roleScopes := make(map[string][]string, len(doc.Roles))
	groupIndex := make(map[string]string)

	for _, entry := range doc.Roles {
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			return nil, fmt.Errorf("roles file %s contains an entry without name", path)
		}
		if len(entry.Scopes) == 0 {
			return nil, fmt.Errorf("role %q must define at least one scope", name)
		}
		roleScopes[name] = dedupe(entry.Scopes)
	}

	if _, ok := roleScopes[doc.DefaultRole]; !ok {
		return nil, fmt.Errorf("default_role %q does not reference a defined role", doc.DefaultRole)
	}

	for role, groups := range groupMap {
		if _, ok := roleScopes[role]; !ok {
			return nil, fmt.Errorf("role group mapping references undefined role %q", role)
		}
		for _, group := range groups {
			group = strings.TrimSpace(group)
			if group == "" {
				continue
			}
			groupIndex[group] = role
		}
	}

	return &Store{
		defaultRole: doc.DefaultRole,
		roleScopes:  roleScopes,
		groupIndex:  groupIndex,
	}, nil
}

// Resolve returns the internal role name and scopes for the provided external groups.
func (s *Store) Resolve(externals []string) (string, []string, bool) {
	for _, value := range externals {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if roleName, ok := s.groupIndex[value]; ok {
			return roleName, s.roleScopes[roleName], true
		}
	}

	return "", nil, false
}

// Default returns the default role name and scopes.
func (s *Store) Default() (string, []string) {
	return s.defaultRole, s.roleScopes[s.defaultRole]
}

func dedupe(values []string) []string {
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
