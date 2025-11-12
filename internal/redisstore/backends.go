package redisstore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	backendHashKey    = "backend:meta:%s"
	backendStatusSet  = "backend:status"
	backendModelsHash = "backend:models:%s"
)

// BackendStatus represents cached information about an Ollama backend.
type BackendStatus struct {
	ID           string           `json:"id"`
	Address      string           `json:"address"`
	Healthy      bool             `json:"healthy"`
	LatencyMS    int64            `json:"latency_ms"`
	Tags         []string         `json:"tags"`
	Models       []string         `json:"models"`
	LoadedModels []string         `json:"loaded_models"`
	ModelMeta    []ModelInfo      `json:"model_meta"`
	Weights      map[string]int64 `json:"weights"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// ModelInfo stores metadata about a single model.
type ModelInfo struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	OwnedBy   string    `json:"owned_by"`
}

// SaveBackend stores backend metadata and health score.
func (s *Store) SaveBackend(ctx context.Context, status BackendStatus, score float64) error {
	key := fmt.Sprintf(backendHashKey, status.ID)
	fields := map[string]any{
		"address":       status.Address,
		"healthy":       boolAsInt(status.Healthy),
		"latency_ms":    status.LatencyMS,
		"tags":          encodeStringSlice(status.Tags),
		"models":        encodeStringSlice(status.Models),
		"loaded_models": encodeStringSlice(status.LoadedModels),
		"models_meta":   encodeModelMeta(status.ModelMeta),
		"updated_at":    status.UpdatedAt.Unix(),
	}
	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, key, fields)
	pipe.ZAdd(ctx, backendStatusSet, redis.Z{Score: score, Member: status.ID})
	_, err := pipe.Exec(ctx)
	return err
}

// DeleteBackend removes backend metadata.
func (s *Store) DeleteBackend(ctx context.Context, id string) error {
	key := fmt.Sprintf(backendHashKey, id)
	pipe := s.client.TxPipeline()
	pipe.Del(ctx, key)
	pipe.Del(ctx, fmt.Sprintf(backendModelsHash, id))
	pipe.ZRem(ctx, backendStatusSet, id)
	_, err := pipe.Exec(ctx)
	return err
}

// ListBackendIDs returns backend IDs sorted by score.
func (s *Store) ListBackendIDs(ctx context.Context, start, stop int64) ([]string, error) {
	return s.client.ZRange(ctx, backendStatusSet, start, stop).Result()
}

// ListBackends loads full status for every backend.
func (s *Store) ListBackends(ctx context.Context) ([]BackendStatus, error) {
	ids, err := s.ListBackendIDs(ctx, 0, -1)
	if err != nil {
		return nil, err
	}
	statuses := make([]BackendStatus, 0, len(ids))
	for _, id := range ids {
		key := fmt.Sprintf(backendHashKey, id)
		values, err := s.client.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		status := BackendStatus{
			ID: id,
		}
		if addr, ok := values["address"]; ok {
			status.Address = addr
		}
		if healthy, ok := values["healthy"]; ok && healthy == "1" {
			status.Healthy = true
		}
		if latency, ok := values["latency_ms"]; ok {
			fmt.Sscan(latency, &status.LatencyMS)
		}
		if tags, ok := values["tags"]; ok && tags != "" {
			status.Tags = decodeStringSlice(tags)
		}
		if models, ok := values["models"]; ok && models != "" {
			status.Models = decodeStringSlice(models)
		} else if legacy, ok := values["available_models"]; ok && legacy != "" {
			status.Models = decodeStringSlice(legacy)
		}
		if loaded, ok := values["loaded_models"]; ok && loaded != "" {
			status.LoadedModels = decodeStringSlice(loaded)
		}
		if rawMeta, ok := values["models_meta"]; ok && rawMeta != "" {
			if meta, err := decodeModelMeta(rawMeta); err == nil {
				status.ModelMeta = meta
			}
		}
		if updated, ok := values["updated_at"]; ok {
			if ts, err := parseUnix(updated); err == nil {
				status.UpdatedAt = ts
			}
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

// GetBackend loads a single backend status by ID.
func (s *Store) GetBackend(ctx context.Context, id string) (BackendStatus, error) {
	key := fmt.Sprintf(backendHashKey, id)
	values, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return BackendStatus{}, err
	}
	if len(values) == 0 {
		return BackendStatus{}, nil
	}
	status := BackendStatus{ID: id}
	if addr, ok := values["address"]; ok {
		status.Address = addr
	}
	if healthy, ok := values["healthy"]; ok && healthy == "1" {
		status.Healthy = true
	}
	if latency, ok := values["latency_ms"]; ok {
		fmt.Sscan(latency, &status.LatencyMS)
	}
	if tags, ok := values["tags"]; ok && tags != "" {
		status.Tags = decodeStringSlice(tags)
	}
	if models, ok := values["models"]; ok && models != "" {
		status.Models = decodeStringSlice(models)
	} else if legacy, ok := values["available_models"]; ok && legacy != "" {
		status.Models = decodeStringSlice(legacy)
	}
	if loaded, ok := values["loaded_models"]; ok && loaded != "" {
		status.LoadedModels = decodeStringSlice(loaded)
	}
	if rawMeta, ok := values["models_meta"]; ok && rawMeta != "" {
		if meta, err := decodeModelMeta(rawMeta); err == nil {
			status.ModelMeta = meta
		}
	}
	if updated, ok := values["updated_at"]; ok {
		if ts, err := parseUnix(updated); err == nil {
			status.UpdatedAt = ts
		}
	}
	return status, nil
}

func parseUnix(raw string) (time.Time, error) {
	var unixTS int64
	if _, err := fmt.Sscan(raw, &unixTS); err != nil {
		return time.Time{}, err
	}
	return time.Unix(unixTS, 0), nil
}

func boolAsInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func encodeStringSlice(values []string) string {
	return strings.Join(values, ",")
}

func decodeStringSlice(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func encodeModelMeta(meta []ModelInfo) string {
	if len(meta) == 0 {
		return ""
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return ""
	}
	return string(data)
}

func decodeModelMeta(raw string) ([]ModelInfo, error) {
	var meta []ModelInfo
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, err
	}
	return meta, nil
}
