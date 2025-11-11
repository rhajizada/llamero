package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/ollama/ollama/api"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/models"
	"github.com/rhajizada/llamero/internal/redisstore"
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
	now := time.Now()

	for _, def := range defs {
		if def.ID == "" || def.Address == "" {
			return fmt.Errorf("backend definition missing id or address")
		}

		desired[def.ID] = struct{}{}

		prev, found := existingByID[def.ID]
		status := prev

		if !found {
			status.Healthy = true
			status.LatencyMS = 0
			status.Models = nil
			status.ModelMeta = nil
		}

		status.ID = def.ID
		status.Address = def.Address
		status.Tags = append([]string(nil), def.Tags...)
		status.Weights = map[string]int64{
			"default": int64(def.Weight),
		}
		status.UpdatedAt = now

		if err := s.store.SaveBackend(ctx, status, 0); err != nil {
			return err
		}
	}

	for id := range existingByID {
		if _, ok := desired[id]; ok {
			continue
		}
		if err := s.store.DeleteBackend(ctx, id); err != nil {
			return err
		}
	}

	return nil
}

// CheckBackends performs a naive health update for all registered backends.
func (s *Service) CheckBackends(ctx context.Context) error {
	backends, err := s.store.ListBackends(ctx)
	if err != nil {
		return err
	}
	for _, backend := range backends {
		start := time.Now()
		modelMeta, err := s.pingBackend(ctx, backend.Address)
		backend.Healthy = err == nil
		backend.LatencyMS = time.Since(start).Milliseconds()
		if err == nil {
			backend.Models = extractModelNames(modelMeta)
			backend.ModelMeta = modelMeta
		}
		backend.UpdatedAt = time.Now()
		if saveErr := s.store.SaveBackend(ctx, backend, 0); saveErr != nil {
			return saveErr
		}
	}
	return nil
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
			ID:        status.ID,
			Address:   status.Address,
			Healthy:   status.Healthy,
			LatencyMS: status.LatencyMS,
			Tags:      append([]string(nil), status.Tags...),
			Models:    append([]string(nil), status.Models...),
			UpdatedAt: status.UpdatedAt,
		})
	}
	return backends, nil
}

func (s *Service) selectBackend(ctx context.Context, model string) (redisstore.BackendStatus, error) {
	statuses, err := s.store.ListBackends(ctx)
	if err != nil {
		return redisstore.BackendStatus{}, err
	}

	var fallback *redisstore.BackendStatus
	for _, status := range statuses {
		if !status.Healthy || status.Address == "" {
			continue
		}
		if model == "" || contains(status.Models, model) {
			return status, nil
		}
		if fallback == nil {
			copy := status
			fallback = &copy
		}
	}
	if fallback != nil {
		return *fallback, nil
	}
	return redisstore.BackendStatus{}, ErrNoHealthyBackends
}

func (s *Service) pingBackend(ctx context.Context, baseURL string) ([]redisstore.ModelInfo, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	client := api.NewClient(parsed, http.DefaultClient)
	return fetchInstalledModels(ctx, client)
}

func fetchInstalledModels(ctx context.Context, client *api.Client) ([]redisstore.ModelInfo, error) {
	metaMap := make(map[string]redisstore.ModelInfo)

	listCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	resp, err := client.List(listCtx)
	if err != nil {
		return nil, err
	}
	addListModels(metaMap, resp.Models)

	runningCtx, cancelRunning := context.WithTimeout(ctx, 5*time.Second)
	defer cancelRunning()
	if running, err := client.ListRunning(runningCtx); err == nil {
		addProcessModels(metaMap, running.Models)
	}

	if len(metaMap) == 0 {
		return nil, nil
	}
	out := make([]redisstore.ModelInfo, 0, len(metaMap))
	for _, info := range metaMap {
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out, nil
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
			OwnedBy:   "library",
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
		return "library"
	}
	return remoteHost
}

func extractModelNames(models []redisstore.ModelInfo) []string {
	if len(models) == 0 {
		return nil
	}
	names := make([]string, 0, len(models))
	for _, m := range models {
		names = append(names, m.Name)
	}
	return names
}

func contains(values []string, target string) bool {
	if target == "" || len(values) == 0 {
		return false
	}
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
