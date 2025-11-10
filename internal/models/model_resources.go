package models

// Model represents an OpenAI-compatible model resource.
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
} // @name Model

// ModelList is the response envelope for listing models.
type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
} // @name ModelList
