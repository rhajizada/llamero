package models

import (
	"time"

	ollamaapi "github.com/ollama/ollama/api"
)

// Backend represents backend metadata returned by the admin API.
type Backend struct {
	ID        string    `json:"id"`
	Address   string    `json:"address"`
	Healthy   bool      `json:"healthy"`
	LatencyMS int64     `json:"latency_ms"`
	Tags      []string  `json:"tags"`
	Models    []string  `json:"models"`
	UpdatedAt time.Time `json:"updated_at"`
} // @name Backend

// BackendProcessDetails re-exports Ollama's model metadata for docs.
type BackendProcessDetails = ollamaapi.ModelDetails // @name BackendProcessDetails

// BackendProcess re-exports the Ollama process model schema for docs.
type BackendProcess = ollamaapi.ProcessModelResponse // @name BackendProcess

// BackendProcessList re-exports the Ollama process list schema for docs.
type BackendProcessList = ollamaapi.ProcessResponse // @name BackendProcessList
