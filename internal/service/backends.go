package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/ollama/ollama/api"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/models"
	"github.com/rhajizada/llamero/internal/redisstore"
)

const (
	defaultModelOwner     = "library"
	backendRequestTimeout = 5 * time.Second
)

// RegisterBackends seeds Redis with backend definitions.
func (s *Service) RegisterBackends(ctx context.Context, defs []config.BackendDefinition) error {
	existing, err := s.store.ListBackends(ctx)
	if err != nil {
		return err
	}

	existingByID := make(map[string]redisstore.BackendStatus, len(existing))
	for _, backend := range existing {
		existingByID[backend.ID] = backend
	}

	desired := make(map[string]struct{}, len(defs))
	seenIDs := make(map[string]struct{}, len(defs))
	seenAddresses := make(map[string]string, len(defs))
	now := time.Now()

	for _, def := range defs {
		id := strings.TrimSpace(def.ID)
		addr := strings.TrimSpace(def.Address)
		if id == "" || addr == "" {
			return errors.New("backend definition missing id or address")
		}
		if _, exists := seenIDs[id]; exists {
			return fmt.Errorf("duplicate backend id %q", id)
		}
		seenIDs[id] = struct{}{}
		if existingID, exists := seenAddresses[addr]; exists {
			return fmt.Errorf("backend address %q reused by %s and %s", addr, existingID, id)
		}
		seenAddresses[addr] = id

		desired[id] = struct{}{}

		prev, found := existingByID[id]
		status := prev

		if !found {
			status.Healthy = true
			status.LatencyMS = 0
			status.Models = nil
			status.LoadedModels = nil
			status.ModelMeta = nil
		}

		status.ID = id
		status.Address = addr
		status.Tags = append([]string(nil), def.Tags...)
		status.Weights = map[string]int64{
			"default": int64(def.Weight),
		}
		status.UpdatedAt = now

		if err = s.store.SaveBackend(ctx, status, 0); err != nil {
			return err
		}
	}

	for id := range existingByID {
		if _, ok := desired[id]; ok {
			continue
		}
		if err = s.store.DeleteBackend(ctx, id); err != nil {
			return err
		}
	}

	return nil
}

// SyncBackends refreshes health and model metadata for every backend.
func (s *Service) SyncBackends(ctx context.Context) error {
	backends, err := s.store.ListBackends(ctx)
	if err != nil {
		return err
	}
	for _, backend := range backends {
		if err = s.syncBackend(ctx, backend); err != nil {
			return err
		}
	}
	return nil
}

// SyncBackendByID refreshes health and model metadata for a specific backend.
func (s *Service) SyncBackendByID(ctx context.Context, backendID string) error {
	backendID = strings.TrimSpace(backendID)
	if backendID == "" {
		return errors.New("backend id is required")
	}
	status, err := s.store.GetBackend(ctx, backendID)
	if err != nil {
		return err
	}
	if status.ID == "" {
		return fmt.Errorf("backend %q not found", backendID)
	}
	return s.syncBackend(ctx, status)
}

// ErrNoHealthyBackends indicates that no registered backend can serve a request.
var ErrNoHealthyBackends = errors.New("no healthy backends available")

// BackendRoute contains the minimum backend data required for proxying.
type BackendRoute struct {
	ID      string
	Address string
}

// LookupBackendRoute fetches backend connection details by identifier.
func (s *Service) LookupBackendRoute(ctx context.Context, backendID string) (BackendRoute, error) {
	backendID = strings.TrimSpace(backendID)
	if backendID == "" {
		return BackendRoute{}, &Error{
			Code:    http.StatusNotFound,
			Message: "backend not found",
		}
	}
	statuses, err := s.store.ListBackends(ctx)
	if err != nil {
		return BackendRoute{}, err
	}
	for _, status := range statuses {
		if status.ID != backendID {
			continue
		}
		if strings.TrimSpace(status.Address) == "" {
			return BackendRoute{}, &Error{
				Code:    http.StatusBadGateway,
				Message: "backend missing address",
			}
		}
		return BackendRoute{
			ID:      status.ID,
			Address: status.Address,
		}, nil
	}
	return BackendRoute{}, &Error{
		Code:    http.StatusNotFound,
		Message: "backend not found",
	}
}

// RouteBackend selects a healthy backend for a given model.
func (s *Service) RouteBackend(ctx context.Context, model string) (BackendRoute, error) {
	status, err := s.selectBackend(ctx, model)
	if err != nil {
		return BackendRoute{}, err
	}
	return BackendRoute{
		ID:      status.ID,
		Address: status.Address,
	}, nil
}

// ListBackends returns all backend statuses from Redis.
func (s *Service) ListBackends(ctx context.Context) ([]models.Backend, error) {
	statuses, err := s.store.ListBackends(ctx)
	if err != nil {
		return nil, err
	}
	backends := make([]models.Backend, 0, len(statuses))
	for _, status := range statuses {
		backends = append(backends, models.Backend{
			ID:           status.ID,
			Address:      status.Address,
			Healthy:      status.Healthy,
			LatencyMS:    status.LatencyMS,
			Tags:         append([]string(nil), status.Tags...),
			Models:       append([]string(nil), status.Models...),
			LoadedModels: append([]string(nil), status.LoadedModels...),
			UpdatedAt:    status.UpdatedAt,
		})
	}
	return backends, nil
}

func (s *Service) selectBackend(ctx context.Context, model string) (redisstore.BackendStatus, error) {
	statuses, err := s.store.ListBackends(ctx)
	if err != nil {
		return redisstore.BackendStatus{}, err
	}

	var (
		healthy    []redisstore.BackendStatus
		loadedHits []redisstore.BackendStatus
		modelHits  []redisstore.BackendStatus
	)

	for _, status := range statuses {
		if !status.Healthy || strings.TrimSpace(status.Address) == "" {
			continue
		}
		healthy = append(healthy, status)
		if model == "" {
			continue
		}
		if contains(status.LoadedModels, model) {
			loadedHits = append(loadedHits, status)
			continue
		}
		if contains(status.Models, model) {
			modelHits = append(modelHits, status)
		}
	}

	if len(healthy) == 0 {
		return redisstore.BackendStatus{}, ErrNoHealthyBackends
	}
	if model == "" {
		return healthy[0], nil
	}
	if len(loadedHits) > 0 {
		return loadedHits[0], nil
	}
	if len(modelHits) > 0 {
		return modelHits[0], nil
	}
	return healthy[0], nil
}

func (s *Service) pingBackend(ctx context.Context, baseURL string) ([]redisstore.ModelInfo, []string, []string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, nil, nil, err
	}
	client := api.NewClient(parsed, http.DefaultClient)
	return fetchInstalledModels(ctx, client)
}

func fetchInstalledModels(ctx context.Context, client *api.Client) ([]redisstore.ModelInfo, []string, []string, error) {
	metaMap := make(map[string]redisstore.ModelInfo)
	var available []string
	var loaded []string

	listCtx, cancel := context.WithTimeout(ctx, backendRequestTimeout)
	defer cancel()
	resp, err := client.List(listCtx)
	if err != nil {
		return nil, nil, nil, err
	}
	addListModels(metaMap, resp.Models)
	for _, model := range resp.Models {
		name := firstModelName(model.Name, model.Model)
		if name == "" || contains(available, name) {
			continue
		}
		available = append(available, name)
	}

	runningCtx, cancelRunning := context.WithTimeout(ctx, backendRequestTimeout)
	defer cancelRunning()
	if running, listErr := client.ListRunning(runningCtx); listErr == nil {
		addProcessModels(metaMap, running.Models)
		for _, model := range running.Models {
			name := firstModelName(model.Name, model.Model)
			if name == "" || contains(loaded, name) {
				continue
			}
			loaded = append(loaded, name)
		}
	}

	if len(metaMap) == 0 {
		return nil, available, loaded, nil
	}
	out := make([]redisstore.ModelInfo, 0, len(metaMap))
	for _, info := range metaMap {
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	sort.Strings(available)
	sort.Strings(loaded)
	return out, available, loaded, nil
}

func addListModels(registry map[string]redisstore.ModelInfo, models []api.ListModelResponse) {
	for _, model := range models {
		name := firstModelName(model.Name, model.Model)
		if name == "" {
			continue
		}
		created := model.ModifiedAt
		if created.IsZero() {
			created = time.Now()
		}
		registry[name] = redisstore.ModelInfo{
			Name:      name,
			CreatedAt: created,
			OwnedBy:   inferOwner(model.RemoteHost),
		}
	}
}

func addProcessModels(registry map[string]redisstore.ModelInfo, models []api.ProcessModelResponse) {
	for _, model := range models {
		name := firstModelName(model.Name, model.Model)
		if name == "" {
			continue
		}
		if _, exists := registry[name]; exists {
			continue
		}
		created := model.ExpiresAt
		if created.IsZero() {
			created = time.Now()
		}
		registry[name] = redisstore.ModelInfo{
			Name:      name,
			CreatedAt: created,
			OwnedBy:   defaultModelOwner,
		}
	}
}

func firstModelName(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func inferOwner(remoteHost string) string {
	if strings.TrimSpace(remoteHost) == "" {
		return defaultModelOwner
	}
	return remoteHost
}

func contains(values []string, target string) bool {
	if target == "" || len(values) == 0 {
		return false
	}
	return slices.Contains(values, target)
}

func (s *Service) syncBackend(ctx context.Context, backend redisstore.BackendStatus) error {
	if strings.TrimSpace(backend.Address) == "" {
		return fmt.Errorf("backend %q missing address", backend.ID)
	}
	start := time.Now()
	modelMeta, available, loaded, err := s.pingBackend(ctx, backend.Address)
	backend.Healthy = err == nil
	backend.LatencyMS = time.Since(start).Milliseconds()
	if err == nil {
		backend.Models = available
		backend.LoadedModels = loaded
		backend.ModelMeta = modelMeta
	}
	backend.UpdatedAt = time.Now()
	return s.store.SaveBackend(ctx, backend, 0)
}
