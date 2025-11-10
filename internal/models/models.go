package models

import (
	"time"

	"github.com/rhajizada/llamero/internal/repository"
)

// User represents a persisted user.
// @name User
type User = repository.User

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
