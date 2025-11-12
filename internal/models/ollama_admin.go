package models

import "time"

// BackendCreateModelRequest matches Ollama's create API payload.
type BackendCreateModelRequest struct {
	Model     string `json:"model"`               // Name of the model to create.
	Modelfile string `json:"modelfile,omitempty"` // Inline Modelfile contents.
	Path      string `json:"path,omitempty"`      // Optional path to an existing Modelfile.
	Quantize  string `json:"quantize,omitempty"`  // Quantization target, e.g. "Q4_0".
	KeepAlive string `json:"keep_alive,omitempty"`
} // @name BackendCreateModelRequest

// BackendCopyModelRequest matches Ollama's copy API payload.
type BackendCopyModelRequest struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
} // @name BackendCopyModelRequest

// BackendPullModelRequest matches Ollama's pull API payload.
type BackendPullModelRequest struct {
	Model    string `json:"model"`
	Insecure bool   `json:"insecure,omitempty"`
	Stream   bool   `json:"stream,omitempty"`
} // @name BackendPullModelRequest

// BackendPushModelRequest matches Ollama's push API payload.
type BackendPushModelRequest struct {
	Model    string `json:"model"`
	Insecure bool   `json:"insecure,omitempty"`
	Stream   bool   `json:"stream,omitempty"`
} // @name BackendPushModelRequest

// BackendDeleteModelRequest matches Ollama's delete API payload.
type BackendDeleteModelRequest struct {
	Model string `json:"model"`
	Force bool   `json:"force,omitempty"`
} // @name BackendDeleteModelRequest

// BackendShowModelRequest matches Ollama's show API payload.
type BackendShowModelRequest struct {
	Model  string `json:"model"`
	System string `json:"system,omitempty"`
} // @name BackendShowModelRequest

// BackendShowModelResponse captures the important fields from Ollama's show response.
type BackendShowModelResponse struct {
	Model      string                  `json:"model"`
	License    string                  `json:"license,omitempty"`
	Modelfile  string                  `json:"modelfile,omitempty"`
	Template   string                  `json:"template,omitempty"`
	Parameters map[string]any          `json:"parameters,omitempty"`
	Details    BackendShowModelDetails `json:"details"`
	ModifiedAt time.Time               `json:"modified_at"`
} // @name BackendShowModelResponse

// BackendShowModelDetails describes metadata extracted from the model.
type BackendShowModelDetails struct {
	Family            string `json:"family,omitempty"`
	ParameterSize     string `json:"parameter_size,omitempty"`
	QuantizationLevel string `json:"quantization_level,omitempty"`
} // @name BackendShowModelDetails

// BackendOperationResponse represents the streaming status envelopes returned by Ollama's admin APIs.
type BackendOperationResponse struct {
	Status string `json:"status"`
	Model  string `json:"model,omitempty"`
	Digest string `json:"digest,omitempty"`
	Detail string `json:"detail,omitempty"`
} // @name BackendOperationResponse

// BackendVersionResponse mirrors Ollama's version endpoint.
type BackendVersionResponse struct {
	Version string `json:"version"`
} // @name BackendVersionResponse
