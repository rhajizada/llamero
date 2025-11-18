package models

import (
	"time"

	ollamaapi "github.com/ollama/ollama/api"
)

// Backend represents backend metadata returned by the admin API.
type Backend struct {
	ID           string    `json:"id"`
	Address      string    `json:"address"`
	Healthy      bool      `json:"healthy"`
	LatencyMS    int64     `json:"latency_ms"`
	Tags         []string  `json:"tags"`
	Models       []string  `json:"models"`        // Installed models available on disk.
	LoadedModels []string  `json:"loaded_models"` // Models currently running in Ollama.
	UpdatedAt    time.Time `json:"updated_at"`
} // @name Backend

type ProcessResponse = []ProcessModelResponse // @name ProcessResponse

type ProcessModelResponse struct {
	Name          string       `json:"name"`
	Model         string       `json:"model"`
	Size          int64        `json:"size"`
	Digest        string       `json:"digest"`
	Details       ModelDetails `json:"details"`
	ExpiresAt     time.Time    `json:"expires_at"`
	SizeVRAM      int64        `json:"size_vram"`
	ContextLength int          `json:"context_length"`
} // @name ProcessModelResponse

type (
	ModelDetails = ollamaapi.ModelDetails // @name ModelDetails
)

type BackendTagsResponse struct {
	Models []OllamaTag `json:"models,omitempty"`
} // @name BackendTagsResponse

type OllamaTag struct {
	Name       string `json:"name,omitempty"`
	Size       int64  `json:"size,omitempty"`
	Digest     string `json:"digest,omitempty"`
	ModifiedAt string `json:"modified_at,omitempty"`
} // @name OllamaTag
