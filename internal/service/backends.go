package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ollama/ollama/api"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/models"
	"github.com/rhajizada/llamero/internal/redisstore"
)

// RegisterBackends seeds Redis with backend definitions.
func (s *Service) RegisterBackends(ctx context.Context, defs []config.BackendDefinition) error {
	for _, def := range defs {
		if def.ID == "" || def.Address == "" {
			return fmt.Errorf("backend definition missing id or address")
		}
		status := redisstore.BackendStatus{
			ID:        def.ID,
			Address:   def.Address,
			Healthy:   true,
			LatencyMS: 0,
			Tags:      def.Tags,
			Models:    nil,
			Weights: map[string]int64{
				"default": int64(def.Weight),
			},
			UpdatedAt: time.Now(),
		}
		if err := s.store.SaveBackend(ctx, status, 0); err != nil {
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
		models, err := s.pingBackend(ctx, backend.Address)
		backend.Healthy = err == nil
		backend.LatencyMS = time.Since(start).Milliseconds()
		if err == nil {
			backend.Models = models
		}
		backend.UpdatedAt = time.Now()
		if saveErr := s.store.SaveBackend(ctx, backend, 0); saveErr != nil {
			return saveErr
		}
	}
	return nil
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

func (s *Service) pingBackend(ctx context.Context, baseURL string) ([]string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	client := api.NewClient(parsed, http.DefaultClient)
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	resp, err := client.List(reqCtx)
	if err != nil {
		return nil, err
	}
	models := make([]string, 0, len(resp.Models))
	for _, m := range resp.Models {
		models = append(models, m.Name)
	}
	return models, nil
}
